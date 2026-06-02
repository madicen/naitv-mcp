package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/madicen/naitv-mcp/internal/instructions"
	"github.com/madicen/naitv-mcp/internal/mcp"
	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/internal/tui"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "serve":
			if err := runServer(os.Args[2:]); err != nil {
				fmt.Fprintf(os.Stderr, "naitv-mcp: %v\n", err)
				os.Exit(1)
			}
			return
		case "seed-demo":
			if err := runSeedDemo(os.Args[2:]); err != nil {
				fmt.Fprintf(os.Stderr, "naitv-mcp: %v\n", err)
				os.Exit(1)
			}
			return
		case "init":
			if err := runInit(os.Args[2:]); err != nil {
				fmt.Fprintf(os.Stderr, "naitv-mcp: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}
	if err := runTUI(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "naitv-mcp: %v\n", err)
		os.Exit(1)
	}
}

func runServer(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	dbPath := fs.String("db", store.DefaultDBPath(), "Path to SQLite database")
	fs.Parse(args)

	st, err := store.Open(*dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()
	return mcp.Run(st)
}

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	dbPath := fs.String("db", store.DefaultDBPath(), "Path to SQLite database")
	out := fs.String("out", "AGENTS.md", "Output file path. Use '-' to write to stdout.")
	fs.Parse(args)

	st, err := store.Open(*dbPath)
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

	if *out == "-" {
		fmt.Print(doc)
		return nil
	}

	if err := os.WriteFile(*out, []byte(doc), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", *out, err)
	}
	fmt.Fprintf(os.Stderr, "Wrote %d of %d entries (init delivery) to %s\n", len(initEntries), len(entries), *out)
	return nil
}

func runTUI(args []string) error {
	fs := flag.NewFlagSet("tui", flag.ExitOnError)
	dbPath := fs.String("db", store.DefaultDBPath(), "Path to SQLite database")
	demo := fs.Bool("demo", false, "Run with seeded demo data (for VHS recordings)")
	fs.Parse(args)

	if *demo {
		if err := configureDemoEnv(); err != nil {
			return fmt.Errorf("demo setup: %w", err)
		}
		// Refresh dbPath after env mutation
		*dbPath = store.DefaultDBPath()
	}

	st, err := store.Open(*dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	m := tui.New(st)
	prog := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = prog.Run()
	return err
}

func runSeedDemo(args []string) error {
	fs := flag.NewFlagSet("seed-demo", flag.ExitOnError)
	fs.Parse(args)
	dbPath := store.DefaultDBPath()
	st, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer st.Close()
	return seedDemoDB(st)
}

func configureDemoEnv() error {
	lipgloss.SetColorProfile(termenv.TrueColor)

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
	// Check if already seeded (idempotent)
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

	// Pending proposals so Review tab has content
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
