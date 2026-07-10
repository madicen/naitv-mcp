package plugin

import (
	"testing"

	"github.com/madicen/naitv-mcp/internal/store"
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

func TestEntryIDs_FromJSON(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	created, err := st.Create(entry.Entry{Kind: "rule", Name: "linked", Body: "x"})
	if err != nil {
		t.Fatal(err)
	}
	fromJSON := EntryIDs(st, entry.Entry{Fields: map[string]string{"entry_ids": `["` + created.ID + `"]`}})
	if len(fromJSON) != 1 || fromJSON[0] != created.ID {
		t.Fatalf("entry_ids = %#v", fromJSON)
	}
}

func TestEntryIDs_MigratesLegacyNames(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	created, err := st.Create(entry.Entry{Kind: "rule", Name: "legacy-link", Body: "x"})
	if err != nil {
		t.Fatal(err)
	}
	track, err := st.Create(entry.Entry{
		Kind: "plugin",
		Name: "legacy-plugin",
		Fields: map[string]string{
			"entry_names": "legacy-link, missing",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ids := EntryIDs(st, track)
	if len(ids) != 1 || ids[0] != created.ID {
		t.Fatalf("migrated ids = %#v", ids)
	}
	got, err := st.Get(track.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Fields["entry_names"] != "" {
		t.Fatalf("expected entry_names removed, got %#v", got.Fields)
	}
	if got.Fields["entry_ids"] == "" {
		t.Fatal("expected entry_ids written during migration")
	}
}
