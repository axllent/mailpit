package linkcheck

// Response represents the Link check response
//
// swagger:model LinkCheckResponse
type Response struct {
	// Total number of errors
	Errors int `json:"Errors"`
	// Tested links
	Links []Link `json:"Links"`
}

// Link struct
type Link struct {
	// Link URL
	URL string `json:"URL"`
	// HTTP status code
	StatusCode int `json:"StatusCode"`
	// HTTP status definition
	Status string `json:"Status"`
}
