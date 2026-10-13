package form

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/madicen/naitv-mcp/internal/tui/zones"
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

func TestTypingIntoNamePersists(t *testing.T) {
	m := NewModel(zone.New())
	m.SetDimensions(120, 40)
	m.Show()
	m.SetKinds([]string{"repo"})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Text: "tab"})
	for _, ch := range "hello" {
		m, _ = m.Update(tea.KeyPressMsg{Code: rune(ch), Text: string(ch)})
	}
	if m.ToEntry().Name != "hello" {
		t.Fatalf("name=%q", m.ToEntry().Name)
	}
}

func TestTabThroughHuhFields(t *testing.T) {
	m := NewModel(zone.New())
	m.SetDimensions(120, 40)
	m.Reset()
	m.SetKinds([]string{"repo"})
	m.Show()
	m.SetDimensions(120, 40)

	tab := tea.KeyPressMsg{Code: tea.KeyTab, Text: "tab"}
	for want := focusName; want <= focusBody; want++ {
		var cmd tea.Cmd
		m, cmd = m.Update(tab)
		_ = cmd
		if m.focusIdx != want {
			t.Fatalf("after tab focusIdx=%d want %d", m.focusIdx, want)
		}
		if f := m.huhField(want); f == nil {
			t.Fatalf("missing huh field %d", want)
		}
	}
	m, _ = m.Update(tab)
	if m.focusIdx != m.focusIdxAddField() {
		t.Fatalf("after body tab focusIdx=%d want add-field %d", m.focusIdx, m.focusIdxAddField())
	}
}

func TestClickHuhFieldFocuses(t *testing.T) {
	zm := zone.New()
	m := NewModel(zm)
	m.SetDimensions(120, 40)
	m.Reset()
	m.SetKinds([]string{"repo"})
	m.Show()
	m.SetDimensions(120, 40)
	_ = zm.Scan(m.View())

	var z *zone.ZoneInfo
	for i := 0; i < 50; i++ {
		z = zm.Get(zones.FormGroup)
		if z != nil && !z.IsZero() {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if z == nil || z.IsZero() {
		t.Fatal("FormGroup zone not registered")
	}
	m, _ = m.Update(tea.MouseClickMsg{
		Button: tea.MouseLeft,
		X:      (z.StartX + z.EndX) / 2,
		Y:      (z.StartY + z.EndY) / 2,
	})
	if m.focusIdx != focusGroup {
		t.Fatalf("focusIdx=%d want group", m.focusIdx)
	}
}

func TestNewEntryFieldsVisibleAfterSetDimensions(t *testing.T) {
	// Matches the root model: OpenNewForm → Show → WindowSize handled only via SetDimensions.
	m := NewModel(zone.New())
	m.SetDimensions(120, 40)
	m.Reset()
	m.SetMode(ModeCreate)
	m.SetKinds([]string{"repo"})
	m.Show()
	m.SetDimensions(120, 40)
	v := m.View()
	for _, label := range []string{"Name", "Group", "Tags", "Body"} {
		i := strings.Index(v, label)
		if i < 0 {
			t.Fatalf("missing %s in form view", label)
		}
		seg := v[max(0, i-32):i]
		if strings.Contains(seg, "38;5;240") {
			t.Fatalf("%s uses dim color 240, which is invisible on the dark background: %q", label, seg)
		}
		if !strings.Contains(seg, "38;5;252") && !strings.Contains(seg, "38;5;205") {
			t.Fatalf("%s is not theme text or accent: %q", label, seg)
		}
	}
	if strings.Contains(v, "38;5;240") {
		t.Fatal("form still contains dim color 240")
	}
}

func TestSetKindsIncludesSentinel(t *testing.T) {
	m := NewModel(zone.New())
	m.SetKinds([]string{"rule", "note"})
	if len(m.kinds) < 2 {
		t.Fatalf("kinds = %#v", m.kinds)
	}
}
