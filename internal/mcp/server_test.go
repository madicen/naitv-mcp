package mcp_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	return connectTestClientWithOpts(t, st, nil)
}

func connectTestClientWithOpts(t *testing.T, st *store.Store, opts *sdkmcp.ClientOptions) (*sdkmcp.ClientSession, context.Context) {
	t.Helper()
	ctx := context.Background()
	server := mcp.NewServer(st)

	ct, stTransport := sdkmcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, stTransport, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test-client", Version: "test"}, opts)
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

func TestServer_GetPrompts(t *testing.T) {
	st := openTestStore(t)
	session, ctx := connectTestClient(t, st)

	var names []string
	for p, err := range session.Prompts(ctx, nil) {
		if err != nil {
			t.Fatalf("Prompts: %v", err)
		}
		names = append(names, p.Name)
	}
	if !contains(strings.Join(names, ","), "load-context") || !contains(strings.Join(names, ","), "propose-learning") {
		t.Fatalf("expected load-context and propose-learning prompts, got %v", names)
	}

	res, err := session.GetPrompt(ctx, &sdkmcp.GetPromptParams{
		Name:      "load-context",
		Arguments: map[string]string{"task": "fix flaky tests"},
	})
	if err != nil {
		t.Fatalf("GetPrompt: %v", err)
	}
	if len(res.Messages) == 0 {
		t.Fatal("expected prompt messages")
	}
	text, ok := res.Messages[0].Content.(*sdkmcp.TextContent)
	if !ok || !contains(text.Text, "fix flaky tests") {
		t.Fatalf("prompt missing task: %#v", res.Messages[0].Content)
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

func TestServer_StaticToolsHappyPaths(t *testing.T) {
	st := openTestStore(t)
	created, err := st.Create(entry.Entry{
		Kind: "note",
		Name: "alpha",
		Body: "body text",
		Tags: []string{"go"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	session, ctx := connectTestClient(t, st)

	cases := []struct {
		name string
		args map[string]any
		want string
	}{
		{"list_entries", map[string]any{}, "alpha"},
		{"get_entry", map[string]any{"id_or_name": created.ID}, "alpha"},
		{"search_entries", map[string]any{"query": "alpha"}, "alpha"},
		{"list_tools", map[string]any{}, "No executable tools"},
		{"export_entries", map[string]any{}, "schema_version"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{Name: tc.name, Arguments: tc.args})
			if err != nil {
				t.Fatalf("CallTool: %v", err)
			}
			if res.IsError {
				t.Fatalf("tool error: %v", res.Content)
			}
			text := res.Content[0].(*sdkmcp.TextContent).Text
			if !contains(text, tc.want) {
				t.Fatalf("output missing %q: %s", tc.want, text)
			}
		})
	}
}

func TestServer_AddEntryValidation(t *testing.T) {
	st := openTestStore(t)
	session, ctx := connectTestClient(t, st)

	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "add_entry",
		Arguments: map[string]any{"kind": "note"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected validation error for missing name")
	}
}

func TestServer_AddAndUpdateEntryHappyPath(t *testing.T) {
	st := openTestStore(t)
	session, ctx := connectTestClient(t, st)

	addRes, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "add_entry",
		Arguments: map[string]any{
			"kind": "note",
			"name": "new-note",
			"body": "hello",
			"tags": "go,test",
		},
	})
	if err != nil || addRes.IsError {
		t.Fatalf("add_entry: %v %#v", err, addRes)
	}
	active, err := st.Create(entry.Entry{Kind: "note", Name: "existing", Body: "old"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	updRes, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "update_entry",
		Arguments: map[string]any{
			"id":   active.ID,
			"body": "new body",
		},
	})
	if err != nil || updRes.IsError {
		t.Fatalf("update_entry: %v %#v", err, updRes)
	}
}

func TestServer_InitializeKindsFilter(t *testing.T) {
	st := openTestStore(t)
	if _, err := st.Create(entry.Entry{Kind: "rule", Name: "r1", Body: "rule"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Create(entry.Entry{Kind: "note", Name: "n1", Body: "note"}); err != nil {
		t.Fatal(err)
	}
	session, ctx := connectTestClient(t, st)
	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "initialize",
		Arguments: map[string]any{"kinds": "rule"},
	})
	if err != nil || res.IsError {
		t.Fatalf("initialize: %v", err)
	}
	text := res.Content[0].(*sdkmcp.TextContent).Text
	if !contains(text, "r1") || contains(text, "n1") {
		t.Fatalf("kinds filter failed: %s", text)
	}
}

func TestServer_ListPluginsEmpty(t *testing.T) {
	st := openTestStore(t)
	session, ctx := connectTestClient(t, st)
	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{Name: "list_plugins"})
	if err != nil || res.IsError {
		t.Fatalf("list_plugins: %v", err)
	}
	text := res.Content[0].(*sdkmcp.TextContent).Text
	if !contains(text, "No plugins installed") {
		t.Fatalf("unexpected: %s", text)
	}
}

func TestServer_GenerateContinueConfig(t *testing.T) {
	st := openTestStore(t)
	session, ctx := connectTestClient(t, st)
	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{Name: "generate_continue_config"})
	if err != nil || res.IsError {
		t.Fatalf("generate_continue_config: %v", err)
	}
	text := res.Content[0].(*sdkmcp.TextContent).Text
	if !contains(text, "mcpServers") {
		t.Fatalf("missing config: %s", text)
	}
}

