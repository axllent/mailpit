package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/axllent/mailpit/mcp/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListMessagesArgs are the arguments for list_messages.
type ListMessagesArgs struct {
	Start int `json:"start,omitempty" jsonschema:"description=Pagination offset (default: 0)"`
	Limit int `json:"limit,omitempty" jsonschema:"description=Number of messages to return (default: 50)"`
}

// RegisterListMessages registers the list_messages tool.
func RegisterListMessages(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_messages",
		Description: "List messages from Mailpit inbox with pagination, ordered from newest to oldest",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[ListMessagesArgs]) (*mcp.CallToolResultFor[any], error) {
		result, err := c.ListMessages(ctx, params.Arguments.Start, params.Arguments.Limit)
		if err != nil {
			return errorResult(err), nil
		}
		return jsonResultWithSummary(result), nil
	})
}

// SearchMessagesArgs are the arguments for search_messages.
type SearchMessagesArgs struct {
	Query    string `json:"query" jsonschema:"description=Search query using Mailpit search syntax (e.g. from:user@example.com subject:test is:unread has:attachment)"`
	Start    int    `json:"start,omitempty" jsonschema:"description=Pagination offset (default: 0)"`
	Limit    int    `json:"limit,omitempty" jsonschema:"description=Number of messages to return (default: 50)"`
	Timezone string `json:"timezone,omitempty" jsonschema:"description=Timezone for before:/after: filters (e.g. America/New_York)"`
}

// RegisterSearchMessages registers the search_messages tool.
func RegisterSearchMessages(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_messages",
		Description: "Search messages using Mailpit search syntax. Supports: from:, to:, subject:, message-id:, tag:, is:read/unread, has:attachment, before:, after:",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[SearchMessagesArgs]) (*mcp.CallToolResultFor[any], error) {
		if params.Arguments.Query == "" {
			return errorResult(fmt.Errorf("query is required")), nil
		}
		result, err := c.SearchMessages(ctx, params.Arguments.Query, params.Arguments.Start, params.Arguments.Limit, params.Arguments.Timezone)
		if err != nil {
			return errorResult(err), nil
		}
		return jsonResultWithSummary(result), nil
	})
}

// GetMessageArgs are the arguments for get_message.
type GetMessageArgs struct {
	ID string `json:"id" jsonschema:"description=Message database ID or 'latest' for the most recent message"`
}

// RegisterGetMessage registers the get_message tool.
func RegisterGetMessage(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_message",
		Description: "Get full details of a specific message including headers, body, and attachments. Marks the message as read.",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[GetMessageArgs]) (*mcp.CallToolResultFor[any], error) {
		if params.Arguments.ID == "" {
			return errorResult(fmt.Errorf("id is required")), nil
		}
		result, err := c.GetMessage(ctx, params.Arguments.ID)
		if err != nil {
			return errorResult(err), nil
		}
		return jsonResultWithMessage(result), nil
	})
}

// GetMessageHeadersArgs are the arguments for get_message_headers.
type GetMessageHeadersArgs struct {
	ID string `json:"id" jsonschema:"description=Message database ID or 'latest' for the most recent message"`
}

// RegisterGetMessageHeaders registers the get_message_headers tool.
func RegisterGetMessageHeaders(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_message_headers",
		Description: "Get all headers of a specific message as key-value pairs",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[GetMessageHeadersArgs]) (*mcp.CallToolResultFor[any], error) {
		if params.Arguments.ID == "" {
			return errorResult(fmt.Errorf("id is required")), nil
		}
		result, err := c.GetMessageHeaders(ctx, params.Arguments.ID)
		if err != nil {
			return errorResult(err), nil
		}
		r, err := jsonResult(result)
		if err != nil {
			return errorResult(err), nil
		}
		return r, nil
	})
}

// GetMessageSourceArgs are the arguments for get_message_source.
type GetMessageSourceArgs struct {
	ID string `json:"id" jsonschema:"description=Message database ID or 'latest' for the most recent message"`
}

// RegisterGetMessageSource registers the get_message_source tool.
func RegisterGetMessageSource(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_message_source",
		Description: "Get the raw RFC 2822 source of a message (full email including headers)",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[GetMessageSourceArgs]) (*mcp.CallToolResultFor[any], error) {
		if params.Arguments.ID == "" {
			return errorResult(fmt.Errorf("id is required")), nil
		}
		result, err := c.GetMessageSource(ctx, params.Arguments.ID)
		if err != nil {
			return errorResult(err), nil
		}
		return textResult(result), nil
	})
}

