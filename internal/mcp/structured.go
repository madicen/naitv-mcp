package mcp

import (
	"github.com/madicen/naitv-mcp/internal/tools"
	"github.com/madicen/naitv-mcp/pkg/entry"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func toolResult(text string, structured any) (*sdkmcp.CallToolResult, any, error) {
	return &sdkmcp.CallToolResult{
		Content:           []sdkmcp.Content{&sdkmcp.TextContent{Text: text}},
		StructuredContent: structured,
	}, nil, nil
}

func structuredEntry(e entry.Entry) map[string]any {
	fields := e.Fields
	if fields == nil {
		fields = map[string]string{}
	}
	return map[string]any{
		"id":         e.ID,
		"kind":       e.Kind,
		"name":       e.Name,
		"body":       e.Body,
		"tags":       e.Tags,
		"fields":     fields,
		"status":     string(e.Status),
		"delivery":   string(e.DeliveryOrDefault()),
		"group":      e.Group,
		"proposed_by": e.ProposedBy,
	}
}

func structuredEntries(entries []entry.Entry) map[string]any {
	out := make([]map[string]any, len(entries))
	for i, e := range entries {
		out[i] = structuredEntry(e)
	}
	return map[string]any{"entries": out, "count": len(entries)}
}

func structuredToolDefs(defs []tools.Def) map[string]any {
	out := make([]map[string]any, len(defs))
	for i, d := range defs {
		out[i] = map[string]any{
			"name":        d.Name,
			"exec":        d.Exec,
			"timeout":     d.Timeout.String(),
			"working_dir": d.WorkingDir,
			"disabled":    d.Disabled,
			"description": d.Description,
		}
	}
	return map[string]any{"tools": out, "count": len(defs)}
}
