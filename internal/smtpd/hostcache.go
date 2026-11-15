package smtpd

import (
	"sync"
	"time"
)

const (
	// defaultHostCacheTTL is how long the host name of an address is remembered.
	defaultHostCacheTTL = 5 * time.Minute

	// defaultHostCacheFailureTTL is how long it is remembered that an address has no host name, or that looking it up
	// took too long. This is shorter, as a lookup which failed may well work soon.
	defaultHostCacheFailureTTL = time.Minute

	// defaultHostCacheMax is how many addresses are remembered. Addresses come from the network, so there must be a
	// limit.
	defaultHostCacheMax = 1000
)

type hostEntry struct {
	host    string
	expires time.Time
}

// hostCache remembers the result of reverse DNS lookups for a short time. A client which connects often (as a test
// suite does) is looked up once instead of for every connection, and one whose lookup takes too long does not make
// every connection wait for the timeout. The zero value is ready to use, with the defaults above.
//
// When it is full a new address is not remembered, it is just looked up each time, as if there was no cache. This is
// better than removing others to make room, as a client could then push out the entries of everyone else by connecting
// from many addresses.
type hostCache struct {
	mu         sync.Mutex
	entries    map[string]hostEntry
	ttl        time.Duration // for a name which was found, defaultHostCacheTTL if zero
	failureTTL time.Duration // for a lookup which found nothing, defaultHostCacheFailureTTL if zero
	max        int           // defaultHostCacheMax if zero
}

// get returns the host name which was remembered for the address, if it still is.
func (c *hostCache) get(ip string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.entries[ip]
	if !ok {
		return "", false
	}

	if !time.Now().Before(e.expires) {
		delete(c.entries, ip)

		return "", false
	}

	return e.host, true
}

// put remembers the host name of an address. found is false if the address has no name, or the lookup failed.
func (c *hostCache) put(ip, host string, found bool) {
	ttl, limit := c.ttl, c.max
	if found {
		if ttl <= 0 {
			ttl = defaultHostCacheTTL
		}
	} else {
		ttl = c.failureTTL
		if ttl <= 0 {
			ttl = defaultHostCacheFailureTTL
		}
	}
	if limit <= 0 {
		limit = defaultHostCacheMax
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.entries == nil {
		c.entries = make(map[string]hostEntry)
	}

	now := time.Now()

	if _, ok := c.entries[ip]; !ok && len(c.entries) >= limit {
		// make room by removing what has expired, which is at most the limit (and only done when it is full)
		for k, e := range c.entries {
			if !now.Before(e.expires) {
				delete(c.entries, k)
			}
		}

		if len(c.entries) >= limit {
			return
		}
	}

	c.entries[ip] = hostEntry{host: host, expires: now.Add(ttl)}
}
