package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"charm.land/fang/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/madicen/naitv-mcp/internal/instructions"
	"github.com/madicen/naitv-mcp/internal/mcp"
	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/internal/tui"
	"github.com/madicen/naitv-mcp/pkg/entry"
	"github.com/spf13/cobra"
)

func main() {
	root := newRootCmd()
	if err := fang.Execute(context.Background(), root); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var demo bool

	root := &cobra.Command{
		Use:   "naitv-mcp",
		Short: "Local MCP server and TUI for managing AI agent context",
		Long:  "Browse, curate, and serve context entries to MCP clients. Run without a subcommand to open the TUI.",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath, err := cmd.Flags().GetString("db")
			if err != nil {
				return err
			}
			return runTUI(dbPath, demo)
		},
	}
	root.Version = mcp.Version
	root.Flags().BoolVar(&demo, "demo", false, "Run with seeded demo data (for VHS recordings)")
	root.PersistentFlags().String("db", store.DefaultDBPath(), "Path to SQLite database")

	root.AddCommand(newServeCmd(), newInitCmd(), newSeedDemoCmd())
	return root
}

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the MCP server over stdio",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath, err := cmd.Flags().GetString("db")
			if err != nil {
				return err
			}
			st, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("open store: %w", err)
			}
			defer st.Close()
			return mcp.Run(st)
		},
	}
}

func newInitCmd() *cobra.Command {
	var out string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Write the initialization bundle to a file",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath, err := cmd.Flags().GetString("db")
			if err != nil {
				return err
			}
			return runInit(dbPath, out)
		},
	}
	cmd.Flags().StringVar(&out, "out", "AGENTS.md", "Output file path. Use '-' to write to stdout.")
	return cmd
}

func newSeedDemoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "seed-demo",
		Short: "Populate the default database with demo data if empty",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := store.Open(store.DefaultDBPath())
			if err != nil {
				return err
			}
			defer st.Close()
			return seedDemoDB(st)
		},
	}
}

func runInit(dbPath, out string) error {
	st, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	entries, err := st.List("", nil)
	if err != nil {
		return fmt.Errorf("list entries: %w", err)
	}

	initEntries := instructions.FilterInit(entries)
	doc := instructions.Render(initEntries)

	if out == "-" {
		fmt.Print(doc)
		return nil
	}

	if err := os.WriteFile(out, []byte(doc), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}
	fmt.Fprintf(os.Stderr, "Wrote %d of %d entries (init delivery) to %s\n", len(initEntries), len(entries), out)
	return nil
}

func runTUI(dbPath string, demo bool) error {
	if demo {
		if err := configureDemoEnv(); err != nil {
			return fmt.Errorf("demo setup: %w", err)
		}
		dbPath = store.DefaultDBPath()
	}

	st, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	m := tui.New(st)
	prog := tea.NewProgram(m)
	_, err = prog.Run()
	return err
}

func configureDemoEnv() error {
	lipgloss.Writer.Profile = colorprofile.TrueColor

	demoRoot := os.Getenv("NAITV_MCP_DEMO_DIR")
	if demoRoot == "" {
		var err error
		demoRoot, err = os.MkdirTemp("", "naitv-mcp-demo-")
		if err != nil {
			return fmt.Errorf("mkdir demo dir: %w", err)
		}
	}
	dbPath := filepath.Join(demoRoot, "context.db")
	if err := os.Setenv("NAITV_MCP_DB", dbPath); err != nil {
		return err
	}
	st, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer st.Close()
	return seedDemoDB(st)
}

func seedDemoDB(st *store.Store) error {
	existing, _ := st.List("", nil)
	if len(existing) > 0 {
		return nil
	}

	active := []entry.Entry{
		{Kind: "repo", Name: "stackadapt-web", Body: "Monolith. Start with `bin/dev`. Postgres runs in Docker.", Tags: []string{"ruby", "work"}, Fields: map[string]string{"path": "~/dev/stackadapt/stackadapt-web", "lang": "ruby"}},
		{Kind: "repo", Name: "jj-tui", Body: "TUI for jj (jujutsu VCS). bubbletea + bubblezone + bubble-overlay.", Tags: []string{"go", "personal"}, Fields: map[string]string{"path": "~/Documents/GitHub/jj-tui", "lang": "go"}},
		{Kind: "repo", Name: "appr-ai-sal", Body: "AI-assisted PR review TUI. Claude + Gemini backends.", Tags: []string{"go", "personal", "ai"}, Fields: map[string]string{"path": "~/Documents/GitHub/appr-ai-sal", "lang": "go"}},
		{Kind: "workflow", Name: "deploy", Body: "Always deploy to staging first.\nTag format: release/YYYY-MM-DD.\nRun smoke tests before prod."},
		{Kind: "workflow", Name: "code-review", Body: "Check for N+1 queries. Verify test coverage. Run rubocop before approving."},
		{Kind: "fact", Name: "slack workspace", Fields: map[string]string{"url": "stackadapt.slack.com"}},
		{Kind: "fact", Name: "1password vault", Fields: map[string]string{"vault": "StackAdapt Team"}},
		{Kind: "note", Name: "onboarding", Body: "VPN required for internal services.\nUse `asdf` for runtime version management.\nSlack #engineering-help for blockers."},
	}

	for _, e := range active {
		if _, err := st.Create(e); err != nil {
			return fmt.Errorf("seed: create %s: %w", e.Name, err)
		}
	}

	pending := []entry.Entry{
		{Kind: "repo", Name: "naitv-mcp", Body: "Local MCP server + TUI for managing agent context.", Tags: []string{"go", "personal", "mcp"}, Fields: map[string]string{"path": "~/Documents/GitHub/naitv-mcp", "lang": "go"}, ProposedBy: "claude"},
		{Kind: "workflow", Name: "incident-response", Body: "Page on-call via PagerDuty.\nCreate incident Slack channel.\nPost updates every 30 min.", ProposedBy: "claude"},
	}
	for _, e := range pending {
		if _, err := st.CreatePending(e); err != nil {
			return fmt.Errorf("seed: pending %s: %w", e.Name, err)
		}
	}

	return nil
}
