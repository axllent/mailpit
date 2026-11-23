package storage

import (
	"net/mail"
	"time"
)

// Message data excluding physical attachments
//
// openapi:model Message
type Message struct {
	// Database ID
	// example: "2FBBJL7M7OKnpVWZgh2ped"
	ID string
	// Message ID
	// example: "3UUdoPhRumhsHven3aBTxj"
	MessageID string
	// From address
	// example: {"Name": "John Doe", "Address": "from@example.com"}
	From *mail.Address
	// To addresses
	// example: [{"Name": "John Doe", "Address": "to@example.com"}]
	To []*mail.Address
	// Cc addresses
	// example: [{"Name": "John Doe", "Address": "cc@example.com"}]
	Cc []*mail.Address
	// Bcc addresses
	// example: [{"Name": "John Doe", "Address": "bcc@example.com"}]
	Bcc []*mail.Address
	// ReplyTo addresses
	// example: [{"Name": "John Doe", "Address": "reply-to@example.com"}]
	ReplyTo []*mail.Address
	// Return-Path
	// example: "return-path@example.com"
	ReturnPath string
	// Message subject
	// example: "Message subject"
	Subject string
	// List-Unsubscribe header information
	ListUnsubscribe ListUnsubscribe
	// Message RFC3339Nano date & time (if set), else date & time received
	// ([extended RFC3339](https://tools.ietf.org/html/rfc3339#section-5.6) format with optional nano seconds)
	// example: 2026-11-17T04:22:55Z
	Date time.Time
	// Message tags
	// example: ["tag1", "tag2"]
	Tags []string
	// Username used for authentication (if provided) with the SMTP or Send API
	// example: "username@example.com"
	Username string
	// Message body text
	// example: "This is the body text of the message."
	Text string
	// Message body HTML
	// example: "<p>This is the body HTML of the message.</p>"
	HTML string
	// Message size in bytes
	// example: 1024
	Size uint64
	// Inline message attachments
	Inline []Attachment
	// Message attachments
	Attachments []Attachment
}

// AttachmentChecksums are the checksums of an attachment file
//
// openapi:model AttachmentChecksums
type AttachmentChecksums struct {
	// MD5 checksum hash of file
	// example: "adae40ffa3f6b0b6492e1b02f1760052"
	MD5 string
	// SHA1 checksum hash of file
	// example: "37c219db300060d15a2a6215b7b1fa523d5365bb"
	SHA1 string
	// SHA256 checksum hash of file
	// example: "1f752e4b2e8840153c9e74f37f099cb2de9d815a179a3abbb487ffc489de8e8c"
	SHA256 string
}

// Attachment struct for inline images and attachments
//
// openapi:model Attachment
type Attachment struct {
	// Attachment part ID
	// example: "1.2"
	PartID string
	// File name
	// example: "document.pdf"
	FileName string
	// Content type
	// example: "application/pdf"
	ContentType string
	// Content ID
	// example: "bd940e14-cb97-4f22-bf7b-84098013fdb1"
	ContentID string
	// Size in bytes
	// example: 1024
	Size uint64
	// File checksums
	Checksums AttachmentChecksums
}

// MessageSummary struct for frontend messages
//
// openapi:model MessageSummary
type MessageSummary struct {
	// Database ID
	// example: "2FBBJL7M7OKnpVWZgh2ped"
	ID string
	// Message ID
	// example: "3UUdoPhRumhsHven3aBTxj"
	MessageID string
	// Read status
	// example: false
	Read bool
	// From address
	// example: {"Name": "John Doe", "Address": "from@example.com"}
	From *mail.Address
	// To addresses
	// example: [{"Name": "John Doe", "Address": "to@example.com"}]
	To []*mail.Address
	// Cc addresses
	// example: [{"Name": "John Doe", "Address": "cc@example.com"}]
	Cc []*mail.Address
	// Bcc addresses
	// example: [{"Name": "John Doe", "Address": "bcc@example.com"}]
	Bcc []*mail.Address
	// ReplyTo addresses
	// example: [{"Name": "John Doe", "Address": "reply-to@example.com"}]
	ReplyTo []*mail.Address
	// Email subject
	// example: "Message subject"
	Subject string
	// Received RFC3339Nano date & time ([extended RFC3339](https://tools.ietf.org/html/rfc3339#section-5.6) format with optional nano seconds)
	// example: 2026-11-17T04:22:55Z
	Created time.Time
	// Username used for authentication (if provided) with the SMTP or Send API
	// example: "username@example.com"
	Username string
	// Message tags
	// example: ["tag1", "tag2"]
	Tags []string
	// Message size in bytes (total)
	// example: 1024
	Size uint64
	// Whether the message has any attachments
	// example: 1
	Attachments int
	// Message snippet includes up to 250 characters
	// example: "This is a message snippet that includes up to 250 characters."
	Snippet string
}

// MailboxStats struct for quick mailbox total/read lookups
type MailboxStats struct {
	Total  uint64
	Unread uint64
	Tags   []string
}

// Metadata struct for storing message metadata.
//
// Field searches (eg: `from:`) use a case-folded copy of this in the mailbox MetaSearchText
// column, so any code that writes Metadata must also write MetaSearchText (see
// createMetaSearchText), otherwise searches will match stale values.
type Metadata struct {
	From     *mail.Address   `json:"From,omitempty"`
	To       []*mail.Address `json:"To,omitempty"`
	Cc       []*mail.Address `json:"Cc,omitempty"`
	Bcc      []*mail.Address `json:"Bcc,omitempty"`
	ReplyTo  []*mail.Address `json:"ReplyTo,omitempty"`
	Username string          `json:"Username,omitempty"`
}

// ListUnsubscribe contains a summary of List-Unsubscribe & List-Unsubscribe-Post headers
// including validation of the link structure
//
// openapi:model ListUnsubscribe
type ListUnsubscribe struct {
	// List-Unsubscribe header value
	// example: "<mailto:unsubscribe@example.com>, <https://example.com/unsubscribe>"
	Header string
	// Detected links, maximum one email and one HTTP(S) link
	// example: ["mailto:unsubscribe@example.com", "https://example.com/unsubscribe"]
	Links []string
	// Validation errors (if any)
	// example: ""
	Errors string
	// List-Unsubscribe-Post value (if set)
	// example: "List-Unsubscribe=One-Click"
	HeaderPost string
}