func TestServer_SearchEntriesErrorPath(t *testing.T) {
	st := openTestStore(t)
	session, ctx := connectTestClient(t, st)
	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{Name: "search_entries"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error for missing query")
	}
}

func TestServer_SetProjectAndExport(t *testing.T) {
	st := openTestStore(t)
	if _, err := st.Create(entry.Entry{
		Kind: "tool", Name: "build", Body: "build project",
		Fields: map[string]string{"exec": "true", "working_dir": "/old"},
	}); err != nil {
		t.Fatal(err)
	}
	session, ctx := connectTestClient(t, st)

	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "set_project",
		Arguments: map[string]any{"project_dir": t.TempDir()},
	})
	if err != nil || res.IsError {
		t.Fatalf("set_project: %v %#v", err, res)
	}
	got, err := st.GetByName("build")
	if err != nil || got.Fields["working_dir"] == "/old" {
		t.Fatalf("working_dir not updated: %#v, %v", got, err)
	}
}

func TestServer_ListAvailablePluginsLocalRegistry(t *testing.T) {
	st := openTestStore(t)
	session, ctx := connectTestClient(t, st)
	regPath := filepath.Join(t.TempDir(), "registry.json")
	if err := os.WriteFile(regPath, []byte(`{"plugins":[{"name":"demo","version":"1.0.0","url":"file:///x","description":"d"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "list_available_plugins",
		Arguments: map[string]any{"registry_url": regPath},
	})
	if err != nil || res.IsError {
		t.Fatalf("list_available_plugins: %v", err)
	}
	text := res.Content[0].(*sdkmcp.TextContent).Text
	if !contains(text, "demo") {
		t.Fatalf("missing demo: %s", text)
	}
}

func TestServer_ToolListChangedAfterApproval(t *testing.T) {
	st := openTestStore(t)
	pending, err := st.CreatePending(entry.Entry{
		Kind: "tool",
		Name: "run-tests",
		Body: "Run tests",
		Fields: map[string]string{
			"exec": "echo ok",
		},
	})
	if err != nil {
		t.Fatalf("CreatePending: %v", err)
	}

	changed := make(chan struct{}, 1)
	session, ctx := connectTestClientWithOpts(t, st, &sdkmcp.ClientOptions{
		ToolListChangedHandler: func(context.Context, *sdkmcp.ToolListChangedRequest) {
			select {
			case changed <- struct{}{}:
			default:
			}
		},
	})

	before, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	beforeNames := toolNames(before.Tools)

	if _, err := st.Approve(pending.ID); err != nil {
		t.Fatalf("Approve: %v", err)
	}

	select {
	case <-changed:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for tools/list_changed notification")
	}

	after, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools after: %v", err)
	}
	afterNames := toolNames(after.Tools)
	if containsAll(beforeNames, "run_tests") {
		t.Fatalf("run_tests already present before approval: %v", beforeNames)
	}
	if !containsAll(afterNames, "run_tests") {
		t.Fatalf("run_tests missing after approval: %v", afterNames)
	}
}

func toolNames(tools []*sdkmcp.Tool) []string {
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}
	return names
}

func containsAll(names []string, want string) bool {
	for _, n := range names {
		if n == want {
			return true
		}
	}
	return false
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
