package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/madicen/naitv-mcp/internal/instructions"
	"github.com/madicen/naitv-mcp/internal/plugin"
	"github.com/madicen/naitv-mcp/internal/setup"
	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/internal/tools"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Version is set at build time via -ldflags.
var Version = "0.1.0"

// Run registers all tools and starts serving over stdio.
func Run(st *store.Store) error {
	server := NewServer(st)
	return server.Run(context.Background(), &sdkmcp.StdioTransport{})
}

// NewServer builds a configured MCP server without starting a transport.
func NewServer(st *store.Store) *sdkmcp.Server {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "naitv-mcp", Version: Version}, nil)
	registerStaticTools(server, st)
	registerResources(server, st)
	registerPrompts(server, st)
	wireResourceNotifications(server, st)
	if err := registerDynamicTools(server, st); err != nil {
		fmt.Fprintf(os.Stderr, "naitv-mcp: dynamic tools: %v\n", err)
	}
	return server
}

func textResult(text string) (*sdkmcp.CallToolResult, any, error) {
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: text}},
	}, nil, nil
}

func toolError(format string, args ...any) (*sdkmcp.CallToolResult, any, error) {
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: fmt.Sprintf(format, args...)}},
		IsError: true,
	}, nil, nil
}

type listEntriesArgs struct {
	Kind string `json:"kind,omitempty" jsonschema:"Entry kind to filter by (e.g. repo, note, doc). Empty means all kinds."`
	Tags string `json:"tags,omitempty" jsonschema:"Comma-separated tags to filter by (AND logic). Empty means no tag filter."`
}

type getEntryArgs struct {
	IDOrName string `json:"id_or_name" jsonschema:"Entry ID or name to look up."`
}

type searchEntriesArgs struct {
	Query string `json:"query" jsonschema:"Search query string."`
}

type addEntryArgs struct {
	Kind  string `json:"kind" jsonschema:"Entry kind (e.g. repo, note, doc, tool)."`
	Name  string `json:"name" jsonschema:"Entry name (must be unique among active entries)."`
	Body  string `json:"body,omitempty" jsonschema:"Free-form body text."`
	Fields string `json:"fields,omitempty" jsonschema:"JSON object of key/value string pairs."`
	Tags  string `json:"tags,omitempty" jsonschema:"Comma-separated tags."`
	Group string `json:"group,omitempty" jsonschema:"Display group for this entry in the TUI."`
	Agent string `json:"agent,omitempty" jsonschema:"Name of the agent proposing this entry."`
}

type updateEntryArgs struct {
	ID     string `json:"id" jsonschema:"ID of the active entry to update."`
	Name   string `json:"name,omitempty" jsonschema:"New name (leave empty to keep existing)."`
	Body   string `json:"body,omitempty" jsonschema:"New body text (leave empty to keep existing)."`
	Fields string `json:"fields,omitempty" jsonschema:"JSON object of key/value pairs to merge in."`
	Tags   string `json:"tags,omitempty" jsonschema:"New comma-separated tags (leave empty to keep existing)."`
	Group  string `json:"group,omitempty" jsonschema:"New display group (leave empty to keep existing)."`
	Agent  string `json:"agent,omitempty" jsonschema:"Name of the agent proposing this update."`
}

type installPluginArgs struct {
	Source string `json:"source" jsonschema:"Plugin source: URL, local path, or registry name."`
}

type listAvailablePluginsArgs struct {
	RegistryURL string `json:"registry_url,omitempty" jsonschema:"Registry URL to query (default: public registry)."`
}

type uninstallPluginArgs struct {
	Name string `json:"name" jsonschema:"Plugin name to uninstall."`
}

type setProjectArgs struct {
	ProjectDir string `json:"project_dir" jsonschema:"Absolute path to the project root."`
	EnableLint string `json:"enable_lint,omitempty" jsonschema:"Set to true to enable the lint tool."`
}

