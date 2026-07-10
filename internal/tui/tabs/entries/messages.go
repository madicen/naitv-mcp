package entries

import (
	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

// EntriesLoadedMsg is sent when entries have been loaded from the store.
type EntriesLoadedMsg struct {
	Entries  []entry.Entry
	Kinds    []string
	Archived bool
}

// EntryDeletedMsg is sent when an entry has been deleted from the store.
type EntryDeletedMsg struct {
	ID   string
	Name string
}

// EntryRestoredMsg is sent when an archived entry has been restored.
type EntryRestoredMsg struct {
	ID string
}

// EntryPurgedMsg is sent when an entry has been permanently removed.
type EntryPurgedMsg struct {
	ID string
}

// HistoryLoadedMsg is sent when entry history has been loaded.
type HistoryLoadedMsg struct {
	Records []store.HistoryRecord
}

// VersionRestoredMsg is sent when a historical version was restored.
type VersionRestoredMsg struct {
	Entry entry.Entry
}

// SearchResultsMsg is sent when a search has completed.
type SearchResultsMsg struct {
	Entries []entry.Entry
	Query   string
}
