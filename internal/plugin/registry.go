package plugin

import (
	"encoding/json"
	"fmt"

	"github.com/madicen/naitv-mcp/internal/xpath"
)

// DefaultRegistryURL is the well-known public registry hosted in madicen/naitv-plugins.
const DefaultRegistryURL = "https://raw.githubusercontent.com/madicen/naitv-plugins/main/registry.json"

// RegistryEntry describes a single plugin listed in the registry.
type RegistryEntry struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Tags        []string `json:"tags,omitempty"`
	Version     string   `json:"version"`
	Author      string   `json:"author,omitempty"`
}

// Registry is the top-level structure of a registry.json file.
type Registry struct {
	Plugins []RegistryEntry `json:"plugins"`
}

// LoadRegistry fetches and parses a plugin registry from the given URL.
// Pass DefaultRegistryURL to use the well-known public registry.
// A local file path can also be passed for testing (url is forwarded to fetchURL
// only when it starts with http/https; otherwise treat as a file path).
func LoadRegistry(url string) (Registry, error) {
	var (
		data []byte
		err  error
	)
	// Reuse the same URL/file detection logic as Load.
	if xpath.IsHTTP(url) {
		data, err = fetchURL(url)
	} else {
		data, err = readFile(url)
	}
	if err != nil {
		return Registry{}, fmt.Errorf("load registry: %w", err)
	}
	var r Registry
	if err := json.Unmarshal(data, &r); err != nil {
		return Registry{}, fmt.Errorf("parse registry: %w", err)
	}
	return r, nil
}

// Find returns the first registry entry with the given name, or nil.
func (r Registry) Find(name string) *RegistryEntry {
	for i := range r.Plugins {
		if r.Plugins[i].Name == name {
			return &r.Plugins[i]
		}
	}
	return nil
}
