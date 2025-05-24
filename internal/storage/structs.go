package storage

import (
	"net/mail"
	"time"
)

// Message data excluding physical attachments
//
// swagger:model Message
type Message struct {
	// Database ID
	ID string
	// Message ID
	MessageID string
	// From address
	From *mail.Address
	// To addresses
	To []*mail.Address
	// Cc addresses
	Cc []*mail.Address
	// Bcc addresses
	Bcc []*mail.Address
	// ReplyTo addresses
	ReplyTo []*mail.Address
	// Return-Path
	ReturnPath string
	// Message subject
	Subject string
	// List-Unsubscribe header information
	ListUnsubscribe ListUnsubscribe
	// Message RFC3339Nano date & time (if set), else date & time received
	// ([extended RFC3339](https://tools.ietf.org/html/rfc3339#section-5.6) format with optional nano seconds)
	Date time.Time
	// Message tags
	Tags []string
	// Message body text
	Text string
	// Message body HTML
	HTML string
	// Message size in bytes
	Size uint64
	// Inline message attachments
	Inline []Attachment
	// Message attachments
	Attachments []Attachment
}

// Attachment struct for inline and attachments
//
// swagger:model Attachment
type Attachment struct {
	// Attachment part ID
	PartID string
	// File name
	FileName string
	// Content type
	ContentType string
	// Content ID
	ContentID string
	// Size in bytes
	Size uint64
}

// MessageSummary struct for frontend messages
//
// swagger:model MessageSummary
type MessageSummary struct {
	// Database ID
	ID string
	// Message ID
	MessageID string
	// Read status
	Read bool
	// From address
	From *mail.Address
	// To address
	To []*mail.Address
	// Cc addresses
	Cc []*mail.Address
	// Bcc addresses
	Bcc []*mail.Address
	// Reply-To address
	ReplyTo []*mail.Address
	// Email subject
	Subject string
	// Received RFC3339Nano date & time ([extended RFC3339](https://tools.ietf.org/html/rfc3339#section-5.6) format with optional nano seconds)
	Created time.Time
	// Message tags
	Tags []string
	// Message size in bytes (total)
	Size uint64
	// Whether the message has any attachments
	Attachments int
	// Message snippet includes up to 250 characters
	Snippet string
}

// MailboxStats struct for quick mailbox total/read lookups
type MailboxStats struct {
	Total  uint64
	Unread uint64
	Tags   []string
}

// DBMailSummary struct for storing mail summary
type DBMailSummary struct {
	From    *mail.Address
	To      []*mail.Address
	Cc      []*mail.Address
	Bcc     []*mail.Address
	ReplyTo []*mail.Address
}

// ListUnsubscribe contains a summary of List-Unsubscribe & List-Unsubscribe-Post headers
// including validation of the link structure
type ListUnsubscribe struct {
	// List-Unsubscribe header value
	Header string
	// Detected links, maximum one email and one HTTP(S) link
	Links []string
	// Validation errors (if any)
	Errors string
	// List-Unsubscribe-Post value (if set)
	HeaderPost string
}
