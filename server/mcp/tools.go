package mcpserver

import (
	"context"
	"fmt"
	"net/mail"
	"time"

	"github.com/axllent/mailpit/internal/htmlcheck"
	"github.com/axllent/mailpit/internal/linkcheck"
	"github.com/axllent/mailpit/internal/spamassassin"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListMessagesInput represents input for listing messages
type ListMessagesInput struct {
	Limit  int    `json:"limit" jsonschema:"maximum number of messages to return"`
	Search string `json:"search,omitempty" jsonschema:"search query"`
	Tag    string `json:"tag,omitempty" jsonschema:"filter by tag"`
}

// ListMessagesOutput represents output for listing messages
type ListMessagesOutput struct {
	Messages []MessageSummary `json:"messages"`
	Total    int              `json:"total"`
}

// MessageSummary represents a message summary
type MessageSummary struct {
	ID      string    `json:"id"`
	From    string    `json:"from"`
	To      []string  `json:"to"`
	Subject string    `json:"subject"`
	Date    time.Time `json:"date"`
	Tags    []string  `json:"tags"`
	Size    int       `json:"size"`
	Read    bool      `json:"read"`
}

// GetMessageInput represents input for getting a message
type GetMessageInput struct {
	ID         string `json:"id" jsonschema:"required,message ID"`
	IncludeRaw bool   `json:"includeRaw,omitempty" jsonschema:"include raw message"`
}

// GetMessageOutput represents output for getting a message
type GetMessageOutput struct {
	ID          string            `json:"id"`
	From        string            `json:"from"`
	To          []string          `json:"to"`
	Cc          []string          `json:"cc,omitempty"`
	Bcc         []string          `json:"bcc,omitempty"`
	Subject     string            `json:"subject"`
	Date        time.Time         `json:"date"`
	HTMLBody    string            `json:"htmlBody,omitempty"`
	TextBody    string            `json:"textBody,omitempty"`
	Headers     map[string]string `json:"headers"`
	Attachments []AttachmentInfo  `json:"attachments"`
	Tags        []string          `json:"tags"`
	Raw         string            `json:"raw,omitempty"`
}

// AttachmentInfo represents attachment information
type AttachmentInfo struct {
	PartID      string `json:"partId"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	Size        int    `json:"size"`
}

// SearchMessagesInput represents input for searching messages
type SearchMessagesInput struct {
	Query    string    `json:"query" jsonschema:"required,search query"`
	DateFrom time.Time `json:"dateFrom,omitempty"`
	DateTo   time.Time `json:"dateTo,omitempty"`
	Limit    int       `json:"limit,omitempty"`
}

// SearchMessagesOutput represents output for searching messages
type SearchMessagesOutput struct {
	Results []MessageSummary `json:"results"`
	Total   int              `json:"total"`
}

// AnalyzeMessageInput represents input for analyzing a message
type AnalyzeMessageInput struct {
	ID string `json:"id" jsonschema:"required,message ID"`
}

// AnalyzeMessageOutput represents output for analyzing a message
type AnalyzeMessageOutput struct {
	ID             string                 `json:"id"`
	HTMLCheck      *HTMLCheckResult       `json:"htmlCheck,omitempty"`
	LinkCheck      *LinkCheckResult       `json:"linkCheck,omitempty"`
	SpamScore      *float64               `json:"spamScore,omitempty"`
	SpamDetails    map[string]interface{} `json:"spamDetails,omitempty"`
	Deliverability string                 `json:"deliverability"`
}

// HTMLCheckResult represents HTML check results
type HTMLCheckResult struct {
	Score       float32                 `json:"score"`
	Summary     string                  `json:"summary"`
	Issues      []string                `json:"issues"`
	ClientStats map[string]interface{} `json:"clientStats"`
}

// LinkCheckResult represents link check results
type LinkCheckResult struct {
	TotalLinks   int      `json:"totalLinks"`
	BrokenLinks  []string `json:"brokenLinks"`
	ValidLinks   []string `json:"validLinks"`
	Warnings     []string `json:"warnings"`
}

// ListMessages lists recent messages in Mailpit
func ListMessages(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[ListMessagesInput]) (*mcp.CallToolResultFor[ListMessagesOutput], error) {
	input := params.Arguments
	// Default limit
	if input.Limit == 0 {
		input.Limit = 50
	}
	if input.Limit > 500 {
		input.Limit = 500
	}

	// Build search query
	search := input.Search
	if input.Tag != "" {
		if search != "" {
			search += " AND "
		}
		search += fmt.Sprintf("tag:%s", input.Tag)
	}

	// Query storage
	messages, total, err := storage.Search(search, "", 0, 0, input.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %v", err)
	}

	// Convert to output format
	summaries := make([]MessageSummary, len(messages))
	for i, msg := range messages {
		summaries[i] = MessageSummary{
			ID:      msg.ID,
			From:    msg.From.String(),
			To:      convertAddresses(msg.To),
			Subject: msg.Subject,
			Date:    msg.Created,
			Tags:    msg.Tags,
			Size:    int(msg.Size),
			Read:    msg.Read,
		}
	}

	output := ListMessagesOutput{
		Messages: summaries,
		Total:    total,
	}

	return &mcp.CallToolResultFor[ListMessagesOutput]{StructuredContent: output}, nil
}

// GetMessage retrieves a specific message by ID
func GetMessage(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[GetMessageInput]) (*mcp.CallToolResultFor[GetMessageOutput], error) {
	input := params.Arguments
	// Get message from storage
	msg, err := storage.GetMessage(input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %v", err)
	}

	output := GetMessageOutput{
		ID:       msg.ID,
		From:     msg.From.String(),
		To:       convertAddresses(msg.To),
		Cc:       convertAddresses(msg.Cc),
		Bcc:      convertAddresses(msg.Bcc),
		Subject:  msg.Subject,
		Date:     msg.Date,
		HTMLBody: msg.HTML,
		TextBody: msg.Text,
		Headers:  convertHeaders(*msg),
		Tags:     msg.Tags,
	}

	// Add attachments
	output.Attachments = make([]AttachmentInfo, len(msg.Attachments))
	for i, att := range msg.Attachments {
		output.Attachments[i] = AttachmentInfo{
			PartID:      att.PartID,
			FileName:    att.FileName,
			ContentType: att.ContentType,
			Size:        int(att.Size),
		}
	}

	// Include raw message if requested
	if input.IncludeRaw {
		raw, err := storage.GetMessageRaw(input.ID)
		if err == nil {
			output.Raw = string(raw)
		}
	}

	return &mcp.CallToolResultFor[GetMessageOutput]{StructuredContent: output}, nil
}

// SearchMessages searches for messages
func SearchMessages(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[SearchMessagesInput]) (*mcp.CallToolResultFor[SearchMessagesOutput], error) {
	input := params.Arguments
	// Default limit
	limit := input.Limit
	if limit == 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	// Build search query with date filters
	query := input.Query
	if !input.DateFrom.IsZero() {
		query += fmt.Sprintf(" after:%s", input.DateFrom.Format("2006-01-02"))
	}
	if !input.DateTo.IsZero() {
		query += fmt.Sprintf(" before:%s", input.DateTo.Format("2006-01-02"))
	}

	// Search messages
	messages, total, err := storage.Search(query, "", 0, 0, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %v", err)
	}

	// Convert to output format
	results := make([]MessageSummary, len(messages))
	for i, msg := range messages {
		results[i] = MessageSummary{
			ID:      msg.ID,
			From:    msg.From.String(),
			To:      convertAddresses(msg.To),
			Subject: msg.Subject,
			Date:    msg.Created,
			Tags:    msg.Tags,
			Size:    int(msg.Size),
			Read:    msg.Read,
		}
	}

	output := SearchMessagesOutput{
		Results: results,
		Total:   total,
	}

	return &mcp.CallToolResultFor[SearchMessagesOutput]{StructuredContent: output}, nil
}

// AnalyzeMessage analyzes a message for various checks
func AnalyzeMessage(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[AnalyzeMessageInput]) (*mcp.CallToolResultFor[AnalyzeMessageOutput], error) {
	input := params.Arguments
	// Get message from storage
	msg, err := storage.GetMessage(input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %v", err)
	}

	output := AnalyzeMessageOutput{
		ID:             input.ID,
		Deliverability: "good", // default
	}

	// HTML check if HTML content exists
	if msg.HTML != "" {
		htmlResult, err := htmlcheck.RunTests(msg.HTML)
		if err == nil {
			output.HTMLCheck = &HTMLCheckResult{
				Score: htmlResult.Total.Supported,
			}
			
			// Extract warnings as issues
			issues := []string{}
			for _, warning := range htmlResult.Warnings {
				issues = append(issues, fmt.Sprintf("%s: %s", warning.Title, warning.Description))
			}
			if len(issues) > 0 {
				output.HTMLCheck.Issues = issues
				output.HTMLCheck.Summary = fmt.Sprintf("%d HTML compatibility issues found", len(issues))
			} else {
				output.HTMLCheck.Summary = "No HTML compatibility issues"
			}

			// Update deliverability based on score
			if htmlResult.Total.Supported < 50 {
				output.Deliverability = "poor"
			} else if htmlResult.Total.Supported < 80 {
				output.Deliverability = "fair"
			}
		}
	}

	// Link check
	linkResult, err := linkcheck.RunTests(msg, false)
	if err == nil {
		output.LinkCheck = &LinkCheckResult{
			TotalLinks: len(linkResult.Links),
		}

		// Extract broken and valid links
		brokenLinks := []string{}
		validLinks := []string{}
		warnings := []string{}

		for _, link := range linkResult.Links {
			if link.StatusCode >= 400 || link.StatusCode == 0 {
				brokenLinks = append(brokenLinks, link.URL)
			} else if link.StatusCode >= 200 && link.StatusCode < 400 {
				validLinks = append(validLinks, link.URL)
			}
		}

		output.LinkCheck.BrokenLinks = brokenLinks
		output.LinkCheck.ValidLinks = validLinks
		output.LinkCheck.Warnings = warnings

		// Update deliverability
		if len(brokenLinks) > 0 {
			if output.Deliverability == "good" {
				output.Deliverability = "fair"
			}
		}
	}

	// SpamAssassin check (skip if not configured)
	raw, err := storage.GetMessageRaw(input.ID)
	if err == nil {
		result, err := spamassassin.Check(raw)
		if err == nil {
			score := result.Score
			if score > 0 {
				output.SpamScore = &score
				output.SpamDetails = map[string]interface{}{
					"rules":  result.Rules,
					"isSpam": result.IsSpam,
				}

				// Update deliverability based on spam score
				if score > 10 {
					output.Deliverability = "spam"
				} else if score > 5 {
					output.Deliverability = "poor"
				} else if score > 2 && output.Deliverability == "good" {
					output.Deliverability = "fair"
				}
			}
		}
	}

	return &mcp.CallToolResultFor[AnalyzeMessageOutput]{StructuredContent: output}, nil
}

// Helper functions

func convertAddresses(addresses []*mail.Address) []string {
	result := make([]string, len(addresses))
	for i, addr := range addresses {
		if addr.Name != "" {
			result[i] = fmt.Sprintf("%s <%s>", addr.Name, addr.Address)
		} else {
			result[i] = addr.Address
		}
	}
	return result
}

func convertHeaders(msg storage.Message) map[string]string {
	headers := make(map[string]string)
	
	// Add common headers
	headers["From"] = msg.From.String()
	headers["Subject"] = msg.Subject
	headers["Date"] = msg.Date.Format(time.RFC1123Z)
	headers["Message-ID"] = msg.MessageID
	
	if len(msg.To) > 0 {
		headers["To"] = joinAddresses(msg.To)
	}
	if len(msg.Cc) > 0 {
		headers["Cc"] = joinAddresses(msg.Cc)
	}
	if len(msg.ReplyTo) > 0 {
		headers["Reply-To"] = joinAddresses(msg.ReplyTo)
	}
	
	return headers
}

func joinAddresses(addresses []*mail.Address) string {
	result := ""
	for i, addr := range addresses {
		if i > 0 {
			result += ", "
		}
		if addr.Name != "" {
			result += fmt.Sprintf("%s <%s>", addr.Name, addr.Address)
		} else {
			result += addr.Address
		}
	}
	return result
}