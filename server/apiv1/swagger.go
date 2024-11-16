package apiv1

// These structs are for the purpose of defining swagger HTTP parameters & responses

// Binary data response which inherits the attachment's content type.
// swagger:response BinaryResponse
type binaryResponse string

// Plain text response
// swagger:response TextResponse
type textResponse string

// HTML response
// swagger:response HTMLResponse
type htmlResponse string

// Server error will return with a 400 status code
// with the error message in the body
// swagger:response ErrorResponse
type errorResponse string

// Not found error will return a 404 status code
// swagger:response NotFoundResponse
type notFoundResponse string

// Plain text "ok" response
// swagger:response OKResponse
type okResponse string

// Plain JSON array response
// swagger:response ArrayResponse
type arrayResponse []string

// JSON error response
// swagger:response jsonErrorResponse
type jsonErrorResponse struct {
	// A JSON-encoded error response
	//
	// in: body
	Body JSONErrorMessage
}
