package tools

import (
	"testing"

	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

func TestSanitizeName(t *testing.T) {
	if SanitizeName("Run Tests") != "run_tests" {
		t.Fatalf("got %q", SanitizeName("Run Tests"))
	}
	if SanitizeName("!!!") != "" {
		t.Fatalf("got %q", SanitizeName("!!!"))
	}
}

func TestIsExecutable(t *testing.T) {
	if !IsExecutable(entry.Entry{Fields: map[string]string{"exec": "true"}}) {
		t.Fatal("expected executable")
	}
	if IsExecutable(entry.Entry{Fields: map[string]string{"exec": "  "}}) {
		t.Fatal("expected non-executable")
	}
}

func TestParseDef_Validation(t *testing.T) {
	if _, err := ParseDef(entry.Entry{Name: "t", Fields: map[string]string{}}); err == nil {
		t.Fatal("expected empty exec error")
	}
	def, err := ParseDef(entry.Entry{
		Name:   "My Tool",
		Body:   "desc",
		Fields: map[string]string{"exec": "echo hi", "timeout": "5s", "params": `[{"name":"x","description":"d"}]`},
	})
	if err != nil {
		t.Fatal(err)
	}
	if def.Name != "my_tool" || def.Timeout.String() != "5s" || len(def.Params) != 1 {
		t.Fatalf("def = %#v", def)
	}
}

func TestListDefs_SkipsInvalid(t *testing.T) {
	st := openStore(t)
	if _, err := st.Create(entry.Entry{Kind: "tool", Name: "good", Fields: map[string]string{"exec": "true"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Create(entry.Entry{Kind: "tool", Name: "bad", Fields: map[string]string{"exec": " ", "params": "not-json"}}); err != nil {
		t.Fatal(err)
	}
	defs, err := ListDefs(st)
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) != 1 || defs[0].Name != "good" {
		t.Fatalf("defs = %#v", defs)
	}
}

func openStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
