package listpane_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/madicen/naitv-mcp/internal/tui/components/listpane"
)

func TestComputeAndRender(t *testing.T) {
	l := listpane.Compute(100, 20, 2, 0)
	if l.ListW != 35 || l.DetailW != 64 || l.ContentH != 18 {
		t.Fatalf("layout = %#v", l)
	}
	rows := []string{"one", "two"}
	out := listpane.RenderList(l.ListW, l.ContentH, rows)
	if !strings.Contains(out, "one") {
		t.Fatalf("render missing row: %q", out)
	}
}

func TestPadRows(t *testing.T) {
	got := listpane.PadRows([]string{"a"}, 3)
	if len(got) != 3 || got[2] != "" {
		t.Fatalf("PadRows = %#v", got)
	}
	trunc := listpane.PadRows([]string{"a", "b", "c", "d"}, 2)
	if len(trunc) != 2 {
		t.Fatalf("truncate = %#v", trunc)
	}
}

func TestDetailViewport(t *testing.T) {
	d := listpane.NewDetail()
	l := listpane.Compute(80, 24, 2, 0)
	d.Resize(l)
	d.SetContent("detail body")
	if !strings.Contains(d.View(), "detail body") {
		t.Fatalf("view = %q", d.View())
	}
	pane := d.RenderPane(l.DetailW, l.ContentH, "extra")
	if !strings.Contains(pane, "extra") {
		t.Fatalf("pane = %q", pane)
	}
}

func TestSelectionClamp(t *testing.T) {
	s := listpane.Selection{Index: 5}
	s.Clamp(3)
	if s.Index != 2 {
		t.Fatalf("index = %d", s.Index)
	}
}

func TestDetailUpdateScroll(t *testing.T) {
	d := listpane.NewDetail()
	l := listpane.Compute(80, 24, 2, 0)
	d.Resize(l)
	d.SetContent(strings.Repeat("line\n", 40))
	_, cmd := d.Update(tea.MouseWheelMsg{Y: -1})
	if cmd != nil {
		t.Fatal("unexpected cmd")
	}
}
