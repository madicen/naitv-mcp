package form
import ("testing"; tea "charm.land/bubbletea/v2"; zone "github.com/lrstanley/bubblezone/v2")
func runFormCmds(m Model, cmd tea.Cmd) Model { for cmd != nil { msg := cmd(); if msg == nil { break }; var n tea.Cmd; m, n = m.Update(msg); cmd = n }; return m }
func TestSaveViaCtrlS(t *testing.T) {
	m := NewModel(zone.New()); m.SetDimensions(120, 40); m.Show(); m.SetKinds(nil)
	for _, ch := range "tool" { r := rune(ch); m, _ = m.Update(tea.KeyPressMsg(tea.Key{Text: string(ch), Code: r, ShiftedCode: r})) }
	var cmd tea.Cmd; m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Text: "tab"}); m = runFormCmds(m, cmd)
	for _, ch := range "typed-entry" { r := rune(ch); m, _ = m.Update(tea.KeyPressMsg(tea.Key{Text: string(ch), Code: r, ShiftedCode: r})) }
	if m.ToEntry().Name != "typed-entry" { t.Fatalf("name=%q", m.ToEntry().Name) }
	m, cmd = m.Update(tea.KeyPressMsg{Text: "ctrl+s"})
	if cmd == nil { t.Fatal("no save cmd") }
	if save, ok := cmd().(SaveMsg); !ok || save.E.Name != "typed-entry" { t.Fatalf("save=%#v", cmd()) }
}
func TestAddRemoveField(t *testing.T) {
	m := NewModel(zone.New()); m.SetDimensions(120,40); m.Show(); m.huhVals.name="x"; m.rebuildHuhForm()
	m.addField(); m.fields[0].Key.SetValue("env"); m.fields[0].Val.SetValue("prod")
	if m.ToEntry().Fields["env"] != "prod" { t.Fatal() }
	m.removeField(0); if len(m.ToEntry().Fields) != 0 { t.Fatal() }
}
