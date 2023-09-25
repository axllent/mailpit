package htmlcheck

// Response represents the HTML check response struct
//
// swagger:model HTMLCheckResponse
type Response struct {
	// List of warnings from tests
	Warnings []Warning `json:"Warnings"`
	// All platforms tested, mainly for the web UI
	Platforms map[string][]string `json:"Platforms"`
	// Total overall score
	Total Total `json:"Total"`
}

// Warning represents a failed test
//
// swagger:model HTMLCheckWarning
type Warning struct {
	// Slug identifier
	Slug string `json:"Slug"`
	// Friendly title
	Title string `json:"Title"`
	// Description
	Description string `json:"Description"`
	// URL to caniemail.com
	URL string `json:"URL"`
	// Category [css, html]
	Category string `json:"Category"`
	// Tags
	Tags []string `json:"Tags"`
	// Keywords
	Keywords string `json:"Keywords"`
	// Test results
	Results []Result `json:"Results"`
	// Notes based on results
	NotesByNumber map[string]string `json:"NotesByNumber"`
	// Test score calculated from results
	Score Score `json:"Score"`
}

// Result struct
//
// swagger:model HTMLCheckResult
type Result struct {
	// Friendly name of result, combining family, platform & version
	Name string `json:"Name"`
	// Platform eg: ios, android, windows
	Platform string `json:"Platform"`
	// Family eg: Outlook, Mozilla Thunderbird
	Family string `json:"Family"`
	// Family version eg: 4.7.1, 2019-10, 10.3
	Version string `json:"Version"`
	// Support [yes, no, partial]
	Support string `json:"Support"`
	// Note number for partially supported if applicable
	NoteNumber string `json:"NoteNumber"` // where applicable
}

// Score struct
//
// swagger:model HTMLCheckScore
type Score struct {
	// Number of matches in the document
	Found int `json:"Found"`
	// Total percentage supported
	Supported float32 `json:"Supported"`
	// Total percentage partially supported
	Partial float32 `json:"Partial"`
	// Total percentage unsupported
	Unsupported float32 `json:"Unsupported"`
}

// Total weighted result for all scores
//
// swagger:model HTMLCheckTotal
type Total struct {
	// Total number of tests done
	Tests int `json:"Tests"`
	// Total number of HTML nodes detected in message
	Nodes int `json:"Nodes"`
	// Overall percentage supported
	Supported float32 `json:"Supported"`
	// Overall percentage partially supported
	Partial float32 `json:"Partial"` // total percentage
	// Overall percentage unsupported
	Unsupported float32 `json:"Unsupported"` // total percentage
}
