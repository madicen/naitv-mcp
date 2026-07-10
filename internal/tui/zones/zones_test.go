package zones_test

import (
	"strings"
	"testing"

	"github.com/madicen/naitv-mcp/internal/tui/zones"
	zone "github.com/lrstanley/bubblezone/v2"
)

func TestZoneIDs(t *testing.T) {
	if zones.EntriesRow(3) != "flat:3" {
		t.Fatal("EntriesRow mismatch")
	}
	if zones.ReviewRow(1) != "proposal:1" {
		t.Fatal("ReviewRow mismatch")
	}
	if zones.FormRemoveField(2) != "form:remove-field:2" {
		t.Fatal("FormRemoveField mismatch")
	}
}

func TestButtonMarksZone(t *testing.T) {
	zm := zone.New()
	out := zones.Button(zm, zones.EntriesNew, "n", "new")
	if !strings.Contains(out, "new") {
		t.Fatalf("button missing label: %q", out)
	}
}
