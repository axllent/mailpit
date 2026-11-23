package apiv1

import (
	"net/http"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/htmlcheck"
	"github.com/axllent/mailpit/internal/linkcheck"
	"github.com/axllent/mailpit/internal/rspamd"
	"github.com/axllent/mailpit/internal/smtpd/chaos"
	"github.com/axllent/mailpit/internal/spamassassin"
	"github.com/axllent/mailpit/internal/stats"
)

// Route describes a single API operation.
//
// Routes is the single source of truth for the HTTP API: the server registers its handlers from it, and the
// OpenAPI specification (server/ui/api/v1/openapi.json) is generated from it by internal/openapi.
// Run `go run ./internal/openapi/gen` after changing anything here, or in any type which it refers to.
type Route struct {
	// Method is the HTTP method.
	Method string
	// Path is the path without the webroot, with a leading slash. Path parameters use the net/http
	// pattern syntax, eg: "/api/v1/message/{id}".
	Path string
	// Handler serves the route. It is nil for a route which is documented here but registered elsewhere.
	Handler http.HandlerFunc
	// SendAPI routes are protected by the send API authentication rather than the UI authentication.
	SendAPI bool
	// Enabled, if set, decides at start-up whether the route is registered. The route is always documented.
	Enabled func() bool

	// ID is the unique OpenAPI operationId.
	ID string
	// Tag groups the operation in the documentation.
	Tag string
	// Summary is the one line summary.
	Summary string
	// Description is the (markdown) description.
	Description string
	// Params is a struct describing the path and query parameters (`path:"name" json:"-"` and `query:"name" json:"-"` field
	// tags, the field doc comments are the descriptions), or nil.
	Params any
	// Body is the type of the JSON request body (ideally the type the handler decodes into), or nil.
	Body any
	// Response is the type of a successful (200) response body.
	Response any
	// ResponseDescription is the description of the successful response. (The descriptions of the error responses
	// are the same for every route, and are set by the generator.)
	ResponseDescription string
	// ResponseExample is an optional example of the successful response body, which must be a value of the Response
	// type (eg: TextBody("ok")). It is for responses which are not generated from a struct with `example:` fields,
	// such as plain text and maps.
	ResponseExample any
	// ContentTypes are the content types of the successful response.
	ContentTypes []string
	// NotFound documents a 404 response.
	NotFound bool
	// JSONErrors documents error responses as JSON rather than plain text (the send API).
	JSONErrors bool
	// ExtraErrors are additional documented error status codes (in addition to 400, and 404 if NotFound).
	ExtraErrors []int
}

// TextBody is the Response type of plain text (or HTML) responses.
type TextBody string

// BinaryBody is the Response type of binary responses, eg: attachments.
type BinaryBody []byte

const (
	jsonType = "application/json"
	textType = "text/plain"
	htmlType = "text/html"
)

func spamAssassinEnabled() bool { return config.EnableSpamAssassin != "" }

func rspamdEnabled() bool { return config.EnableRspamd != "" }

// ---- Documentation-only types: path and query parameters, and response bodies ----

type messageParams struct {
	// Message database ID or "latest"
	ID string `path:"id" json:"-"`
}

type attachmentParams struct {
	// Message database ID or "latest"
	ID string `path:"id" json:"-"`
	// Attachment part ID
	PartID string `path:"partID" json:"-"`
}

type releaseParams struct {
	// Message database ID
	ID string `path:"id" json:"-"`
}

type listParams struct {
	// Pagination offset
	Start int `query:"start" default:"0" json:"-"`
	// Limit number of results
	Limit int `query:"limit" default:"50" json:"-"`
}

type readStatusParams struct {
	// Optional [timezone identifier](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) used only for `before:` & `after:` searches (eg: "Pacific/Auckland").
	TZ string `query:"tz" json:"-"`
}

type searchParams struct {
	// Search query
	Query string `query:"query" required:"true" json:"-"`
	// Pagination offset
	Start string `query:"start" default:"0" json:"-"`
	// Limit results
	Limit string `query:"limit" default:"50" json:"-"`
	// Optional [timezone identifier](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) used only for `before:` & `after:` searches (eg: "Pacific/Auckland").
	TZ string `query:"tz" json:"-"`
}

type deleteSearchParams struct {
	// Search query
	Query string `query:"query" required:"true" json:"-"`
	// [Timezone identifier](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) used only for `before:` & `after:` searches (eg: "Pacific/Auckland").
	TZ string `query:"tz" json:"-"`
}

