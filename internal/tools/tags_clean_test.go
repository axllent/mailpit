package tools

import (
	"strings"
	"testing"
)

// A tag cut at 100 characters between two words used to end with a space.
func TestCleanTagTruncation(t *testing.T) {
	a99 := strings.Repeat("a", 99)

	tests := []struct {
		name, in, want string
	}{
		// the cut falls on the space between two words
		{"exploit: the 100th character is a space", a99 + " b", a99},
		{"exploit: a long gap before it", a99 + "     b c", a99},
		{"exploit: replaced characters before it", a99 + "!!!b", a99},
		{"exploit: longer words after it", a99 + " " + strings.Repeat("b", 50), a99},
		{"exploit: several words, the cut after a space", strings.Repeat("abcd ", 20) + "efgh", strings.TrimSpace(strings.Repeat("abcd ", 20))},

		// controls: the same lengths where the cut is not at a space
		{"control: exactly 100", strings.Repeat("a", 100), strings.Repeat("a", 100)},
		{"control: 101 characters", strings.Repeat("a", 101), strings.Repeat("a", 100)},
		{"control: a space at the 101st position", strings.Repeat("a", 100) + " b", strings.Repeat("a", 100)},
		{"control: the cut inside a word", a99 + "bcd", a99 + "b"},
		{"control: a space inside the limit", "a b", "a b"},
		{"control: 98 characters and a word", strings.Repeat("a", 98) + " b", strings.Repeat("a", 98) + " b"},
		{"control: surrounding space", "  tag  ", "tag"},
		{"control: runs of spaces", "a   b", "a b"},
		{"control: invalid characters", "a<b>c&d", "a b c d"},
		{"control: empty", "   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanTag(tt.in)
			if got != tt.want {
				t.Errorf("CleanTag(%q) = %q, want %q", tt.in, got, tt.want)
			}
			if len(got) > 100 || got != strings.TrimSpace(got) {
				t.Errorf("CleanTag(%q) = %q (%d bytes), which is not trimmed or is too long", tt.in, got, len(got))
			}
			// the result is stable, so a stored tag is not changed when it is cleaned again
			if again := CleanTag(got); again != got {
				t.Errorf("cleaning %q again gives %q", got, again)
			}
		})
	}
}

func FuzzCleanTag(f *testing.F) {
	for _, s := range []string{"tag", strings.Repeat("a", 99) + " b", "a  b\t\nc", "日本語 tag", "<>&"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		got := CleanTag(s)
		if len(got) > 100 || got != strings.TrimSpace(got) || strings.Contains(got, "  ") {
			t.Fatalf("CleanTag(%q) = %q", s, got)
		}
		if tagsInvalidChars.MatchString(got) {
			t.Fatalf("CleanTag(%q) = %q has invalid characters", s, got)
		}
		if CleanTag(got) != got {
			t.Fatalf("CleanTag(%q) = %q is not stable", s, got)
		}
	})
}
