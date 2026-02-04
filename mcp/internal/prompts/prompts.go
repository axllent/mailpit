// Package prompts provides MCP prompt implementations for Mailpit.
package prompts

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterAllPrompts registers all MCP prompts.
func RegisterAllPrompts(s *mcp.Server) {
	registerAnalyzeLatestEmail(s)
	registerDebugEmailDelivery(s)
	registerCheckEmailQuality(s)
	registerSearchEmails(s)
	registerComposeTestEmail(s)
	registerAnalyzeEmailHeaders(s)
	registerCompareEmails(s)
	registerSummarizeInbox(s)
}

// registerAnalyzeLatestEmail registers the analyze_latest_email prompt.
func registerAnalyzeLatestEmail(s *mcp.Server) {
	s.AddPrompt(&mcp.Prompt{
		Name:        "analyze_latest_email",
		Description: "Analyze the most recent email for potential issues and provide a comprehensive review",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.GetPromptParams) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Description: "Analyze the latest email in Mailpit",
			Messages: []*mcp.PromptMessage{{
				Role: mcp.Role("user"),
				Content: &mcp.TextContent{
					Text: `Please fetch and analyze the latest email from Mailpit. Provide a comprehensive analysis including:

1. **Basic Information**
   - Subject, From, To, Date
   - Message ID and size
   - Any CC, BCC, or Reply-To addresses

2. **Content Analysis**
   - Does it have both HTML and plain text versions?
   - Character count and content length
   - Are there any images (inline or linked)?
   - Any attachments? List them with sizes.

3. **Technical Assessment**
   - Check the message structure (MIME parts)
   - Look for any encoding issues
   - Verify proper header formatting

4. **Potential Issues**
   - Missing recommended headers
   - Possible spam trigger content
   - Broken or missing elements

Use the get_message tool with id="latest" to fetch the email, then provide your analysis.`,
				},
			}},
		}, nil
	})
}

// registerDebugEmailDelivery registers the debug_email_delivery prompt.
func registerDebugEmailDelivery(s *mcp.Server) {
	s.AddPrompt(&mcp.Prompt{
		Name:        "debug_email_delivery",
		Description: "Debug email delivery and rendering issues for a specific message",
		Arguments: []*mcp.PromptArgument{{
			Name:        "message_id",
			Description: "Message ID to debug (optional, defaults to 'latest')",
			Required:    false,
		}},
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.GetPromptParams) (*mcp.GetPromptResult, error) {
		messageID := "latest"
		if id, ok := params.Arguments["message_id"]; ok && id != "" {
			messageID = id
		}

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Debug email delivery for message: %s", messageID),
			Messages: []*mcp.PromptMessage{{
				Role: mcp.Role("user"),
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Help me debug email delivery issues for message ID: %s

Please perform the following analysis:

1. **Fetch Message Details**
   - Use get_message to retrieve the full message
   - Use get_message_headers for detailed header analysis

2. **Header Analysis**
   - Check for SPF, DKIM, DMARC authentication headers
   - Verify Return-Path configuration
   - Look for X-Spam-* headers if present
   - Check Message-ID format
   - Analyze Received headers for routing path

3. **Content Checks**
   - Run check_html to test email client compatibility
   - Run check_links to validate all URLs and images
   - Run check_spam for SpamAssassin analysis (if enabled)

4. **Common Issues to Look For**
   - Missing or malformed headers
   - Content-Type issues
   - Character encoding problems
   - Suspicious patterns that might trigger spam filters

5. **Recommendations**
   - Provide specific, actionable fixes for any issues found
   - Prioritize by severity (critical, warning, suggestion)

Start by fetching the message details.`, messageID),
				},
			}},
		}, nil
	})
}

// registerCheckEmailQuality registers the check_email_quality prompt.
func registerCheckEmailQuality(s *mcp.Server) {
	s.AddPrompt(&mcp.Prompt{
		Name:        "check_email_quality",
		Description: "Perform a comprehensive quality check on an email including HTML compatibility, links, and spam score",
		Arguments: []*mcp.PromptArgument{{
			Name:        "message_id",
			Description: "Message ID to check (optional, defaults to 'latest')",
			Required:    false,
		}},
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.GetPromptParams) (*mcp.GetPromptResult, error) {
		messageID := "latest"
		if id, ok := params.Arguments["message_id"]; ok && id != "" {
			messageID = id
		}

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Quality check for message: %s", messageID),
			Messages: []*mcp.PromptMessage{{
				Role: mcp.Role("user"),
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Perform a comprehensive quality check on message ID: %s

Run ALL of the following checks and compile a quality report:

1. **HTML Compatibility Check** (use check_html)
   - Test against major email clients
   - Identify any CSS/HTML features with poor support
   - Note which clients will have rendering issues

2. **Link Validation** (use check_links with follow=true)
   - Check all URLs return valid responses
   - Verify image links are accessible
   - Flag any broken or suspicious links

3. **Spam Analysis** (use check_spam)
   - Get the SpamAssassin score
   - Review triggered spam rules
   - Identify content that might cause delivery issues

4. **Content Review** (use get_message)
   - Does it have both HTML and text versions?
   - Are images using alt text?
   - Is the content-to-image ratio reasonable?
   - Any missing common headers?

**Quality Report Format:**

## Email Quality Score: [X/100]

### Passed Checks
- List items that passed

### Warnings
- List items that need attention

### Critical Issues
- List items that must be fixed

### Recommendations
1. Prioritized list of improvements

Please run all checks and provide the comprehensive report.`, messageID),
				},
			}},
		}, nil
	})
}

