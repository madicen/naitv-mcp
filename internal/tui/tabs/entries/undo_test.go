package entries

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

func TestUpdate_UndoHistoryArchiveRequests(t *testing.T) {
	m := NewModel(zone.New())
	m.entries = []entry.Entry{{ID: "e1", Kind: "note", Name: "n1"}}
	m.buildFlatItems()
	m.SetDimensions(100, 30)

	_, req, _ := m.Update(pressKey("u"))
	if req == nil || !req.Undo {
		t.Fatalf("undo = %#v", req)
	}
	_, req, _ = m.Update(tea.KeyPressMsg(tea.Key{Text: "H", Code: 'h', ShiftedCode: 'H', Mod: tea.ModShift}))
	if req == nil || !req.ShowHistory {
		t.Fatalf("history = %#v", req)
	}

	m.showArchived = true
	_, req, _ = m.Update(pressKey("v"))
	if req == nil || !req.RestoreEntry {
		t.Fatalf("restore = %#v", req)
	}
	_, req, _ = m.Update(tea.KeyPressMsg(tea.Key{Text: "P", Code: 'p', ShiftedCode: 'P', Mod: tea.ModShift}))
	if req == nil || !req.PurgeEntry {
		t.Fatalf("purge = %#v", req)
	}
}

func TestHistoryLoadedShowsDetail(t *testing.T) {
	m := NewModel(zone.New())
	m.SetDimensions(100, 30)
	m, _, _ = m.Update(HistoryLoadedMsg{Records: []store.HistoryRecord{{
		ID: "h1", Action: "update", Snapshot: entry.Entry{Name: "snap"},
	}}})
	if !m.showHistory || m.View() == "" {
		t.Fatal("expected history detail view")
	}
	_, req, _ := m.Update(pressKey("enter"))
	if req == nil || !req.RestoreHistory {
		t.Fatalf("restore history = %#v", req)
	}
}

func TestLoadArchivedCmd(t *testing.T) {
	st, _ := store.Open(t.TempDir() + "/t.db")
	defer st.Close()
	e, _ := st.Create(entry.Entry{Kind: "note", Name: "arch", Body: "x"})
	_ = st.Delete(e.ID)
	msg := LoadArchivedCmd(st, "")().(EntriesLoadedMsg)
	if len(msg.Entries) != 1 || !msg.Archived {
		t.Fatalf("msg = %#v", msg)
	}
}
