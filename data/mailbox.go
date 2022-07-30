package data

import "time"

// MailboxSummary struct
type MailboxSummary struct {
	Name        string
	Slug        string
	Total       int
	Unread      int
	LastMessage time.Time
}

// WebsocketNotification struct for responses
type WebsocketNotification struct {
	Type string
	Data interface{}
}

// MailboxStats struct for quick mailbox total/read lookups
type MailboxStats struct {
	Total  int
	Unread int
}
