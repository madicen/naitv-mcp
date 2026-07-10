package kinddropdown

import "testing"

func TestKindAtFilterIndex(t *testing.T) {
	kinds := []string{"", "fact", "rule"}

	cases := []struct {
		idx  int
		want string
	}{
		{0, ""},
		{1, "fact"},
		{2, "rule"},
		{3, ""},
		{-1, ""},
	}
	for _, c := range cases {
		if got := KindAtFilterIndex(c.idx, kinds); got != c.want {
			t.Errorf("KindAtFilterIndex(%d) = %q, want %q", c.idx, got, c.want)
		}
	}
}

func TestBuildFilterPreservesSelection(t *testing.T) {
	d := BuildFilter(nil, []string{"fact", "rule"}, "rule")
	if got := d.SelectedIndex(); got != 2 {
		t.Fatalf("SelectedIndex() = %d, want 2", got)
	}

	d = BuildFilter(nil, []string{"fact", "note", "rule"}, "rule")
	if got := d.SelectedIndex(); got != 3 {
		t.Fatalf("after grow, SelectedIndex() = %d, want 3", got)
	}
	if got := KindAtFilterIndex(d.SelectedIndex(), []string{"fact", "note", "rule"}); got != "rule" {
		t.Fatalf("KindAtFilterIndex(selected) = %q, want \"rule\"", got)
	}
}

func TestBuildFilterAllSelection(t *testing.T) {
	d := BuildFilter(nil, []string{"fact", "rule"}, "")
	if got := d.SelectedIndex(); got != 0 {
		t.Fatalf("empty selectedKind should map to index 0 (All), got %d", got)
	}
	if got := KindAtFilterIndex(0, []string{"fact", "rule"}); got != "" {
		t.Fatalf("KindAtFilterIndex(0) = %q, want \"\"", got)
	}
}

func TestIsNewKindIndex(t *testing.T) {
	if !IsNewKindIndex(2, 2) {
		t.Fatal("index 2 with 2 kinds should be the sentinel")
	}
	if IsNewKindIndex(1, 2) {
		t.Fatal("index 1 with 2 kinds should not be the sentinel")
	}
}
