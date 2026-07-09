package form

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	dropdownv2 "github.com/madicen/bubble-dropdown/v2"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

func TestSetKindExisting(t *testing.T) {
	m := NewModel(nil)
	m.SetKinds([]string{"fact", "rule"})
	m.setKind("rule")
	if m.newKindMode {
		t.Fatalf("selecting an existing kind should not enable new-kind mode")
	}
	if got := m.selectedKind(); got != "rule" {
		t.Fatalf("selectedKind() = %q, want \"rule\"", got)
	}
}

func TestSetKindNew(t *testing.T) {
	m := NewModel(nil)
	m.SetKinds([]string{"fact", "rule"})
	m.setKind("brandnew")
	if !m.newKindMode {
		t.Fatalf("an unknown kind should enable new-kind mode")
	}
	if got := m.selectedKind(); got != "brandnew" {
		t.Fatalf("selectedKind() = %q, want \"brandnew\"", got)
	}
}

func TestSetKindEmptyDefaultsToFirst(t *testing.T) {
	m := NewModel(nil)
	m.SetKinds([]string{"fact", "rule"})
	m.setKind("")
	if m.newKindMode {
		t.Fatalf("empty kind with existing kinds should not enable new-kind mode")
	}
	if got := m.selectedKind(); got != "fact" {
		t.Fatalf("selectedKind() = %q, want \"fact\" (first existing)", got)
	}
}

func TestSetKindEmptyNoKindsEntersNewMode(t *testing.T) {
	m := NewModel(nil)
	m.SetKinds(nil)
	m.setKind("")
	if !m.newKindMode {
		t.Fatalf("empty kind with no existing kinds should enable new-kind mode")
	}
}

func TestPopulateFromSelectsExistingKind(t *testing.T) {
	m := NewModel(nil)
	m.SetKinds([]string{"fact", "rule"})
	m.PopulateFrom(entry.Entry{Kind: "rule", Name: "n"})
	if m.newKindMode {
		t.Fatalf("populating with an existing kind should not enable new-kind mode")
	}
	if got := m.ToEntry().Kind; got != "rule" {
		t.Fatalf("ToEntry().Kind = %q, want \"rule\"", got)
	}
}

func TestPopulateFromNewKind(t *testing.T) {
	m := NewModel(nil)
	m.SetKinds([]string{"fact", "rule"})
	m.PopulateFrom(entry.Entry{Kind: "snippet", Name: "n"})
	if !m.newKindMode {
		t.Fatalf("populating with an unknown kind should enable new-kind mode")
	}
	if got := m.ToEntry().Kind; got != "snippet" {
		t.Fatalf("ToEntry().Kind = %q, want \"snippet\"", got)
	}
}

func TestChooseSentinelEntersNewKindMode(t *testing.T) {
	m := NewModel(nil)
	m.SetKinds([]string{"fact", "rule"})
	// The sentinel is the last option index (== len(ddKinds)).
	m, _ = m.handleKindChosen(dropdownv2.ItemChosenMsg{Index: len(m.ddKinds)})
	if !m.newKindMode {
		t.Fatalf("choosing the sentinel should enable new-kind mode")
	}
	m.kind.SetValue("custom")
	if got := m.ToEntry().Kind; got != "custom" {
		t.Fatalf("ToEntry().Kind = %q, want \"custom\"", got)
	}
}

// TestChooseExistingViaUpdate exercises the full open -> navigate -> confirm
// flow so the dropdown is genuinely open when ItemChosenMsg is applied.
func TestChooseExistingViaUpdate(t *testing.T) {
	m := NewModel(nil)
	m.SetKinds([]string{"fact", "rule"})
	m.Show() // visible, focus on the Kind dropdown

	// Open the panel.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !m.kindDD.Open() {
		t.Fatalf("Enter on the focused trigger should open the panel")
	}
	// Move cursor to the second option ("Rule") and confirm.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("confirming an option should emit a command")
	}
	m, _ = m.Update(cmd())
	if m.kindDD.Open() {
		t.Fatalf("the panel should close after a choice")
	}
	if got := m.ToEntry().Kind; got != "rule" {
		t.Fatalf("ToEntry().Kind = %q, want \"rule\"", got)
	}
}
