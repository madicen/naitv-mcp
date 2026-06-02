package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/madicen/naitv-mcp/internal/instructions"
	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/pkg/entry"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Run registers all tools and starts serving over stdio.
func Run(st *store.Store) error {
	s := server.NewMCPServer("naitv-mcp", "0.1.0")

	// initialize
	s.AddTool(
		mcp.NewTool("initialize",
			mcp.WithDescription("Return the user's standing instructions, rendered from context entries marked for initialization (rules, tooling preferences, workflows, agent roles, repos, facts, notes). Call this at the start of a session to work the way the user prefers. Entries marked on-demand are not included here; fetch those with get_entry or search_entries when relevant."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			entries, err := st.List("", nil)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("initialize error: %v", err)), nil
			}
			return mcp.NewToolResultText(instructions.Render(instructions.FilterInit(entries))), nil
		},
	)

	// list_entries
	s.AddTool(
		mcp.NewTool("list_entries",
			mcp.WithDescription("List active context entries, optionally filtered by kind and/or tags."),
			mcp.WithString("kind", mcp.Description("Entry kind to filter by (e.g. repo, note, doc). Empty means all kinds.")),
			mcp.WithString("tags", mcp.Description("Comma-separated tags to filter by (AND logic). Empty means no tag filter.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			kind, _ := req.Params.Arguments["kind"].(string)
			tagsStr, _ := req.Params.Arguments["tags"].(string)
			entries, err := st.List(kind, parseTags(tagsStr))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("list_entries error: %v", err)), nil
			}
			return mcp.NewToolResultText(formatEntries(entries)), nil
		},
	)

	// get_entry
	s.AddTool(
		mcp.NewTool("get_entry",
			mcp.WithDescription("Get a single context entry by ID or name."),
			mcp.WithString("id_or_name",
				mcp.Description("Entry ID or name to look up."),
				mcp.Required(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			idOrName, _ := req.Params.Arguments["id_or_name"].(string)
			if idOrName == "" {
				return mcp.NewToolResultError("id_or_name is required"), nil
			}
			e, err := st.Get(idOrName)
			if err != nil {
				// Try by name
				e, err = st.GetByName(idOrName)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("entry not found: %s", idOrName)), nil
				}
			}
			return mcp.NewToolResultText(formatEntry(e)), nil
		},
	)

	// search_entries
	s.AddTool(
		mcp.NewTool("search_entries",
			mcp.WithDescription("Full-text search over active context entries."),
			mcp.WithString("query",
				mcp.Description("Search query string."),
				mcp.Required(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			query, _ := req.Params.Arguments["query"].(string)
			if query == "" {
				return mcp.NewToolResultError("query is required"), nil
			}
			entries, err := st.Search(query)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("search error: %v", err)), nil
			}
			return mcp.NewToolResultText(formatEntries(entries)), nil
		},
	)

	// add_entry
	s.AddTool(
		mcp.NewTool("add_entry",
			mcp.WithDescription("Propose a new context entry for review. The entry is queued as pending until approved in the TUI."),
			mcp.WithString("kind",
				mcp.Description("Entry kind (e.g. repo, note, doc)."),
				mcp.Required(),
			),
			mcp.WithString("name",
				mcp.Description("Entry name (must be unique among active entries)."),
				mcp.Required(),
			),
			mcp.WithString("body", mcp.Description("Free-form body text.")),
			mcp.WithString("fields", mcp.Description("JSON object of key/value string pairs (e.g. {\"path\": \"~/dev/foo\", \"lang\": \"go\"}).")),
			mcp.WithString("tags", mcp.Description("Comma-separated tags.")),
			mcp.WithString("agent", mcp.Description("Name of the agent proposing this entry.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			kind, _ := req.Params.Arguments["kind"].(string)
			name, _ := req.Params.Arguments["name"].(string)
			if kind == "" {
				return mcp.NewToolResultError("kind is required"), nil
			}
			if name == "" {
				return mcp.NewToolResultError("name is required"), nil
			}

			body, _ := req.Params.Arguments["body"].(string)
			fieldsStr, _ := req.Params.Arguments["fields"].(string)
			tagsStr, _ := req.Params.Arguments["tags"].(string)
			agent, _ := req.Params.Arguments["agent"].(string)

			fields, err := parseFields(fieldsStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid fields JSON: %v", err)), nil
			}

			proposal := entry.Entry{
				Kind:       kind,
				Name:       name,
				Body:       body,
				Tags:       parseTags(tagsStr),
				Fields:     fields,
				ProposedBy: agent,
			}

			created, err := st.CreatePending(proposal)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("add_entry error: %v", err)), nil
			}

			result := map[string]string{
				"status":      "queued",
				"proposal_id": created.ID,
				"message":     "Queued for review in naitv-mcp TUI. Run 'naitv-mcp tui' to approve.",
			}
			b, _ := json.Marshal(result)
			return mcp.NewToolResultText(string(b)), nil
		},
	)

	// update_entry
	s.AddTool(
		mcp.NewTool("update_entry",
			mcp.WithDescription("Propose an update to an existing active entry. The update is queued as pending until approved in the TUI."),
			mcp.WithString("id",
				mcp.Description("ID of the active entry to update."),
				mcp.Required(),
			),
			mcp.WithString("name", mcp.Description("New name (leave empty to keep existing).")),
			mcp.WithString("body", mcp.Description("New body text (leave empty to keep existing).")),
			mcp.WithString("fields", mcp.Description("JSON object of key/value pairs to merge in (leave empty to keep existing).")),
			mcp.WithString("tags", mcp.Description("New comma-separated tags (leave empty to keep existing).")),
			mcp.WithString("agent", mcp.Description("Name of the agent proposing this update.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id, _ := req.Params.Arguments["id"].(string)
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}

			name, _ := req.Params.Arguments["name"].(string)
			body, _ := req.Params.Arguments["body"].(string)
			fieldsStr, _ := req.Params.Arguments["fields"].(string)
			tagsStr, _ := req.Params.Arguments["tags"].(string)
			agent, _ := req.Params.Arguments["agent"].(string)

			fields, err := parseFields(fieldsStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid fields JSON: %v", err)), nil
			}

			proposal := entry.Entry{
				TargetID:   id,
				Name:       name,
				Body:       body,
				Tags:       parseTags(tagsStr),
				Fields:     fields,
				ProposedBy: agent,
			}

			created, err := st.CreatePending(proposal)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("update_entry error: %v", err)), nil
			}

			result := map[string]string{
				"status":      "queued",
				"proposal_id": created.ID,
				"message":     "Queued for review in naitv-mcp TUI. Run 'naitv-mcp tui' to approve.",
			}
			b, _ := json.Marshal(result)
			return mcp.NewToolResultText(string(b)), nil
		},
	)

	return server.ServeStdio(s)
}

