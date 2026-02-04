package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/axllent/mailpit/mcp/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CheckHTMLArgs are the arguments for check_html.
type CheckHTMLArgs struct {
	ID string `json:"id" jsonschema:"description=Message database ID or 'latest' for the most recent message"`
}

// RegisterCheckHTML registers the check_html tool.
func RegisterCheckHTML(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "check_html",
		Description: "Check email HTML/CSS compatibility across different email clients (Outlook, Gmail, Apple Mail, etc.). Uses caniemail.com database.",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[CheckHTMLArgs]) (*mcp.CallToolResultFor[any], error) {
		if params.Arguments.ID == "" {
			return errorResult(fmt.Errorf("id is required")), nil
		}
		result, err := c.CheckHTML(ctx, params.Arguments.ID)
		if err != nil {
			return errorResult(err), nil
		}
		return formatHTMLCheckResult(result), nil
	})
}

// CheckLinksArgs are the arguments for check_links.
type CheckLinksArgs struct {
	ID     string `json:"id" jsonschema:"description=Message database ID or 'latest' for the most recent message"`
	Follow bool   `json:"follow,omitempty" jsonschema:"description=Follow redirects when checking links (default: false)"`
}

// RegisterCheckLinks registers the check_links tool.
func RegisterCheckLinks(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "check_links",
		Description: "Validate all links and images in a message. Checks HTTP status codes for each URL.",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[CheckLinksArgs]) (*mcp.CallToolResultFor[any], error) {
		if params.Arguments.ID == "" {
			return errorResult(fmt.Errorf("id is required")), nil
		}
		result, err := c.CheckLinks(ctx, params.Arguments.ID, params.Arguments.Follow)
		if err != nil {
			return errorResult(err), nil
		}
		return formatLinkCheckResult(result), nil
	})
}

// CheckSpamArgs are the arguments for check_spam.
type CheckSpamArgs struct {
	ID string `json:"id" jsonschema:"description=Message database ID or 'latest' for the most recent message"`
}

// RegisterCheckSpam registers the check_spam tool.
func RegisterCheckSpam(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "check_spam",
		Description: "Run SpamAssassin analysis on a message. Returns spam score and triggered rules. Requires SpamAssassin to be enabled in Mailpit.",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[CheckSpamArgs]) (*mcp.CallToolResultFor[any], error) {
		if params.Arguments.ID == "" {
			return errorResult(fmt.Errorf("id is required")), nil
		}
		result, err := c.CheckSpam(ctx, params.Arguments.ID)
		if err != nil {
			return errorResult(err), nil
		}
		return formatSpamCheckResult(result), nil
	})
}

// formatHTMLCheckResult formats an HTML check result.
func formatHTMLCheckResult(result *client.HTMLCheckResponse) *mcp.CallToolResultFor[any] {
	var sb strings.Builder

	sb.WriteString("=== HTML Email Compatibility Check ===\n\n")

	if result.Total != nil {
		sb.WriteString("OVERALL COMPATIBILITY:\n")
		sb.WriteString(fmt.Sprintf("  Supported:   %.1f%%\n", result.Total.Supported))
		sb.WriteString(fmt.Sprintf("  Partial:     %.1f%%\n", result.Total.Partial))
		sb.WriteString(fmt.Sprintf("  Unsupported: %.1f%%\n", result.Total.Unsupported))
		sb.WriteString(fmt.Sprintf("  HTML Nodes:  %d\n", result.Total.Nodes))
		sb.WriteString(fmt.Sprintf("  Tests Run:   %d\n\n", result.Total.Tests))
	}

	if len(result.Warnings) > 0 {
		sb.WriteString(fmt.Sprintf("COMPATIBILITY WARNINGS (%d):\n\n", len(result.Warnings)))
		for i, w := range result.Warnings {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, w.Title))
			sb.WriteString(fmt.Sprintf("   Category: %s\n", w.Category))
			if w.Description != "" {
				sb.WriteString(fmt.Sprintf("   Description: %s\n", w.Description))
			}
			if w.Score != nil {
				sb.WriteString(fmt.Sprintf("   Support: %.0f%% supported, %.0f%% partial, %.0f%% unsupported\n",
					w.Score.Supported, w.Score.Partial, w.Score.Unsupported))
			}
			if w.URL != "" {
				sb.WriteString(fmt.Sprintf("   More info: %s\n", w.URL))
			}

			// Show problematic clients
			if len(w.Results) > 0 {
				unsupported := []string{}
				partial := []string{}
				for _, r := range w.Results {
					switch r.Support {
					case "no":
						unsupported = append(unsupported, r.Name)
					case "partial":
						partial = append(partial, r.Name)
					}
				}
				if len(unsupported) > 0 && len(unsupported) <= 10 {
					sb.WriteString(fmt.Sprintf("   Not supported in: %s\n", strings.Join(unsupported, ", ")))
				} else if len(unsupported) > 10 {
					sb.WriteString(fmt.Sprintf("   Not supported in: %d email clients\n", len(unsupported)))
				}
				if len(partial) > 0 && len(partial) <= 10 {
					sb.WriteString(fmt.Sprintf("   Partial support in: %s\n", strings.Join(partial, ", ")))
				}
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("No compatibility warnings found. Your email should render correctly across all tested email clients.\n")
	}

	// Platform summary
	if len(result.Platforms) > 0 {
		sb.WriteString("\nTESTED PLATFORMS:\n")
		for platform, clients := range result.Platforms {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", platform, strings.Join(clients, ", ")))
		}
	}

	return textResult(sb.String())
}

