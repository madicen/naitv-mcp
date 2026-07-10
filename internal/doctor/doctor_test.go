package doctor_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/madicen/naitv-mcp/internal/doctor"
	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

func TestRun_AllChecksPass(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "context.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := st.Create(entry.Entry{Kind: "rule", Name: "demo", Body: "ok"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	st.Close()

	out, err := runDoctor(dbPath, false)
	if err != nil {
		t.Fatalf("Run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "All checks passed") {
		t.Fatalf("output missing success: %q", out)
	}
}

func TestRun_MissingDatabaseStat(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nested", "missing.db")
	out, err := runDoctor(dbPath, false)
	if err == nil {
		t.Fatalf("expected error when database file is missing at stat time, out=%q", out)
	}
	if !strings.Contains(out, "cannot stat") {
		t.Fatalf("expected stat error in output: %q", out)
	}
}

func runDoctor(dbPath string, rebuildFTS bool) (string, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	old := os.Stdout
	os.Stdout = w
	runErr := doctor.Run(dbPath, rebuildFTS)
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String(), runErr
}
