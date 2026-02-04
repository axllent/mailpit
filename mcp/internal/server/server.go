package server

import (
	"github.com/axllent/mailpit/mcp/internal/client"
	"github.com/axllent/mailpit/mcp/internal/prompts"
	"github.com/axllent/mailpit/mcp/internal/resources"
	"github.com/axllent/mailpit/mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Version is set at build time.
var Version = "dev"

// New creates a new MCP server with all tools, resources, and prompts registered.
func New(mailpitClient *client.Client) *mcp.Server {
	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "mailpit-mcp-server",
			Version: Version,
		},
		nil,
	)

	// Register all tools
	registerAllTools(s, mailpitClient)

	// Register all resources
	resources.RegisterAllResources(s, mailpitClient)

	// Register all prompts
	prompts.RegisterAllPrompts(s)

	return s
}

// registerAllTools registers all MCP tools.
func registerAllTools(s *mcp.Server, c *client.Client) {
	tools.RegisterAllMessageTools(s, c)
	tools.RegisterAllContentTools(s, c)
	tools.RegisterAllValidationTools(s, c)
	tools.RegisterAllTagTools(s, c)
	tools.RegisterAllTestingTools(s, c)
	tools.RegisterAllSystemTools(s, c)
}