func registerStaticTools(s *sdkmcp.Server, st *store.Store) {
	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "initialize",
		Description: "Return the user's standing instructions, rendered from context entries marked for initialization (rules, tooling preferences, workflows, agent roles, repos, facts, notes). Call this at the start of a session to work the way the user prefers. Entries marked on-demand are not included here; fetch those with get_entry or search_entries when relevant.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, _ struct{}) (*sdkmcp.CallToolResult, any, error) {
		entries, err := st.List("", nil)
		if err != nil {
			return toolError("initialize error: %v", err)
		}
		return textResult(instructions.Render(instructions.FilterInit(entries)))
	})

	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "list_entries",
		Description: "List active context entries, optionally filtered by kind and/or tags.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, args listEntriesArgs) (*sdkmcp.CallToolResult, any, error) {
		entries, err := st.List(args.Kind, parseTags(args.Tags))
		if err != nil {
			return toolError("list_entries error: %v", err)
		}
		return textResult(formatEntries(entries))
	})

	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "get_entry",
		Description: "Get a single context entry by ID or name.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, args getEntryArgs) (*sdkmcp.CallToolResult, any, error) {
		if args.IDOrName == "" {
			return toolError("id_or_name is required")
		}
		e, err := st.Get(args.IDOrName)
		if err != nil {
			e, err = st.GetByName(args.IDOrName)
			if err != nil {
				return toolError("entry not found: %s", args.IDOrName)
			}
		}
		return textResult(formatEntry(e))
	})

	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "search_entries",
		Description: "Full-text search over active context entries.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, args searchEntriesArgs) (*sdkmcp.CallToolResult, any, error) {
		if args.Query == "" {
			return toolError("query is required")
		}
		entries, err := st.Search(args.Query)
		if err != nil {
			return toolError("search error: %v", err)
		}
		return textResult(formatEntries(entries))
	})

	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "add_entry",
		Description: "Propose a new context entry for review. The entry is queued as pending until approved in the TUI. To propose an executable tool, set kind=\"tool\" and include an \"exec\" field containing the shell command.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, args addEntryArgs) (*sdkmcp.CallToolResult, any, error) {
		if args.Kind == "" {
			return toolError("kind is required")
		}
		if args.Name == "" {
			return toolError("name is required")
		}
		fields, err := parseFields(args.Fields)
		if err != nil {
			return toolError("invalid fields JSON: %v", err)
		}
		result, err := proposeEntry(st, entryProposalSpec{
			Kind:       args.Kind,
			Name:       args.Name,
			Group:      args.Group,
			Body:       args.Body,
			Tags:       parseTags(args.Tags),
			Fields:     fields,
			ProposedBy: args.Agent,
		})
		if err != nil {
			return toolError("add_entry error: %v", err)
		}
		text, err := marshalProposalResult(result)
		if err != nil {
			return toolError("%v", err)
		}
		return textResult(text)
	})

	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "update_entry",
		Description: "Propose an update to an existing active entry. The update is queued as pending until approved in the TUI.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, args updateEntryArgs) (*sdkmcp.CallToolResult, any, error) {
		if args.ID == "" {
			return toolError("id is required")
		}
		fields, err := parseFields(args.Fields)
		if err != nil {
			return toolError("invalid fields JSON: %v", err)
		}
		result, err := proposeEntry(st, entryProposalSpec{
			TargetID:   args.ID,
			Name:       args.Name,
			Group:      args.Group,
			Body:       args.Body,
			Tags:       parseTags(args.Tags),
			Fields:     fields,
			ProposedBy: args.Agent,
		})
		if err != nil {
			return toolError("update_entry error: %v", err)
		}
		text, err := marshalProposalResult(result)
		if err != nil {
			return toolError("%v", err)
		}
		return textResult(text)
	})

	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "list_tools",
		Description: "List all active executable tool entries (kind=tool entries with an exec field).",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, _ struct{}) (*sdkmcp.CallToolResult, any, error) {
		defs, err := tools.ListDefs(st)
		if err != nil {
			return toolError("list_tools error: %v", err)
		}
		return textResult(formatToolDefs(defs))
	})

	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "install_plugin",
		Description: "Install a naitv-mcp plugin from a URL, local file path, or plugin name looked up in the registry. Proposes all plugin entries as pending.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, args installPluginArgs) (*sdkmcp.CallToolResult, any, error) {
		if args.Source == "" {
			return toolError("source is required")
		}
		result, err := plugin.Install(st, args.Source)
		if err != nil {
			return toolError("install_plugin: %v", err)
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "Plugin %q v%s installed.\n\n", result.Manifest.Name, result.Manifest.Version)
		if len(result.Proposed) > 0 {
			fmt.Fprintf(&sb, "Proposed %d entries (pending TUI approval):\n", len(result.Proposed))
			for _, n := range result.Proposed {
				fmt.Fprintf(&sb, "  + %s\n", n)
			}
		}
		if len(result.Skipped) > 0 {
			fmt.Fprintf(&sb, "\nSkipped %d entries (already exist):\n", len(result.Skipped))
			for _, n := range result.Skipped {
				fmt.Fprintf(&sb, "  ~ %s\n", n)
			}
		}
		sb.WriteString("\nReview and approve entries in the naitv-mcp TUI (Review tab).")
		sb.WriteString("\nRestart naitv-mcp serve after approving to activate executable tools.")
		if len(result.Proposed) > 0 {
			sb.WriteString("\nThen call set_project to point the tools at your project directory.")
		}
		return textResult(sb.String())
	})

	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "list_plugins",
		Description: "List all installed plugins with version, source, and entry count.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, _ struct{}) (*sdkmcp.CallToolResult, any, error) {
		pluginEntries, err := st.List("plugin", nil)
		if err != nil {
			return toolError("list_plugins: %v", err)
		}
		if len(pluginEntries) == 0 {
			return textResult("No plugins installed. Use install_plugin to install one.")
		}
		var sb strings.Builder
		for _, pe := range pluginEntries {
			fmt.Fprintf(&sb, "plugin: %s  v%s\n", pe.Name, pe.Fields["version"])
			if pe.Fields["author"] != "" {
				fmt.Fprintf(&sb, "  author:  %s\n", pe.Fields["author"])
			}
			fmt.Fprintf(&sb, "  source:  %s\n", pe.Fields["source"])
			fmt.Fprintf(&sb, "  entries: %s\n", pe.Fields["entry_count"])
			if pe.Body != "" {
				fmt.Fprintf(&sb, "  desc:    %s\n", pe.Body)
			}
			sb.WriteString("\n")
		}
		return textResult(strings.TrimRight(sb.String(), "\n"))
	})

	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "list_available_plugins",
		Description: fmt.Sprintf("Fetch and display plugins available in the naitv-plugins registry. Default: %s", plugin.DefaultRegistryURL),
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, args listAvailablePluginsArgs) (*sdkmcp.CallToolResult, any, error) {
		registryURL := args.RegistryURL
		if registryURL == "" {
			registryURL = plugin.DefaultRegistryURL
		}
		reg, err := plugin.LoadRegistry(registryURL)
		if err != nil {
			return toolError("list_available_plugins: %v", err)
		}
		if len(reg.Plugins) == 0 {
			return textResult("Registry contains no plugins.")
		}
		return textResult(formatRegistry(reg))
	})

	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "uninstall_plugin",
		Description: "Remove an installed plugin and all of its entries (active and pending) from naitv-mcp.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, args uninstallPluginArgs) (*sdkmcp.CallToolResult, any, error) {
		if args.Name == "" {
			return toolError("name is required")
		}
		result, err := plugin.Uninstall(st, args.Name)
		if err != nil {
			return toolError("uninstall_plugin: %v", err)
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "Plugin %q uninstalled.\n", result.Name)
		if len(result.Removed) > 0 {
			fmt.Fprintf(&sb, "\nRemoved %d entries:\n", len(result.Removed))
			for _, n := range result.Removed {
				fmt.Fprintf(&sb, "  - %s\n", n)
			}
		}
		if len(result.Missing) > 0 {
			sb.WriteString("\nNot found (may have been manually deleted):\n")
			for _, n := range result.Missing {
				fmt.Fprintf(&sb, "  ? %s\n", n)
			}
		}
		sb.WriteString("\nRestart naitv-mcp serve for changes to take effect.")
		return textResult(sb.String())
	})

	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "set_project",
		Description: "Update the working_dir field on all active executable tool entries to point at the given project root.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, args setProjectArgs) (*sdkmcp.CallToolResult, any, error) {
		projectDir, err := setup.ResolveDir(args.ProjectDir)
		if err != nil {
			return toolError("project_dir: %v", err)
		}
		if projectDir == "" {
			return toolError("project_dir is required")
		}
		enableLint := args.EnableLint == "true"
		result, err := setup.SetProject(st, projectDir, enableLint, false)
		if err != nil {
			return toolError("set_project: %v", err)
		}
		var sb strings.Builder
		if len(result.Updated) > 0 {
			fmt.Fprintf(&sb, "Updated working_dir → %s on:\n", projectDir)
			for _, n := range result.Updated {
				fmt.Fprintf(&sb, "  ✓ %s\n", n)
			}
		}
		if len(result.Skipped) > 0 {
			sb.WriteString("\nAlready correct (skipped):\n")
			for _, n := range result.Skipped {
				fmt.Fprintf(&sb, "  ~ %s\n", n)
			}
		}
		if len(result.Updated) > 0 {
			sb.WriteString("\nRestart naitv-mcp serve for the changes to take effect.")
		} else {
			sb.WriteString("No changes needed — all tools already point at the correct directory.")
		}
		return textResult(sb.String())
	})

	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "generate_continue_config",
		Description: "Generate the text of a .continue/config.yaml file wired to use this naitv-mcp server.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, _ struct{}) (*sdkmcp.CallToolResult, any, error) {
		defs, _ := tools.ListDefs(st)
		toolNames := make([]string, 0, len(defs))
		for _, d := range defs {
			toolNames = append(toolNames, d.Name)
		}
		binaryPath, err := os.Executable()
		if err != nil {
			binaryPath = "naitv-mcp"
		}
		return textResult(setup.ContinueConfig(toolNames, binaryPath))
	})

	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "export_entries",
		Description: "Export all entries as JSON (schema_version, exported_at, entries). Use for backup or sync between machines.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, _ struct{}) (*sdkmcp.CallToolResult, any, error) {
		var buf strings.Builder
		if err := st.ExportJSON(&buf); err != nil {
			return toolError("export_entries error: %v", err)
		}
		return textResult(buf.String())
	})
}

