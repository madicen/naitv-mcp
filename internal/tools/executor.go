package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/madicen/naitv-mcp/internal/xpath"
)

// Result holds the output from running an executable tool.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
	// Error is non-empty when the process could not be started or timed out,
	// or when JSON I/O validation fails.
	// It is distinct from a non-zero exit code — both may occur together.
	Error string
	// IOMode is the mode used for this run (text or json). Set by Run/RunJSON.
	IOMode string
}

// Run executes a Def with string arguments (for {param} interpolation).
// When def.IOMode is json, args are also marshaled as a JSON object on stdin.
// The provided context is respected for cancellation; the Def's Timeout
// is applied as an additional hard deadline.
func Run(ctx context.Context, def Def, args map[string]string) Result {
	var jsonArgs map[string]any
	if def.IsJSON() {
		jsonArgs = make(map[string]any, len(args))
		for k, v := range args {
			jsonArgs[k] = v
		}
	}
	return execute(ctx, def, args, jsonArgs)
}

// RunJSON executes a Def in JSON I/O mode with a full argument object.
// Prefer this when the MCP client supplies non-string JSON values.
// String placeholders in exec/working_dir are filled from string-valued args
// (non-strings are JSON-encoded for interpolation only).
func RunJSON(ctx context.Context, def Def, args map[string]any) Result {
	if !def.IsJSON() {
		def.IOMode = IOModeJSON
	}
	strArgs := make(map[string]string, len(args))
	for k, v := range args {
		switch t := v.(type) {
		case string:
			strArgs[k] = t
		case nil:
			strArgs[k] = ""
		default:
			b, err := json.Marshal(t)
			if err != nil {
				return Result{IOMode: IOModeJSON, Error: fmt.Sprintf("marshal arg %q: %v", k, err)}
			}
			strArgs[k] = string(b)
		}
	}
	return execute(ctx, def, strArgs, args)
}

func execute(ctx context.Context, def Def, strArgs map[string]string, jsonArgs map[string]any) Result {
	mode := def.IOMode
	if mode == "" {
		mode = IOModeText
	}

	if def.Disabled {
		return Result{IOMode: mode, Error: fmt.Sprintf("tool %q is disabled", def.Name)}
	}

	cmdStr := interpolate(def.Exec, strArgs)

	tctx, cancel := context.WithTimeout(ctx, def.Timeout)
	defer cancel()

	c := exec.CommandContext(tctx, "sh", "-c", cmdStr) //nolint:gosec // exec is user-approved
	c.Env = filterEnv(def.EnvAllowlist)

	// Interpolate working_dir from runtime args (e.g. {project_root}) then
	// expand ~. If placeholders remain unresolved (agent didn't pass the param),
	// fall back to empty string so the process inherits the server's CWD.
	workDir := interpolate(def.WorkingDir, strArgs)
	if strings.Contains(workDir, "{") {
		workDir = "" // unresolved placeholder — use server CWD
	}
	if workDir != "" {
		c.Dir = xpath.ExpandHome(workDir)
	}

	if def.IsJSON() {
		var payload []byte
		if jsonArgs == nil {
			payload = []byte("{}")
		} else {
			var err error
			payload, err = json.Marshal(jsonArgs)
			if err != nil {
				return Result{IOMode: mode, Error: fmt.Sprintf("marshal JSON stdin: %v", err)}
			}
		}
		c.Stdin = bytes.NewReader(append(payload, '\n'))
	}

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	start := time.Now()
	runErr := c.Run()
	dur := time.Since(start)

	r := Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: dur,
		IOMode:   mode,
	}

	if runErr != nil {
		if tctx.Err() == context.DeadlineExceeded {
			r.Error = fmt.Sprintf("timed out after %s", def.Timeout)
		} else if exitErr, ok := runErr.(*exec.ExitError); ok {
			r.ExitCode = exitErr.ExitCode()
			// Non-zero exit is expected (e.g. test failures) — not a hard error.
		} else {
			r.Error = runErr.Error()
		}
	}

	if def.IsJSON() && r.Error == "" && r.ExitCode == 0 {
		trimmed := bytes.TrimSpace(stdout.Bytes())
		if len(trimmed) == 0 {
			r.Error = "json io: stdout is empty (expected a JSON value)"
		} else if !json.Valid(trimmed) {
			r.Error = "json io: stdout is not valid JSON"
		} else {
			r.Stdout = string(trimmed)
		}
	}

	return r
}

// Format returns a human-readable summary of the result, suitable for
// returning to the model as tool output. In JSON mode on success, returns
// the JSON stdout only (no timing footer).
func (r Result) Format() string {
	if strings.EqualFold(r.IOMode, IOModeJSON) {
		return r.formatJSON()
	}

	var sb strings.Builder

	if r.Error != "" {
		fmt.Fprintf(&sb, "⚠ error: %s\n\n", r.Error)
	}

	if r.Stdout != "" {
		sb.WriteString(r.Stdout)
		if !strings.HasSuffix(r.Stdout, "\n") {
			sb.WriteString("\n")
		}
	}

	if r.Stderr != "" {
		sb.WriteString("\nstderr:\n")
		sb.WriteString(r.Stderr)
		if !strings.HasSuffix(r.Stderr, "\n") {
			sb.WriteString("\n")
		}
	}

	if r.ExitCode != 0 {
		fmt.Fprintf(&sb, "\nexit code: %d", r.ExitCode)
	}

	fmt.Fprintf(&sb, "\n(completed in %s)", r.Duration.Round(time.Millisecond))
	return sb.String()
}

func (r Result) formatJSON() string {
	if r.Error == "" && r.ExitCode == 0 {
		return r.Stdout
	}

	var sb strings.Builder
	if r.Error != "" {
		fmt.Fprintf(&sb, "⚠ error: %s\n", r.Error)
	}
	if r.Stderr != "" {
		sb.WriteString(r.Stderr)
		if !strings.HasSuffix(r.Stderr, "\n") {
			sb.WriteString("\n")
		}
	}
	if r.Stdout != "" {
		sb.WriteString(r.Stdout)
		if !strings.HasSuffix(r.Stdout, "\n") {
			sb.WriteString("\n")
		}
	}
	if r.ExitCode != 0 {
		fmt.Fprintf(&sb, "exit code: %d\n", r.ExitCode)
	}
	fmt.Fprintf(&sb, "(completed in %s)", r.Duration.Round(time.Millisecond))
	return sb.String()
}

// interpolate replaces {name} placeholders in template with values from args.
// Unknown placeholders are left as-is.
func interpolate(template string, args map[string]string) string {
	result := template
	for k, v := range args {
		result = strings.ReplaceAll(result, "{"+k+"}", v)
	}
	return result
}

var defaultEnvAllowlist = []string{"PATH", "HOME"}

func filterEnv(allowlist []string) []string {
	if len(allowlist) == 0 {
		allowlist = defaultEnvAllowlist
	}
	allowed := make(map[string]bool, len(allowlist))
	for _, k := range allowlist {
		allowed[k] = true
	}
	var out []string
	for _, e := range os.Environ() {
		key, _, _ := strings.Cut(e, "=")
		if allowed[key] {
			out = append(out, e)
		}
	}
	return out
}

// ShellCommandLine returns the exact sh -c invocation for a tool definition.
func ShellCommandLine(def Def, args map[string]string) string {
	cmdStr := interpolate(def.Exec, args)
	return fmt.Sprintf("sh -c %q", cmdStr)
}
