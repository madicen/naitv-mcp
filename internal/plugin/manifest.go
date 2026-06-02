// Package plugin handles loading, parsing, and managing naitv-mcp plugins.
//
// A plugin is a JSON manifest that bundles a set of entry definitions. Plugins
// are fetched from a URL or local file and proposed to the store as pending
// entries, preserving the human-in-the-loop approval gate. Plugin metadata is
// tracked in the store as active kind=plugin entries so they can be listed and
// uninstalled later.
package plugin

import "github.com/madicen/naitv-mcp/pkg/entry"

// Manifest is a parsed plugin manifest.
type Manifest struct {
	Name        string      `json:"name"`
	Version     string      `json:"version"`
	Description string      `json:"description"`
	Author      string      `json:"author,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	Entries     []EntrySpec `json:"entries"`
}

// EntrySpec is one entry definition within a plugin manifest.
// It mirrors entry.Entry fields and is converted to one on install.
type EntrySpec struct {
	Kind     string            `json:"kind"`
	Name     string            `json:"name"`
	Body     string            `json:"body,omitempty"`
	Delivery entry.Delivery    `json:"delivery,omitempty"` // "init" or "on-demand"; default "init"
	Tags     []string          `json:"tags,omitempty"`
	Fields   map[string]string `json:"fields,omitempty"`
}

// ToEntry converts an EntrySpec to an entry.Entry suitable for st.CreatePending.
// proposedBy is typically "plugin:<manifest-name>" so the TUI Review tab shows
// which plugin is the source of the proposal.
func (s EntrySpec) ToEntry(proposedBy string) entry.Entry {
	d := entry.DeliveryInit
	if s.Delivery == entry.DeliveryOnDemand {
		d = entry.DeliveryOnDemand
	}
	return entry.Entry{
		Kind:       s.Kind,
		Name:       s.Name,
		Body:       s.Body,
		Delivery:   d,
		Tags:       s.Tags,
		Fields:     s.Fields,
		ProposedBy: proposedBy,
	}
}
