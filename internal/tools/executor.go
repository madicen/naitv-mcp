package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Result holds the output from running an executable tool.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
	// Error is non-empty when the process could not be started or timed out.
	// It is distinct from a non-zero exit code — both may occur together.
	Error string
}

// Run executes a Def with the given arguments and returns the output.
// args maps placeholder names to their values (e.g. {"file": "main.go"}).
// The provided context is respected for cancellation; the Def's Timeout
// is applied as an additional hard deadline.
func Run(ctx context.Context, def Def, args map[string]string) Result {
	if def.Disabled {
		return Result{Error: fmt.Sprintf("tool %q is disabled", def.Name)}
	}

	cmdStr := interpolate(def.Exec, args)

	tctx, cancel := context.WithTimeout(ctx, def.Timeout)
	defer cancel()

	c := exec.CommandContext(tctx, "sh", "-c", cmdStr) //nolint:gosec // exec is user-approved
	c.Env = os.Environ()

	// Interpolate working_dir from runtime args (e.g. {project_root}) then
	// expand ~. If placeholders remain unresolved (agent didn't pass the param),
	// fall back to empty string so the process inherits the server's CWD.
	workDir := interpolate(def.WorkingDir, args)
	if strings.Contains(workDir, "{") {
		workDir = "" // unresolved placeholder — use server CWD
	}
	if workDir != "" {
		c.Dir = expandHome(workDir)
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

	return r
}

// Format returns a human-readable summary of the result, suitable for
// returning to the model as tool output.
func (r Result) Format() string {
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

// interpolate replaces {name} placeholders in template with values from args.
// Unknown placeholders are left as-is.
func interpolate(template string, args map[string]string) string {
	result := template
	for k, v := range args {
		result = strings.ReplaceAll(result, "{"+k+"}", v)
	}
	return result
}

// expandHome replaces a leading ~ with the user's home directory.
// If os.UserHomeDir fails, the path is returned unchanged.
func expandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}
