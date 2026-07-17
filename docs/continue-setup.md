# Continue + naitv-mcp: Loop Engineering Setup

This guide wires naitv-mcp into Continue for VS Code to implement the
[Loop Engineering](https://www.langchain.com/blog/the-art-of-loop-engineering)
pattern with smaller local models.

The key principle: **naitv-mcp is a server, not a CLI tool**. Every interaction
with it — including setup — happens through an agent talking to the running
MCP server. You never reach into the database directly.

---

## How the loops map to this stack

| Loop | What it does | Who provides it |
|------|-------------|-----------------|
| 1. Agent loop | Model calls tools in a loop until task is complete | Continue |
| 2. Verification loop | Output scored; retried with feedback if it fails | naitv-mcp `build`, `vet`, `test`, … |
| 3. Event-driven loop | Events trigger agent runs | Continue (file watchers, slash commands) |
| 4. Hill climbing loop | Runs improve the harness over time | naitv-mcp proposal/approval workflow |

---

## Step 1 — Bootstrap: wire Continue to naitv-mcp

Before anything else, connect Continue to the running naitv-mcp server.
Create `.continue/config.yaml` in your project with at minimum:

```yaml
models:
  - name: local-model
    provider: ollama
    model: qwen2.5-coder:14b   # or whatever model you use

mcpServers:
  - name: naitv-mcp
    command: /absolute/path/to/naitv-mcp   # `which naitv-mcp` to find it
    args: ["serve"]

systemMessage: |
  You are working inside Continue with access to naitv-mcp tools.
  Call initialize from naitv-mcp at the start of every session.
```

Start the server: `naitv-mcp serve`

Restart Continue. Ask the model: **"What tools do you have?"** — you should
see `initialize`, `install_plugin`, `set_project`, `generate_continue_config`,
and the other naitv-mcp tools listed.

---

## Step 2 — Install the Go loop-engineering plugin

Once Continue is connected, tell the agent to install the plugin and set up
your project. The agent does all the work through the MCP protocol:

```
Install the loop-engineering-go plugin and set it up for the Go project
at ~/dev/myproject. Then generate my Continue config.
```

The agent will:

1. Call `install_plugin(source="loop-engineering-go")` — fetches the plugin
   from the naitv-plugins registry and proposes all loop-engineering entries
   (rules, workflows, and verification tools) as pending entries in naitv-mcp.

2. Call `set_project(project_dir="~/dev/myproject")` — restores
   `working_dir={project_root}` on loop-engineering tools (pass the path on
   each build/vet/test call) and sets absolute `working_dir` on any other
   executable tools.

3. Call `generate_continue_config()` — returns a complete `.continue/config.yaml`
   pre-populated with the binary path and the list of tools in your store.

4. Show you both results.

> **Local install (before the naitv-plugins repo is live):** pass the path to
> the plugin file directly:
> `install_plugin(source="./plugins/loop-engineering-go.json")`

---

## Step 3 — Review proposals in the TUI

The plugin installer creates entries as **pending proposals**, not active entries.
Open the TUI to review them:

```sh
naitv-mcp
```

Go to the **Review** tab. You'll see all the proposed rules, workflows, and
tool entries. For each one you can approve, edit, or reject. Executable tool
proposals (the Go verification tools) show a **⚠ EXECUTABLE TOOL** warning —
check the `exec` field before approving.

Approve everything you want. On the next `naitv-mcp serve` restart, approved
tool entries are registered as live MCP tools.

---

## Step 4 — Save the generated config

Take the config text from `generate_continue_config` and save it:

```sh
# The agent showed you the config — copy it to:
.continue/config.yaml
```

Or ask the agent to put it in the right place:

```
Save the config you generated as .continue/config.yaml in my project root.
```

Restart Continue. The model will now call `initialize` at session start,
receive your approved rules and workflows, and run verification tools
autonomously.

---

## Ongoing: the hill-climbing loop

After each session the model proposes new entries via `add_entry`. Review and
approve them in the TUI. Over time the knowledge base compounds:

- Session 1: basic rules and tools
- Session 2: the model notices a project convention and proposes a `fact` entry
- Session 3: you corrected a mistake → model proposes a new `rule`
- Session N: the model barely needs guidance for this project

---

## Updating the project directory

If you move the project or set up a new one, ask the agent:

```
Update naitv-mcp to point all verification tools at ~/dev/newproject.
```

The agent calls `set_project(project_dir="~/dev/newproject")`. For
loop-engineering tools this restores `working_dir={project_root}` (the path
is passed on each build/vet/test call). Restart `naitv-mcp serve` to apply.

---

## Enabling golangci-lint

Once `golangci-lint` is installed, ask the agent:

```
Enable the lint tool in naitv-mcp.
```

The agent calls `set_project(project_dir=".", enable_lint="true")`. Restart
`naitv-mcp serve`.

---

## Reference — naitv-mcp tools the model can call

### Knowledge base
| Tool | Description |
|------|-------------|
| `initialize` | Load init-delivery entries as standing instructions |
| `list_entries` | List active entries, filter by kind/tags |
| `get_entry` | Fetch a single entry by ID or name |
| `search_entries` | Full-text search over active entries |
| `add_entry` | Propose a new entry (queued for review) |
| `update_entry` | Propose an update to an existing entry |
| `list_tools` | List all registered executable tools and their params |

### Plugin management (agent-callable)
| Tool | Description |
|------|-------------|
| `install_plugin` | Fetch a plugin by name, URL, or file path and propose its entries |
| `list_plugins` | List installed plugins with version and source |
| `list_available_plugins` | Show what's available in the registry |
| `uninstall_plugin` | Remove a plugin and all its entries |

### Project setup (agent-callable)
| Tool | Description |
|------|-------------|
| `set_project` | Restore `working_dir={project_root}` on parameterized tools; set absolute dir on others; optionally enable lint |
| `generate_continue_config` | Return a ready-to-use `.continue/config.yaml` as text |

### Verification (Go — registered after approving loop-engineering-go plugin entries)
| Tool | Description |
|------|-------------|
| `build` | `go build ./...` |
| `vet` | `go vet ./...` |
| `test` | `go test -timeout 120s ./...` |
| `test_run` | `go test -run {pattern} {pkg}` |
| `test_race` | `go test -race -timeout 180s ./...` |
| `lint` | `golangci-lint run ./...` (disabled until enabled) |
