package apiv1

import "github.com/axllent/mailpit/storage"

// The following structs & aliases are provided for easy import
// and understanding of the JSON structure.

// MessageSummary - summary of a single message
type MessageSummary = storage.MessageSummary

// MessagesSummary - summary of a list of messages
type MessagesSummary struct {
	Total    int              `json:"total"`
	Unread   int              `json:"unread"`
	Count    int              `json:"count"`
	Start    int              `json:"start"`
	Messages []MessageSummary `json:"messages"`
}

// Message data
type Message = storage.Message

// Attachment summary
type Attachment = storage.Attachment
