package entries

import (
	"sort"

	tea "charm.land/bubbletea/v2"
	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

// loadEntriesMsg loads entries filtered by kind and the distinct kind set.
func loadEntriesMsg(st *store.Store, kind string) tea.Msg {
	entries, _ := st.List(kind, nil)
	// collect distinct kinds for sub-tabs
	kindSet := map[string]struct{}{}
	all, _ := st.List("", nil)
	for _, e := range all {
		kindSet[e.Kind] = struct{}{}
	}
	var kinds []string
	for k := range kindSet {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	return EntriesLoadedMsg{Entries: entries, Kinds: kinds}
}

// LoadEntriesCmd loads entries from the store filtered by kind.
func LoadEntriesCmd(st *store.Store, kind string) tea.Cmd {
	return func() tea.Msg {
		return loadEntriesMsg(st, kind)
	}
}

// ToggleDeliveryCmd flips the delivery mode of an entry between init and
// on-demand, then reloads the entries list (filtered by kind).
func ToggleDeliveryCmd(st *store.Store, id, kind string) tea.Cmd {
	return func() tea.Msg {
		if e, err := st.Get(id); err == nil {
			next := entry.DeliveryInit
			if e.DeliveryOrDefault() == entry.DeliveryInit {
				next = entry.DeliveryOnDemand
			}
			_ = st.SetDelivery(id, next)
		}
		return loadEntriesMsg(st, kind)
	}
}

// DeleteEntryCmd deletes an entry from the store.
func DeleteEntryCmd(st *store.Store, id string) tea.Cmd {
	return func() tea.Msg {
		_ = st.Delete(id)
		return EntryDeletedMsg{ID: id}
	}
}

// SearchCmd performs a full-text search in the store.
func SearchCmd(st *store.Store, query string) tea.Cmd {
	return func() tea.Msg {
		results, _ := st.Search(query)
		return SearchResultsMsg{Entries: results, Query: query}
	}
}
