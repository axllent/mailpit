package storage

import (
	"net/mail"
	"time"

	"github.com/jhillyerd/enmime"
)

// Message struct for loading messages. It does not include physical attachments.
type Message struct {
	ID          string
	Read        bool
	From        *mail.Address
	To          []*mail.Address
	Cc          []*mail.Address
	Bcc         []*mail.Address
	Subject     string
	Date        time.Time
	Text        string
	HTML        string
	Size        int
	Inline      []Attachment
	Attachments []Attachment
}

// Attachment struct for inline and attachments
type Attachment struct {
	PartID      string
	FileName    string
	ContentType string
	ContentID   string
	Size        int
}

// MessageSummary struct for frontend messages
type MessageSummary struct {
	ID          string
	Read        bool
	From        *mail.Address
	To          []*mail.Address
	Cc          []*mail.Address
	Bcc         []*mail.Address
	Subject     string
	Created     time.Time
	Size        int
	Attachments int
}

// MailboxStats struct for quick mailbox total/read lookups
type MailboxStats struct {
	Total  int
	Unread int
}

// AttachmentSummary returns a summary of the attachment without any binary data
func AttachmentSummary(a *enmime.Part) Attachment {
	o := Attachment{}
	o.PartID = a.PartID
	o.FileName = a.FileName
	if o.FileName == "" {
		o.FileName = a.ContentID
	}
	o.ContentType = a.ContentType
	o.ContentID = a.ContentID
	o.Size = len(a.Content)

	return o
}
