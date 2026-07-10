package plugin

import (
	"testing"

	"github.com/madicen/naitv-mcp/pkg/entry"
)

func TestParseEntryIDs(t *testing.T) {
	ids := parseEntryIDs(entry.Entry{Fields: map[string]string{
		"entry_ids": `["id1","id2"]`,
	}})
	if len(ids) != 2 || ids[0] != "id1" {
		t.Fatalf("parseEntryIDs = %#v", ids)
	}
}

func TestValidateManifest_RequiresVersion(t *testing.T) {
	err := validateManifest(Manifest{Name: "demo"}, "src")
	if err == nil {
		t.Fatal("expected version required error")
	}
}
