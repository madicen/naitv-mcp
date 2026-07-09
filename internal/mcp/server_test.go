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

func TestServer_InitializeHandshake(t *testing.T) {
	st := openTestStore(t)
	if _, err := st.Create(entry.Entry{Kind: "rule", Name: "use-jj", Body: "Use jj for version control."}); err != nil {
		t.Fatalf("Create: %v", err)
	}

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
	defer session.Close()

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

func TestServer_ListEntriesErrorPath(t *testing.T) {
	st := openTestStore(t)
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
	defer session.Close()

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
