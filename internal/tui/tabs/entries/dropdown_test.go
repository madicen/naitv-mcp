package entries

import "testing"

func TestKindAtIndex(t *testing.T) {
	m := &Model{kinds: []string{"", "fact", "rule"}}

	cases := []struct {
		idx  int
		want string
	}{
		{0, ""},     // "All"
		{1, "fact"}, // empty kind is filtered out, so index 1 -> fact
		{2, "rule"},
		{3, ""},  // out of range
		{-1, ""}, // negative
	}
	for _, c := range cases {
		if got := m.kindAtIndex(c.idx); got != c.want {
			t.Errorf("kindAtIndex(%d) = %q, want %q", c.idx, got, c.want)
		}
	}
}

func TestRefreshKindDropdownPreservesSelection(t *testing.T) {
	m := &Model{kinds: []string{"fact", "rule"}, selectedKind: "rule"}
	m.refreshKindDropdown()
	// Options are ["All", "Fact", "Rule"], so "rule" is index 2.
	if got := m.kindDD.SelectedIndex(); got != 2 {
		t.Fatalf("after refresh, SelectedIndex() = %d, want 2", got)
	}

	// Growing the kind set keeps the same selection (now at a new index).
	m.kinds = []string{"fact", "note", "rule"}
	m.refreshKindDropdown()
	// Options become ["All", "Fact", "Note", "Rule"]; "rule" is index 3.
	if got := m.kindDD.SelectedIndex(); got != 3 {
		t.Fatalf("after grow+refresh, SelectedIndex() = %d, want 3", got)
	}
	if got := m.kindAtIndex(m.kindDD.SelectedIndex()); got != "rule" {
		t.Fatalf("kindAtIndex(selected) = %q, want \"rule\"", got)
	}
}

func TestRefreshKindDropdownAllSelection(t *testing.T) {
	m := &Model{kinds: []string{"fact", "rule"}, selectedKind: ""}
	m.refreshKindDropdown()
	if got := m.kindDD.SelectedIndex(); got != 0 {
		t.Fatalf("empty selectedKind should map to index 0 (All), got %d", got)
	}
	if got := m.kindAtIndex(0); got != "" {
		t.Fatalf("kindAtIndex(0) = %q, want \"\"", got)
	}
}
