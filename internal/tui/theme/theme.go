// Package theme holds shared visual constants for the TUI so embedded
// components and the various tab packages share one source of truth.
package theme

// Accent is the lipgloss color used as the application's primary accent (the
// pink already used for active tabs, titles, and selections). Centralizing it
// here lets embedded components such as bubble-dropdown match the rest of the
// TUI, and gives a future theme system a single place to hook into.
const Accent = "205"
