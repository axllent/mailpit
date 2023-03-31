package apiv1

import (
	"github.com/axllent/mailpit/storage"
)

// MessagesSummary is a summary of a list of messages
type MessagesSummary struct {
	// Total number of messages in mailbox
	Total int `json:"total"`

	// Total number of unread messages in mailbox
	Unread int `json:"unread"`

	// Number of results returned
	Count int `json:"count"`

	// Pagination offset
	Start int `json:"start"`

	// All current tags
	Tags []string `json:"tags"`

	// Messages summary
	// in:body
	Messages []storage.MessageSummary `json:"messages"`
}

// The following structs & aliases are provided for easy import
// and understanding of the JSON structure.

// MessageSummary - summary of a single message
type MessageSummary = storage.MessageSummary

// Message data
type Message = storage.Message

// Attachment summary
type Attachment = storage.Attachment