// registerSearchEmails registers the search_emails prompt.
func registerSearchEmails(s *mcp.Server) {
	s.AddPrompt(&mcp.Prompt{
		Name:        "search_emails",
		Description: "Help construct and execute a search query to find specific emails",
		Arguments: []*mcp.PromptArgument{{
			Name:        "criteria",
			Description: "What you're looking for (e.g., 'emails from john about invoices last week')",
			Required:    true,
		}},
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.GetPromptParams) (*mcp.GetPromptResult, error) {
		criteria := params.Arguments["criteria"]

		return &mcp.GetPromptResult{
			Description: "Search for emails matching criteria",
			Messages: []*mcp.PromptMessage{{
				Role: mcp.Role("user"),
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Help me find emails matching this criteria: "%s"

**Available Search Syntax:**
- from:email@example.com - Sender address (partial match supported)
- to:email@example.com - Recipient address
- subject:keyword - Subject contains text
- message-id:id - Specific Message-ID header
- tag:tagname - Has specific tag
- is:read / is:unread - Read status
- has:attachment - Has attachments
- before:YYYY-MM-DD - Before date
- after:YYYY-MM-DD - After date

**Search Tips:**
- Multiple terms are combined with AND logic
- Use quotes for exact phrases
- Partial email matches work (e.g., from:@example.com)

**Your Task:**
1. Analyze my criteria and construct an appropriate search query
2. Execute the search using search_messages
3. Present the results in a clear format
4. If no results, suggest alternative search terms

Please construct the search query and execute it.`, criteria),
				},
			}},
		}, nil
	})
}

// registerComposeTestEmail registers the compose_test_email prompt.
func registerComposeTestEmail(s *mcp.Server) {
	s.AddPrompt(&mcp.Prompt{
		Name:        "compose_test_email",
		Description: "Help compose and send a test email for a specific scenario",
		Arguments: []*mcp.PromptArgument{{
			Name:        "scenario",
			Description: "Type of email: welcome, notification, newsletter, transactional, plain, or custom description",
			Required:    true,
		}},
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.GetPromptParams) (*mcp.GetPromptResult, error) {
		scenario := params.Arguments["scenario"]

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Compose test email for scenario: %s", scenario),
			Messages: []*mcp.PromptMessage{{
				Role: mcp.Role("user"),
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Help me create and send a test email for this scenario: "%s"

**Standard Scenarios:**
- welcome: Welcome/onboarding email with branding
- notification: Alert or notification message
- newsletter: Newsletter with sections and images
- transactional: Receipt, confirmation, or order update
- plain: Simple plain text email

**Email Requirements:**
1. Create appropriate HTML content with inline CSS
2. Include a plain text version
3. Use realistic placeholder content
4. Follow email best practices:
   - Proper heading hierarchy
   - Alt text for images (use placeholder image references)
   - Mobile-responsive design hints
   - Clear call-to-action if applicable

**Your Task:**
1. Based on the scenario, design an appropriate email
2. Show me the HTML and text content you'll use
3. Use send_message to deliver it with:
   - from_email: test@example.com
   - to: [{email: "recipient@example.com"}]
   - Appropriate subject line
   - Both HTML and text content
4. After sending, retrieve and display the sent message

Please create the email content and send it.`, scenario),
				},
			}},
		}, nil
	})
}

