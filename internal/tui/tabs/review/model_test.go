package review

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

func TestUpdate_ApproveRejectRequests(t *testing.T) {
	m := NewModel(zone.New())
	m.proposals = []entry.Entry{{ID: "p1", Kind: "note", Name: "demo", Body: "body"}}
	m.SetDimensions(100, 30)

	_, req, _ := m.Update(pressKey("a"))
	if req == nil || !req.ApproveSelected {
		t.Fatalf("approve req = %#v", req)
	}
	_, req, _ = m.Update(pressKey("r"))
	if req == nil || !req.RejectSelected {
		t.Fatalf("reject req = %#v", req)
	}
	_, req, _ = m.Update(tea.KeyPressMsg(tea.Key{Text: "A", Code: 'a', ShiftedCode: 'A', Mod: tea.ModShift}))
	if req == nil || !req.ApproveAll {
		t.Fatalf("approve-all req = %#v", req)
	}
}

func TestUpdate_ProposalApprovedRemovesRow(t *testing.T) {
	m := NewModel(zone.New())
	m.proposals = []entry.Entry{{ID: "p1", Name: "a"}, {ID: "p2", Name: "b"}}
	m.SetDimensions(100, 30)
	m, _, _ = m.Update(ProposalApprovedMsg{Entry: entry.Entry{ID: "p1"}})
	if len(m.proposals) != 1 || m.proposals[0].ID != "p2" {
		t.Fatalf("proposals = %#v", m.proposals)
	}
}

func TestFormatProposalDetail_UpdateDiff(t *testing.T) {
	m := NewModel(zone.New())
	m.targets = map[string]entry.Entry{
		"t1": {ID: "t1", Kind: "note", Name: "demo", Body: "old body", Fields: map[string]string{"exec": "old"}},
	}
	p := entry.Entry{
		ID:       "p1",
		Kind:     "note",
		Name:     "demo",
		TargetID: "t1",
		Body:     "new body",
		Fields:   map[string]string{"exec": "new"},
	}
	text := m.formatProposalDetail(p)
	for _, want := range []string{"demo", "new body", "field exec"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
}

func TestToggleMarkdownRenders(t *testing.T) {
	m := NewModel(zone.New())
	m.proposals = []entry.Entry{{ID: "p1", Kind: "note", Name: "md", Body: "# Title"}}
	m.SetDimensions(100, 30)
	m, _, _ = m.Update(pressKey("m"))
	if !m.renderMarkdown {
		t.Fatal("expected markdown mode")
	}
	view := m.View()
	if view == "" {
		t.Fatal("expected view")
	}
}

func pressKey(s string) tea.KeyPressMsg {
	if len(s) == 1 {
		r := rune(s[0])
		return tea.KeyPressMsg(tea.Key{Text: s, Code: r, ShiftedCode: r})
	}
	return tea.KeyPressMsg{Text: s}
}
