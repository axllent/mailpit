package linkcheck

import (
	"container/list"
	"context"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/time/rate"
)

// Per-domain rate-limiter parameters. The bucket starts full so a single
// fresh check of a 100-link newsletter completes without waiting; refill
// caps sustained traffic to any one registered domain at 1 req/s across
// all concurrent API calls.
const (
	perDomainBurst       = 100
	perDomainRefill      = rate.Limit(1)
	perDomainConcurrency = 2

	// limiterRegistryCap bounds memory regardless of attacker effort.
	// Eviction prefers buckets at full capacity (safe to drop).
	limiterRegistryCap = 10000

	// resultCacheTTL deduplicates repeated checks of the same URL so a
	// user retesting the same email doesn't drain the rate limiter twice
	// and an attacker can't multiply outbound load by looping the API.
	resultCacheTTL = 60 * time.Second
)

type domainState struct {
	limiter *rate.Limiter
	sem     chan struct{}
	lruElem *list.Element
}

type registry struct {
	mu      sync.Mutex
	entries map[string]*domainState
	lru     *list.List // front = most recently used
}

func newRegistry() *registry {
	return &registry{
		entries: make(map[string]*domainState),
		lru:     list.New(),
	}
}

// get returns the state for a registered domain, creating it on demand.
// When the registry is at capacity, prefers to evict entries whose bucket
// is at full capacity (no security cost — recreating yields identical state).
func (r *registry) get(domain string) *domainState {
	r.mu.Lock()
	defer r.mu.Unlock()

	if st, ok := r.entries[domain]; ok {
		r.lru.MoveToFront(st.lruElem)
		return st
	}

	if len(r.entries) >= limiterRegistryCap {
		r.evictLocked()
	}

	st := &domainState{
		limiter: rate.NewLimiter(perDomainRefill, perDomainBurst),
		sem:     make(chan struct{}, perDomainConcurrency),
	}
	st.lruElem = r.lru.PushFront(domainKey{domain: domain, state: st})
	r.entries[domain] = st

	return st
}

type domainKey struct {
	domain string
	state  *domainState
}

// evictLocked drops one entry. Caller must hold r.mu.
// Walks the LRU from the back looking for a full bucket; if none, drops the LRU.
func (r *registry) evictLocked() {
	for e := r.lru.Back(); e != nil; e = e.Prev() {
		k := e.Value.(domainKey)
		if k.state.limiter.Tokens() >= float64(perDomainBurst) {
			r.lru.Remove(e)
			delete(r.entries, k.domain)
			return
		}
	}

	e := r.lru.Back()
	if e == nil {
		return
	}

	k := e.Value.(domainKey)
	r.lru.Remove(e)
	delete(r.entries, k.domain)
}

type cachedResult struct {
	link    Link
	expires time.Time
}

type resultCache struct {
	mu      sync.Mutex
	entries map[string]cachedResult
}

func newResultCache() *resultCache {
	return &resultCache{entries: make(map[string]cachedResult)}
}

func (c *resultCache) get(u string) (Link, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[u]
	if !ok {
		return Link{}, false
	}
	if time.Now().After(e.expires) {
		delete(c.entries, u)
		return Link{}, false
	}

	return e.link, true
}

func (c *resultCache) put(u string, l Link) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[u] = cachedResult{link: l, expires: time.Now().Add(resultCacheTTL)}
	// Opportunistic sweep: when the cache grows past a threshold,
	// drop expired entries. Avoids unbounded growth without a goroutine.
	if len(c.entries) > 2*limiterRegistryCap {
		now := time.Now()
		for k, v := range c.entries {
			if now.After(v.expires) {
				delete(c.entries, k)
			}
		}
	}
}

var (
	domainRegistry = newRegistry()
	linkCache      = newResultCache()
)

// registeredDomain returns the eTLD+1 for a URL's host, or the lowercased
// host if no registered domain can be determined (e.g. IP literals).
// Subdomains share the same key so wildcard-DNS bypass is closed.
func registeredDomain(rawurl string) string {
	u, err := url.Parse(rawurl)
	if err != nil {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return ""
	}
	d, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		return host
	}

	return d
}

// acquireDomainSlot blocks until both a rate-limit token and a per-domain
// concurrency slot are available, or ctx is cancelled. Returns a release
// function that must be called when the request completes.
func acquireDomainSlot(ctx context.Context, domain string, warned *sync.Map) (release func(), err error) {
	if config.DisableLinkCheckRateLimit {
		return func() {}, nil
	}
	st := domainRegistry.get(domain)
	if st.limiter.Tokens() < 1 {
		if _, alreadyWarned := warned.LoadOrStore(domain, struct{}{}); !alreadyWarned {
			logger.Log().Warnf("[link-check] rate limiting active for %s - use --disable-link-check-rate-limit to disable", domain)
		}
	}
	if err := st.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	select {
	case st.sem <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return func() { <-st.sem }, nil
}

// cachedLink returns a previously-checked result if still fresh.
func cachedLink(u string) (Link, bool) {
	if config.DisableLinkCheckRateLimit {
		return Link{}, false
	}
	return linkCache.get(u)
}

// storeLink caches a result so repeat checks of the same URL skip the
// rate limiter and the outbound HEAD.
func storeLink(u string, l Link) {
	if config.DisableLinkCheckRateLimit {
		return
	}

	linkCache.put(u, l)
}
