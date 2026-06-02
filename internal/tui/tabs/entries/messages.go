package entries

import "github.com/madicen/naitv-mcp/pkg/entry"

// EntriesLoadedMsg is sent when entries have been loaded from the store.
type EntriesLoadedMsg struct {
	Entries []entry.Entry
	Kinds   []string
}

// EntryDeletedMsg is sent when an entry has been deleted from the store.
type EntryDeletedMsg struct {
	ID string
}

// SearchResultsMsg is sent when a search has completed.
type SearchResultsMsg struct {
	Entries []entry.Entry
	Query   string
}
