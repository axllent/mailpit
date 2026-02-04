package tools

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/axllent/mailpit/mcp/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetMessageHTMLArgs are the arguments for get_message_html.
type GetMessageHTMLArgs struct {
	ID string `json:"id" jsonschema:"description=Message database ID or 'latest' for the most recent message"`
}

// RegisterGetMessageHTML registers the get_message_html tool.
func RegisterGetMessageHTML(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_message_html",
		Description: "Get the rendered HTML content of a message. Inline images are linked to the API. Returns 404 if message has no HTML part.",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[GetMessageHTMLArgs]) (*mcp.CallToolResultFor[any], error) {
		if params.Arguments.ID == "" {
			return errorResult(fmt.Errorf("id is required")), nil
		}
		result, err := c.GetMessageHTML(ctx, params.Arguments.ID)
		if err != nil {
			return errorResult(err), nil
		}
		return textResult(result), nil
	})
}

// GetMessageTextArgs are the arguments for get_message_text.
type GetMessageTextArgs struct {
	ID string `json:"id" jsonschema:"description=Message database ID or 'latest' for the most recent message"`
}

// RegisterGetMessageText registers the get_message_text tool.
func RegisterGetMessageText(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_message_text",
		Description: "Get the plain text content of a message",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[GetMessageTextArgs]) (*mcp.CallToolResultFor[any], error) {
		if params.Arguments.ID == "" {
			return errorResult(fmt.Errorf("id is required")), nil
		}
		result, err := c.GetMessageText(ctx, params.Arguments.ID)
		if err != nil {
			return errorResult(err), nil
		}
		if result == "" {
			return textResult("(Message has no text content)"), nil
		}
		return textResult(result), nil
	})
}

// GetAttachmentArgs are the arguments for get_attachment.
type GetAttachmentArgs struct {
	MessageID string `json:"message_id" jsonschema:"description=Message database ID or 'latest' for the most recent message"`
	PartID    string `json:"part_id" jsonschema:"description=Attachment part ID (from message details)"`
}

// RegisterGetAttachment registers the get_attachment tool.
func RegisterGetAttachment(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_attachment",
		Description: "Download an attachment from a message. Returns base64-encoded content for binary files.",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[GetAttachmentArgs]) (*mcp.CallToolResultFor[any], error) {
		if params.Arguments.MessageID == "" {
			return errorResult(fmt.Errorf("message_id is required")), nil
		}
		if params.Arguments.PartID == "" {
			return errorResult(fmt.Errorf("part_id is required")), nil
		}
		data, err := c.GetAttachment(ctx, params.Arguments.MessageID, params.Arguments.PartID)
		if err != nil {
			return errorResult(err), nil
		}

		// Check if content is text-like (simple heuristic)
		isText := true
		for _, b := range data {
			if b == 0 || (b < 32 && b != 9 && b != 10 && b != 13) {
				isText = false
				break
			}
		}

		if isText {
			return textResult(string(data)), nil
		}

		// Return base64 for binary content
		encoded := base64.StdEncoding.EncodeToString(data)
		return textResult(fmt.Sprintf("Base64-encoded attachment (%d bytes):\n%s", len(data), encoded)), nil
	})
}

// RegisterAllContentTools registers all content-related tools.
func RegisterAllContentTools(s *mcp.Server, c *client.Client) {
	RegisterGetMessageHTML(s, c)
	RegisterGetMessageText(s, c)
	RegisterGetAttachment(s, c)
}
