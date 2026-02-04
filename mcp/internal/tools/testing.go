package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/axllent/mailpit/mcp/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SendMessageArgs are the arguments for send_message.
type SendMessageArgs struct {
	FromEmail string            `json:"from_email" jsonschema:"description=Sender email address (required)"`
	FromName  string            `json:"from_name,omitempty" jsonschema:"description=Sender display name"`
	To        []RecipientArg    `json:"to,omitempty" jsonschema:"description=To recipients"`
	Cc        []RecipientArg    `json:"cc,omitempty" jsonschema:"description=CC recipients"`
	Bcc       []string          `json:"bcc,omitempty" jsonschema:"description=BCC recipient email addresses"`
	Subject   string            `json:"subject,omitempty" jsonschema:"description=Email subject"`
	Text      string            `json:"text,omitempty" jsonschema:"description=Plain text body"`
	HTML      string            `json:"html,omitempty" jsonschema:"description=HTML body"`
	Tags      []string          `json:"tags,omitempty" jsonschema:"description=Tags to apply to the message"`
	Headers   map[string]string `json:"headers,omitempty" jsonschema:"description=Custom headers as key-value pairs"`
}

// RecipientArg represents an email recipient.
type RecipientArg struct {
	Email string `json:"email" jsonschema:"description=Recipient email address"`
	Name  string `json:"name,omitempty" jsonschema:"description=Recipient display name"`
}

// RegisterSendMessage registers the send_message tool.
func RegisterSendMessage(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "send_message",
		Description: "Send a test email message via Mailpit's HTTP API. Useful for testing email templates and workflows.",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[SendMessageArgs]) (*mcp.CallToolResultFor[any], error) {
		args := params.Arguments
		if args.FromEmail == "" {
			return errorResult(fmt.Errorf("from_email is required")), nil
		}

		// Build the request
		sendReq := &client.SendMessageRequest{
			From: &client.SendAddress{
				Email: args.FromEmail,
				Name:  args.FromName,
			},
			Subject: args.Subject,
			Text:    args.Text,
			HTML:    args.HTML,
			Tags:    args.Tags,
			Headers: args.Headers,
			Bcc:     args.Bcc,
		}

		// Convert recipients
		for _, r := range args.To {
			sendReq.To = append(sendReq.To, &client.SendAddress{
				Email: r.Email,
				Name:  r.Name,
			})
		}
		for _, r := range args.Cc {
			sendReq.Cc = append(sendReq.Cc, &client.SendAddress{
				Email: r.Email,
				Name:  r.Name,
			})
		}

		result, err := c.SendMessage(ctx, sendReq)
		if err != nil {
			return errorResult(err), nil
		}

		var sb strings.Builder
		sb.WriteString("Message sent successfully!\n\n")
		sb.WriteString(fmt.Sprintf("Message ID: %s\n", result.ID))
		sb.WriteString(fmt.Sprintf("From: %s\n", formatAddress(args.FromName, args.FromEmail)))
		if len(args.To) > 0 {
			addrs := make([]string, len(args.To))
			for i, r := range args.To {
				addrs[i] = formatAddress(r.Name, r.Email)
			}
			sb.WriteString(fmt.Sprintf("To: %s\n", strings.Join(addrs, ", ")))
		}
		sb.WriteString(fmt.Sprintf("Subject: %s\n", args.Subject))

		return textResult(sb.String()), nil
	})
}

// ReleaseMessageArgs are the arguments for release_message.
type ReleaseMessageArgs struct {
	ID string   `json:"id" jsonschema:"description=Message database ID or 'latest' for the most recent message"`
	To []string `json:"to" jsonschema:"description=Email addresses to relay the message to"`
}

// RegisterReleaseMessage registers the release_message tool.
func RegisterReleaseMessage(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "release_message",
		Description: "Release (relay) a captured message via the configured external SMTP server. Requires SMTP relay to be configured in Mailpit.",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[ReleaseMessageArgs]) (*mcp.CallToolResultFor[any], error) {
		if params.Arguments.ID == "" {
			return errorResult(fmt.Errorf("id is required")), nil
		}
		if len(params.Arguments.To) == 0 {
			return errorResult(fmt.Errorf("to is required (at least one recipient)")), nil
		}

		err := c.ReleaseMessage(ctx, params.Arguments.ID, params.Arguments.To)
		if err != nil {
			return errorResult(err), nil
		}

		return textResult(fmt.Sprintf("Message %s released to: %s", params.Arguments.ID, strings.Join(params.Arguments.To, ", "))), nil
	})
}

