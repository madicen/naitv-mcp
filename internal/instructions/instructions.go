// Package instructions renders active context entries into a single
// initialization document that can be fed to an AI agent (Cursor, Claude,
// a local model, etc.) so it works the way the user prefers.
package instructions

import (
	"fmt"
	"sort"
	"strings"

	"github.com/madicen/naitv-mcp/pkg/entry"
)

// kindSection describes how a given entry kind is presented in the rendered
// document: its heading and a short intro that frames the entries for the model.
type kindSection struct {
	kind    string
	heading string
	intro   string
}

// orderedSections lists the kinds we render first, in priority order. Rules and
// tool preferences come first because they are the strongest constraints; repos
// and background context come last. Any kind not listed here is rendered
// afterwards under a generic section.
var orderedSections = []kindSection{
	{kind: "rule", heading: "Rules", intro: "Follow these rules at all times."},
	{kind: "tool", heading: "Tooling preferences", intro: "Prefer these tools and conventions."},
	{kind: "workflow", heading: "Workflows", intro: "When performing these activities, follow the steps below."},
	{kind: "agent", heading: "Agent roles", intro: "Adopt these roles/personas when relevant."},
	{kind: "repo", heading: "Repositories", intro: "Reference these repositories for patterns, paths, and conventions."},
	{kind: "fact", heading: "Facts", intro: "Useful facts to keep in mind."},
	{kind: "note", heading: "Notes", intro: "Background context."},
}

const docHeader = `# Agent initialization

This document was generated from the naitv-mcp knowledge base. It describes how
the user wants AI agents to operate. Treat it as standing instructions for every
session.`

// FilterInit returns only the entries whose delivery mode is "init" (the ones
// that belong in the initialization bundle). Entries marked on-demand are
// excluded so agents fetch them directly instead.
func FilterInit(entries []entry.Entry) []entry.Entry {
	out := make([]entry.Entry, 0, len(entries))
	for _, e := range entries {
		if e.DeliveryOrDefault() == entry.DeliveryInit {
			out = append(out, e)
		}
	}
	return out
}

// FilterInitByKinds returns init-delivery entries optionally limited to kinds.
func FilterInitByKinds(entries []entry.Entry, kinds []string) []entry.Entry {
	filtered := FilterInit(entries)
	if len(kinds) == 0 {
		return filtered
	}
	allowed := make(map[string]struct{}, len(kinds))
	for _, k := range kinds {
		k = strings.TrimSpace(k)
		if k != "" {
			allowed[k] = struct{}{}
		}
	}
	if len(allowed) == 0 {
		return filtered
	}
	out := make([]entry.Entry, 0, len(filtered))
	for _, e := range filtered {
		if _, ok := allowed[e.Kind]; ok {
			out = append(out, e)
		}
	}
	return out
}

// Render turns the given active entries into a markdown initialization document.
// Entries are grouped by kind; well-known kinds are ordered and introduced, and
// any remaining kinds are appended under their own headings.
func Render(entries []entry.Entry) string {
	byKind := map[string][]entry.Entry{}
	for _, e := range entries {
		byKind[e.Kind] = append(byKind[e.Kind], e)
	}

	var sb strings.Builder
	sb.WriteString(docHeader)
	sb.WriteString("\n")

	rendered := map[string]bool{}

	for _, sec := range orderedSections {
		group := byKind[sec.kind]
		if len(group) == 0 {
			continue
		}
		writeSection(&sb, sec.heading, sec.intro, group)
		rendered[sec.kind] = true
	}

	// Any remaining kinds, in stable alphabetical order.
	remaining := make([]string, 0, len(byKind))
	for kind := range byKind {
		if !rendered[kind] {
			remaining = append(remaining, kind)
		}
	}
	sort.Strings(remaining)
	for _, kind := range remaining {
		writeSection(&sb, titleCase(kind), "", byKind[kind])
	}

	if len(entries) == 0 {
		sb.WriteString("\n_No context entries yet. Add some via the naitv-mcp TUI or the add_entry tool._\n")
	}

	return sb.String()
}

func writeSection(sb *strings.Builder, heading, intro string, entries []entry.Entry) {
	// Stable ordering within a section by name.
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })

	fmt.Fprintf(sb, "\n## %s\n\n", heading)
	if intro != "" {
		fmt.Fprintf(sb, "%s\n\n", intro)
	}
	for _, e := range entries {
		writeEntry(sb, e)
	}
}

func writeEntry(sb *strings.Builder, e entry.Entry) {
	fmt.Fprintf(sb, "### %s\n\n", e.Name)

	if body := strings.TrimSpace(e.Body); body != "" {
		// Multi-line bodies often read as a list of steps/instructions; keep
		// them as-is so newlines are preserved in the markdown.
		sb.WriteString(body)
		sb.WriteString("\n")
	}

	if len(e.Fields) > 0 {
		keys := make([]string, 0, len(e.Fields))
		for k := range e.Fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		sb.WriteString("\n")
		for _, k := range keys {
			fmt.Fprintf(sb, "- **%s**: %s\n", k, e.Fields[k])
		}
	}

	if len(e.Tags) > 0 {
		fmt.Fprintf(sb, "\n_tags: %s_\n", strings.Join(e.Tags, ", "))
	}

	sb.WriteString("\n")
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
