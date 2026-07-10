package integration_tests

import (
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/internal/tui"
)

func newTestDB(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil { t.Fatalf("open test store: %v", err) }
	t.Cleanup(func() { st.Close() })
	return st
}

func newTestModel(t *testing.T, st *store.Store) *tui.Model {
	t.Helper()
	m := tui.New(st)
	m.SetDimensions(120, 40)
	if cmd := m.Init(); cmd != nil { runCmds(m, cmd) }
	return m
}

func updateModel(m *tui.Model, msg tea.Msg) *tui.Model {
	next, _ := m.Update(msg)
	return next.(*tui.Model)
}

func runPendingCmds(m *tui.Model, msg tea.Msg, maxRounds int) *tui.Model {
	next, cmd := m.Update(msg)
	m = next.(*tui.Model)
	for i := 0; i < maxRounds && cmd != nil; i++ {
		result := cmd()
		if result == nil { break }
		next, cmd = m.Update(result)
		m = next.(*tui.Model)
	}
	return m
}

func runCmds(m *tui.Model, cmd tea.Cmd) {
	if cmd == nil { return }
	if msg := cmd(); msg != nil { _, _ = m.Update(msg) }
}

func key(s string) tea.KeyPressMsg {
	switch s {
	case "ctrl+s": return tea.KeyPressMsg{Text: "ctrl+s"}
	case "enter": return tea.KeyPressMsg{Code: tea.KeyEnter, Text: "enter"}
	case "esc": return tea.KeyPressMsg{Code: tea.KeyEsc, Text: "esc"}
	case "tab": return tea.KeyPressMsg{Code: tea.KeyTab, Text: "tab"}
	}
	if len(s) == 1 {
		r := rune(s[0])
		return tea.KeyPressMsg(tea.Key{Text: s, Code: r, ShiftedCode: r})
	}
	return tea.KeyPressMsg{Text: s}
}

func keyType(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }

func typeString(m *tui.Model, s string) *tui.Model {
	for _, r := range s { m = runPendingCmds(m, key(string(r)), 8) }
	return m
}

func pressTab(m *tui.Model) *tui.Model { return runPendingCmds(m, key("tab"), 25) }

func clickZone(t *testing.T, m *tui.Model, zoneID string) *tui.Model {
	t.Helper()
	_ = m.View().Content
	var z = m.ZoneManager().Get(zoneID)
	for i := 0; (z == nil || z.IsZero()) && i < 50; i++ {
		time.Sleep(time.Millisecond)
		z = m.ZoneManager().Get(zoneID)
	}
	if z == nil || z.IsZero() { t.Fatalf("zone %q not found", zoneID) }
	return runPendingCmds(m, tea.MouseClickMsg{Button: tea.MouseLeft, X: (z.StartX + z.EndX) / 2, Y: (z.StartY + z.EndY) / 2}, 8)
}
