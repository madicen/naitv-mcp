package plugin

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/internal/xpath"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

type InstallResult struct {
	Manifest Manifest
	Source   string
	Proposed []string
	Skipped  []string
}

type UninstallResult struct {
	Name    string
	Removed []string
	Missing []string
}

func Install(st *store.Store, source string) (*InstallResult, error) {
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

	installed, _ := st.List("plugin", nil)
	for _, pe := range installed {
		if pe.Name == m.Name {
			return nil, fmt.Errorf("plugin %q is already installed (version %s); uninstall first to reinstall", m.Name, pe.Fields["version"])
		}
	}

	existing, _ := st.List("", nil)
	pending, _ := st.ListPending()
	nameSet := make(map[string]bool, len(existing)+len(pending))
	nameToID := make(map[string]string, len(existing)+len(pending))
	for _, e := range existing {
		nameSet[e.Name] = true
		nameToID[e.Name] = e.ID
	}
	for _, e := range pending {
		nameSet[e.Name] = true
		nameToID[e.Name] = e.ID
	}

	result := &InstallResult{Manifest: m, Source: resolvedSource}
	proposedBy := "plugin:" + m.Name
	var entryIDs []string
	for _, spec := range m.Entries {
		if nameSet[spec.Name] {
			result.Skipped = append(result.Skipped, spec.Name)
			if id, ok := nameToID[spec.Name]; ok {
				entryIDs = append(entryIDs, id)
			}
			continue
		}
		e := spec.ToEntry(proposedBy)
		created, err := st.CreatePending(e)
		if err != nil {
			return result, fmt.Errorf("propose %q: %w", spec.Name, err)
		}
		result.Proposed = append(result.Proposed, spec.Name)
		entryIDs = append(entryIDs, created.ID)
		nameSet[spec.Name] = true
	}

	idsJSON, _ := json.Marshal(entryIDs)
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
			"entry_ids":   string(idsJSON),
		},
	})
	return result, nil
}

func Uninstall(st *store.Store, name string) (*UninstallResult, error) {
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

	entryIDs := EntryIDs(st, *track)
	result := &UninstallResult{Name: name}

	for _, id := range entryIDs {
		e, err := st.Get(id)
		if err != nil {
			result.Missing = append(result.Missing, id)
			continue
		}
		if err := st.Delete(id); err != nil {
			result.Missing = append(result.Missing, e.Name)
			continue
		}
		result.Removed = append(result.Removed, e.Name)
	}
	_ = st.Delete(track.ID)
	return result, nil
}

func parseEntryIDs(track entry.Entry) []string {
	raw := track.Fields["entry_ids"]
	if raw == "" {
		return nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil
	}
	return ids
}

// EntryIDs returns linked entry IDs for a plugin tracker. Legacy trackers that
// only stored entry_names are migrated opportunistically on read.
func EntryIDs(st *store.Store, track entry.Entry) []string {
	if ids := parseEntryIDs(track); len(ids) > 0 {
		return ids
	}
	return migrateLegacyEntryNames(st, track)
}

func migrateLegacyEntryNames(st *store.Store, track entry.Entry) []string {
	raw := track.Fields["entry_names"]
	if raw == "" {
		return nil
	}
	active, _ := st.List("", nil)
	pending, _ := st.ListPending()
	nameToID := make(map[string]string, len(active)+len(pending))
	for _, e := range active {
		nameToID[e.Name] = e.ID
	}
	for _, e := range pending {
		nameToID[e.Name] = e.ID
	}
	var ids []string
	for _, n := range strings.Split(raw, ",") {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if id, ok := nameToID[n]; ok {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	if b, err := json.Marshal(ids); err == nil {
		track.Fields["entry_ids"] = string(b)
		delete(track.Fields, "entry_names")
		_, _ = st.Update(track)
	}
	return ids
}
