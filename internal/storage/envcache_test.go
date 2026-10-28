package storage

import (
	"strconv"
	"testing"

	"github.com/jhillyerd/enmime/v2"
)

func imageEnvelope(contentSize int) *enmime.Envelope {
	return &enmime.Envelope{
		Inlines: []*enmime.Part{{PartID: "1", ContentType: "image/png", Content: make([]byte, contentSize)}},
	}
}

func cachedEnvelope(id string) (*envelopeCacheEntry, bool) {
	envelopeCache.Lock()
	defer envelopeCache.Unlock()
	e, ok := envelopeCache.entries[id]

	return e, ok
}

func storeEnvelope(id string, env *enmime.Envelope) {
	envelopeCache.Lock()
	storeEnvelopeLocked(id, env, 1024)
	envelopeCache.Unlock()
}

func TestEnvelopeCacheOnlyStoresMessagesWithImages(t *testing.T) {
	invalidateAllEnvelopeCache()
	defer invalidateAllEnvelopeCache()

	storeEnvelope("plain", &enmime.Envelope{Text: "hello"})
	storeEnvelope("pdf", &enmime.Envelope{
		Attachments: []*enmime.Part{{PartID: "1", ContentType: "application/pdf", Content: []byte("pdf")}},
	})
	storeEnvelope("inline-image", imageEnvelope(10))
	storeEnvelope("attached-image", &enmime.Envelope{
		Attachments: []*enmime.Part{{PartID: "1", ContentType: "image/jpeg", Content: []byte("jpg")}},
	})

	for id, want := range map[string]bool{
		"plain":          false,
		"pdf":            false,
		"inline-image":   true,
		"attached-image": true,
	} {
		if _, ok := cachedEnvelope(id); ok != want {
			t.Errorf("%s: cached = %v, want %v", id, ok, want)
		}
	}
}

func TestEnvelopeCacheIsBounded(t *testing.T) {
	invalidateAllEnvelopeCache()
	defer invalidateAllEnvelopeCache()

	const size = 1 << 20 // 1 MiB
	const capacity = envelopeCacheMaxBytes / size

	for i := range capacity * 2 {
		storeEnvelope(strconv.Itoa(i), imageEnvelope(size))
	}

	envelopeCache.Lock()
	total := envelopeCache.bytes
	var sum uint64
	for _, e := range envelopeCache.entries {
		sum += e.size
	}
	count := len(envelopeCache.entries)
	envelopeCache.Unlock()

	if total > envelopeCacheMaxBytes {
		t.Errorf("cache holds %d bytes, limit is %d", total, envelopeCacheMaxBytes)
	}
	if total != sum {
		t.Errorf("byte accounting drifted: tracked %d, actual %d", total, sum)
	}
	if count != capacity {
		t.Errorf("expected %d entries, got %d", capacity, count)
	}

	// the oldest entries are evicted, the newest are kept
	if _, ok := cachedEnvelope("0"); ok {
		t.Error("oldest entry should have been evicted")
	}
	if _, ok := cachedEnvelope(strconv.Itoa(capacity*2 - 1)); !ok {
		t.Error("newest entry should be cached")
	}

	// control: invalidating returns the accounting to zero
	invalidateAllEnvelopeCache()
	if envelopeCache.bytes != 0 {
		t.Errorf("expected 0 bytes after invalidation, got %d", envelopeCache.bytes)
	}
}

func TestEnvelopeCacheKeepsOversizedNewestEntry(t *testing.T) {
	invalidateAllEnvelopeCache()
	defer invalidateAllEnvelopeCache()

	storeEnvelope("small", imageEnvelope(1<<20))
	storeEnvelope("huge", imageEnvelope(envelopeCacheMaxBytes+1))

	if _, ok := cachedEnvelope("huge"); !ok {
		t.Error("an oversized message should still be cached so its parts are not re-parsed")
	}
	if _, ok := cachedEnvelope("small"); ok {
		t.Error("older entries should be evicted to make room for the oversized message")
	}

	// a following message evicts the oversized one rather than growing the cache
	storeEnvelope("next", imageEnvelope(1<<20))
	if _, ok := cachedEnvelope("huge"); ok {
		t.Error("oversized entry should be evicted by the next message")
	}
	if _, ok := cachedEnvelope("next"); !ok {
		t.Error("newest entry should be cached")
	}
}
