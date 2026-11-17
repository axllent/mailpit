package rspamd

import (
	"context"
	"sync"
	"time"
)

const (
	// maxConcurrentChecks is how many messages are sent to Rspamd at once. Every check holds the
	// whole message in memory (up to the maximum message size) while it is sent, so unlimited
	// concurrent requests could use a lot of memory and overload the Rspamd server.
	maxConcurrentChecks = 4

	// maxCachedResults is the most results kept.
	maxCachedResults = 1000
)

var (
	// resultCacheTTL is how long a result is kept, a variable so that the tests can shorten it.
	// A message never changes, so the result is only out of date if Rspamd itself has changed.
	resultCacheTTL = 60 * time.Second

	checkSlots = make(chan struct{}, maxConcurrentChecks)

	cacheMu sync.Mutex
	cache   = make(map[string]cachedResult)
)

type cachedResult struct {
	result  Result
	expires time.Time
}

// CheckMessage returns the Rspamd result for the message with the given ID. The message itself is
// only loaded, and sent to Rspamd, if there is no recent result, and no more than
// maxConcurrentChecks are done at once, any others wait their turn (or for ctx to be done).
// Only results without an error are kept, so a failed check is tried again next time.
func CheckMessage(ctx context.Context, id string, load func() ([]byte, error)) (Result, error) {
	if r, ok := cachedResultFor(id); ok {
		return r, nil
	}

	select {
	case checkSlots <- struct{}{}:
		defer func() { <-checkSlots }()
	case <-ctx.Done():
		return Result{}, ctx.Err()
	}

	// the same message may have been checked while this request waited
	if r, ok := cachedResultFor(id); ok {
		return r, nil
	}

	raw, err := load()
	if err != nil {
		return Result{}, err
	}

	r, err := Check(raw)
	if err == nil && r.Error == "" {
		storeResult(id, r)
	}

	return r, err
}

func cachedResultFor(id string) (Result, bool) {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	c, ok := cache[id]
	if !ok {
		return Result{}, false
	}
	if time.Now().After(c.expires) {
		delete(cache, id)
		return Result{}, false
	}

	return c.result, true
}

func storeResult(id string, r Result) {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	if len(cache) >= maxCachedResults {
		now := time.Now()
		for k, v := range cache {
			if now.After(v.expires) {
				delete(cache, k)
			}
		}
		if len(cache) >= maxCachedResults {
			return // full of recent results, so this one is not kept
		}
	}

	cache[id] = cachedResult{result: r, expires: time.Now().Add(resultCacheTTL)}
}
