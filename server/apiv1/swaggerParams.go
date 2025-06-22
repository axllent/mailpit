// Package apiv1 provides the API v1 endpoints for Mailpit.
//
// These structs are for the purpose of defining swagger HTTP parameters in go-swagger
// in order to generate a spec file. They are lowercased to avoid exporting them as public types.
//
//nolint:unused
package apiv1

import "github.com/axllent/mailpit/internal/smtpd/chaos"

// swagger:parameters setChaosParams
type setChaosParams struct {
	// in: body
	Body chaos.Triggers
}

// swagger:parameters AttachmentParams
type attachmentParams struct {
	// Message database ID or "latest"
	//
	// in: path
	// required: true
	ID string

	// Attachment part ID
	//
	// in: path
	// required: true
	PartID string
}

// swagger:parameters DownloadRawParams
type downloadRawParams struct {
	// Message database ID or "latest"
	//
	// in: path
	// required: true
	ID string
}

// swagger:parameters GetMessageParams
type getMessageParams struct {
	// Message database ID or "latest"
	//
	// in: path
	// required: true
	ID string
}

// swagger:parameters GetHeadersParams
type getHeadersParams struct {
	// Message database ID or "latest"
	//
	// in: path
	// required: true
	ID string
}

// swagger:parameters GetMessagesParams
type getMessagesParams struct {
	// Pagination offset
	//
	// in: query
	// name: start
	// required: false
	// default: 0
	// type: integer
	Start int `json:"start"`

	// Limit number of results
	//
	// in: query
	// name: limit
	// required: false
	// default: 50
	// type: integer
	Limit int `json:"limit"`
}

// swagger:parameters SetReadStatusParams
type setReadStatusParams struct {
	// in: body
	Body struct {
		// Read status
		//
		// required: false
		// default: false
		// example: true
		Read bool

		// Optional array of message database IDs
		//
		// required: false
		// default: []
		// example: ["4oRBnPtCXgAqZniRhzLNmS", "hXayS6wnCgNnt6aFTvmOF6"]
		IDs []string

		// Optional messages matching a search
		//
		// required: false
		// example: tag:backups
		Search string
	}

	// Optional [timezone identifier](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) used only for `before:` & `after:` searches (eg: "Pacific/Auckland").
	//
	// in: query
	// required: false
	// type string
	TZ string `json:"tz"`
}

// swagger:parameters DeleteMessagesParams
type deleteMessagesParams struct {
	// Delete request
	// in: body
	Body struct {
		// Array of message database IDs
		//
		// required: false
		// example: ["4oRBnPtCXgAqZniRhzLNmS", "hXayS6wnCgNnt6aFTvmOF6"]
		IDs []string
	}
}

// swagger:parameters SearchParams
type searchParams struct {
	// Search query
	//
	// in: query
	// required: true
	// type: string
	Query string `json:"query"`

	// Pagination offset
	//
	// in: query
	// required: false
	// default: 0
	// type integer
	Start string `json:"start"`

	// Limit results
	//
	// in: query
	// required: false
	// default: 50
	// type integer
	Limit string `json:"limit"`

	// Optional [timezone identifier](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) used only for `before:` & `after:` searches (eg: "Pacific/Auckland").
	//
	// in: query
	// required: false
	// type string
	TZ string `json:"tz"`
}

// swagger:parameters DeleteSearchParams
type deleteSearchParams struct {
	// Search query
	//
	// in: query
	// required: true
	// type: string
	Query string `json:"query"`

	// [Timezone identifier](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) used only for `before:` & `after:` searches (eg: "Pacific/Auckland").
	//
	// in: query
	// required: false
	// type string
	TZ string `json:"tz"`
}

// swagger:parameters HTMLCheckParams
type htmlCheckParams struct {
	// Message database ID or "latest"
	//
	// in: path
	// description: Message database ID or "latest"
	// required: true
	ID string
}

// swagger:parameters LinkCheckParams
type linkCheckParams struct {
	// Message database ID or "latest"
	//
	// in: path
	// required: true
	ID string

	// Follow redirects
	//
	// in: query
	// required: false
	// default: false
	Follow string `json:"follow"`
}

// swagger:parameters ReleaseMessageParams
type releaseMessageParams struct {
	// Message database ID
	//
	// in: path
	// description: Message database ID
	// required: true
	ID string

	// in: body
	Body struct {
		// Array of email addresses to relay the message to
		//
		// required: true
		// example: ["user1@example.com", "user2@example.com"]
		To []string
	}
}

