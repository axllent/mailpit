package apiv1

// These structs are for the purpose of defining swagger HTTP parameters & responses

// Application information
// swagger:response InfoResponse
type infoResponse struct {
	// Application information
	//
	// in: body
	Body appInformation
}

// Web UI configuration
// swagger:response WebUIConfigurationResponse
type webUIConfigurationResponse struct {
	// Web UI configuration settings
	//
	// in: body
	Body webUIConfiguration
}

// Message summary
// swagger:response MessagesSummaryResponse
type messagesSummaryResponse struct {
	// The message summary
	// in: body
	Body MessagesSummary
}

// Message headers
// swagger:model MessageHeaders
type messageHeaders map[string][]string

// swagger:parameters DeleteMessages
type deleteMessagesParams struct {
	// in: body
	Body *deleteMessagesRequestBody
}

// Delete request
// swagger:model DeleteRequest
type deleteMessagesRequestBody struct {
	// Array of message database IDs
	//
	// required: false
	// example: ["5dec4247-812e-4b77-9101-e25ad406e9ea", "8ac66bbc-2d9a-4c41-ad99-00aa75fa674e"]
	IDs []string `json:"ids"`
}

// swagger:parameters SetReadStatus
type setReadStatusParams struct {
	// in: body
	Body *setReadStatusRequestBody
}

// Set read status request
// swagger:model setReadStatusRequestBody
type setReadStatusRequestBody struct {
	// Read status
	//
	// required: false
	// default: false
	// example: true
	Read bool `json:"read"`

	// Array of message database IDs
	//
	// required: false
	// example: ["5dec4247-812e-4b77-9101-e25ad406e9ea", "8ac66bbc-2d9a-4c41-ad99-00aa75fa674e"]
	IDs []string `json:"ids"`
}

// swagger:parameters SetTags
type setTagsParams struct {
	// in: body
	Body *setTagsRequestBody
}

// Set tags request
// swagger:model setTagsRequestBody
type setTagsRequestBody struct {
	// Array of tag names to set
	//
	// required: true
	// example: ["Tag 1", "Tag 2"]
	Tags []string `json:"tags"`

	// Array of message database IDs
	//
	// required: true
	// example: ["5dec4247-812e-4b77-9101-e25ad406e9ea", "8ac66bbc-2d9a-4c41-ad99-00aa75fa674e"]
	IDs []string `json:"ids"`
}

// swagger:parameters ReleaseMessage
type releaseMessageParams struct {
	// Message database ID
	//
	// in: path
	// description: Message database ID
	// required: true
	ID string

	// in: body
	Body *releaseMessageRequestBody
}

// Release request
// swagger:model releaseMessageRequestBody
type releaseMessageRequestBody struct {
	// Array of email addresses to relay the message to
	//
	// required: true
	// example: ["user1@example.com", "user2@example.com"]
	To []string `json:"to"`
}

// swagger:parameters HTMLCheck
type htmlCheckParams struct {
	// Message database ID or "latest"
	//
	// in: path
	// description: Message database ID or "latest"
	// required: true
	ID string
}

// swagger:parameters LinkCheck
type linkCheckParams struct {
	// Message database ID or "latest"
	//
	// in: path
	// description: Message database ID or "latest"
	// required: true
	ID string

	// Follow redirects
	//
	// in: query
	// description: Follow redirects
	// required: false
	// default: false
	Follow string `json:"follow"`
}

// Binary data response inherits the attachment's content type
// swagger:response BinaryResponse
type binaryResponse string

// Plain text response
// swagger:response TextResponse
type textResponse string

// HTML response
// swagger:response HTMLResponse
type htmlResponse string

// HTTP error response will return with a >= 400 response code
// swagger:response ErrorResponse
type errorResponse string

// Plain text "ok" response
// swagger:response OKResponse
type okResponse string

// Plain JSON array response
// swagger:response ArrayResponse
type arrayResponse []string
