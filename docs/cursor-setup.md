# Cursor + naitv-mcp: Loop Engineering Setup

This guide wires naitv-mcp into Cursor to implement the
[Loop Engineering](https://www.langchain.com/blog/the-art-of-loop-engineering)
pattern — agents that verify their own output, never return broken code, and
compound knowledge over time.

## How the loops map to this stack

| Loop | What it does | Who provides it |
|------|--------------|-----------------|
| 1. Agent loop | Model calls tools in a cycle until the task is complete | Cursor |
| 2. Verification loop | Build/vet/test run after every change; failures loop back | naitv-mcp exec tools |
| 3. Event-driven loop | Events trigger agent runs | Cursor (file watchers, slash commands) |
| 4. Hill climbing loop | Sessions improve the knowledge base over time | naitv-mcp proposal/approval workflow |

---

## Step 1 — Build the binary

```bash
cd ~/path/to/naitv-mcp
make build        # → bin/naitv-mcp
```

---

## Step 2 — Wire Cursor to the MCP server

Add a `.cursor/mcp.json` at your project root (or in Cursor's global settings):

```json
{
  "mcpServers": {
    "naitv-mcp": {
      "command": "/absolute/path/to/naitv-mcp/bin/naitv-mcp",
      "args": ["serve"]
    }
  }
}
```

Use `which naitv-mcp` if you installed it globally via `go install` or Homebrew;
otherwise use the `bin/naitv-mcp` path from the repo.

Restart Cursor. In the chat, ask: **"What MCP tools do you have?"** — you should
see `initialize`, `list_entries`, `search_entries`, `add_entry`, `update_entry`,
`install_plugin`, `list_plugins`, `list_available_plugins`, and `uninstall_plugin`.

---

## Step 3 — Install the loop-engineering-go plugin

Tell the agent:

```
Install the plugin at https://raw.githubusercontent.com/madicen/naitv-mcp-plugins/main/plugins/loop-engineering-go.json
```

The agent calls `install_plugin` and proposes all plugin entries as **pending**
in naitv-mcp. Nothing is active yet.

---

## Step 4 — Review and approve proposals in the TUI

```bash
naitv-mcp    # opens the terminal UI
```

Go to the **Review** tab. You'll see the proposed rules, workflows, and tools
from the plugin. For each one: approve (`a`), edit before approving (`e`), or
reject (`r`). Press `A` to approve all at once.

Entries with `kind: tool` and an `exec` field show an **⚠ EXECUTABLE TOOL**
warning — review the `exec` command before approving.

Once approved, the verification tools (`build`, `vet`, `test`, etc.) are
registered as live MCP tools on the next server restart.

---

## Step 5 — Restart the MCP server

Cursor needs to reconnect to pick up the newly registered tools. The easiest
way is to reload Cursor's MCP servers from the settings panel, or just restart
Cursor. Verify the new tools appear: `build`, `vet`, `test`, `test_run`,
`test_race`, `lint`.

---

## Step 6 — Register your project

The `load-project-context` rule tells the agent to look up a `kind=project`
entry for the current repo at session start. Add one so the agent always knows
where your project lives and how to build it.

Either tell the agent:

```
Add a project entry for this repo. The path is /absolute/path/to/myproject.
```

Or create it yourself in the TUI (`n` for new entry):

| Field | Value |
|-------|-------|
| Kind | `project` |
| Name | `my-repo-name` |
| Group | `my-repo-name` |
| Body | Brief description |
| Fields | `path = /absolute/path/to/myproject` |

Optional field overrides:

| Field | Example |
|-------|---------|
| `build_cmd` | `make build` |
| `test_cmd` | `make test` |
| `lint_cmd` | `golangci-lint run ./...` |

If the agent can't find a project entry it falls back to the current working
directory and proposes creating one at the end of the session.

---

## Step 7 — Test it

Open a Go project in Cursor and give the agent a task. It should:

1. Call `initialize` and load your rules and workflows.
2. Look up the project entry to get `project_root`.
3. Do the work.
4. Before responding, call `build`, `vet`, and `test` — fixing failures and
   looping until all three pass.
5. End with a verification summary:
   ```
   ✓ build passed
   ✓ vet passed
   ✓ test passed  (N tests across M packages)
   ```

If the agent skips verification, the `verify-before-done` rule is the lever —
you can strengthen the wording or move it to `on-demand` and reference it
explicitly in your prompt.

---

## Ongoing: hill climbing

After sessions, the agent proposes new entries via `add_entry`. Review them in
the TUI's **Review** tab. Good candidates to watch for:

- A rule you corrected the agent on
- A project-specific convention it inferred (file layout, migration pattern, etc.)
- A codebase fact it had to figure out the hard way

Approve the ones that are accurate. Over time the knowledge base compounds and
the agent needs less hand-holding per session.

---

## Enabling golangci-lint

The `lint` tool is installed disabled by default (not everyone has
`golangci-lint`). Once it's installed:

1. Find the `lint` entry in the TUI's **Entries** tab.
2. Press `e` to edit it.
3. In the `fields` section, change `disabled` from `true` to `false`.
4. Save and restart the MCP server.

---

## MCP tool reference

### Knowledge base

| Tool | Description |
|------|-------------|
| `initialize` | Returns all `init`-delivery entries as standing instructions. |
| `list_entries` | List active entries, filter by `kind` and/or `tags`. |
| `get_entry` | Fetch a single entry by ID or name. |
| `search_entries` | Full-text search over active entries. |
| `add_entry` | Propose a new entry (queued as pending for review). |
| `update_entry` | Propose an update to an existing entry (queued for review). |

### Plugin management

| Tool | Description |
|------|-------------|
| `install_plugin` | Fetch a plugin by name, URL, or file path and propose its entries. |
| `list_plugins` | List installed plugins with version and source URL. |
| `list_available_plugins` | Show what's available in the registry. |
| `uninstall_plugin` | Remove a plugin and all its entries. |

### Verification (registered after approving loop-engineering-go)

| Tool | Params | Description |
|------|--------|-------------|
| `build` | `project_root` | `go build ./...` |
| `vet` | `project_root` | `go vet ./...` |
| `test` | `project_root` | `go test -timeout 120s ./...` |
| `test_run` | `project_root`, `pattern`, `pkg` | Run a targeted subset of tests. |
| `test_race` | `project_root` | `go test -race ./...` |
| `lint` | `project_root` | `golangci-lint run ./...` (disabled until enabled) |