func registerDynamicTools(s *sdkmcp.Server, st *store.Store) error {
	defs, err := tools.ListDefs(st)
	if err != nil {
		return err
	}
	for _, def := range defs {
		registerOne(s, def)
	}
	if len(defs) > 0 {
		fmt.Fprintf(os.Stderr, "naitv-mcp: registered %d dynamic tool(s)\n", len(defs))
	}
	return nil
}

func registerOne(s *sdkmcp.Server, def tools.Def) {
	desc := def.Description
	if desc == "" {
		desc = fmt.Sprintf("Run: %s", def.Exec)
	}
	if def.Disabled {
		desc = "[disabled] " + desc
	}

	tool := &sdkmcp.Tool{
		Name:        def.Name,
		Description: desc,
		InputSchema: dynamicToolInputSchema(def.Params),
	}

	captured := def
	s.AddTool(tool, func(ctx context.Context, req *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error) {
		args := make(map[string]string, len(captured.Params))
		var raw map[string]any
		if len(req.Params.Arguments) > 0 {
			_ = json.Unmarshal(req.Params.Arguments, &raw)
		}
		for _, p := range captured.Params {
			if v, ok := raw[p.Name].(string); ok {
				args[p.Name] = v
			}
		}
		result := tools.Run(ctx, captured, args)
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: result.Format()}},
		}, nil
	})
}

func dynamicToolInputSchema(params []tools.Param) map[string]any {
	properties := map[string]any{}
	required := []string{}
	for _, p := range params {
		prop := map[string]any{
			"type":        "string",
			"description": p.Description,
		}
		properties[p.Name] = prop
		if p.Required {
			required = append(required, p.Name)
		}
	}
	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

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