// formatLinkCheckResult formats a link check result.
func formatLinkCheckResult(result *client.LinkCheckResponse) *mcp.CallToolResultFor[any] {
	var sb strings.Builder

	sb.WriteString("=== Link Validation Results ===\n\n")

	total := len(result.Links)
	errors := result.Errors

	sb.WriteString(fmt.Sprintf("Total links checked: %d\n", total))
	if errors > 0 {
		sb.WriteString(fmt.Sprintf("Errors found: %d\n\n", errors))
	} else {
		sb.WriteString("Errors found: 0\n\n")
	}

	if len(result.Links) > 0 {
		// Group by status
		working := []*client.Link{}
		broken := []*client.Link{}

		for _, link := range result.Links {
			if link.StatusCode >= 200 && link.StatusCode < 400 {
				working = append(working, link)
			} else {
				broken = append(broken, link)
			}
		}

		if len(broken) > 0 {
			sb.WriteString("BROKEN LINKS:\n")
			for _, link := range broken {
				sb.WriteString(fmt.Sprintf("  [%d %s] %s\n", link.StatusCode, link.Status, link.URL))
			}
			sb.WriteString("\n")
		}

		if len(working) > 0 {
			sb.WriteString("WORKING LINKS:\n")
			for _, link := range working {
				sb.WriteString(fmt.Sprintf("  [%d] %s\n", link.StatusCode, link.URL))
			}
		}
	} else {
		sb.WriteString("No links found in the message.\n")
	}

	return textResult(sb.String())
}

// formatSpamCheckResult formats a spam check result.
func formatSpamCheckResult(result *client.SpamAssassinResponse) *mcp.CallToolResultFor[any] {
	var sb strings.Builder

	sb.WriteString("=== SpamAssassin Analysis ===\n\n")

	if result.Error != "" {
		sb.WriteString(fmt.Sprintf("Error: %s\n", result.Error))
		return textResult(sb.String())
	}

	if result.IsSpam {
		sb.WriteString("SPAM DETECTED\n\n")
	} else {
		sb.WriteString("NOT SPAM\n\n")
	}

	sb.WriteString(fmt.Sprintf("Spam Score: %.1f\n\n", result.Score))

	if len(result.Rules) > 0 {
		sb.WriteString("TRIGGERED RULES:\n")
		for _, rule := range result.Rules {
			scoreStr := fmt.Sprintf("%.1f", rule.Score)
			if rule.Score > 0 {
				scoreStr = "+" + scoreStr
			}
			sb.WriteString(fmt.Sprintf("  [%s] %s\n", scoreStr, rule.Name))
			if rule.Description != "" {
				sb.WriteString(fmt.Sprintf("         %s\n", rule.Description))
			}
		}
	} else {
		sb.WriteString("No spam rules triggered.\n")
	}

	return textResult(sb.String())
}

// RegisterAllValidationTools registers all validation-related tools.
func RegisterAllValidationTools(s *mcp.Server, c *client.Client) {
	RegisterCheckHTML(s, c)
	RegisterCheckLinks(s, c)
	RegisterCheckSpam(s, c)
}
