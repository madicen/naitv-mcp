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

func Load(source string) (Manifest, error) {
	var data []byte
	var err error
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
	if err := validateManifest(m, source); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

func validateManifest(m Manifest, source string) error {
	if m.Name == "" {
		return fmt.Errorf("plugin manifest %q: name field is required", source)
	}
	if m.Version == "" {
		return fmt.Errorf("plugin manifest %q: version field is required", source)
	}
	for i, spec := range m.Entries {
		if spec.Kind == "" {
			return fmt.Errorf("plugin manifest %q: entry[%d] missing kind", source, i)
		}
		if spec.Name == "" {
			return fmt.Errorf("plugin manifest %q: entry[%d] missing name", source, i)
		}
	}
	return nil
}

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
