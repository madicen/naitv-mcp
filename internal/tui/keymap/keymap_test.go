package keymap_test

import (
	"testing"

	"github.com/madicen/naitv-mcp/internal/tui/keymap"
)

func TestDefaultKeymapsExposeHelp(t *testing.T) {
	cases := []struct {
		name string
		full func() string
	}{
		{"entries", func() string { return keymap.DefaultEntries.EntriesActions()[0].Help().Key }},
		{"review", func() string { return keymap.DefaultReview.ReviewActions()[0].Help().Key }},
		{"plugins", func() string { return keymap.DefaultPlugins.PluginActions(true)[0].Help().Key }},
		{"form", func() string { return keymap.DefaultForm.FormActions()[0].Help().Key }},
	}
	for _, tc := range cases {
		if tc.full() == "" {
			t.Fatalf("%s keymap missing help", tc.name)
		}
	}
}
