package form

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

func TestPopulateFromAndToEntry(t *testing.T) {
	m := NewModel(zone.New())
	m.SetDimensions(120, 40)
	src := entry.Entry{
		Kind:   "tool",
		Name:   "demo-tool",
		Body:   "body",
		Tags:   []string{"a", "b"},
		Fields: map[string]string{"exec": "true", "env": "prod"},
		Group:  "grp",
	}
	m.PopulateFrom(src)
	got := m.ToEntry()
	if got.Name != src.Name || got.Body != src.Body || got.Fields["env"] != "prod" {
		t.Fatalf("ToEntry = %#v", got)
	}
	if len(got.Tags) != 2 {
		t.Fatalf("tags = %#v", got.Tags)
	}
}

func TestCancelEmitsCancelMsg(t *testing.T) {
	m := NewModel(zone.New())
	m.SetDimensions(120, 40)
	m.Show()
	m, cmd := m.Update(tea.KeyPressMsg{Text: "esc"})
	if cmd == nil {
		t.Fatal("expected cancel cmd")
	}
	if _, ok := cmd().(CancelMsg); !ok {
		t.Fatalf("msg = %T", cmd())
	}
}

func TestModeCreateSetsPendingFields(t *testing.T) {
	m := NewModel(zone.New())
	m.SetMode(ModeCreate)
	m.huhVals.name = "x"
	m.huhVals.body = "y"
	e := m.ToEntry()
	if e.Name != "x" || e.Body != "y" {
		t.Fatalf("entry = %#v", e)
	}
}

func TestSetKindsIncludesSentinel(t *testing.T) {
	m := NewModel(zone.New())
	m.SetKinds([]string{"rule", "note"})
	if len(m.kinds) < 2 {
		t.Fatalf("kinds = %#v", m.kinds)
	}
}
