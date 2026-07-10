package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/madicen/naitv-mcp/pkg/entry"
)

func TestFilterEnv_DefaultAllowlist(t *testing.T) {
	t.Setenv("PATH", "/bin")
	t.Setenv("HOME", "/home/test")
	t.Setenv("SECRET_TOKEN", "nope")

	got := filterEnv(nil)
	for _, e := range got {
		key, _, _ := strings.Cut(e, "=")
		if key == "SECRET_TOKEN" {
			t.Fatalf("SECRET_TOKEN leaked into env: %v", got)
		}
	}
	has := func(key string) bool {
		for _, e := range got {
			k, _, _ := strings.Cut(e, "=")
			if k == key {
				return true
			}
		}
		return false
	}
	if !has("PATH") || !has("HOME") {
		t.Fatalf("expected PATH and HOME in filtered env, got %v", got)
	}
}

func TestShellCommandLine(t *testing.T) {
	line := ShellCommandLine(Def{Exec: "echo {msg}"}, map[string]string{"msg": "hi"})
	if line != `sh -c "echo hi"` {
		t.Fatalf("ShellCommandLine = %q", line)
	}
}

func TestParseDef_EnvAllowlist(t *testing.T) {
	def, err := ParseDef(entry.Entry{
		Kind:   "tool",
		Name:   "test-tool",
		Fields: map[string]string{"exec": "true", "env_allowlist": "PATH, CUSTOM"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(def.EnvAllowlist) != 2 || def.EnvAllowlist[0] != "PATH" || def.EnvAllowlist[1] != "CUSTOM" {
		t.Fatalf("EnvAllowlist = %#v", def.EnvAllowlist)
	}
}

func TestRun_DisabledAndEcho(t *testing.T) {
	disabled := Run(context.Background(), Def{Name: "x", Disabled: true}, nil)
	if disabled.Error == "" {
		t.Fatal("expected disabled error")
	}
	got := Run(context.Background(), Def{Name: "echo", Exec: "echo hello", Timeout: 5 * time.Second}, nil)
	if got.Error != "" {
		t.Fatalf("run error: %s", got.Error)
	}
	if !strings.Contains(got.Stdout, "hello") {
		t.Fatalf("stdout = %q", got.Stdout)
	}
}

func TestResultFormat(t *testing.T) {
	text := Result{
		Stdout:   "ok",
		Stderr:   "warn",
		ExitCode: 2,
		Duration: 1500 * time.Millisecond,
	}.Format()
	for _, want := range []string{"ok", "stderr:", "warn", "exit code: 2", "completed in"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %q", want, text)
		}
	}
	errText := Result{Error: "boom"}.Format()
	if !strings.Contains(errText, "error: boom") {
		t.Fatalf("error format = %q", errText)
	}
}
