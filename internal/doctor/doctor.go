package doctor

import (
	"fmt"
	"os"
	"strings"

	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/internal/tools"
)

func Run(dbPath string, rebuildFTS bool) error {
	var issues int
	fmt.Printf("Database: %s\n", dbPath)
	if info, err := os.Stat(dbPath); err != nil {
		fmt.Printf("  ✗ cannot stat: %v\n", err)
		issues++
	} else {
		fmt.Printf("  ✓ exists (%d bytes)\n", info.Size())
	}
	st, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer st.Close()
	if err := st.IntegrityCheck(); err != nil {
		fmt.Printf("  ✗ integrity_check: %v\n", err)
		issues++
	} else {
		fmt.Println("  ✓ PRAGMA integrity_check ok")
	}
	if outOfSync, err := st.FTSOutOfSync(); err != nil {
		fmt.Printf("  ✗ FTS check: %v\n", err)
		issues++
	} else if outOfSync {
		if rebuildFTS {
			if err := st.RebuildFTS(); err != nil {
				fmt.Printf("  ✗ FTS rebuild: %v\n", err)
				issues++
			} else {
				fmt.Println("  ✓ FTS index rebuilt")
			}
		} else {
			fmt.Println("  ⚠ FTS index may be out of sync (use --rebuild-fts)")
			issues++
		}
	} else {
		fmt.Println("  ✓ FTS index in sync")
	}
	orphans, _ := st.OrphanProposals()
	if len(orphans) > 0 {
		fmt.Printf("  ⚠ %d orphaned proposals\n", len(orphans))
		issues++
	} else {
		fmt.Println("  ✓ no orphaned proposals")
	}
	defs, _ := tools.ListDefs(st)
	bad := 0
	for _, d := range defs {
		if strings.TrimSpace(d.Exec) == "" {
			bad++
		}
	}
	if bad > 0 {
		fmt.Printf("  ⚠ %d tool(s) missing exec\n", bad)
		issues++
	} else {
		fmt.Printf("  ✓ %d executable tool(s) ok\n", len(defs))
	}
	bin, _ := os.Executable()
	fmt.Printf("\nCursor mcp.json snippet:\n{\n  \"mcpServers\": {\n    \"naitv-mcp\": {\"command\": %q, \"args\": [\"serve\"]}\n  }\n}\n", bin)
	if issues > 0 {
		return fmt.Errorf("%d issue(s) found", issues)
	}
	fmt.Println("\nAll checks passed.")
	return nil
}
