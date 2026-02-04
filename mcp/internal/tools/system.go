package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/axllent/mailpit/mcp/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterGetInfo registers the get_info tool.
func RegisterGetInfo(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_info",
		Description: "Get Mailpit server information including version, database stats, message counts, and runtime statistics",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[EmptyArgs]) (*mcp.CallToolResultFor[any], error) {
		result, err := c.GetInfo(ctx)
		if err != nil {
			return errorResult(err), nil
		}
		return formatAppInfo(result), nil
	})
}

// RegisterGetWebUIConfig registers the get_webui_config tool.
func RegisterGetWebUIConfig(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_webui_config",
		Description: "Get Mailpit web UI configuration including enabled features (SpamAssassin, Chaos, relay settings)",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[EmptyArgs]) (*mcp.CallToolResultFor[any], error) {
		result, err := c.GetWebUIConfig(ctx)
		if err != nil {
			return errorResult(err), nil
		}
		return formatWebUIConfig(result), nil
	})
}

// formatAppInfo formats application information.
func formatAppInfo(info *client.AppInfo) *mcp.CallToolResultFor[any] {
	var sb strings.Builder

	sb.WriteString("=== Mailpit Server Information ===\n\n")

	sb.WriteString("VERSION:\n")
	sb.WriteString(fmt.Sprintf("  Current:  %s\n", info.Version))
	if info.LatestVersion != "" && info.LatestVersion != info.Version {
		sb.WriteString(fmt.Sprintf("  Latest:   %s (update available)\n", info.LatestVersion))
	}

	sb.WriteString("\nDATABASE:\n")
	sb.WriteString(fmt.Sprintf("  Path: %s\n", info.Database))
	sb.WriteString(fmt.Sprintf("  Size: %s\n", formatSize(info.DatabaseSize)))

	sb.WriteString("\nMESSAGES:\n")
	sb.WriteString(fmt.Sprintf("  Total:  %d\n", info.Messages))
	sb.WriteString(fmt.Sprintf("  Unread: %d\n", info.Unread))

	if len(info.Tags) > 0 {
		sb.WriteString("\nTAGS:\n")
		for tag, count := range info.Tags {
			sb.WriteString(fmt.Sprintf("  %s: %d messages\n", tag, count))
		}
	}

	if info.RuntimeStats != nil {
		sb.WriteString("\nRUNTIME STATISTICS:\n")
		sb.WriteString(fmt.Sprintf("  Uptime:           %s\n", formatDuration(info.RuntimeStats.Uptime)))
		sb.WriteString(fmt.Sprintf("  Memory Usage:     %s\n", formatSize(info.RuntimeStats.Memory)))
		sb.WriteString(fmt.Sprintf("  SMTP Accepted:    %d (%s)\n", info.RuntimeStats.SMTPAccepted, formatSize(info.RuntimeStats.SMTPAcceptedSize)))
		sb.WriteString(fmt.Sprintf("  SMTP Rejected:    %d\n", info.RuntimeStats.SMTPRejected))
		if info.RuntimeStats.SMTPIgnored > 0 {
			sb.WriteString(fmt.Sprintf("  SMTP Ignored:     %d\n", info.RuntimeStats.SMTPIgnored))
		}
		sb.WriteString(fmt.Sprintf("  Messages Deleted: %d\n", info.RuntimeStats.MessagesDeleted))
	}

	return textResult(sb.String())
}

// formatWebUIConfig formats web UI configuration.
func formatWebUIConfig(config *client.WebUIConfig) *mcp.CallToolResultFor[any] {
	var sb strings.Builder

	sb.WriteString("=== Mailpit Configuration ===\n\n")

	if config.Label != "" {
		sb.WriteString(fmt.Sprintf("Instance Label: %s\n\n", config.Label))
	}

	sb.WriteString("FEATURES:\n")
	sb.WriteString(fmt.Sprintf("  SpamAssassin:      %s\n", enabledStr(config.SpamAssassin)))
	sb.WriteString(fmt.Sprintf("  Chaos Testing:     %s\n", enabledStr(config.ChaosEnabled)))
	sb.WriteString(fmt.Sprintf("  Ignore Duplicates: %s\n", enabledStr(config.DuplicatesIgnored)))
	sb.WriteString(fmt.Sprintf("  Hide Delete All:   %s\n", enabledStr(config.HideDeleteAllButton)))

	if config.MessageRelay != nil {
		sb.WriteString("\nMESSAGE RELAY:\n")
		sb.WriteString(fmt.Sprintf("  Enabled: %s\n", enabledStr(config.MessageRelay.Enabled)))
		if config.MessageRelay.Enabled {
			sb.WriteString(fmt.Sprintf("  SMTP Server: %s\n", config.MessageRelay.SMTPServer))
			if config.MessageRelay.AllowedRecipients != "" {
				sb.WriteString(fmt.Sprintf("  Allowed Recipients: %s\n", config.MessageRelay.AllowedRecipients))
			}
			if config.MessageRelay.BlockedRecipients != "" {
				sb.WriteString(fmt.Sprintf("  Blocked Recipients: %s\n", config.MessageRelay.BlockedRecipients))
			}
			if config.MessageRelay.OverrideFrom != "" {
				sb.WriteString(fmt.Sprintf("  Override From: %s\n", config.MessageRelay.OverrideFrom))
			}
			if config.MessageRelay.ReturnPath != "" {
				sb.WriteString(fmt.Sprintf("  Return-Path: %s\n", config.MessageRelay.ReturnPath))
			}
			sb.WriteString(fmt.Sprintf("  Preserve Message-IDs: %s\n", enabledStr(config.MessageRelay.PreserveMessageIDs)))
		}
	}

	return textResult(sb.String())
}

// enabledStr returns "Enabled" or "Disabled" based on the boolean value.
func enabledStr(enabled bool) string {
	if enabled {
		return "Enabled"
	}
	return "Disabled"
}

// formatDuration formats seconds as a human-readable duration.
func formatDuration(seconds uint64) string {
	d := time.Duration(seconds) * time.Second
	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute
	d -= minutes * time.Minute
	secs := d / time.Second

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, secs)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, secs)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, secs)
	}
	return fmt.Sprintf("%ds", secs)
}

// RegisterAllSystemTools registers all system-related tools.
func RegisterAllSystemTools(s *mcp.Server, c *client.Client) {
	RegisterGetInfo(s, c)
	RegisterGetWebUIConfig(s, c)
}
