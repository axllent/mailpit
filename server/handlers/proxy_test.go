package handlers

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// seedAssets replaces the global assets cache with a single entry for id
// containing an empty asset set. Callers must not run other tests in parallel
// with tests that touch the assets cache.
func seedAssets(id string) {
	assetsMutex.Lock()
	assets = map[string]MessageAssets{
		id: {
			ID:      id,
			Created: time.Now(),
			Assets:  []string{},
			index:   map[string]struct{}{},
		},
	}
	assetsMutex.Unlock()
}

func snapshotAssets(id string) []string {
	assetsMutex.Lock()
	defer assetsMutex.Unlock()
	out := make([]string, len(assets[id].Assets))
	copy(out, assets[id].Assets)
	return out
}

// TestRecordAssetUniqueURLsScaleLinearly is the security regression test for
// the O(N²) tools.InArray scan previously used by the CSS rewriter. Feeding
// many unique URLs must produce O(N) work; at N = 100000 the previous
// implementation exceeded 30s on one CPU, so any wall clock above a few
// seconds here indicates the quadratic scan has come back.
func TestRecordAssetUniqueURLsScaleLinearly(t *testing.T) {
	const id = "msg-quadratic"
	const n = 100000

	seedAssets(id)

	start := time.Now()
	for i := 0; i < n; i++ {
		recordAsset(id, fmt.Sprintf("https://example.test/asset/%06d", i))
	}
	elapsed := time.Since(start)

	got := snapshotAssets(id)
	if len(got) != n {
		t.Fatalf("recorded %d unique URLs, want %d", len(got), n)
	}

	// Linear-scaling budget. 5s on any modern CI CPU is generous for
	// 100000 map inserts; the previous O(N²) InArray scan exceeded 30s.
	if elapsed > 5*time.Second {
		t.Fatalf("recording %d unique URLs took %s, want linear-scale time budget (<5s)", n, elapsed)
	}
}

// TestRecordAssetDuplicatesControl is the matched benign control for the
// quadratic-scan regression. With many repeated URLs the dedup path must keep
// only the first, and time is unrelated to the raw call count.
func TestRecordAssetDuplicatesControl(t *testing.T) {
	const id = "msg-duplicates"
	const n = 100000

	seedAssets(id)

	for i := 0; i < n; i++ {
		recordAsset(id, "https://example.test/asset/repeated")
	}

	got := snapshotAssets(id)
	if len(got) != 1 {
		t.Fatalf("recorded %d assets, want 1 after dedup", len(got))
	}
	if got[0] != "https://example.test/asset/repeated" {
		t.Fatalf("kept unexpected asset %q", got[0])
	}
}

// TestRecordAssetCaseInsensitiveDedup preserves the previous case-insensitive
// membership semantics of tools.InArray (strings.EqualFold). The map key is
// strings.ToLower(url); URLs that differ only in case must collapse to one
// entry, and the first-seen casing is kept.
func TestRecordAssetCaseInsensitiveDedup(t *testing.T) {
	const id = "msg-case"

	seedAssets(id)

	recordAsset(id, "https://EXAMPLE.test/Asset/One")
	recordAsset(id, "https://example.test/asset/one")
	recordAsset(id, "HTTPS://Example.Test/Asset/One")
	recordAsset(id, "https://example.test/asset/two")

	got := snapshotAssets(id)
	if len(got) != 2 {
		t.Fatalf("recorded %d assets, want 2 (case-insensitive dedup)", len(got))
	}
	if got[0] != "https://EXAMPLE.test/Asset/One" {
		t.Fatalf("first entry %q, want original casing preserved", got[0])
	}
	if got[1] != "https://example.test/asset/two" {
		t.Fatalf("second entry %q, want the distinct URL", got[1])
	}
}

// TestRecordAssetNoOpForUnknownMessage guards a subtle invariant: if the CSS
// rewriter is called for a message ID that is not in the cache (expired by the
// cleanup goroutine mid-request), recordAsset must be a no-op rather than
// creating a fresh index-less entry.
func TestRecordAssetNoOpForUnknownMessage(t *testing.T) {
	assetsMutex.Lock()
	assets = map[string]MessageAssets{}
	assetsMutex.Unlock()

	recordAsset("nonexistent", "https://example.test/asset/one")

	assetsMutex.Lock()
	_, present := assets["nonexistent"]
	assetsMutex.Unlock()

	if present {
		t.Fatal("recordAsset created an entry for an unknown message ID")
	}
}

// TestHasAssetCaseInsensitive checks that the outer-URL guard at the top of
// ProxyHandler continues to accept URL casings that differ from the stored
// entry, matching the previous tools.InArray(uri, links) semantics.
func TestHasAssetCaseInsensitive(t *testing.T) {
	const id = "msg-outer"
	seedAssets(id)

	// Prime the index directly so we don't have to build from storage.
	assetsMutex.Lock()
	entry := assets[id]
	entry.Assets = []string{"https://example.test/style.css"}
	entry.index[strings.ToLower("https://example.test/style.css")] = struct{}{}
	assets[id] = entry
	assetsMutex.Unlock()

	cases := []struct {
		name string
		url  string
		want bool
	}{
		{"exact", "https://example.test/style.css", true},
		{"mixed case host", "https://Example.Test/style.css", true},
		{"mixed case path", "https://example.test/STYLE.CSS", true},
		{"different path", "https://example.test/other.css", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := hasAsset(id, tc.url)
			if err != nil {
				t.Fatalf("hasAsset returned error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("hasAsset(%q) = %v, want %v", tc.url, got, tc.want)
			}
		})
	}
}
