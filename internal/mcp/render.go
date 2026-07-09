package mcp

import (
	"fmt"
	"strings"

	"github.com/madicen/naitv-mcp/internal/plugin"
	"github.com/madicen/naitv-mcp/internal/tools"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

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

func formatToolDefs(defs []tools.Def) string {
	if len(defs) == 0 {
		return "No executable tools defined. Add a tool entry with an exec field via add_entry or the TUI."
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
	return strings.TrimRight(sb.String(), "\n")
}