// DeleteMessagesArgs are the arguments for delete_messages.
type DeleteMessagesArgs struct {
	IDs []string `json:"ids,omitempty" jsonschema:"description=Array of message IDs to delete. If empty, ALL messages will be deleted."`
}

// RegisterDeleteMessages registers the delete_messages tool.
func RegisterDeleteMessages(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "delete_messages",
		Description: "Delete specific messages by ID, or all messages if no IDs provided. WARNING: Deletion is permanent.",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[DeleteMessagesArgs]) (*mcp.CallToolResultFor[any], error) {
		err := c.DeleteMessages(ctx, params.Arguments.IDs)
		if err != nil {
			return errorResult(err), nil
		}
		if len(params.Arguments.IDs) == 0 {
			return textResult("All messages deleted successfully"), nil
		}
		return textResult(fmt.Sprintf("Deleted %d message(s) successfully", len(params.Arguments.IDs))), nil
	})
}

// DeleteSearchArgs are the arguments for delete_search.
type DeleteSearchArgs struct {
	Query    string `json:"query" jsonschema:"description=Search query to match messages for deletion"`
	Timezone string `json:"timezone,omitempty" jsonschema:"description=Timezone for before:/after: filters"`
}

// RegisterDeleteSearch registers the delete_search tool.
func RegisterDeleteSearch(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "delete_search",
		Description: "Delete all messages matching a search query. WARNING: Deletion is permanent.",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[DeleteSearchArgs]) (*mcp.CallToolResultFor[any], error) {
		if params.Arguments.Query == "" {
			return errorResult(fmt.Errorf("query is required")), nil
		}
		err := c.DeleteSearch(ctx, params.Arguments.Query, params.Arguments.Timezone)
		if err != nil {
			return errorResult(err), nil
		}
		return textResult(fmt.Sprintf("Deleted messages matching query: %s", params.Arguments.Query)), nil
	})
}

// SetReadStatusArgs are the arguments for set_read_status.
type SetReadStatusArgs struct {
	IDs    []string `json:"ids,omitempty" jsonschema:"description=Array of message IDs to update. If empty and no search provided, updates all messages."`
	Read   bool     `json:"read" jsonschema:"description=Read status to set (true=read, false=unread)"`
	Search string   `json:"search,omitempty" jsonschema:"description=Optional search query to match messages instead of using IDs"`
}

// RegisterSetReadStatus registers the set_read_status tool.
func RegisterSetReadStatus(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "set_read_status",
		Description: "Mark messages as read or unread by IDs, search query, or all messages",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[SetReadStatusArgs]) (*mcp.CallToolResultFor[any], error) {
		err := c.SetReadStatus(ctx, params.Arguments.IDs, params.Arguments.Read, params.Arguments.Search)
		if err != nil {
			return errorResult(err), nil
		}
		status := "read"
		if !params.Arguments.Read {
			status = "unread"
		}
		return textResult(fmt.Sprintf("Messages marked as %s", status)), nil
	})
}

// jsonResultWithSummary formats a MessagesSummary result.
func jsonResultWithSummary(result *client.MessagesSummary) *mcp.CallToolResultFor[any] {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d messages (%d unread) out of %d total\n\n",
		result.MessagesCount, result.MessagesUnreadCount, result.Total))

	for i, msg := range result.Messages {
		from := "unknown"
		if msg.From != nil {
			from = formatAddress(msg.From.Name, msg.From.Address)
		}
		status := "○"
		if msg.Read {
			status = "●"
		}
		sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1+result.Start, status, msg.Subject))
		sb.WriteString(fmt.Sprintf("   ID: %s\n", msg.ID))
		sb.WriteString(fmt.Sprintf("   From: %s\n", from))
		sb.WriteString(fmt.Sprintf("   Date: %s | Size: %s", msg.Created.Format("2006-01-02 15:04:05"), formatSize(msg.Size)))
		if msg.Attachments > 0 {
			sb.WriteString(fmt.Sprintf(" | Attachments: %d", msg.Attachments))
		}
		if len(msg.Tags) > 0 {
			sb.WriteString(fmt.Sprintf(" | Tags: %s", strings.Join(msg.Tags, ", ")))
		}
		sb.WriteString("\n\n")
	}

	return textResult(sb.String())
}

