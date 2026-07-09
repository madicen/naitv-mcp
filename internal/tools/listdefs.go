package tools

import (
	"fmt"

	"github.com/madicen/naitv-mcp/internal/store"
)

// ListDefs returns parsed executable tool definitions from active tool entries.
func ListDefs(st *store.Store) ([]Def, error) {
	entries, err := st.List("tool", nil)
	if err != nil {
		return nil, fmt.Errorf("list tool entries: %w", err)
	}

	var defs []Def
	for _, e := range entries {
		if !IsExecutable(e) {
			continue
		}
		def, err := ParseDef(e)
		if err != nil {
			continue
		}
		defs = append(defs, def)
	}
	return defs, nil
}
