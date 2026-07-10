package xpath

import "testing"

func TestExpandHome(t *testing.T) {
	t.Setenv("HOME", "/home/test")
	if got := ExpandHome("~/dev"); got != "/home/test/dev" {
		t.Fatalf("ExpandHome = %q", got)
	}
	if ExpandHome("/abs") != "/abs" {
		t.Fatal("absolute path changed")
	}
}

func TestIsHTTP(t *testing.T) {
	if !IsHTTP("https://example.com/x") {
		t.Fatal("expected https URL")
	}
	if IsHTTP("/local/path") {
		t.Fatal("path should not be HTTP")
	}
}

func TestIsFilePath(t *testing.T) {
	for _, p := range []string{"/abs", "./rel", "../up", "~/home"} {
		if !IsFilePath(p) {
			t.Fatalf("%q should be file path", p)
		}
	}
}