// jsonResultWithMessage formats a Message result.
func jsonResultWithMessage(msg *client.Message) *mcp.CallToolResultFor[any] {
	var sb strings.Builder

	// Header section
	sb.WriteString(fmt.Sprintf("Subject: %s\n", msg.Subject))
	sb.WriteString(fmt.Sprintf("ID: %s\n", msg.ID))
	sb.WriteString(fmt.Sprintf("Message-ID: %s\n", msg.MessageID))
	sb.WriteString(fmt.Sprintf("Date: %s\n", msg.Date.Format("2006-01-02 15:04:05 MST")))
	sb.WriteString(fmt.Sprintf("Size: %s\n\n", formatSize(msg.Size)))

	// Addresses
	if msg.From != nil {
		sb.WriteString(fmt.Sprintf("From: %s\n", formatAddress(msg.From.Name, msg.From.Address)))
	}
	if len(msg.To) > 0 {
		addrs := make([]string, len(msg.To))
		for i, a := range msg.To {
			addrs[i] = formatAddress(a.Name, a.Address)
		}
		sb.WriteString(fmt.Sprintf("To: %s\n", strings.Join(addrs, ", ")))
	}
	if len(msg.Cc) > 0 {
		addrs := make([]string, len(msg.Cc))
		for i, a := range msg.Cc {
			addrs[i] = formatAddress(a.Name, a.Address)
		}
		sb.WriteString(fmt.Sprintf("Cc: %s\n", strings.Join(addrs, ", ")))
	}
	if len(msg.ReplyTo) > 0 {
		addrs := make([]string, len(msg.ReplyTo))
		for i, a := range msg.ReplyTo {
			addrs[i] = formatAddress(a.Name, a.Address)
		}
		sb.WriteString(fmt.Sprintf("Reply-To: %s\n", strings.Join(addrs, ", ")))
	}
	if msg.ReturnPath != "" {
		sb.WriteString(fmt.Sprintf("Return-Path: %s\n", msg.ReturnPath))
	}

	// Tags
	if len(msg.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("\nTags: %s\n", strings.Join(msg.Tags, ", ")))
	}

	// Attachments
	if len(msg.Attachments) > 0 {
		sb.WriteString(fmt.Sprintf("\nAttachments (%d):\n", len(msg.Attachments)))
		for _, a := range msg.Attachments {
			sb.WriteString(fmt.Sprintf("  - %s (%s, %s, PartID: %s)\n", a.FileName, a.ContentType, formatSize(a.Size), a.PartID))
		}
	}
	if len(msg.Inline) > 0 {
		sb.WriteString(fmt.Sprintf("\nInline attachments (%d):\n", len(msg.Inline)))
		for _, a := range msg.Inline {
			sb.WriteString(fmt.Sprintf("  - %s (%s, %s, PartID: %s)\n", a.FileName, a.ContentType, formatSize(a.Size), a.PartID))
		}
	}

	// Body content
	sb.WriteString("\n--- Text Body ---\n")
	if msg.Text != "" {
		sb.WriteString(msg.Text)
	} else {
		sb.WriteString("(empty)")
	}
	sb.WriteString("\n\n--- HTML Body ---\n")
	if msg.HTML != "" {
		// Truncate HTML for readability
		html := msg.HTML
		if len(html) > 5000 {
			html = html[:5000] + "\n... (truncated, use get_message_html for full content)"
		}
		sb.WriteString(html)
	} else {
		sb.WriteString("(empty)")
	}

	return textResult(sb.String())
}

// RegisterAllMessageTools registers all message-related tools.
func RegisterAllMessageTools(s *mcp.Server, c *client.Client) {
	RegisterListMessages(s, c)
	RegisterSearchMessages(s, c)
	RegisterGetMessage(s, c)
	RegisterGetMessageHeaders(s, c)
	RegisterGetMessageSource(s, c)
	RegisterDeleteMessages(s, c)
	RegisterDeleteSearch(s, c)
	RegisterSetReadStatus(s, c)
}