type linkCheckParams struct {
	// Message database ID or "latest"
	ID string `path:"id" json:"-"`
	// Follow redirects
	Follow string `query:"follow" default:"false" json:"-"`
}

type renameTagParams struct {
	// The url-encoded tag name to rename
	Tag string `path:"tag" json:"-"`
}

type deleteTagParams struct {
	// The url-encoded tag name to delete
	Tag string `path:"tag" json:"-"`
}

type viewHTMLParams struct {
	// Message database ID or "latest"
	ID string `path:"id" json:"-"`
	// If this is route is to be embedded in an iframe, set embed to `1` in the URL to add `target="_blank"` and `rel="noreferrer noopener"` to all links.
	//
	// In addition, a small script will be added to the end of the document to post (postMessage()) the height of the document back to the parent window for optional iframe height resizing.
	//
	// Note that this will also *transform* the message into a full HTML document (if it isn't already), so this option is useful for viewing but not programmatic testing.
	Embed string `query:"embed" json:"-"`
}

// sendResponse is the response of a successfully sent message
//
// openapi:model SendResponse
type sendResponse struct {
	// Database ID
	// example: iAfZVVe2UQfNSG5BAjgYwa
	ID string
}

// Routes is the HTTP API.
var Routes = []Route{
	{
		Method:              http.MethodGet,
		Path:                "/api/v1/info",
		Handler:             AppInfo,
		ID:                  "AppInformation",
		Tag:                 "application",
		Summary:             "Get application information",
		Description:         "Returns basic runtime information, message totals and latest release version.",
		Response:            stats.AppInformation{},
		ContentTypes:        []string{jsonType},
		ResponseDescription: "Application information",
	},
	{
		Method:  http.MethodGet,
		Path:    "/api/v1/webui",
		Handler: WebUIConfig,
		ID:      "WebUIConfigurationResponse",
		Tag:     "application",
		Summary: "Get web UI configuration",
		Description: "Returns configuration settings for the web UI.\n" +
			"Intended for web UI only!",
		Response:            WebUIConfiguration{},
		ContentTypes:        []string{jsonType},
		ResponseDescription: "Web UI configuration response",
	},
	{
		Method:              http.MethodGet,
		Path:                "/api/v1/messages",
		Handler:             GetMessages,
		ID:                  "GetMessagesParams",
		Tag:                 "messages",
		Summary:             "List messages",
		Description:         "Returns messages from the mailbox ordered from newest to oldest.",
		Params:              listParams{},
		Response:            MessagesSummary{},
		ContentTypes:        []string{jsonType},
		ResponseDescription: "Summary of messages",
	},
	{
		Method:  http.MethodPut,
		Path:    "/api/v1/messages",
		Handler: SetReadStatus,
		ID:      "SetReadStatusParams",
		Tag:     "messages",
		Summary: "Set read status",
		Description: "You can optionally provide an array of IDs or a search string.\n" +
			"If neither IDs nor search is provided then all mailbox messages are updated.",
		Params:              readStatusParams{},
		Body:                SetReadStatusRequest{},
		Response:            TextBody(""),
		ContentTypes:        []string{textType},
		ResponseDescription: "Plain text \"ok\" response",
		ResponseExample:     TextBody("ok"),
	},
	{
		Method:              http.MethodDelete,
		Path:                "/api/v1/messages",
		Handler:             DeleteMessages,
		ID:                  "DeleteMessagesParams",
		Tag:                 "messages",
		Summary:             "Delete messages",
		Description:         "Delete individual or all messages. If no IDs are provided then all messages are deleted.",
		Body:                DeleteMessagesRequest{},
		Response:            TextBody(""),
		ContentTypes:        []string{textType},
		ResponseDescription: "Plain text \"ok\" response",
		ResponseExample:     TextBody("ok"),
	},
	{
		Method:              http.MethodGet,
		Path:                "/api/v1/search",
		Handler:             Search,
		ID:                  "SearchParams",
		Tag:                 "messages",
		Summary:             "Search messages",
		Description:         "Returns messages matching [a search](https://mailpit.axllent.org/docs/usage/search-filters/), sorted by received date (descending).",
		Params:              searchParams{},
		Response:            MessagesSummary{},
		ContentTypes:        []string{jsonType},
		ResponseDescription: "Summary of messages",
	},
	{
		Method:              http.MethodDelete,
		Path:                "/api/v1/search",
		Handler:             DeleteSearch,
		ID:                  "DeleteSearchParams",
		Tag:                 "messages",
		Summary:             "Delete messages by search",
		Description:         "Delete all messages matching [a search](https://mailpit.axllent.org/docs/usage/search-filters/).",
		Params:              deleteSearchParams{},
		Response:            TextBody(""),
		ContentTypes:        []string{textType},
		ResponseDescription: "Plain text \"ok\" response",
		ResponseExample:     TextBody("ok"),
	},
	{
		Method:  http.MethodPost,
		Path:    "/api/v1/send",
		Handler: SendMessageHandler,
		SendAPI: true,
		ID:      "SendMessageParams",
		Tag:     "message",
		Summary: "Send a message",
		Description: "Send a message via the HTTP API.\n" +
			"\n" +
			"A message (and so a request) which is larger than the maximum message size is rejected with a 413 response, a\n" +
			"request which says that it is too large is rejected without being read. If too many messages are being sent at once a\n" +
			"request waits for its turn for a few seconds, and then gets a 503 response with a Retry-After header.",
		Body:                SendRequest{},
		Response:            sendResponse{},
		ContentTypes:        []string{jsonType},
		ResponseDescription: "Confirmation message for HTTP send API",
		JSONErrors:          true,
		ExtraErrors:         []int{http.StatusRequestEntityTooLarge, http.StatusServiceUnavailable},
	},
	{
		Method:  http.MethodGet,
		Path:    "/api/v1/message/{id}",
		Handler: GetMessage,
		ID:      "GetMessageParams",
		Tag:     "message",
		Summary: "Get message summary",
		Description: "Returns the summary of a message, marking the message as read.\n" +
			"\n" +
			"The ID can be set to `latest` to return the latest message.",
		Params:              messageParams{},
		Response:            Message{},
		ContentTypes:        []string{jsonType},
		ResponseDescription: "Message",
		NotFound:            true,
	},
	{
		Method:  http.MethodGet,
		Path:    "/api/v1/message/{id}/headers",
		Handler: GetHeaders,
		ID:      "GetHeadersParams",
		Tag:     "message",
		Summary: "Get message headers",
		Description: "Returns the message headers as an array. Note that header keys are returned alphabetically.\n" +
			"\n" +
			"The ID can be set to `latest` to return the latest message headers.",
		Params:              messageParams{},
		Response:            map[string][]string{},
		ContentTypes:        []string{jsonType},
		ResponseDescription: "Message headers",
		ResponseExample: map[string][]string{
			"Content-Type": {"text/plain; charset=utf-8"},
			"Date":         {"Mon, 02 Jan 2006 15:04:05 -0700"},
			"From":         {"John Doe <john@example.com>"},
			"Message-Id":   {"<CA+12345@example.com>"},
			"Subject":      {"Message subject"},
			"To":           {"Jane Doe <jane@example.com>"},
		},
		NotFound: true,
	},
	{
		Method:  http.MethodGet,
		Path:    "/api/v1/message/{id}/part/{partID}",
		Handler: DownloadAttachment,
		ID:      "AttachmentParams",
		Tag:     "message",
		Summary: "Get message attachment",
		Description: "This will return the attachment part using the appropriate Content-Type.\n" +
			"\n" +
			"The ID can be set to `latest` to reference the latest message.",
		Params:              attachmentParams{},
		Response:            BinaryBody{},
		ContentTypes:        []string{"application/*", "image/*", "text/*"},
		ResponseDescription: "Binary data response which inherits the attachment's content type.",
		NotFound:            true,
	},
	{
		Method:  http.MethodGet,
		Path:    "/api/v1/message/{id}/part/{partID}/thumb",
		Handler: Thumbnail,
		ID:      "ThumbnailParams",
		Tag:     "message",
		Summary: "Get an attachment image thumbnail",
		Description: "This will return a cropped 180x120 JPEG thumbnail of an image attachment.\n" +
			"If the image is smaller than 180x120 then the image is padded. If the attachment is not an image then a blank image is returned.\n" +
			"\n" +
			"The ID can be set to `latest` to return the latest message.",
		Params:              attachmentParams{},
		Response:            BinaryBody{},
		ContentTypes:        []string{"image/jpeg"},
		ResponseDescription: "Binary data response which inherits the attachment's content type.",
	},
	{
		Method:  http.MethodGet,
		Path:    "/api/v1/message/{id}/raw",
		Handler: DownloadRaw,
		ID:      "DownloadRawParams",
		Tag:     "message",
		Summary: "Get message source",
		Description: "Returns the full email source as plain text.\n" +
			"\n" +
			"The ID can be set to `latest` to return the latest message source.",
		Params:              messageParams{},
		Response:            TextBody(""),
		ContentTypes:        []string{textType},
		ResponseDescription: "Plain text response",
		ResponseExample: TextBody("Date: Mon, 02 Jan 2006 15:04:05 -0700\n" +
			"From: John Doe <john@example.com>\n" +
			"To: Jane Doe <jane@example.com>\n" +
			"Subject: Message subject\n" +
			"Message-Id: <CA+12345@example.com>\n" +
			"Content-Type: text/plain; charset=utf-8\n" +
			"\n" +
			"Message body"),
		NotFound: true,
	},
	{
		Method:  http.MethodPost,
		Path:    "/api/v1/message/{id}/release",
		Handler: ReleaseMessage,
		ID:      "ReleaseMessageParams",
		Tag:     "message",
		Summary: "Release message",
		Description: "Release a message via a pre-configured external SMTP server. This is only enabled if message relaying has been configured.\n" +
			"\n" +
			"The ID can be set to `latest` to reference the latest message.",
		Params:              releaseParams{},
		Body:                ReleaseRequest{},
		Response:            TextBody(""),
		ContentTypes:        []string{textType},
		ResponseDescription: "Plain text \"ok\" response",
		ResponseExample:     TextBody("ok"),
		NotFound:            true,
	},
	{
		Method:  http.MethodGet,
		Path:    "/api/v1/message/{id}/html-check",
		Handler: HTMLCheck,
		ID:      "HTMLCheckParams",
		Tag:     "other",
		Summary: "HTML check",
		Description: "Returns the summary of the message HTML checker.\n" +
			"\n" +
			"The ID can be set to `latest` to return the latest message.",
		Params:              messageParams{},
		Response:            htmlcheck.Response{},
		ContentTypes:        []string{jsonType},
		ResponseDescription: "HTMLCheckResponse",
		NotFound:            true,
	},
	{
		Method:  http.MethodGet,
		Path:    "/api/v1/message/{id}/link-check",
		Handler: LinkCheck,
		ID:      "LinkCheckParams",
		Tag:     "other",
		Summary: "Link check",
		Description: "Returns the summary of the message Link checker.\n" +
			"\n" +
			"The ID can be set to `latest` to return the latest message.",
		Params:              linkCheckParams{},
		Response:            linkcheck.Response{},
		ContentTypes:        []string{jsonType},
		ResponseDescription: "LinkCheckResponse",
		NotFound:            true,
	},
	{
		Method:  http.MethodGet,
		Path:    "/api/v1/message/{id}/sa-check",
		Handler: SpamAssassinCheck,
		Enabled: spamAssassinEnabled,
		ID:      "SpamAssassinCheckParams",
		Tag:     "other",
		Summary: "SpamAssassin check",
		Description: "Returns the SpamAssassin summary (if enabled) of the message.\n" +
			"\n" +
			"The ID can be set to `latest` to return the latest message.",
		Params:              messageParams{},
		Response:            spamassassin.Result{},
		ContentTypes:        []string{jsonType},
		ResponseDescription: "SpamAssassinResponse",
		NotFound:            true,
	},
	{
		Method:  http.MethodGet,
		Path:    "/api/v1/message/{id}/rspamd-check",
		Handler: RspamdCheck,
		Enabled: rspamdEnabled,
		ID:      "RspamdCheckParams",
		Tag:     "other",
		Summary: "Rspamd check",
		Description: "Returns the Rspamd summary (if enabled) of the message.\n" +
			"\n" +
			"The ID can be set to `latest` to return the latest message.",
		Params:              messageParams{},
		Response:            rspamd.Result{},
		ContentTypes:        []string{jsonType},
		ResponseDescription: "RspamdResponse",
		NotFound:            true,
	},
	{
		Method:              http.MethodGet,
		Path:                "/api/v1/tags",
		Handler:             GetAllTags,
		ID:                  "GetAllTags",
		Tag:                 "tags",
		Summary:             "Get all current tags",
		Description:         "Returns a JSON array of all unique message tags.",
		Response:            []string{},
		ContentTypes:        []string{jsonType},
		ResponseDescription: "Plain JSON array response",
	},
	{
		Method:  http.MethodPut,
		Path:    "/api/v1/tags",
		Handler: SetMessageTags,
		ID:      "SetTagsParams",
		Tag:     "tags",
		Summary: "Set message tags",
		Description: "This will overwrite any existing tags for selected message database IDs. To remove all tags from a message, pass an empty tags array.\n" +
			"A message can have at most 100 tags, passing more is an error.",
		Body:                SetTagsRequest{},
		Response:            TextBody(""),
		ContentTypes:        []string{textType},
		ResponseDescription: "Plain text \"ok\" response",
		ResponseExample:     TextBody("ok"),
	},
	{
		Method:              http.MethodPut,
		Path:                "/api/v1/tags/{tag}",
		Handler:             RenameTag,
		ID:                  "RenameTagParams",
		Tag:                 "tags",
		Summary:             "Rename a tag",
		Description:         "Renames an existing tag.",
		Params:              renameTagParams{},
		Body:                RenameTagRequest{},
		Response:            TextBody(""),
		ContentTypes:        []string{textType},
		ResponseDescription: "Plain text \"ok\" response",
		ResponseExample:     TextBody("ok"),
	},
	{
		Method:              http.MethodDelete,
		Path:                "/api/v1/tags/{tag}",
		Handler:             DeleteTag,
		ID:                  "DeleteTagParams",
		Tag:                 "tags",
		Summary:             "Delete a tag",
		Description:         "Deletes a tag. This will not delete any messages with the tag, but will remove the tag from any messages containing the tag.",
		Params:              deleteTagParams{},
		Response:            TextBody(""),
		ContentTypes:        []string{textType},
		ResponseDescription: "Plain text \"ok\" response",
		ResponseExample:     TextBody("ok"),
	},
	{
		Method:  http.MethodGet,
		Path:    "/api/v1/chaos",
		Handler: GetChaos,
		ID:      "getChaos",
		Tag:     "testing",
		Summary: "Get Chaos triggers",
		Description: "Returns the current Chaos triggers configuration.\n" +
			"This API route will return an error if Chaos is not enabled at runtime.",
		Response:            chaos.Triggers{},
		ContentTypes:        []string{jsonType},
		ResponseDescription: "Response for the Chaos triggers configuration",
	},
	{
		Method:  http.MethodPut,
		Path:    "/api/v1/chaos",
		Handler: SetChaos,
		ID:      "setChaosParams",
		Tag:     "testing",
		Summary: "Set Chaos triggers",
		Description: "Set the Chaos triggers configuration and return the updated values.\n" +
			"This API route will return an error if Chaos is not enabled at runtime.\n" +
			"\n" +
			"If any triggers are omitted from the request, then those are reset to their\n" +
			"default values with a 0% probability (ie: disabled).\n" +
			"Setting a blank `{}` will reset all triggers to their default values.",
		Body:                chaos.Triggers{},
		Response:            chaos.Triggers{},
		ContentTypes:        []string{jsonType},
		ResponseDescription: "Response for the Chaos triggers configuration",
	},
	{
		Method:  http.MethodGet,
		Path:    "/view/{id}.html",
		ID:      "GetMessageHTMLParams",
		Tag:     "testing",
		Summary: "Render message HTML part",
		Description: "Renders just the message's HTML part which can be used for UI integration testing.\n" +
			"Attached inline images are modified to link to the API provided they exist.\n" +
			"Note that is the message does not contain a HTML part then an 404 error is returned.\n" +
			"A HTML part which is larger than 10 MiB is not rendered, and a 413 error is returned.\n" +
			"\n" +
			"The ID can be set to `latest` to return the latest message.",
		Params:              viewHTMLParams{},
		Response:            TextBody(""),
		ContentTypes:        []string{htmlType},
		ResponseDescription: "HTML response",
		NotFound:            true,
	},
	{
		Method:  http.MethodGet,
		Path:    "/view/{id}.txt",
		ID:      "GetMessageTextParams",
		Tag:     "testing",
		Summary: "Render message text part",
		Description: "Renders just the message's text part which can be used for UI integration testing.\n" +
			"\n" +
			"The ID can be set to `latest` to return the latest message.",
		Params:              messageParams{},
		Response:            TextBody(""),
		ContentTypes:        []string{textType},
		ResponseDescription: "Plain text response",
		NotFound:            true,
	},
}
