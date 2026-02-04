package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/axllent/mailpit/mcp/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// EmptyArgs represents no arguments.
type EmptyArgs struct{}

// RegisterListTags registers the list_tags tool.
func RegisterListTags(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_tags",
		Description: "Get all unique message tags currently in use",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[EmptyArgs]) (*mcp.CallToolResultFor[any], error) {
		result, err := c.ListTags(ctx)
		if err != nil {
			return errorResult(err), nil
		}
		if len(result) == 0 {
			return textResult("No tags found."), nil
		}
		return textResult(fmt.Sprintf("Tags (%d):\n  - %s", len(result), strings.Join(result, "\n  - "))), nil
	})
}

// SetTagsArgs are the arguments for set_tags.
type SetTagsArgs struct {
	IDs  []string `json:"ids" jsonschema:"description=Array of message database IDs to tag"`
	Tags []string `json:"tags" jsonschema:"description=Array of tag names to set. Pass empty array to remove all tags."`
}

// RegisterSetTags registers the set_tags tool.
func RegisterSetTags(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "set_tags",
		Description: "Set tags on messages. This overwrites existing tags. Pass empty tags array to remove all tags.",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[SetTagsArgs]) (*mcp.CallToolResultFor[any], error) {
		if len(params.Arguments.IDs) == 0 {
			return errorResult(fmt.Errorf("ids is required")), nil
		}
		err := c.SetTags(ctx, params.Arguments.IDs, params.Arguments.Tags)
		if err != nil {
			return errorResult(err), nil
		}
		if len(params.Arguments.Tags) == 0 {
			return textResult(fmt.Sprintf("Removed all tags from %d message(s)", len(params.Arguments.IDs))), nil
		}
		return textResult(fmt.Sprintf("Set tags [%s] on %d message(s)", strings.Join(params.Arguments.Tags, ", "), len(params.Arguments.IDs))), nil
	})
}

// RenameTagArgs are the arguments for rename_tag.
type RenameTagArgs struct {
	OldName string `json:"old_name" jsonschema:"description=Current tag name to rename"`
	NewName string `json:"new_name" jsonschema:"description=New name for the tag"`
}

// RegisterRenameTag registers the rename_tag tool.
func RegisterRenameTag(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "rename_tag",
		Description: "Rename an existing tag. Updates all messages with this tag.",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[RenameTagArgs]) (*mcp.CallToolResultFor[any], error) {
		if params.Arguments.OldName == "" {
			return errorResult(fmt.Errorf("old_name is required")), nil
		}
		if params.Arguments.NewName == "" {
			return errorResult(fmt.Errorf("new_name is required")), nil
		}
		err := c.RenameTag(ctx, params.Arguments.OldName, params.Arguments.NewName)
		if err != nil {
			return errorResult(err), nil
		}
		return textResult(fmt.Sprintf("Renamed tag '%s' to '%s'", params.Arguments.OldName, params.Arguments.NewName)), nil
	})
}

// DeleteTagArgs are the arguments for delete_tag.
type DeleteTagArgs struct {
	Name string `json:"name" jsonschema:"description=Tag name to delete"`
}

// RegisterDeleteTag registers the delete_tag tool.
func RegisterDeleteTag(s *mcp.Server, c *client.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "delete_tag",
		Description: "Delete a tag. Removes the tag from all messages but does not delete the messages themselves.",
	}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[DeleteTagArgs]) (*mcp.CallToolResultFor[any], error) {
		if params.Arguments.Name == "" {
			return errorResult(fmt.Errorf("name is required")), nil
		}
		err := c.DeleteTag(ctx, params.Arguments.Name)
		if err != nil {
			return errorResult(err), nil
		}
		return textResult(fmt.Sprintf("Deleted tag '%s'", params.Arguments.Name)), nil
	})
}

// RegisterAllTagTools registers all tag-related tools.
func RegisterAllTagTools(s *mcp.Server, c *client.Client) {
	RegisterListTags(s, c)
	RegisterSetTags(s, c)
	RegisterRenameTag(s, c)
	RegisterDeleteTag(s, c)
}
