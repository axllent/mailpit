// Package resources provides MCP resource implementations for Mailpit.
package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/axllent/mailpit/mcp/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterAllResources registers all MCP resources.
func RegisterAllResources(s *mcp.Server, c *client.Client) {
	registerInfoResource(s, c)
	registerLatestMessageResource(s, c)
	registerTagsResource(s, c)
	registerConfigResource(s, c)
}

// registerInfoResource registers the mailpit://info resource.
func registerInfoResource(s *mcp.Server, c *client.Client) {
	s.AddResource(&mcp.Resource{
		URI:         "mailpit://info",
		Name:        "Mailpit Server Info",
		Description: "Current Mailpit server information and statistics",
		MIMEType:    "application/json",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.ReadResourceParams) (*mcp.ReadResourceResult, error) {
		info, err := c.GetInfo(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get server info: %w", err)
		}

		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal info: %w", err)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			}},
		}, nil
	})
}

// registerLatestMessageResource registers the mailpit://messages/latest resource.
func registerLatestMessageResource(s *mcp.Server, c *client.Client) {
	s.AddResource(&mcp.Resource{
		URI:         "mailpit://messages/latest",
		Name:        "Latest Message",
		Description: "Summary of the most recent email message",
		MIMEType:    "text/plain",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.ReadResourceParams) (*mcp.ReadResourceResult, error) {
		msg, err := c.GetMessage(ctx, "latest")
		if err != nil {
			return nil, fmt.Errorf("failed to get latest message: %w", err)
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Subject: %s\n", msg.Subject))
		sb.WriteString(fmt.Sprintf("ID: %s\n", msg.ID))
		sb.WriteString(fmt.Sprintf("Date: %s\n", msg.Date.Format("2006-01-02 15:04:05 MST")))

		if msg.From != nil {
			from := msg.From.Address
			if msg.From.Name != "" {
				from = fmt.Sprintf("%s <%s>", msg.From.Name, msg.From.Address)
			}
			sb.WriteString(fmt.Sprintf("From: %s\n", from))
		}

		if len(msg.To) > 0 {
			addrs := make([]string, len(msg.To))
			for i, a := range msg.To {
				if a.Name != "" {
					addrs[i] = fmt.Sprintf("%s <%s>", a.Name, a.Address)
				} else {
					addrs[i] = a.Address
				}
			}
			sb.WriteString(fmt.Sprintf("To: %s\n", strings.Join(addrs, ", ")))
		}

		if len(msg.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(msg.Tags, ", ")))
		}

		sb.WriteString(fmt.Sprintf("\nAttachments: %d\n", len(msg.Attachments)))

		if msg.Text != "" {
			preview := msg.Text
			if len(preview) > 500 {
				preview = preview[:500] + "..."
			}
			sb.WriteString(fmt.Sprintf("\nPreview:\n%s\n", preview))
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      params.URI,
				MIMEType: "text/plain",
				Text:     sb.String(),
			}},
		}, nil
	})
}

// registerTagsResource registers the mailpit://tags resource.
func registerTagsResource(s *mcp.Server, c *client.Client) {
	s.AddResource(&mcp.Resource{
		URI:         "mailpit://tags",
		Name:        "Message Tags",
		Description: "All current message tags",
		MIMEType:    "application/json",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.ReadResourceParams) (*mcp.ReadResourceResult, error) {
		tags, err := c.ListTags(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get tags: %w", err)
		}

		data, err := json.MarshalIndent(tags, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tags: %w", err)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			}},
		}, nil
	})
}

// registerConfigResource registers the mailpit://config resource.
func registerConfigResource(s *mcp.Server, c *client.Client) {
	s.AddResource(&mcp.Resource{
		URI:         "mailpit://config",
		Name:        "Mailpit Configuration",
		Description: "Current Mailpit configuration and enabled features",
		MIMEType:    "application/json",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.ReadResourceParams) (*mcp.ReadResourceResult, error) {
		config, err := c.GetWebUIConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get config: %w", err)
		}

		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			}},
		}, nil
	})
}
