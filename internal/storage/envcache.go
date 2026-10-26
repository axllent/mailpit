package storage

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jhillyerd/enmime/v2"
)

const (
	envelopeCacheTTL = 5 * time.Second

	// envelopeCacheMaxBytes caps the combined estimated size of cached envelopes
	// (see envelopeSize), regardless of request rate. The most recent envelope is
	// always kept, so a single message larger than this is still cached.
	envelopeCacheMaxBytes = 50 << 20
)

type envelopeCacheEntry struct {
	env     *enmime.Envelope
	rawSize uint64 // size of the raw message
	size    uint64 // estimated memory retained by env
	created time.Time
	timer   *time.Timer // removes the entry once the TTL expires
}

// envelopeCache provides a short-lived cache for parsed enmime envelopes.
// When a user opens a message with N image attachments, the browser fires
// N+1 concurrent requests (1 message + N thumbnails), each of which would
// otherwise trigger a full re-parse of the raw email. This cache ensures
// only one parse occurs per message, with a TTL of a few seconds to cover
// the burst and then free the memory.
var envelopeCache = struct {
	sync.Mutex
	entries  map[string]*envelopeCacheEntry
	inflight map[string]*envelopeInflight
	bytes    uint64 // combined size of all entries
}{
	entries:  make(map[string]*envelopeCacheEntry),
	inflight: make(map[string]*envelopeInflight),
}

type envelopeInflight struct {
	done chan struct{} // closed when parse completes
	env  *enmime.Envelope
	size uint64
	err  error
}

// getCachedEnvelope returns a parsed envelope for the given message ID,
// using the cache if available and deduplicating concurrent parses so
// only one goroutine does the actual work.
func getCachedEnvelope(id string) (*enmime.Envelope, uint64, error) {
	envelopeCache.Lock()

	// check for a valid cached entry
	if entry, ok := envelopeCache.entries[id]; ok {
		if time.Since(entry.created) < envelopeCacheTTL {
			envelopeCache.Unlock()
			return entry.env, entry.rawSize, nil
		}
		delete(envelopeCache.entries, id)
	}

	// another goroutine is already parsing this message - wait for it
	if inf, ok := envelopeCache.inflight[id]; ok {
		envelopeCache.Unlock()
		<-inf.done
		return inf.env, inf.size, inf.err
	}

	// we are the first - register as inflight and release the lock
	inf := &envelopeInflight{done: make(chan struct{})}
	envelopeCache.inflight[id] = inf
	envelopeCache.Unlock()

	// ensure waiters are always unblocked, even on panic
	defer func() {
		if r := recover(); r != nil {
			inf.err = fmt.Errorf("panic parsing message %s: %v", id, r)
		}
		close(inf.done)

		envelopeCache.Lock()
		delete(envelopeCache.inflight, id)
		if inf.err == nil {
			storeEnvelopeLocked(id, inf.env, inf.size)
		}
		envelopeCache.Unlock()
	}()

	inf.env, inf.size, inf.err = parseEnvelope(id)

	return inf.env, inf.size, inf.err
}

// removeEnvelopeLocked removes an entry from the cache and stops its timer.
// The caller must hold envelopeCache's lock.
func removeEnvelopeLocked(id string) {
	if entry, ok := envelopeCache.entries[id]; ok {
		entry.timer.Stop()
		envelopeCache.bytes -= entry.size
		delete(envelopeCache.entries, id)
	}
}

// envelopeParts returns every attachment, inline and other part of an envelope.
func envelopeParts(env *enmime.Envelope) []*enmime.Part {
	parts := make([]*enmime.Part, 0, len(env.Attachments)+len(env.Inlines)+len(env.OtherParts))
	parts = append(parts, env.Attachments...)
	parts = append(parts, env.Inlines...)

	return append(parts, env.OtherParts...)
}

// hasImagePart returns true if any attachment, inline or other part is an image.
// Only these cause the browser to fire a burst of requests for the same message
// (one per inline image or attachment thumbnail).
func hasImagePart(env *enmime.Envelope) bool {
	for _, p := range envelopeParts(env) {
		if strings.HasPrefix(p.ContentType, "image/") {
			return true
		}
	}

	return false
}

// envelopeSize estimates the memory retained by an envelope, which is dominated
// by the decoded bodies and part content.
func envelopeSize(env *enmime.Envelope) uint64 {
	size := uint64(len(env.Text) + len(env.HTML))
	for _, p := range envelopeParts(env) {
		size += uint64(len(p.Content))
	}

	return size
}

// storeEnvelopeLocked adds an entry to the cache and schedules its removal
// after the TTL, so entries are freed even if never requested again.
// Only messages with at least one image part are cached. The combined size of
// the cache is bounded by envelopeCacheMaxBytes, evicting the oldest entries
// first, but the newest entry is always kept.
// The caller must hold envelopeCache's lock.
func storeEnvelopeLocked(id string, env *enmime.Envelope, rawSize uint64) {
	if !hasImagePart(env) {
		return
	}

	removeEnvelopeLocked(id)

	size := envelopeSize(env)

	for len(envelopeCache.entries) > 0 && envelopeCache.bytes+size > envelopeCacheMaxBytes {
		var oldestID string
		var oldest time.Time
		for k, e := range envelopeCache.entries {
			if oldestID == "" || e.created.Before(oldest) {
				oldestID, oldest = k, e.created
			}
		}
		removeEnvelopeLocked(oldestID)
	}

	entry := &envelopeCacheEntry{env: env, rawSize: rawSize, size: size, created: time.Now()}
	entry.timer = time.AfterFunc(envelopeCacheTTL, func() {
		envelopeCache.Lock()
		defer envelopeCache.Unlock()
		// only remove our own entry, not a newer one stored under the same ID
		if envelopeCache.entries[id] == entry {
			removeEnvelopeLocked(id)
		}
	})
	envelopeCache.entries[id] = entry
	envelopeCache.bytes += size
}

// parseEnvelope fetches a raw message from the database and parses it with enmime.
func parseEnvelope(id string) (*enmime.Envelope, uint64, error) {
	raw, err := GetMessageRaw(id)
	if err != nil {
		return nil, 0, err
	}

	env, err := envelopeParser.ReadEnvelope(bytes.NewReader(raw))
	if err != nil {
		return nil, 0, err
	}

	return env, uint64(len(raw)), nil
}

// invalidateEnvelopeCache removes a specific message from the cache,
// e.g. when a message is deleted.
func invalidateEnvelopeCache(id string) {
	envelopeCache.Lock()
	removeEnvelopeLocked(id)
	envelopeCache.Unlock()
}

// invalidateAllEnvelopeCache clears the entire cache,
// e.g. when all messages are deleted.
func invalidateAllEnvelopeCache() {
	envelopeCache.Lock()
	for _, entry := range envelopeCache.entries {
		entry.timer.Stop()
	}
	envelopeCache.entries = make(map[string]*envelopeCacheEntry)
	envelopeCache.bytes = 0
	envelopeCache.Unlock()
}
