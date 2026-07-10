package mcp_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/madicen/naitv-mcp/internal/mcp"
	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/pkg/entry"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func connectTestClient(t *testing.T, st *store.Store) (*sdkmcp.ClientSession, context.Context) {
	t.Helper()
	ctx := context.Background()
	server := mcp.NewServer(st)

	ct, stTransport := sdkmcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, stTransport, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test-client", Version: "test"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	return session, ctx
}

func TestServer_InitializeHandshake(t *testing.T) {
	st := openTestStore(t)
	if _, err := st.Create(entry.Entry{Kind: "rule", Name: "use-jj", Body: "Use jj for version control."}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	session, ctx := connectTestClient(t, st)

	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "initialize",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool initialize: %v", err)
	}
	if res.IsError {
		t.Fatalf("initialize returned error")
	}
	text, ok := res.Content[0].(*sdkmcp.TextContent)
	if !ok || text.Text == "" {
		t.Fatalf("expected text content from initialize")
	}
	if !contains(text.Text, "use-jj") {
		t.Errorf("initialize output missing entry name: %q", text.Text)
	}
}

func TestServer_ReadResources(t *testing.T) {
	st := openTestStore(t)
	created, err := st.Create(entry.Entry{Kind: "rule", Name: "use-jj", Body: "Use jj for version control."})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	session, ctx := connectTestClient(t, st)

	bundle, err := session.ReadResource(ctx, &sdkmcp.ReadResourceParams{URI: "naitv://bundle"})
	if err != nil {
		t.Fatalf("ReadResource bundle: %v", err)
	}
	if len(bundle.Contents) == 0 {
		t.Fatal("expected bundle contents")
	}
	bundleText := bundle.Contents[0].Text
	if !contains(bundleText, "use-jj") {
		t.Errorf("bundle missing entry: %q", bundleText)
	}

	entryURI := "naitv://entry/" + created.ID
	got, err := session.ReadResource(ctx, &sdkmcp.ReadResourceParams{URI: entryURI})
	if err != nil {
		t.Fatalf("ReadResource entry: %v", err)
	}
	if len(got.Contents) == 0 {
		t.Fatal("expected entry contents")
	}
	if !contains(got.Contents[0].Text, "use-jj") {
		t.Errorf("entry resource missing name: %q", got.Contents[0].Text)
	}

	_, err = session.ReadResource(ctx, &sdkmcp.ReadResourceParams{URI: "naitv://entry/nonexistent"})
	if err == nil {
		t.Fatal("expected error for missing entry resource")
	}
}

func TestServer_ListEntriesErrorPath(t *testing.T) {
	st := openTestStore(t)
	session, ctx := connectTestClient(t, st)

	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "get_entry",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result for missing id_or_name")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
