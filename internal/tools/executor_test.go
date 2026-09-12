package tools

import (
	"context"
	"encoding/json"
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

func TestParseDef_IOMode(t *testing.T) {
	def, err := ParseDef(entry.Entry{
		Kind:   "tool",
		Name:   "json-tool",
		Fields: map[string]string{"exec": "true", "io_mode": "json"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !def.IsJSON() || def.IOMode != IOModeJSON {
		t.Fatalf("IOMode = %q", def.IOMode)
	}

	def, err = ParseDef(entry.Entry{
		Kind:   "tool",
		Name:   "camel",
		Fields: map[string]string{"exec": "true", "ioMode": "json"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !def.IsJSON() {
		t.Fatal("expected ioMode alias to set JSON")
	}

	_, err = ParseDef(entry.Entry{
		Kind:   "tool",
		Name:   "bad",
		Fields: map[string]string{"exec": "true", "io_mode": "binary"},
	})
	if err == nil {
		t.Fatal("expected invalid io_mode error")
	}
}

func TestRunJSON_RoundTrip(t *testing.T) {
	def := Def{
		Name:    "upper",
		Exec:    `python3 -c 'import json,sys; d=json.load(sys.stdin); print(json.dumps({"result": d["input"].upper()}))'`,
		IOMode:  IOModeJSON,
		Timeout: 10 * time.Second,
	}
	got := RunJSON(context.Background(), def, map[string]any{"input": "hello"})
	if got.Error != "" {
		t.Fatalf("run error: %s\nstderr: %s", got.Error, got.Stderr)
	}
	if got.ExitCode != 0 {
		t.Fatalf("exit %d stderr=%q", got.ExitCode, got.Stderr)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(got.Stdout), &out); err != nil {
		t.Fatalf("stdout not JSON: %v (%q)", err, got.Stdout)
	}
	if out["result"] != "HELLO" {
		t.Fatalf("result = %#v", out["result"])
	}
	if got.Format() != got.Stdout {
		t.Fatalf("Format should be raw JSON on success, got %q", got.Format())
	}
}

func TestRun_TextModeUnchanged(t *testing.T) {
	got := Run(context.Background(), Def{
		Name:    "echo",
		Exec:    "echo hello",
		IOMode:  IOModeText,
		Timeout: 5 * time.Second,
	}, nil)
	if got.Error != "" || !strings.Contains(got.Stdout, "hello") {
		t.Fatalf("text mode broke: %#v", got)
	}
	if !strings.Contains(got.Format(), "completed in") {
		t.Fatalf("text Format missing footer: %q", got.Format())
	}
}

func TestRunJSON_NonZeroExit(t *testing.T) {
	got := RunJSON(context.Background(), Def{
		Name:    "fail",
		Exec:    `python3 -c 'import sys; print("boom", file=sys.stderr); sys.exit(2)'`,
		IOMode:  IOModeJSON,
		Timeout: 5 * time.Second,
	}, map[string]any{})
	if got.ExitCode != 2 {
		t.Fatalf("exit = %d", got.ExitCode)
	}
	if !strings.Contains(got.Stderr, "boom") {
		t.Fatalf("stderr = %q", got.Stderr)
	}
	if !strings.Contains(got.Format(), "exit code: 2") {
		t.Fatalf("Format = %q", got.Format())
	}
}

func TestRunJSON_MalformedStdout(t *testing.T) {
	got := RunJSON(context.Background(), Def{
		Name:    "bad-json",
		Exec:    `echo not-json`,
		IOMode:  IOModeJSON,
		Timeout: 5 * time.Second,
	}, map[string]any{})
	if got.Error == "" || !strings.Contains(got.Error, "not valid JSON") {
		t.Fatalf("expected JSON validation error, got %#v", got)
	}
}

func TestRunJSON_Timeout(t *testing.T) {
	got := RunJSON(context.Background(), Def{
		Name:    "slow",
		Exec:    `sleep 5`,
		IOMode:  IOModeJSON,
		Timeout: 200 * time.Millisecond,
	}, map[string]any{})
	if got.Error == "" || !strings.Contains(got.Error, "timed out") {
		t.Fatalf("expected timeout, got %#v", got)
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
