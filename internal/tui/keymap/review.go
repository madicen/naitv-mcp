package keymap

import "charm.land/bubbles/v2/key"

// Review holds key bindings for the review tab.
type Review struct {
	Down, Up, Approve, Reject, Edit, ApproveAll, Back, ToggleMarkdown key.Binding
}

// DefaultReview is the default review tab keymap.
var DefaultReview = Review{
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("↓/j", "down"),
	),
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("↑/k", "up"),
	),
	Approve: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "approve"),
	),
	Reject: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "reject"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	ApproveAll: key.NewBinding(
		key.WithKeys("A"),
		key.WithHelp("A", "approve all"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "entries"),
	),
	ToggleMarkdown: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "markdown"),
	),
}

// ReviewActions returns bindings shown in the review action bar.
func (k Review) ReviewActions() []key.Binding {
	return []key.Binding{k.Approve, k.Reject, k.Edit, k.ApproveAll, k.Back}
}
