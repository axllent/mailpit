// Package client provides a typed HTTP client for the Mailpit API.
package client

import "time"

// Address represents an email address with optional name.
type Address struct {
	Name    string `json:"Name,omitempty"`
	Address string `json:"Address"`
}

// Attachment represents an email attachment.
type Attachment struct {
	PartID      string            `json:"PartID"`
	FileName    string            `json:"FileName"`
	ContentType string            `json:"ContentType"`
	ContentID   string            `json:"ContentID,omitempty"`
	Size        uint64            `json:"Size"`
	Checksums   map[string]string `json:"Checksums,omitempty"`
}

// ListUnsubscribe contains List-Unsubscribe header information.
type ListUnsubscribe struct {
	Header     string   `json:"Header,omitempty"`
	HeaderPost string   `json:"HeaderPost,omitempty"`
	Links      []string `json:"Links,omitempty"`
	Errors     string   `json:"Errors,omitempty"`
}

// Message represents a full email message.
type Message struct {
	ID              string           `json:"ID"`
	MessageID       string           `json:"MessageID"`
	From            *Address         `json:"From"`
	To              []*Address       `json:"To"`
	Cc              []*Address       `json:"Cc,omitempty"`
	Bcc             []*Address       `json:"Bcc,omitempty"`
	ReplyTo         []*Address       `json:"ReplyTo,omitempty"`
	Subject         string           `json:"Subject"`
	Date            time.Time        `json:"Date"`
	Tags            []string         `json:"Tags,omitempty"`
	Text            string           `json:"Text,omitempty"`
	HTML            string           `json:"HTML,omitempty"`
	Size            uint64           `json:"Size"`
	Attachments     []*Attachment    `json:"Attachments,omitempty"`
	Inline          []*Attachment    `json:"Inline,omitempty"`
	ListUnsubscribe *ListUnsubscribe `json:"ListUnsubscribe,omitempty"`
	ReturnPath      string           `json:"ReturnPath,omitempty"`
	Username        string           `json:"Username,omitempty"`
}

// MessageSummary is a lightweight message representation.
type MessageSummary struct {
	ID          string     `json:"ID"`
	MessageID   string     `json:"MessageID"`
	From        *Address   `json:"From"`
	To          []*Address `json:"To"`
	Cc          []*Address `json:"Cc,omitempty"`
	Bcc         []*Address `json:"Bcc,omitempty"`
	ReplyTo     []*Address `json:"ReplyTo,omitempty"`
	Subject     string     `json:"Subject"`
	Created     time.Time  `json:"Created"`
	Tags        []string   `json:"Tags,omitempty"`
	Size        uint64     `json:"Size"`
	Attachments int        `json:"Attachments"`
	Read        bool       `json:"Read"`
	Snippet     string     `json:"Snippet,omitempty"`
	Username    string     `json:"Username,omitempty"`
}

// MessagesSummary represents a paginated list of messages.
type MessagesSummary struct {
	Total               uint64            `json:"total"`
	Unread              uint64            `json:"unread"`
	MessagesCount       uint64            `json:"messages_count"`
	MessagesUnreadCount uint64            `json:"messages_unread"`
	Start               int               `json:"start"`
	Tags                []string          `json:"tags"`
	Messages            []*MessageSummary `json:"messages"`
}

// RuntimeStats contains server runtime statistics.
type RuntimeStats struct {
	Uptime           uint64 `json:"Uptime"`
	Memory           uint64 `json:"Memory"`
	SMTPAccepted     uint64 `json:"SMTPAccepted"`
	SMTPAcceptedSize uint64 `json:"SMTPAcceptedSize"`
	SMTPRejected     uint64 `json:"SMTPRejected"`
	SMTPIgnored      uint64 `json:"SMTPIgnored"`
	MessagesDeleted  uint64 `json:"MessagesDeleted"`
}

// AppInfo represents application information and statistics.
type AppInfo struct {
	Version       string           `json:"Version"`
	LatestVersion string           `json:"LatestVersion,omitempty"`
	Database      string           `json:"Database"`
	DatabaseSize  uint64           `json:"DatabaseSize"`
	Messages      uint64           `json:"Messages"`
	Unread        uint64           `json:"Unread"`
	Tags          map[string]int64 `json:"Tags,omitempty"`
	RuntimeStats  *RuntimeStats    `json:"RuntimeStats,omitempty"`
}

// MessageRelay contains relay configuration.
type MessageRelay struct {
	Enabled            bool   `json:"Enabled"`
	SMTPServer         string `json:"SMTPServer,omitempty"`
	AllowedRecipients  string `json:"AllowedRecipients,omitempty"`
	BlockedRecipients  string `json:"BlockedRecipients,omitempty"`
	OverrideFrom       string `json:"OverrideFrom,omitempty"`
	ReturnPath         string `json:"ReturnPath,omitempty"`
	PreserveMessageIDs bool   `json:"PreserveMessageIDs,omitempty"`
}

// WebUIConfig contains web UI configuration.
type WebUIConfig struct {
	Label               string        `json:"Label,omitempty"`
	SpamAssassin        bool          `json:"SpamAssassin"`
	ChaosEnabled        bool          `json:"ChaosEnabled"`
	DuplicatesIgnored   bool          `json:"DuplicatesIgnored"`
	HideDeleteAllButton bool          `json:"HideDeleteAllButton,omitempty"`
	MessageRelay        *MessageRelay `json:"MessageRelay,omitempty"`
}