// swagger:parameters SendMessageParams
type sendMessageParams struct {
	// in: body
	// Body SendRequest
	Body struct {
		// "From" recipient
		// required: true
		From struct {
			// Optional name
			// example: John Doe
			Name string
			// Email address
			// example: john@example.com
			// required: true
			Email string
		}

		// "To" recipients
		To []struct {
			// Optional name
			// example: Jane Doe
			Name string
			// Email address
			// example: jane@example.com
			// required: true
			Email string
		}

		// Cc recipients
		Cc []struct {
			// Optional name
			// example: Manager
			Name string
			// Email address
			// example: manager@example.com
			// required: true
			Email string
		}

		// Bcc recipients email addresses only
		// example: ["jack@example.com"]
		Bcc []string

		// Optional Reply-To recipients
		ReplyTo []struct {
			// Optional name
			// example: Secretary
			Name string
			// Email address
			// example: secretary@example.com
			// required: true
			Email string
		}

		// Subject
		// example: Mailpit message via the HTTP API
		Subject string

		// Message body (text)
		// example: Mailpit is awesome!
		Text string

		// Message body (HTML)
		// example: <div style="text-align:center"><p style="font-family: arial; font-size: 24px;">Mailpit is <b>awesome</b>!</p><p><img src="cid:mailpit-logo" /></p></div>
		HTML string

		// Attachments
		Attachments []struct {
			// Base64-encoded string of the file content
			// required: true
			// example: iVBORw0KGgoAAAANSUhEUgAAAEEAAAA8CAMAAAAOlSdoAAAACXBIWXMAAAHrAAAB6wGM2bZBAAAAS1BMVEVHcEwRfnUkZ2gAt4UsSF8At4UtSV4At4YsSV4At4YsSV8At4YsSV4At4YsSV4sSV4At4YsSV4At4YtSV4At4YsSV4At4YtSV8At4YsUWYNAAAAGHRSTlMAAwoXGiktRE5dbnd7kpOlr7zJ0d3h8PD8PCSRAAACWUlEQVR42pXT4ZaqIBSG4W9rhqQYocG+/ys9Y0Z0Br+x3j8zaxUPewFh65K+7yrIMeIY4MT3wPfEJCidKXEMnLaVkxDiELiMz4WEOAZSFghxBIypCOlKiAMgXfIqTnBgSm8CIQ6BImxEUxEckClVQiHGj4Ba4AQHikAIClwTE9KtIghAhUJwoLkmLnCiAHJLRKgIMsEtVUKbBUIwoAg2C4QgQBE6l4VCnApBgSKYLLApCnCa0+96AEMW2BQcmC+Pr3nfp7o5Exy49gIADcIqUELGfeA+bp93LmAJp8QJoEcN3C7NY3sbVANixMyI0nku20/n5/ZRf3KI2k6JEDWQtxcbdGuAqu3TAXG+/799Oyyas1B1MnMiA+XyxHp9q0PUKGPiRAau1fZbLRZV09wZcT8/gHk8QQAxXn8VgaDqcUmU6O/r28nbVwXAqca2mRNtPAF5+zoP2MeN9Fy4NgC6RfcbgE7XITBRYTtOE3U3C2DVff7pk+PkUxgAbvtnPXJaD6DxulMLwOhPS/M3MQkgg1ZFrIXnmfaZoOfpKiFgzeZD/WuKqQEGrfJYkyWf6vlG3xUgTuscnkNkQsb599q124kdpMUjCa/XARHs1gZymVtGt3wLkiFv8rUgTxitYCex5EVGec0Y9VmoDTFBSQte2TfXGXlf7hbdaUM9Sk7fisEN9qfBBTK+FZcvM9fQSdkl2vj4W2oX/bRogO3XasiNH7R0eW7fgRM834ImTg+Lg6BEnx4vz81rhr+MYPBBQg1v8GndEOrthxaCTxNAOut8WKLGZQl+MPz88Q9tAO/hVuSeqQAAAABJRU5ErkJggg==
			Content string
			// Filename
			// required: true
			// example: mailpit.png
			Filename string
			// Optional Content Type for the the attachment.
			// If this field is not set (or empty) then the content type is automatically detected.
			// required: false
			// example: image/png
			ContentType string
			// Optional Content-ID (`cid`) for attachment.
			// If this field is set then the file is attached inline.
			// required: false
			// example: mailpit-logo
			ContentID string
		}

		// Mailpit tags
		// example: ["Tag 1","Tag 2"]
		Tags []string

		// Optional headers in {"key":"value"} format
		// example: {"X-IP":"1.2.3.4"}
		Headers map[string]string
	}
}

// swagger:parameters SetTagsParams
type setTagsParams struct {
	// in: body
	Body struct {
		// Array of tag names to set
		//
		// required: true
		// example: ["Tag 1", "Tag 2"]
		Tags []string

		// Array of message database IDs
		//
		// required: true
		// example: ["4oRBnPtCXgAqZniRhzLNmS", "hXayS6wnCgNnt6aFTvmOF6"]
		IDs []string
	}
}

// swagger:parameters RenameTagParams
type renameTagParams struct {
	// The url-encoded tag name to rename
	//
	// in: path
	// required: true
	// type: string
	Tag string

	// in: body
	Body struct {
		// New name
		//
		// required: true
		// example: New name
		Name string
	}
}

// swagger:parameters DeleteTagParams
type deleteTagParams struct {
	// The url-encoded tag name to delete
	//
	// in: path
	// required: true
	Tag string
}

// swagger:parameters GetMessageHTMLParams
type getMessageHTMLParams struct {
	// Message database ID or "latest"
	//
	// in: path
	// required: true
	ID string

	// If this is route is to be embedded in an iframe, set embed to `1` in the URL to add `target="_blank"` and `rel="noreferrer noopener"` to all links.
	//
	// In addition, a small script will be added to the end of the document to post (postMessage()) the height of the document back to the parent window for optional iframe height resizing.
	//
	// Note that this will also *transform* the message into a full HTML document (if it isn't already), so this option is useful for viewing but not programmatic testing.
	//
	// in: query
	// required: false
	// type: string
	Embed string `json:"embed"`
}

// swagger:parameters GetMessageTextParams
type getMessageTextParams struct {
	// Message database ID or "latest"
	//
	// in: path
	// required: true
	ID string
}

// swagger:parameters SpamAssassinCheckParams
type spamAssassinCheckParams struct {
	// Message database ID or "latest"
	//
	// in: path
	// required: true
	ID string
}

// swagger:parameters ThumbnailParams
type thumbnailParams struct {
	// Message database ID or "latest"
	//
	// in: path
	// required: true
	ID string

	// Attachment part ID
	//
	// in: path
	// required: true
	PartID string
}
