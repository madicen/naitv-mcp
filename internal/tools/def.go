// Package tools provides the executable-tool system for naitv-mcp.
//
// # How it works
//
// A context entry with kind="tool" and a non-empty "exec" field in its Fields
// map is treated as an executable tool rather than a plain tooling-preference
// note. When naitv-mcp starts (naitv-mcp serve), it scans all active tool
// entries, parses them into Def values, and registers each as a live MCP tool.
// From that point the model can call them like any other tool.
//
// # Defining a tool entry
//
// Create an entry (via the TUI or add_entry) with:
//
//	kind:  tool
//	name:  run-tests          (becomes the MCP tool name, sanitised)
//	body:  Run the test suite and return output.  (used as the MCP description)
//	fields:
//	  exec:        go test ./...
//	  working_dir: ~/dev/myproject   (optional; ~ is expanded)
//	  timeout:     60s               (optional; default 30s)
//	  params:      [{"name":"pkg","description":"Package path","required":false}]
//	  disabled:    false             (optional; set "true" to temporarily disable)
//
// The "exec" value is a shell command template. Embed {param_name} placeholders
// for each entry in "params"; the model supplies values when it calls the tool.
//
// # Agent proposals
//
// Agents may call add_entry with kind="tool" and an exec field, which queues the
// definition as a pending proposal — just like any other entry. Approve it in the
// Review tab and it goes live on the next server restart. This is the human-in-the-
// loop gate that prevents arbitrary code from being executed without your sign-off.
package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/madicen/naitv-mcp/pkg/entry"
)

// Param describes a single named parameter for an executable tool.
// Parameters map to {name} placeholders in the exec template and are
// declared as MCP tool parameters so the model knows what to supply.
type Param struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// Def is an executable tool definition parsed from a tool entry.
type Def struct {
	// EntryID is the source entry's ID (for traceability).
	EntryID string
	// Name is the sanitised MCP tool name derived from the entry name.
	Name string
	// Description is used as the MCP tool description (entry body).
	Description string
	// Exec is the shell command template. Supports {param_name} placeholders.
	Exec string
	// WorkingDir is the directory to run the command in. ~ is expanded.
	// If empty, the process inherits the server's working directory.
	WorkingDir string
	// Timeout is the maximum time the command may run. Defaults to 30 s.
	Timeout time.Duration
	// Params are the declared parameters for this tool.
	Params []Param
	// Disabled prevents execution while keeping the tool registered.
	// Set the "disabled" field to "true" in the entry to use this.
	Disabled bool
}

// IsExecutable reports whether the entry should be treated as an executable
// tool (i.e. it has a non-empty "exec" field). Entries without this field are
// plain tooling-preference notes and are rendered into the init bundle instead.
func IsExecutable(e entry.Entry) bool {
	v, ok := e.Fields["exec"]
	return ok && strings.TrimSpace(v) != ""
}

// ParseDef builds a Def from an active tool entry. Returns an error if the
// entry does not describe a valid executable tool.
func ParseDef(e entry.Entry) (Def, error) {
	exec := strings.TrimSpace(e.Fields["exec"])
	if exec == "" {
		return Def{}, fmt.Errorf("entry %q: exec field is empty", e.Name)
	}

	name := SanitizeName(e.Name)
	if name == "" {
		return Def{}, fmt.Errorf("entry %q: name produces an empty MCP tool name after sanitisation", e.Name)
	}

	d := Def{
		EntryID:     e.ID,
		Name:        name,
		Description: strings.TrimSpace(e.Body),
		Exec:        exec,
		Timeout:     30 * time.Second,
	}

	if wd := strings.TrimSpace(e.Fields["working_dir"]); wd != "" {
		d.WorkingDir = wd
	}

	if ts := strings.TrimSpace(e.Fields["timeout"]); ts != "" {
		if dur, err := time.ParseDuration(ts); err == nil && dur > 0 {
			d.Timeout = dur
		}
	}

	if ps := strings.TrimSpace(e.Fields["params"]); ps != "" {
		var params []Param
		if err := json.Unmarshal([]byte(ps), &params); err != nil {
			return Def{}, fmt.Errorf("entry %q: invalid params JSON: %w", e.Name, err)
		}
		d.Params = params
	}

	if v := strings.TrimSpace(e.Fields["disabled"]); v == "true" || v == "1" || v == "yes" {
		d.Disabled = true
	}

	return d, nil
}

// SanitizeName converts an entry name to a valid MCP tool name.
// Spaces and hyphens become underscores; non-alphanumeric/underscore
// characters are dropped; the result is lowercased.
func SanitizeName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_':
			b.WriteRune(r)
		case r == ' ' || r == '-':
			b.WriteRune('_')
		}
	}
	return b.String()
}
