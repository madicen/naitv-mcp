// Package markdown renders entry bodies with glamour, cached per entry revision.
package markdown

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
)

// Renderer caches rendered markdown keyed by entry id and body hash.
type Renderer struct {
	mu    sync.RWMutex
	cache map[string]string
	width int
}

// NewRenderer creates a markdown renderer for the given terminal width.
func NewRenderer(width int) *Renderer {
	return &Renderer{cache: make(map[string]string), width: width}
}

// SetWidth updates the wrap width used for rendering.
func (r *Renderer) SetWidth(width int) {
	if width < 1 {
		width = 1
	}
	r.mu.Lock()
	r.width = width
	r.cache = make(map[string]string)
	r.mu.Unlock()
}

// RenderBody returns glamour-rendered markdown for body, using cache when possible.
func (r *Renderer) RenderBody(entryID, body string) string {
	if body == "" {
		return ""
	}
	key := entryID + ":" + hash(body)
	r.mu.RLock()
	if cached, ok := r.cache[key]; ok {
		r.mu.RUnlock()
		return cached
	}
	width := r.width
	r.mu.RUnlock()

	rendered, err := renderMarkdown(body, width)
	if err != nil {
		rendered = body
	}

	r.mu.Lock()
	r.cache[key] = rendered
	r.mu.Unlock()
	return rendered
}

func renderMarkdown(body string, width int) (string, error) {
	r, err := glamour.NewTermRenderer(
		glamour.WithEnvironmentConfig(),
		glamour.WithWordWrap(width),
		glamour.WithStyles(styles.DarkStyleConfig),
	)
	if err != nil {
		return "", fmt.Errorf("glamour: %w", err)
	}
	out, err := r.Render(body)
	if err != nil {
		return "", err
	}
	return out, nil
}

func hash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}