// parseTags splits a comma-separated string into a slice of trimmed, non-empty strings.
func parseTags(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// parseFields unmarshals a JSON object string into a map. Empty string returns nil.
func parseFields(s string) (map[string]string, error) {
	if s == "" {
		return nil, nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	return m, nil
}

// formatEntries renders a slice of entries as a human-readable text block.
func formatEntries(entries []entry.Entry) string {
	if len(entries) == 0 {
		return "No entries found."
	}
	var sb strings.Builder
	for i, e := range entries {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(formatEntry(e))
	}
	return sb.String()
}

// formatEntry renders a single entry as a human-readable text block.
func formatEntry(e entry.Entry) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "[%s] %s\n", e.Kind, e.Name)

	if len(e.Tags) > 0 {
		fmt.Fprintf(&sb, "  tags: %s\n", strings.Join(e.Tags, ", "))
	}

	if len(e.Fields) > 0 {
		parts := make([]string, 0, len(e.Fields))
		for k, v := range e.Fields {
			parts = append(parts, fmt.Sprintf("%s: %s", k, v))
		}
		fmt.Fprintf(&sb, "  %s\n", strings.Join(parts, "  "))
	}

	if e.Body != "" {
		for _, line := range strings.Split(e.Body, "\n") {
			fmt.Fprintf(&sb, "  %s\n", line)
		}
	}

	return sb.String()
}
