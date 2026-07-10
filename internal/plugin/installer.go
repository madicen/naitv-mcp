package plugin

import (
	"fmt"
	"strings"

	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/internal/xpath"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

// InstallResult summarizes the outcome of an Install call.
type InstallResult struct {
	Manifest Manifest // parsed manifest that was installed
	Source   string   // resolved URL or file path
	Proposed []string // entry names queued as pending proposals
	Skipped  []string // entry names that already existed (not re-proposed)
}

// UninstallResult summarizes the outcome of an Uninstall call.
type UninstallResult struct {
	Name    string
	Removed []string // entry names that were deleted
	Missing []string // entry names not found (already manually deleted)
}

// Install fetches a plugin manifest from source and proposes all its entries
// as pending in st, preserving the human-in-the-loop approval gate.
//
// source may be:
//   - a plugin name ("loop-engineering-go")  → resolved via DefaultRegistryURL
//   - a URL ("https://...")                  → fetched directly
//   - a local file path ("./plugins/foo.json", "~/foo.json", "/abs/path")
//
// Returns an error if the plugin is already installed. Call Uninstall first
// to reinstall.
func Install(st *store.Store, source string) (*InstallResult, error) {
	// Resolve a plain name to a URL via the default registry.
	resolvedSource := source
	if !xpath.IsHTTP(source) && !xpath.IsFilePath(source) {
		reg, err := LoadRegistry(DefaultRegistryURL)
		if err != nil {
			return nil, fmt.Errorf("registry lookup for %q: %w", source, err)
		}
		re := reg.Find(source)
		if re == nil {
			return nil, fmt.Errorf("plugin %q not found in registry", source)
		}
		resolvedSource = re.URL
	}

	m, err := Load(resolvedSource)
	if err != nil {
		return nil, err
	}

	// Reject if already installed.
	installed, _ := st.List("plugin", nil)
	for _, pe := range installed {
		if pe.Name == m.Name {
			return nil, fmt.Errorf("plugin %q is already installed (version %s); uninstall first to reinstall", m.Name, pe.Fields["version"])
		}
	}

	// Build a name index across all active + pending entries to skip duplicates.
	existing, _ := st.List("", nil)
	pending, _ := st.ListPending()
	nameSet := make(map[string]bool, len(existing)+len(pending))
	for _, e := range existing {
		nameSet[e.Name] = true
	}
	for _, e := range pending {
		nameSet[e.Name] = true
	}

	result := &InstallResult{Manifest: m, Source: resolvedSource}
	proposedBy := "plugin:" + m.Name
	for _, spec := range m.Entries {
		if nameSet[spec.Name] {
			result.Skipped = append(result.Skipped, spec.Name)
			continue
		}
		e := spec.ToEntry(proposedBy)
		if _, err := st.CreatePending(e); err != nil {
			return result, fmt.Errorf("propose %q: %w", spec.Name, err)
		}
		result.Proposed = append(result.Proposed, spec.Name)
		nameSet[spec.Name] = true
	}

	// Create an active plugin tracking entry so list_plugins / Plugins tab
	// can show it and uninstall can find it later.
	allNames := append(result.Proposed, result.Skipped...)
	_, _ = st.Create(entry.Entry{
		Kind:  "plugin",
		Name:  m.Name,
		Body:  m.Description,
		Tags:  m.Tags,
		Fields: map[string]string{
			"version":     m.Version,
			"source":      resolvedSource,
			"author":      m.Author,
			"entry_count": fmt.Sprintf("%d", len(m.Entries)),
			"entry_names": strings.Join(allNames, ","),
		},
	})
	return result, nil
}

// Uninstall removes the named plugin's tracking entry and all entries it
// originally proposed (both active and pending) from st.
func Uninstall(st *store.Store, name string) (*UninstallResult, error) {
	// Find the plugin tracking entry.
	pluginEntries, _ := st.List("plugin", nil)
	var track *entry.Entry
	for i, pe := range pluginEntries {
		if pe.Name == name {
			track = &pluginEntries[i]
			break
		}
	}
	if track == nil {
		return nil, fmt.Errorf("plugin %q is not installed", name)
	}

	// Parse the comma-separated entry_names field.
	var entryNames []string
	for _, n := range strings.Split(track.Fields["entry_names"], ",") {
		n = strings.TrimSpace(n)
		if n != "" {
			entryNames = append(entryNames, n)
		}
	}

	// Build a name→ID index across active + pending entries.
	active, _ := st.List("", nil)
	pending, _ := st.ListPending()
	nameToID := make(map[string]string, len(active)+len(pending))
	for _, e := range active {
		nameToID[e.Name] = e.ID
	}
	for _, e := range pending {
		nameToID[e.Name] = e.ID
	}

	result := &UninstallResult{Name: name}
	for _, n := range entryNames {
		id, ok := nameToID[n]
		if !ok {
			result.Missing = append(result.Missing, n)
			continue
		}
		if err := st.Delete(id); err != nil {
			result.Missing = append(result.Missing, n)
			continue
		}
		result.Removed = append(result.Removed, n)
	}
	_ = st.Delete(track.ID)
	return result, nil
}
