package html2text

import (
	"strings"
	"testing"
	"time"
)

// TestStripLinearTime checks that adversarial input is not much slower than benign
// input of the same size, eg: many "</" without a nearby ">" must not make Strip
// quadratic. A quadratic implementation is several hundred times slower at this size.
func TestStripLinearTime(t *testing.T) {
	const size = 1 << 20

	control := strings.Repeat("</p>", size/4)
	exploits := map[string]string{
		"close tag without >": strings.Repeat("</", size/2),
		"close tag name":      strings.Repeat("</p", size/3),
		"open angle":          strings.Repeat("<", size),
		"br prefix":           strings.Repeat("<b", size/2),
		"img prefix":          strings.Repeat("<i", size/2),
	}

	// fastest of a few runs, to reduce noise from other tests or a busy machine
	timeStrip := func(h string) time.Duration {
		best := time.Duration(0)
		for range 3 {
			start := time.Now()
			_, _ = Strip(h, true)
			if d := time.Since(start); best == 0 || d < best {
				best = d
			}
		}

		return best
	}

	limit := 10 * timeStrip(control)
	for name, h := range exploits {
		if d := timeStrip(h); d > limit {
			t.Errorf("%s: Strip took %v on %d bytes, want at most %v (10x the benign control)", name, d, len(h), limit)
		}
	}
}
