package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/madicen/naitv-mcp/internal/xpath"
)

// Load fetches and parses a plugin manifest from a URL or local file path.
//
// URL sources (http:// or https://) are fetched with a 30 s timeout.
// All other values are treated as file system paths; a leading ~ is expanded
// to the current user's home directory.
//
// Returns an error if the source cannot be read, the JSON cannot be parsed, or
// the manifest is missing its required name field.
func Load(source string) (Manifest, error) {
	var (
		data []byte
		err  error
	)
	if xpath.IsHTTP(source) {
		data, err = fetchURL(source)
	} else {
		data, err = readFile(source)
	}
	if err != nil {
		return Manifest{}, fmt.Errorf("load plugin %q: %w", source, err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("parse plugin %q: %w", source, err)
	}
	if m.Name == "" {
		return Manifest{}, fmt.Errorf("plugin manifest %q: name field is required", source)
	}
	return m, nil
}

// fetchURL performs an HTTP GET and returns the response body.
func fetchURL(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func readFile(path string) ([]byte, error) {
	return os.ReadFile(xpath.ExpandHome(path))
}