// HTMLCheckScore represents compatibility scoring.
type HTMLCheckScore struct {
	Found       int     `json:"Found"`
	Supported   float64 `json:"Supported"`
	Partial     float64 `json:"Partial"`
	Unsupported float64 `json:"Unsupported"`
}

// HTMLCheckResult represents a single client compatibility result.
type HTMLCheckResult struct {
	Family     string `json:"Family"`
	Platform   string `json:"Platform"`
	Version    string `json:"Version,omitempty"`
	Name       string `json:"Name"`
	Support    string `json:"Support"`
	NoteNumber string `json:"NoteNumber,omitempty"`
}

// HTMLCheckWarning represents a compatibility warning.
type HTMLCheckWarning struct {
	Slug          string             `json:"Slug"`
	Title         string             `json:"Title"`
	Description   string             `json:"Description,omitempty"`
	URL           string             `json:"URL,omitempty"`
	Category      string             `json:"Category"`
	Tags          []string           `json:"Tags,omitempty"`
	Keywords      string             `json:"Keywords,omitempty"`
	Score         *HTMLCheckScore    `json:"Score,omitempty"`
	Results       []*HTMLCheckResult `json:"Results,omitempty"`
	NotesByNumber map[string]string  `json:"NotesByNumber,omitempty"`
}

// HTMLCheckTotal represents total compatibility scores.
type HTMLCheckTotal struct {
	Nodes       int     `json:"Nodes"`
	Tests       int     `json:"Tests"`
	Supported   float64 `json:"Supported"`
	Partial     float64 `json:"Partial"`
	Unsupported float64 `json:"Unsupported"`
}

// HTMLCheckResponse is the response from HTML compatibility checking.
type HTMLCheckResponse struct {
	Total     *HTMLCheckTotal     `json:"Total,omitempty"`
	Warnings  []*HTMLCheckWarning `json:"Warnings,omitempty"`
	Platforms map[string][]string `json:"Platforms,omitempty"`
}

// Link represents a checked link.
type Link struct {
	URL        string `json:"URL"`
	StatusCode int    `json:"StatusCode"`
	Status     string `json:"Status"`
}

// LinkCheckResponse is the response from link checking.
type LinkCheckResponse struct {
	Errors int     `json:"Errors"`
	Links  []*Link `json:"Links,omitempty"`
}

// SpamRule represents a SpamAssassin rule.
type SpamRule struct {
	Name        string  `json:"Name"`
	Score       float64 `json:"Score"`
	Description string  `json:"Description,omitempty"`
}

// SpamAssassinResponse is the response from spam checking.
type SpamAssassinResponse struct {
	IsSpam bool        `json:"IsSpam"`
	Score  float64     `json:"Score"`
	Rules  []*SpamRule `json:"Rules,omitempty"`
	Error  string      `json:"Error,omitempty"`
}

// ChaosTrigger represents a chaos testing trigger.
type ChaosTrigger struct {
	Probability int `json:"Probability"`
	ErrorCode   int `json:"ErrorCode"`
}

// ChaosTriggers contains all chaos triggers.
type ChaosTriggers struct {
	Sender         *ChaosTrigger `json:"Sender,omitempty"`
	Recipient      *ChaosTrigger `json:"Recipient,omitempty"`
	Authentication *ChaosTrigger `json:"Authentication,omitempty"`
}

// SendMessageRequest is the request body for sending a message.
type SendMessageRequest struct {
	From        *SendAddress      `json:"From"`
	To          []*SendAddress    `json:"To,omitempty"`
	Cc          []*SendAddress    `json:"Cc,omitempty"`
	Bcc         []string          `json:"Bcc,omitempty"`
	ReplyTo     []*SendAddress    `json:"ReplyTo,omitempty"`
	Subject     string            `json:"Subject,omitempty"`
	Text        string            `json:"Text,omitempty"`
	HTML        string            `json:"HTML,omitempty"`
	Attachments []*SendAttachment `json:"Attachments,omitempty"`
	Tags        []string          `json:"Tags,omitempty"`
	Headers     map[string]string `json:"Headers,omitempty"`
}

// SendAddress is an address for sending messages.
type SendAddress struct {
	Email string `json:"Email"`
	Name  string `json:"Name,omitempty"`
}

// SendAttachment is an attachment for sending messages.
type SendAttachment struct {
	Filename    string `json:"Filename"`
	Content     string `json:"Content"`
	ContentType string `json:"ContentType,omitempty"`
	ContentID   string `json:"ContentID,omitempty"`
}

// SendMessageResponse is the response from sending a message.
type SendMessageResponse struct {
	ID string `json:"ID"`
}

// ReleaseRequest is the request body for releasing a message.
type ReleaseRequest struct {
	To []string `json:"To"`
}

// SetTagsRequest is the request body for setting tags.
type SetTagsRequest struct {
	IDs  []string `json:"IDs"`
	Tags []string `json:"Tags"`
}

// RenameTagRequest is the request body for renaming a tag.
type RenameTagRequest struct {
	Name string `json:"Name"`
}

// SetReadStatusRequest is the request body for setting read status.
type SetReadStatusRequest struct {
	IDs    []string `json:"IDs,omitempty"`
	Read   bool     `json:"Read"`
	Search string   `json:"Search,omitempty"`
}

// DeleteMessagesRequest is the request body for deleting messages.
type DeleteMessagesRequest struct {
	IDs []string `json:"IDs,omitempty"`
}

// MessageHeaders is a map of header name to values.
type MessageHeaders map[string][]string