// registerAnalyzeEmailHeaders registers the analyze_email_headers prompt.
func registerAnalyzeEmailHeaders(s *mcp.Server) {
	s.AddPrompt(&mcp.Prompt{
		Name:        "analyze_email_headers",
		Description: "Perform deep analysis of email headers for routing, authentication, and debugging",
		Arguments: []*mcp.PromptArgument{{
			Name:        "message_id",
			Description: "Message ID to analyze (optional, defaults to 'latest')",
			Required:    false,
		}},
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.GetPromptParams) (*mcp.GetPromptResult, error) {
		messageID := "latest"
		if id, ok := params.Arguments["message_id"]; ok && id != "" {
			messageID = id
		}

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Analyze headers for message: %s", messageID),
			Messages: []*mcp.PromptMessage{{
				Role: mcp.Role("user"),
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Perform a deep analysis of email headers for message ID: %s

Use get_message_headers to fetch all headers, then analyze:

1. **Routing Analysis**
   - Parse all Received headers in order
   - Trace the complete path from origin to destination
   - Calculate hop times and identify any delays
   - Note the originating IP and mail servers

2. **Authentication Headers**
   - SPF result and details
   - DKIM signature status
   - DMARC policy result
   - ARC headers if present

3. **Sender Information**
   - From vs Return-Path vs Sender
   - Reply-To configuration
   - Any discrepancies that might affect replies

4. **Client & Server Info**
   - User-Agent or X-Mailer
   - Mail server software identified
   - Any custom X-* headers

5. **Timestamps**
   - Date header vs Received times
   - Time zone handling
   - Any suspicious time discrepancies

6. **Security Flags**
   - TLS encryption indicators
   - Spam scores if present
   - Any security warnings

**Output Format:**
Provide a structured analysis with:
- Summary of key findings
- Detailed breakdown by category
- Any red flags or concerns
- Recommendations if issues found

Please fetch and analyze the headers.`, messageID),
				},
			}},
		}, nil
	})
}

// registerCompareEmails registers the compare_emails prompt.
func registerCompareEmails(s *mcp.Server) {
	s.AddPrompt(&mcp.Prompt{
		Name:        "compare_emails",
		Description: "Compare two emails to identify differences (useful for A/B testing or debugging)",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "id1",
				Description: "First message ID",
				Required:    true,
			},
			{
				Name:        "id2",
				Description: "Second message ID",
				Required:    true,
			},
		},
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.GetPromptParams) (*mcp.GetPromptResult, error) {
		id1 := params.Arguments["id1"]
		id2 := params.Arguments["id2"]

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Compare messages %s and %s", id1, id2),
			Messages: []*mcp.PromptMessage{{
				Role: mcp.Role("user"),
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Compare these two emails and highlight all differences:
- Message 1: %s
- Message 2: %s

Fetch both messages using get_message and compare:

1. **Metadata Differences**
   - Subject line changes
   - From/To/CC differences
   - Date/time comparison
   - Size differences

2. **Header Differences**
   - Changed headers
   - Added/removed headers
   - Value changes in common headers

3. **Content Differences**
   - Text body changes
   - HTML body changes
   - Highlight specific text that changed

4. **Structure Differences**
   - Attachment changes (added/removed/modified)
   - MIME structure differences
   - Inline image changes

5. **Tag Differences**
   - Different tags applied

**Output Format:**
Use a clear diff-style format:
- [=] Unchanged
- [+] Added in message 2
- [-] Removed in message 2
- [~] Modified between messages

This is useful for:
- A/B testing email templates
- Debugging email generation
- Tracking changes between versions

Please fetch both messages and provide the comparison.`, id1, id2),
				},
			}},
		}, nil
	})
}

// registerSummarizeInbox registers the summarize_inbox prompt.
func registerSummarizeInbox(s *mcp.Server) {
	s.AddPrompt(&mcp.Prompt{
		Name:        "summarize_inbox",
		Description: "Get a comprehensive summary of the current Mailpit inbox state",
		Arguments: []*mcp.PromptArgument{{
			Name:        "limit",
			Description: "Number of recent messages to include in detail (default: 10)",
			Required:    false,
		}},
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.GetPromptParams) (*mcp.GetPromptResult, error) {
		limit := "10"
		if l, ok := params.Arguments["limit"]; ok && l != "" {
			limit = l
		}

		return &mcp.GetPromptResult{
			Description: "Summarize Mailpit inbox",
			Messages: []*mcp.PromptMessage{{
				Role: mcp.Role("user"),
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Provide a comprehensive summary of the Mailpit inbox.

Use these tools to gather information:
- get_info: Get server stats and totals
- list_messages with limit=%s: Get recent messages
- list_tags: Get all tags in use

**Summary Sections:**

1. **Overview**
   - Total message count
   - Unread count
   - Database size
   - Server uptime

2. **Recent Activity**
   - List the %s most recent messages with:
     - Subject (truncated if long)
     - From address
     - Date/time
     - Read status
     - Tags

3. **Top Senders**
   - Identify the most frequent senders
   - Count messages per sender

4. **Subject Patterns**
   - Common words or patterns in subjects
   - Group similar emails if patterns found

5. **Tag Distribution**
   - List all tags with message counts
   - Identify untagged message percentage

6. **Insights**
   - Any notable patterns
   - Potential issues (many unread, similar subjects suggesting duplicates, etc.)
   - Suggestions for organization

Please gather the data and provide the summary report.`, limit, limit),
				},
			}},
		}, nil
	})
}
