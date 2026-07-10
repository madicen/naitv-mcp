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
		e, _ := st.Get(id)
		_ = st.Delete(id)
		return EntryDeletedMsg{ID: id, Name: e.Name}
	}
}

// SearchCmd performs a full-text search in the store.
func SearchCmd(st *store.Store, query string) tea.Cmd {
	return func() tea.Msg {
		results, _ := st.Search(query)
		return SearchResultsMsg{Entries: results, Query: query}
	}
}

func loadEntriesMsgArchived(st *store.Store, kind string) tea.Msg {
	entries, _ := st.ListArchived(kind)
	kindSet := map[string]struct{}{}
	all, _ := st.List("", nil)
	for _, e := range all {
		kindSet[e.Kind] = struct{}{}
	}
	archived, _ := st.ListArchived("")
	for _, e := range archived {
		kindSet[e.Kind] = struct{}{}
	}
	var kinds []string
	for k := range kindSet {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	return EntriesLoadedMsg{Entries: entries, Kinds: kinds, Archived: true}
}

// LoadArchivedCmd loads archived entries filtered by kind.
func LoadArchivedCmd(st *store.Store, kind string) tea.Cmd {
	return func() tea.Msg {
		return loadEntriesMsgArchived(st, kind)
	}
}

// RestoreEntryCmd restores an archived entry.
func RestoreEntryCmd(st *store.Store, id, kind string) tea.Cmd {
	return func() tea.Msg {
		_ = st.Restore(id)
		return loadEntriesMsgArchived(st, kind)
	}
}

// PurgeEntryCmd permanently removes an entry.
func PurgeEntryCmd(st *store.Store, id, kind string) tea.Cmd {
	return func() tea.Msg {
		_ = st.Purge(id)
		return loadEntriesMsgArchived(st, kind)
	}
}

// LoadHistoryCmd loads version history for an entry.
func LoadHistoryCmd(st *store.Store, entryID string) tea.Cmd {
	return func() tea.Msg {
		records, _ := st.History(entryID)
		return HistoryLoadedMsg{Records: records}
	}
}

// RestoreVersionCmd restores a historical snapshot.
func RestoreVersionCmd(st *store.Store, historyID, kind string, archived bool) tea.Cmd {
	return func() tea.Msg {
		_, _ = st.RestoreVersion(historyID)
		if archived {
			return loadEntriesMsgArchived(st, kind)
		}
		return loadEntriesMsg(st, kind)
	}
}

// UndoCmd restores the most recent history snapshot for an entry.
func UndoCmd(st *store.Store, historyID, kind string, archived bool) tea.Cmd {
	return func() tea.Msg {
		if _, err := st.RestoreVersion(historyID); err != nil {
			return EntriesLoadedMsg{}
		}
		if archived {
			return loadEntriesMsgArchived(st, kind)
		}
		return loadEntriesMsg(st, kind)
	}
}