// RegisterGetChaos registers the get_chaos tool.
func RegisterGetChaos(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_chaos",
		Description: "Get current Chaos testing triggers configuration. Chaos allows simulating SMTP failures. Requires --enable-chaos flag.",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[EmptyArgs]) (*mcp.CallToolResultFor[any], error) {
		result, err := c.GetChaos(ctx)
		if err != nil {
			return errorResult(err), nil
		}
		return formatChaosResult(result), nil
	})
}

// SetChaosArgs are the arguments for set_chaos.
type SetChaosArgs struct {
	SenderProbability    int `json:"sender_probability,omitempty" jsonschema:"description=Probability (0-100) of rejecting at MAIL FROM stage"`
	SenderErrorCode      int `json:"sender_error_code,omitempty" jsonschema:"description=SMTP error code (400-599) for sender rejection"`
	RecipientProbability int `json:"recipient_probability,omitempty" jsonschema:"description=Probability (0-100) of rejecting at RCPT TO stage"`
	RecipientErrorCode   int `json:"recipient_error_code,omitempty" jsonschema:"description=SMTP error code (400-599) for recipient rejection"`
	AuthProbability      int `json:"auth_probability,omitempty" jsonschema:"description=Probability (0-100) of rejecting authentication"`
	AuthErrorCode        int `json:"auth_error_code,omitempty" jsonschema:"description=SMTP error code (400-599) for auth rejection"`
}

// RegisterSetChaos registers the set_chaos tool.
func RegisterSetChaos(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "set_chaos",
		Description: "Set Chaos testing triggers to simulate SMTP failures. Set probability to 0 to disable a trigger. Requires --enable-chaos flag.",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[SetChaosArgs]) (*mcp.CallToolResultFor[any], error) {
		args := params.Arguments
		triggers := &client.ChaosTriggers{}

		if args.SenderProbability > 0 || args.SenderErrorCode > 0 {
			triggers.Sender = &client.ChaosTrigger{
				Probability: args.SenderProbability,
				ErrorCode:   args.SenderErrorCode,
			}
		}
		if args.RecipientProbability > 0 || args.RecipientErrorCode > 0 {
			triggers.Recipient = &client.ChaosTrigger{
				Probability: args.RecipientProbability,
				ErrorCode:   args.RecipientErrorCode,
			}
		}
		if args.AuthProbability > 0 || args.AuthErrorCode > 0 {
			triggers.Authentication = &client.ChaosTrigger{
				Probability: args.AuthProbability,
				ErrorCode:   args.AuthErrorCode,
			}
		}

		result, err := c.SetChaos(ctx, triggers)
		if err != nil {
			return errorResult(err), nil
		}

		var sb strings.Builder
		sb.WriteString("Chaos triggers updated!\n\n")
		chaosText := formatChaosResult(result)
		sb.WriteString(chaosText.Content[0].(*mcp.TextContent).Text)
		return textResult(sb.String()), nil
	})
}

// formatChaosResult formats chaos triggers.
func formatChaosResult(result *client.ChaosTriggers) *mcp.CallToolResultFor[any] {
	var sb strings.Builder
	sb.WriteString("=== Chaos Testing Configuration ===\n\n")

	formatTrigger := func(name string, t *client.ChaosTrigger) {
		if t == nil || t.Probability == 0 {
			sb.WriteString(fmt.Sprintf("%s: Disabled\n", name))
		} else {
			sb.WriteString(fmt.Sprintf("%s: %d%% probability, error code %d\n", name, t.Probability, t.ErrorCode))
		}
	}

	formatTrigger("Sender (MAIL FROM)", result.Sender)
	formatTrigger("Recipient (RCPT TO)", result.Recipient)
	formatTrigger("Authentication", result.Authentication)

	return textResult(sb.String())
}

// RegisterAllTestingTools registers all testing-related tools.
func RegisterAllTestingTools(s *mcp.Server, c *client.Client) {
	RegisterSendMessage(s, c)
	RegisterReleaseMessage(s, c)
	RegisterGetChaos(s, c)
	RegisterSetChaos(s, c)
}
