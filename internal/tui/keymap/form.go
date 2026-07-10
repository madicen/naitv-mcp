package keymap

import "charm.land/bubbles/v2/key"

// Form holds key bindings for the entry form modal.
type Form struct {
	Save, Cancel, Next, Prev, Submit, EditBody key.Binding
}

// DefaultForm is the default form keymap.
var DefaultForm = Form{
	Save: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "save"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	Next: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next field"),
	),
	Prev: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev field"),
	),
	Submit: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "submit"),
	),
	EditBody: key.NewBinding(
		key.WithKeys("ctrl+e"),
		key.WithHelp("ctrl+e", "edit body"),
	),
}

// FormActions returns bindings shown in the form footer.
func (k Form) FormActions() []key.Binding {
	return []key.Binding{k.Save, k.EditBody, k.Cancel}
}
