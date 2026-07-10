package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

func TestRunInit_WritesFilteredBundle(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	for _, e := range []entry.Entry{
		{Kind: "rule", Name: "rule-one", Body: "rule", Delivery: entry.DeliveryInit},
		{Kind: "note", Name: "note-one", Body: "note", Delivery: entry.DeliveryInit},
	} {
		if _, err := st.Create(e); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}
	st.Close()

	outPath := filepath.Join(t.TempDir(), "AGENTS.md")
	if err := runInit(dbPath, outPath, "rule"); err != nil {
		t.Fatalf("runInit: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "rule-one") || strings.Contains(text, "note-one") {
		t.Fatalf("unexpected init output: %s", text)
	}
}

func TestRunInit_Stdout(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := st.Create(entry.Entry{Kind: "rule", Name: "demo", Body: "ok", Delivery: entry.DeliveryInit}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	st.Close()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err = runInit(dbPath, "-", "")
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	if err != nil {
		t.Fatalf("runInit: %v", err)
	}
	if !strings.Contains(buf.String(), "demo") {
		t.Fatalf("stdout missing demo: %q", buf.String())
	}
}

func TestSeedDemoDB_PopulatesOnce(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()
	if err := seedDemoDB(st); err != nil {
		t.Fatalf("seedDemoDB: %v", err)
	}
	first, err := st.List("", nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(first) == 0 {
		t.Fatal("expected seeded entries")
	}
	if err := seedDemoDB(st); err != nil {
		t.Fatalf("seedDemoDB second: %v", err)
	}
	second, err := st.List("", nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(second) != len(first) {
		t.Fatalf("seed ran twice: %d -> %d", len(first), len(second))
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := st.Create(entry.Entry{Kind: "note", Name: "export-me", Body: "body"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	exportPath := filepath.Join(t.TempDir(), "export.json")
	f, err := os.Create(exportPath)
	if err != nil {
		t.Fatalf("Create export: %v", err)
	}
	if err := st.ExportJSON(f); err != nil {
		t.Fatalf("ExportJSON: %v", err)
	}
	f.Close()
	st.Close()

	dbPath2 := filepath.Join(t.TempDir(), "import.db")
	st2, err := store.Open(dbPath2)
	if err != nil {
		t.Fatalf("Open2: %v", err)
	}
	defer st2.Close()
	in, err := os.Open(exportPath)
	if err != nil {
		t.Fatalf("Open export: %v", err)
	}
	defer in.Close()
	n, err := st2.ImportJSON(in, store.ImportMerge)
	if err != nil || n != 1 {
		t.Fatalf("ImportJSON n=%d err=%v", n, err)
	}
	got, err := st2.GetByName("export-me")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if got.Body != "body" {
		t.Fatalf("body = %q", got.Body)
	}
}
