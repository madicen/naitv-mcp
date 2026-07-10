package mcp

import (
	"context"
	"fmt"

	"github.com/madicen/naitv-mcp/internal/store"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerPrompts(s *sdkmcp.Server, st *store.Store) {
	s.AddPrompt(&sdkmcp.Prompt{
		Name:        "load-context",
		Description: "Load standing instructions and fetch on-demand entries relevant to a task",
		Arguments: []*sdkmcp.PromptArgument{
			{
				Name:        "task",
				Description: "Brief description of what you are about to work on",
				Required:    true,
			},
		},
	}, func(ctx context.Context, req *sdkmcp.GetPromptRequest) (*sdkmcp.GetPromptResult, error) {
		task := req.Params.Arguments["task"]
		if task == "" {
			task = "(unspecified)"
		}
		text := fmt.Sprintf(`You are starting work on: %s

1. Call the initialize tool to load standing instructions from naitv-mcp.
2. Call search_entries with a query derived from the task above.
3. For any promising results, call get_entry to fetch full details before proceeding.
4. Follow every rule and workflow from initialize for the rest of this session.`, task)
		return &sdkmcp.GetPromptResult{
			Description: "Load naitv-mcp context for a task",
			Messages: []*sdkmcp.PromptMessage{{
				Role:    "user",
				Content: &sdkmcp.TextContent{Text: text},
			}},
		}, nil
	})

	s.AddPrompt(&sdkmcp.Prompt{
		Name:        "propose-learning",
		Description: "Summarize session learnings and propose new context entries",
	}, func(ctx context.Context, req *sdkmcp.GetPromptRequest) (*sdkmcp.GetPromptResult, error) {
		text := `Review what you learned during this session that would help future agents.

1. Call search_entries to check whether the knowledge base already covers each lesson.
2. For gaps, call add_entry to propose new entries (rules, workflows, facts, or notes).
3. Be specific and actionable — vague summaries are not useful.
4. Tell the user that proposals are pending approval in the naitv-mcp TUI Review tab.`
		return &sdkmcp.GetPromptResult{
			Description: "Propose knowledge-base entries from session learnings",
			Messages: []*sdkmcp.PromptMessage{{
				Role:    "user",
				Content: &sdkmcp.TextContent{Text: text},
			}},
		}, nil
	})

	_ = st // reserved for future dynamic prompt content
}
