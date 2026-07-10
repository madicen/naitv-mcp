// Package zones provides typed zone ID constructors and button rendering helpers.
package zones

import (
	"fmt"

	"github.com/madicen/naitv-mcp/internal/tui/theme"
	zone "github.com/lrstanley/bubblezone/v2"
)

// Tab zone IDs.
const (
	TabEntries = "tab:entries"
	TabReview  = "tab:review"
	TabPlugins = "tab:plugins"
)

// Entries action zones.
const (
	EntriesNew      = "action:new"
	EntriesEdit     = "action:edit"
	EntriesDelete   = "action:delete"
	EntriesDelivery = "action:delivery"
	EntriesCopy     = "action:copy"
	EntriesSearch   = "action:search"
	EntriesReview   = "action:review"
	EntriesKindDD   = "entries:kind-dd"
)

func EntriesRow(i int) string { return fmt.Sprintf("flat:%d", i) }

// Review zones.
const (
	ReviewApprove     = "action:approve"
	ReviewReject      = "action:reject"
	ReviewEdit        = "action:edit-review"
	ReviewApproveAll  = "action:approve-all"
	ReviewDetailApprove = "detail:approve"
	ReviewDetailReject  = "detail:reject"
	ReviewDetailEdit    = "detail:edit"
)

func ReviewRow(i int) string { return fmt.Sprintf("proposal:%d", i) }

func PluginRow(i int) string { return fmt.Sprintf("plugin:row:%d", i) }

const (
	PluginModeInstalled = "plugin:mode:installed"
	PluginModeBrowse    = "plugin:mode:browse"
	PluginActInstall    = "plugin:action:install"
	PluginActUninstall  = "plugin:action:uninstall"
	PluginActTab        = "plugin:action:tab"
	PluginActRefresh    = "plugin:action:refresh"
)

const (
	FormSave   = "form:save"
	FormCancel = "form:cancel"
	FormAddFld = "form:add-field"
	FormKindDD = "form:kind-dd"
)

func FormRemoveField(i int) string { return fmt.Sprintf("form:remove-field:%d", i) }

// Button renders a zone-marked action hint: [key] label.
func Button(zm *zone.Manager, zoneID, keyLabel, actionLabel string) string {
	label := theme.KeyHint.Render("["+keyLabel+"]") + theme.Hint.Render(" "+actionLabel)
	if zm != nil {
		return zm.Mark(zoneID, label)
	}
	return label
}
