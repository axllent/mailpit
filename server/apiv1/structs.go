package apiv1

import (
	"github.com/axllent/mailpit/internal/storage"
)

// The following structs & aliases are provided for easy import
// and understanding of the JSON structure.

// MessageSummary - summary of a single message
type MessageSummary = storage.MessageSummary

// Message data
type Message = storage.Message

// Attachment summary
type Attachment = storage.Attachment
