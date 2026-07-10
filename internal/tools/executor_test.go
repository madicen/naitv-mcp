package tools

import (
	"strings"
	"testing"

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
