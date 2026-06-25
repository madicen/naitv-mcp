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
	"github.com/madicen/naitv-mcp/pkg/entry"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Run registers all tools and starts serving over stdio.
func Run(st *store.Store) error {
	s := server.NewMCPServer("naitv-mcp", "0.1.0")

	registerStaticTools(s, st)

	if err := registerDynamicTools(s, st); err != nil {
		// Non-fatal: log and continue — bad entries shouldn't block the server.
		fmt.Fprintf(os.Stderr, "naitv-mcp: dynamic tools: %v\n", err)
	}

	return server.ServeStdio(s)
}

// registerStaticTools adds the built-in knowledge-base tools.
func registerStaticTools(s *server.MCPServer, st *store.Store) {
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
			mcp.WithDescription("Propose a new context entry for review. The entry is queued as pending until approved in the TUI. To propose an executable tool, set kind=\"tool\" and include an \"exec\" field containing the shell command. Optionally include \"working_dir\", \"timeout\" (e.g. \"30s\"), \"params\" (JSON array of {name,description,required}), and \"disabled\" (\"true\"/\"false\"). Executable tool proposals appear with a warning in the Review tab since they run shell commands on approval."),
			mcp.WithString("kind",
				mcp.Description("Entry kind (e.g. repo, note, doc, tool)."),
				mcp.Required(),
			),
			mcp.WithString("name",
				mcp.Description("Entry name (must be unique among active entries)."),
				mcp.Required(),
			),
			mcp.WithString("body", mcp.Description("Free-form body text.")),
			mcp.WithString("fields", mcp.Description("JSON object of key/value string pairs (e.g. {\"path\": \"~/dev/foo\", \"lang\": \"go\"}). For executable tools: include \"exec\" (required), \"working_dir\", \"timeout\", \"params\", \"disabled\".")),
			mcp.WithString("tags", mcp.Description("Comma-separated tags.")),
			mcp.WithString("group", mcp.Description("Display group for this entry in the TUI (e.g. the current project name). Overrides the default plugin-based grouping. Leave empty for General.")),
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
			group, _ := req.Params.Arguments["group"].(string)
			agent, _ := req.Params.Arguments["agent"].(string)

			fields, err := parseFields(fieldsStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid fields JSON: %v", err)), nil
			}

			proposal := entry.Entry{
				Kind:       kind,
				Name:       name,
				Group:      group,
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
			mcp.WithString("group", mcp.Description("New display group (leave empty to keep existing).")),
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
			group, _ := req.Params.Arguments["group"].(string)
			agent, _ := req.Params.Arguments["agent"].(string)

			fields, err := parseFields(fieldsStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid fields JSON: %v", err)), nil
			}

			proposal := entry.Entry{
				TargetID:   id,
				Name:       name,
				Group:      group,
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

	// list_tools
	s.AddTool(
		mcp.NewTool("list_tools",
			mcp.WithDescription("List all active executable tool entries (kind=tool entries with an exec field). Shows the MCP tool name, exec command, parameters, and whether the tool is disabled. Use this to discover what tools are available before calling them, or to check configuration."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			entries, err := st.List("tool", nil)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("list_tools error: %v", err)), nil
			}

			var defs []tools.Def
			for _, e := range entries {
				if !tools.IsExecutable(e) {
					continue
				}
				def, err := tools.ParseDef(e)
				if err != nil {
					continue
				}
				defs = append(defs, def)
			}

			if len(defs) == 0 {
				return mcp.NewToolResultText("No executable tools defined. Add a tool entry with an exec field via add_entry or the TUI."), nil
			}

			var sb strings.Builder
			for _, d := range defs {
				status := "active"
				if d.Disabled {
					status = "disabled"
				}
				fmt.Fprintf(&sb, "tool: %s  [%s]\n", d.Name, status)
				fmt.Fprintf(&sb, "  exec:    %s\n", d.Exec)
				fmt.Fprintf(&sb, "  timeout: %s\n", d.Timeout)
				if d.WorkingDir != "" {
					fmt.Fprintf(&sb, "  dir:     %s\n", d.WorkingDir)
				}
				if len(d.Params) > 0 {
					fmt.Fprintf(&sb, "  params:\n")
					for _, p := range d.Params {
						req := ""
						if p.Required {
							req = " (required)"
						}
						fmt.Fprintf(&sb, "    {%s}%s — %s\n", p.Name, req, p.Description)
					}
				}
				if d.Description != "" {
					fmt.Fprintf(&sb, "  description: %s\n", d.Description)
				}
				sb.WriteString("\n")
			}
			return mcp.NewToolResultText(strings.TrimRight(sb.String(), "\n")), nil
		},
	)

	// install_plugin
	s.AddTool(
		mcp.NewTool("install_plugin",
			mcp.WithDescription("Install a naitv-mcp plugin from a URL, local file path, or plugin name looked up in the registry. Proposes all plugin entries as pending — review and approve them in the TUI Review tab. Plugin metadata is tracked in the store so you can list and uninstall later."),
			mcp.WithString("source",
				mcp.Description("Plugin source: a URL (https://...), local file path (./plugins/foo.json or ~/plugins/foo.json), or plugin name to look up in the registry (e.g. \"loop-engineering-go\")."),
				mcp.Required(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			source, _ := req.Params.Arguments["source"].(string)
			if source == "" {
				return mcp.NewToolResultError("source is required"), nil
			}

			result, err := plugin.Install(st, source)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("install_plugin: %v", err)), nil
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
			return mcp.NewToolResultText(sb.String()), nil
		},
	)

	// list_plugins
	s.AddTool(
		mcp.NewTool("list_plugins",
			mcp.WithDescription("List all installed plugins with version, source, and entry count."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			pluginEntries, err := st.List("plugin", nil)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("list_plugins: %v", err)), nil
			}
			if len(pluginEntries) == 0 {
				return mcp.NewToolResultText("No plugins installed. Use install_plugin to install one."), nil
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
			return mcp.NewToolResultText(strings.TrimRight(sb.String(), "\n")), nil
		},
	)

	// list_available_plugins
	s.AddTool(
		mcp.NewTool("list_available_plugins",
			mcp.WithDescription(fmt.Sprintf(
				"Fetch and display plugins available in the naitv-plugins registry. Pass a custom registry_url to use a non-default registry. Default: %s",
				plugin.DefaultRegistryURL,
			)),
			mcp.WithString("registry_url",
				mcp.Description("Registry URL to query (default: the public naitv-plugins registry)."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			registryURL, _ := req.Params.Arguments["registry_url"].(string)
			if registryURL == "" {
				registryURL = plugin.DefaultRegistryURL
			}
			reg, err := plugin.LoadRegistry(registryURL)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("list_available_plugins: %v", err)), nil
			}
			if len(reg.Plugins) == 0 {
				return mcp.NewToolResultText("Registry contains no plugins."), nil
			}
			return mcp.NewToolResultText(formatRegistry(reg)), nil
		},
	)

	// uninstall_plugin
	s.AddTool(
		mcp.NewTool("uninstall_plugin",
			mcp.WithDescription("Remove an installed plugin and all of its entries (active and pending) from naitv-mcp."),
			mcp.WithString("name",
				mcp.Description("Plugin name to uninstall (as shown by list_plugins)."),
				mcp.Required(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name, _ := req.Params.Arguments["name"].(string)
			if name == "" {
				return mcp.NewToolResultError("name is required"), nil
			}

			result, err := plugin.Uninstall(st, name)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("uninstall_plugin: %v", err)), nil
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
			return mcp.NewToolResultText(sb.String()), nil
		},
	)

	// set_project
	s.AddTool(
		mcp.NewTool("set_project",
			mcp.WithDescription("Update the working_dir field on all active executable tool entries to point at the given project root. This is a direct write (not a proposal) since it only changes a path. Also optionally enables or disables the lint tool."),
			mcp.WithString("project_dir",
				mcp.Description("Absolute path to the project root (e.g. /home/user/dev/myapp). Use '.' to use the current directory of the naitv-mcp process."),
				mcp.Required(),
			),
			mcp.WithString("enable_lint",
				mcp.Description("Set to \"true\" to enable the lint tool (sets disabled=false). Requires golangci-lint to be installed."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			rawDir, _ := req.Params.Arguments["project_dir"].(string)
			enableLintStr, _ := req.Params.Arguments["enable_lint"].(string)

			projectDir, err := setup.ResolveDir(rawDir)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("project_dir: %v", err)), nil
			}
			if projectDir == "" {
				return mcp.NewToolResultError("project_dir is required"), nil
			}

			enableLint := enableLintStr == "true"

			result, err := setup.SetProject(st, projectDir, enableLint, false)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("set_project: %v", err)), nil
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
			return mcp.NewToolResultText(sb.String()), nil
		},
	)

	// generate_continue_config
	s.AddTool(
		mcp.NewTool("generate_continue_config",
			mcp.WithDescription("Generate the text of a .continue/config.yaml file wired to use this naitv-mcp server. Returns the config as a string — display it to the user so they can save it to their project as .continue/config.yaml."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			toolEntries, _ := st.List("tool", nil)
			var toolNames []string
			for _, e := range toolEntries {
				if tools.IsExecutable(e) {
					def, err := tools.ParseDef(e)
					if err != nil {
						continue
					}
					toolNames = append(toolNames, def.Name)
				}
			}

			binaryPath, err := os.Executable()
			if err != nil {
				binaryPath = "naitv-mcp"
			}

			config := setup.ContinueConfig(toolNames, binaryPath)
			return mcp.NewToolResultText(config), nil
		},
	)
}

// registerDynamicTools loads active tool entries with an exec field from the
// store and registers each as a live MCP tool. Entries that fail to parse are
// logged and skipped so a bad definition doesn't prevent the server from starting.
func registerDynamicTools(s *server.MCPServer, st *store.Store) error {
	entries, err := st.List("tool", nil)
	if err != nil {
		return fmt.Errorf("load tool entries: %w", err)
	}

	registered := 0
	for _, e := range entries {
		if !tools.IsExecutable(e) {
			continue
		}
		def, err := tools.ParseDef(e)
		if err != nil {
			fmt.Fprintf(os.Stderr, "naitv-mcp: skipping tool entry %q: %v\n", e.Name, err)
			continue
		}
		registerOne(s, def)
		registered++
	}

	if registered > 0 {
		fmt.Fprintf(os.Stderr, "naitv-mcp: registered %d dynamic tool(s)\n", registered)
	}
	return nil
}

// registerOne registers a single Def as an MCP tool. def is passed by value so
// the closure captures an independent copy — safe across loop iterations.
func registerOne(s *server.MCPServer, def tools.Def) {
	desc := def.Description
	if desc == "" {
		desc = fmt.Sprintf("Run: %s", def.Exec)
	}
	if def.Disabled {
		desc = "[disabled] " + desc
	}

	opts := []mcp.ToolOption{mcp.WithDescription(desc)}

	for _, p := range def.Params {
		propOpts := []mcp.PropertyOption{mcp.Description(p.Description)}
		if p.Required {
			propOpts = append(propOpts, mcp.Required())
		}
		opts = append(opts, mcp.WithString(p.Name, propOpts...))
	}

	s.AddTool(
		mcp.NewTool(def.Name, opts...),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := make(map[string]string, len(def.Params))
			for _, p := range def.Params {
				if v, ok := req.Params.Arguments[p.Name].(string); ok {
					args[p.Name] = v
				}
			}
			result := tools.Run(ctx, def, args)
			return mcp.NewToolResultText(result.Format()), nil
		},
	)
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

// formatRegistry formats a Registry for display in tool responses.
func formatRegistry(reg plugin.Registry) string {
	var sb strings.Builder
	sb.WriteString("Available plugins:\n\n")
	for _, p := range reg.Plugins {
		fmt.Fprintf(&sb, "  %s  v%s\n", p.Name, p.Version)
		if p.Description != "" {
			fmt.Fprintf(&sb, "    %s\n", p.Description)
		}
		if len(p.Tags) > 0 {
			fmt.Fprintf(&sb, "    tags: %s\n", strings.Join(p.Tags, ", "))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("Install: install_plugin(source=\"<name>\")")
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
