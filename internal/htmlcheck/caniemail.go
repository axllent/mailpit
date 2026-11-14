// Package htmlcheck is used for parsing HTML and returning
// HTML compatibility errors and warnings
package htmlcheck

import (
	"embed"
	"encoding/json"
	"regexp"
	"strings"
	"sync"
)

//go:embed caniemail-data.json
var embeddedFS embed.FS

var (
	// cie is populated exactly once by loadJSONData and must be treated as read-only afterwards
	cie     = CanIEmail{}
	cieOnce sync.Once
	cieErr  error

	noteMatch = regexp.MustCompile(` #(\d+)$`)

	// LimitFamilies will limit results to families if set
	LimitFamilies = []string{}

	// LimitPlatforms will limit results to platforms if set
	LimitPlatforms = []string{}

	// LimitClients will limit results to clients if set
	LimitClients = []string{}
)

// CanIEmail struct for JSON data
type CanIEmail struct {
	APIVersion     string `json:"api_version"`
	LastUpdateDate string `json:"last_update_date"`
	// NiceNames map[string]string `json:"last_update_date"`
	NiceNames struct {
		Family   map[string]string `json:"family"`
		Platform map[string]string `json:"platform"`
		Support  map[string]string `json:"support"`
		Category map[string]string `json:"category"`
	} `json:"nicenames"`
	Data []JSONResult `json:"data"`
}

// JSONResult struct for CanIEmail Data
type JSONResult struct {
	Slug           string            `json:"slug"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	URL            string            `json:"url"`
	Category       string            `json:"category"`
	Tags           []string          `json:"tags"`
	Keywords       string            `json:"keywords"`
	LastTestDate   string            `json:"last_test_date"`
	TestURL        string            `json:"test_url"`
	TestResultsURL string            `json:"test_results_url"`
	Stats          map[string]any    `json:"stats"`
	Notes          string            `json:"notes"`
	NotesByNumber  map[string]string `json:"notes_by_num"`
}

// Load the JSON data. It is safe for concurrent use: the data is parsed once into a
// local value and only published (to be read by all callers) after a successful parse,
// and any error is returned to every caller.
func loadJSONData() error {
	cieOnce.Do(func() {
		b, err := embeddedFS.ReadFile("caniemail-data.json")
		if err != nil {
			cieErr = err
			return
		}

		var data CanIEmail
		if err := json.Unmarshal(b, &data); err != nil {
			cieErr = err
			return
		}

		cie = data
	})

	return cieErr
}

// noteNumber returns the note number from a support string such as "y #12", or an empty
// string if it has none.
func noteNumber(support string) string {
	m := noteMatch.FindStringSubmatch(support)
	if len(m) != 2 {
		return ""
	}

	return strings.TrimSpace(m[1])
}
