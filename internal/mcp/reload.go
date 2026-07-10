package mcp

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/internal/tools"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type dynamicToolRegistry struct {
	server       *sdkmcp.Server
	store        *store.Store
	dynamicTools map[string]tools.Def
	lastDBMod    time.Time
	mu           sync.Mutex
}

func newDynamicToolRegistry(server *sdkmcp.Server, st *store.Store) *dynamicToolRegistry {
	return &dynamicToolRegistry{
		server:       server,
		store:        st,
		dynamicTools: make(map[string]tools.Def),
	}
}

func (r *dynamicToolRegistry) wire() {
	r.initDBModTime()
	r.refresh()
	st := r.store
	st.OnChange(func() {
		r.refresh()
		r.notifyResources()
	})
	r.server.AddReceivingMiddleware(r.receivingMiddleware)
}

func (r *dynamicToolRegistry) initDBModTime() {
	mod, err := r.store.ModTime()
	if err == nil {
		r.lastDBMod = mod
	}
}

func (r *dynamicToolRegistry) receivingMiddleware(next sdkmcp.MethodHandler) sdkmcp.MethodHandler {
	return func(ctx context.Context, method string, req sdkmcp.Request) (sdkmcp.Result, error) {
		if method == "tools/list" {
			r.maybeRefreshFromDB()
		}
		return next(ctx, method, req)
	}
}

func (r *dynamicToolRegistry) maybeRefreshFromDB() {
	mod, err := r.store.ModTime()
	if err != nil || mod.Equal(r.lastDBMod) {
		return
	}
	r.lastDBMod = mod
	r.refresh()
	r.notifyResources()
}

func (r *dynamicToolRegistry) refresh() {
	defs, err := tools.ListDefs(r.store)
	if err != nil {
		fmt.Fprintf(os.Stderr, "naitv-mcp: dynamic tools: %v\n", err)
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	want := make(map[string]tools.Def, len(defs))
	for _, d := range defs {
		want[d.Name] = d
	}

	var toRemove []string
	for name, old := range r.dynamicTools {
		newDef, ok := want[name]
		if !ok || !toolDefEqual(old, newDef) {
			toRemove = append(toRemove, name)
		}
	}
	if len(toRemove) > 0 {
		r.server.RemoveTools(toRemove...)
		for _, name := range toRemove {
			delete(r.dynamicTools, name)
		}
	}

	var added int
	for name, def := range want {
		old, ok := r.dynamicTools[name]
		if ok && toolDefEqual(old, def) {
			continue
		}
		registerOne(r.server, def)
		r.dynamicTools[name] = def
		added++
	}

	if added > 0 && len(r.dynamicTools) == added && len(toRemove) == 0 {
		fmt.Fprintf(os.Stderr, "naitv-mcp: registered %d dynamic tool(s)\n", added)
	}
}

func (r *dynamicToolRegistry) notifyResources() {
	ctx := context.Background()
	_ = r.server.ResourceUpdated(ctx, &sdkmcp.ResourceUpdatedNotificationParams{URI: bundleResourceURI})
}

func toolDefEqual(a, b tools.Def) bool {
	if a.Name != b.Name || a.Exec != b.Exec || a.WorkingDir != b.WorkingDir ||
		a.Description != b.Description || a.Disabled != b.Disabled || a.Timeout != b.Timeout {
		return false
	}
	if len(a.Params) != len(b.Params) {
		return false
	}
	for i := range a.Params {
		if a.Params[i] != b.Params[i] {
			return false
		}
	}
	if len(a.EnvAllowlist) != len(b.EnvAllowlist) {
		return false
	}
	for i := range a.EnvAllowlist {
		if a.EnvAllowlist[i] != b.EnvAllowlist[i] {
			return false
		}
	}
	return true
}
