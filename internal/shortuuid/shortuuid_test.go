package shortuuid

import (
	"regexp"
	"testing"
)

// alphanumeric matches IDs that contain only digits and ASCII letters.
var alphanumeric = regexp.MustCompile(`^[0-9A-Za-z]+$`)

// TestLength verifies that every generated ID is exactly 22 characters long,
// including when the UUID encodes to a value with leading zero-padding.
func TestLength(t *testing.T) {
	for range 100 {
		id := New()
		if len(id) != length {
			t.Errorf("expected length %d, got %d: %q", length, len(id), id)
		}
	}
}

// TestAlphanumeric verifies that no ID contains hyphens, underscores, or any
// other non-alphanumeric character that would be unsafe in a URL path segment.
func TestAlphanumeric(t *testing.T) {
	for range 100 {
		id := New()
		if !alphanumeric.MatchString(id) {
			t.Errorf("non-alphanumeric characters in ID: %q", id)
		}
	}
}

// TestUnique verifies that IDs are unique across a large sample. Collisions are
// cryptographically implausible given the 122-bit UUID entropy, so any hit here
// indicates a bug in the encoding (e.g. truncation, constant output).
func TestUnique(t *testing.T) {
	seen := make(map[string]struct{}, 1000000)
	for range 1000000 {
		id := New()
		if _, exists := seen[id]; exists {
			t.Fatalf("duplicate ID generated: %q", id)
		}
		seen[id] = struct{}{}
	}
}

// BenchmarkNew measures the cost of generating a single ID, including UUID generation.
func BenchmarkNew(b *testing.B) {
	for b.Loop() {
		_ = New()
	}
}
