package postmark

import (
	"time"
)

// PostmarkEmailRequest represents a single email request in Postmark format
type PostmarkEmailRequest struct {
	From          string                 `json:"From"`
	To            string                 `json:"To"`
	Cc            string                 `json:"Cc,omitempty"`
	Bcc           string                 `json:"Bcc,omitempty"`
	Subject       string                 `json:"Subject"`
	Tag           string                 `json:"Tag,omitempty"`
	HtmlBody      string                 `json:"HtmlBody,omitempty"`
	TextBody      string                 `json:"TextBody,omitempty"`
	ReplyTo       string                 `json:"ReplyTo,omitempty"`
	Headers       []PostmarkHeader       `json:"Headers,omitempty"`
	Attachments   []PostmarkAttachment   `json:"Attachments,omitempty"`
	MessageStream string                 `json:"MessageStream,omitempty"`
	Metadata      map[string]string      `json:"Metadata,omitempty"`
}

// PostmarkHeader represents an email header
type PostmarkHeader struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

// PostmarkAttachment represents an email attachment
type PostmarkAttachment struct {
	Name        string `json:"Name"`
	Content     string `json:"Content"`     // Base64 encoded
	ContentType string `json:"ContentType"`
	ContentID   string `json:"ContentID,omitempty"`
}

// PostmarkEmailResponse represents the response for a single email
type PostmarkEmailResponse struct {
	To          string    `json:"To"`
	SubmittedAt time.Time `json:"SubmittedAt"`
	MessageID   string    `json:"MessageID"`
	ErrorCode   int       `json:"ErrorCode"`
	Message     string    `json:"Message"`
}

// PostmarkBatchRequest represents a batch of email requests
type PostmarkBatchRequest []PostmarkEmailRequest

// PostmarkBatchResponse represents the response for a batch of emails
type PostmarkBatchResponse []PostmarkEmailResponse

// PostmarkErrorResponse represents an error response
type PostmarkErrorResponse struct {
	ErrorCode int    `json:"ErrorCode"`
	Message   string `json:"Message"`
}