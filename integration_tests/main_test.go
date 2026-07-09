package integration_tests

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/internal/tui"
)

func newTestDB(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func newTestModel(t *testing.T, st *store.Store) *tui.Model {
	t.Helper()
	m := tui.New(st)
	m.SetDimensions(120, 40)
	// Drive Init commands
	cmd := m.Init()
	if cmd != nil {
		runCmds(m, cmd)
	}
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
		if result == nil {
			break
		}
		next, cmd = m.Update(result)
		m = next.(*tui.Model)
	}
	return m
}

func runCmds(m *tui.Model, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	if msg == nil {
		return
	}
	_, _ = m.Update(msg)
}

func key(s string) tea.KeyPressMsg {
	if len(s) == 1 {
		return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
	}
	return tea.KeyPressMsg{Text: s}
}

func keyType(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}
