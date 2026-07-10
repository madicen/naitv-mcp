# naitv-mcp Improvement Plan

**Audience:** an executing coding agent (Claude Opus 4.8). This plan was produced from a full-repo audit (July 2026). Every file:line reference below was verified against the current tree. Execute phases in order; each phase is independently shippable and ends with a verification gate.

---

## How to work this plan

- **Build/test loop:** `make build` → `make test` (store unit + integration). Lint with `golangci-lint run` (config: `.golangci.yml`). All must pass before a phase is considered done.
- **VCS:** the repo is colocated jj+git. Make one commit per numbered work item (`jj commit -m "..."`, or git if jj is unavailable). Do not squash phases together.
- **Scope discipline:** do not redesign beyond what an item specifies. When an item says "extract X", preserve behavior exactly; integration tests are the safety net.
- **Verify library versions before pinning.** Version numbers below were researched July 2026; check for newer patch releases at execution time.
- **Screenshots:** after TUI-visible changes (Phases 3, 5), regenerate GIFs with `make vhs` if `vhs`/`ffmpeg`/`ttyd` are available; otherwise note it in the commit message and trigger the Screenshots workflow.

### Current state summary (for orientation)

~7.5k lines of Go. Entry point `cmd/naitv-mcp/main.go` (hand-rolled `os.Args` dispatch). Core: `internal/store` (SQLite+FTS5 via modernc, ULID ids, pending/approve workflow), `internal/mcp/server.go` (13 static tools + dynamic executable tools, mcp-go v0.9.0), `internal/instructions` (renders init bundle), `internal/tools` (executable tool defs + `sh -c` runner), `internal/plugin` (JSON manifest install/uninstall), `internal/setup` (set_project, continue config). TUI: root `internal/tui/model.go` composes four tabs (entries, review, plugins, form) held as concrete structs; shared `theme` (one constant) and a **dead** `mouse` package. Estimated 250–300 duplicated TUI lines across 10 clusters (indexed in Appendix A).

---

## Phase 0 — Hygiene, dead code, docs (half a day)

**0.1 Delete tombstone files.** `cmd/naitv-mcp/seed_loop_go.go`, `cmd/naitv-mcp/init_continue.go`, `cmd/naitv-mcp/set_project.go` are empty `package main` files containing only comments. Two of them reference a `setup` MCP tool that is **not registered anywhere** (grep `"setup"` in `internal/mcp/server.go` — no hits). Delete all three.

**0.2 Delete dead `internal/tui/mouse/zones.go`.** Imported by nothing (verified). It has already drifted from the real zone IDs (missing `action:delivery`, `action:copy`). Its *concept* (central zone-ID registry) is resurrected properly in item 3.4 — do not keep this file around.

**0.3 Fix stale `setup` references in `internal/setup/go.go`.** `ContinueConfig` (:76) emits a commented slash-command referencing a `setup` tool with `stack="go"` (:127–130) that doesn't exist. Remove or rewrite that block to reference real tools.

**0.4 README corrections.**
- MCP tools table (`README.md:146-154`) lists 6 tools; the server registers 13 static tools. Add `list_tools`, `install_plugin`, `list_plugins`, `list_available_plugins`, `uninstall_plugin`, `set_project`, `generate_continue_config`, and a paragraph on dynamic executable tools (`kind=tool` entries with an `exec` field become MCP tools after approval).
- Project layout (`README.md:206-219`) omits `internal/tools/`, `internal/plugin/`, `internal/setup/`. Add them.
- Document the plugins TUI tab (currently undocumented; it is fully wired: `internal/tui/model.go:16,23,42,60`).

**0.5 Repo hygiene.** Add `.DS_Store` and `bin/` to `.gitignore` if missing; remove committed `.DS_Store` files and the committed `bin/naitv-mcp` binary from tracking.

**0.6 Drop-in dependency bumps.** `modernc.org/sqlite` v1.34.5 → latest (v1.52.x as of research; drop-in). Run full test suite.

**Gate:** `make build && make test && golangci-lint run` green; `grep -rn "setup tool" cmd internal` returns nothing stale.

---

## Phase 1 — Dependency modernization (2–4 days)

Order matters: Bubble Tea v2 first (touches every TUI file), then the MCP SDK, then CLI. The Charm v2 ecosystem lives at `charm.land/*` import paths.

**1.1 Bubble Tea v2 + Bubbles v2 + Lip Gloss v2 + bubblezone/v2.**
- Migrate `github.com/charmbracelet/{bubbletea,bubbles,lipgloss}` v1 → `charm.land/{bubbletea,bubbles,lipgloss}/v2`, and `lrstanley/bubblezone` → `github.com/lrstanley/bubblezone/v2`. Follow the upstream `UPGRADE_GUIDE_V2.md` docs.
- Key mechanical changes: `Init()` returns `(tea.Model, tea.Cmd)`; `View()` returns a `tea.View` struct (alt-screen/mouse mode become declarative fields — remove imperative equivalents in `internal/tui/model.go` and `cmd/naitv-mcp/main.go`); `tea.KeyMsg` → `tea.KeyPressMsg` with `Code/Text/Mod`; mouse events split into `MouseClickMsg`/`MouseReleaseMsg`/`MouseWheelMsg`/`MouseMotionMsg` — this **replaces the hand-rolled release-gating** copy-pasted at `entries/model.go:251-253` and `plugins/model.go:210-214` (handle `MouseReleaseMsg` directly).
- Replace `atotto/clipboard` with native OSC52 `tea.SetClipboard` for the copy action (`internal/tui/model.go` `handleEntriesRequest` copy branch, ~:426-437). Drop the dependency.
- Use `tea.WithWindowSize` in integration tests instead of manual size messages where applicable.
- Keep behavior identical; all integration tests must pass.

**1.2 MCP SDK: migrate mcp-go v0.9.0 → official `github.com/modelcontextprotocol/go-sdk` (v1.6.x).**
- Rationale: mcp-go v0.9.0 is ~46 minor versions stale with a fully changed API; the official SDK is stable (API-frozen v1.x), spec-tracking (handles 2025-11-25 and the 2026-07-28 revision negotiation), and supports stdio + streamable HTTP, resources, prompts, elicitation, notifications.
- Rewrite `internal/mcp/server.go` registration layer: `mcp.AddTool(server, &mcp.Tool{...}, handler)` with **typed argument structs** (schemas derived by reflection) replacing every `req.Params.Arguments["x"].(string)` cast (currently at :58, :78, :135 and every handler). Business logic (store/instructions/tools/plugin calls) carries over unchanged.
- Do the Phase 4 refactor items *during* this rewrite where they fall out naturally (shared proposal handler, shared tool-scan) — don't port duplication forward.
- Keep stdio as the served transport; structure `Run` so a streamable-HTTP entry point can be added later (Phase 5.8).

**1.3 CLI: cobra + fang.** Replace the hand-rolled `os.Args[1]` switch (`cmd/naitv-mcp/main.go:19-46`) with `spf13/cobra` commands (`serve`, `init`, `seed-demo`, root = TUI) wrapped in `charm.land/fang/v2` (`fang.Execute`) for styled help, `--version` (wire GoReleaser ldflags version instead of the hardcoded "0.1.0" in `server.go:22`), man pages, and completions. Move the shared `--db` flag to a persistent root flag. Preserve exact flag names (`--db`, `--out`, `--demo`) and the demo env behavior (`configureDemoEnv`, `main.go:139`).

**Gate:** full test suite green; manual smoke: `naitv-mcp --demo` renders all tabs, mouse clicks work, `naitv-mcp serve` handshakes with an MCP client (or the SDK's inmemory transport in a new server_test.go), `naitv-mcp --help` is styled, `--version` prints the ldflags version.

---

## Phase 2 — Store hardening (1–2 days)

All in `internal/store/store.go` (+ new files). Add unit tests for every item in `store_test.go`.

**2.1 Versioned migrations.** Replace the ad-hoc `hasColumn`/`ALTER TABLE` approach (`migrate`, :99-112) with `PRAGMA user_version`-based embedded migrations (a small `migrations.go` with an ordered `[]func(*sql.Tx) error` or embedded SQL files; ~50 lines, no new dependency). Fold the existing `delivery`/`grp` column additions into migration 1 so existing DBs upgrade cleanly. All subsequent items in this phase are new migrations.

**2.2 Indices.** There are currently **zero** explicit indices. Add: `idx_entries_status(status)`, `idx_entries_kind(kind)`, `idx_entries_name(name)`, `idx_entries_target(target_id)`.

**2.3 Enforce name uniqueness among active entries.** `add_entry` documents "must be unique among active entries" (`server.go:124`) but nothing enforces it. Add partial unique index `CREATE UNIQUE INDEX idx_entries_active_name ON entries(name) WHERE status='active'` in a migration that first reports/renames duplicates (`name`, `name (2)`, …). Surface a typed `ErrNameConflict` from `Create`/`Approve` so the MCP handler and TUI can show a useful message.

**2.4 Transactional `ApproveAll`.** `ApproveAll` (:557) loops `Approve` non-transactionally; a mid-loop failure leaves partial state. Wrap in a single transaction (refactor `Approve` to take a `*sql.Tx`-backed internal helper).

**2.5 Move tag filtering into SQL.** `List` loads all rows then filters in Go (`matchesTags`, :247). Use SQLite JSON1: `EXISTS (SELECT 1 FROM json_each(entries.tags) WHERE json_each.value = ?)`.

**2.6 De-duplicate SQL.** `Create` (:331) and `CreatePending` (:416) share a verbatim 14-column INSERT — extract one `insertEntry(status, proposedAt, ...)` helper. `Search` (:291-299) hand-rewrites the column list that `selectCols` (:211) already defines — reuse the constant (aliased). Stop swallowing `json.Marshal` errors in `marshalTags`/`marshalFields` (:149, :158).

**2.7 Soft delete + history (foundation for Phase 5 undo/history).**
- Change `Delete` (:393) to set `status='archived'` (new status) by default; add `Purge` for hard delete. Update FTS triggers so archived entries drop out of the index (or filter `status='active'` in `Search` — it already does; verify).
- New `entry_history` table: `(id, entry_id, snapshot_json, action, actor, created_at)`. Write a snapshot on every `Update`, `Approve` (merge path), `Delete`, `SetDelivery`. Add `History(entryID)` and `RestoreVersion(historyID)`.

**2.8 Export/import.** `ExportJSON(w io.Writer)` (all entries + schema version) and `ImportJSON(r io.Reader, mode merge|replace)` on the store. CLI wiring in Phase 5.2.

**Gate:** `make test-store` green including new tests: unique-name conflict, transactional ApproveAll rollback (inject failure), tag filter parity with old behavior, soft-delete excludes from `List`/`Search`/`initialize`, history snapshot on update, export→import round-trip.

---

## Phase 3 — TUI refactor: kill duplication, adopt bubbles/huh (3–5 days)

The audit found ~250–300 duplicated lines across 10 clusters (Appendix A). This phase eliminates them via shared packages, then swaps hand-rolled widgets for library components. Do 3.1–3.5 (structure) before 3.6–3.9 (widgets).

**3.1 Introduce a `Tab` interface.** Root model (`internal/tui/model.go:36-48`) holds tabs as concrete values and hand-writes per-tab switches in `Update` (:283-305), `View` (:315-324), `SetDimensions` (:350-353), plus ~10 copies of the same 6-line message-forwarding ritual (:127-244, Cluster J). Define in new `internal/tui/tab.go`:

```go
type Tab interface {
    Update(tea.Msg) (Tab, tea.Cmd)
    View() string
    SetDimensions(w, h int)
    Title() string
}
```

Unify the two child→parent protocols (entries/review/plugins return `(Model, *Request, tea.Cmd)`; form emits `SaveMsg`/`CancelMsg`) into **one**: tabs return commands that produce typed request messages the root handles in a single `switch`. Root stores `tabs []Tab` + `active int`; `Update`/`View`/`SetDimensions` become loops. Move the dropdown message routing hack (`model.go:109-123`, where root manually dispatches `bubbledropdown` messages to form-or-entries) into the active tab. Delete the hand-rolled `itoa` (`model.go:561-579`) — use `strconv.Itoa`.

**3.2 Shared list-pane component: `internal/tui/components/listpane`.** One component owning: selection index + j/k/↑/↓ navigation (Cluster B: `entries/model.go:194-203`, `review/model.go:100-109`, `plugins/model.go:180-189`), viewport sync, row rendering via a `RenderRow(i, selected bool, width int) string` callback, per-row bubblezone marks + click hit-testing (Clusters C+D: `entries/model.go:294-306`, `review/model.go:162-168`, `plugins/model.go:230-236`), scroll indicator, and height padding. Prefer building this **on `charm.land/bubbles/v2/list`** (selection, filtering, pagination for free) with a thin zone-aware wrapper; only fall back to a custom component if list's styling model fights the existing look. Entries' group-collapse (`buildFlatItems`, `entries/model.go:409-474`) sits on top as flat items with header rows.

**3.3 Shared layout: `internal/tui/layout`.** One source for the 35/65 split (`SplitWidths(w) (listW, detailW int)`) currently computed in **five places** (Cluster A: `entries/model.go:319-320`, `entries/view_helpers.go:80-81`, `review/model.go:181-182`, `review/view_helpers.go:33-34`, `plugins/model.go:393-394`) and for content-height constants replacing the scattered `h-2`/`h-3`/`h-4` magic offsets and the fragile `SetContentTop(2)` coupling (`model.go:66` ↔ `entries/model.go:124`). Also: one shared `Truncate(s string, max int) string` (Cluster E — four hand-rolled copies) built on lipgloss/x/ansi width functions (rune-count truncation breaks on wide glyphs). Delete plugins' 29-line reimplementation of `lipgloss.JoinHorizontal` (`plugins/model.go:408-436`) — use the library like every other tab.

**3.4 Expand `internal/tui/theme` into a real theme + zone registry.** `theme.Accent` exists but the pink `"205"` is hardcoded ~12× and `"240"/"39"/"252"/"220"/"196"` have no names (Cluster G: styles redefined in `entries/view_helpers.go:13-27`, `review/view_helpers.go:9-17`, `plugins/view_helpers.go:26-39`). Define named colors + shared styles (`Selected`, `Dim`, `Pane`, `ActionBtn`, …) once. Add `internal/tui/zones` with typed zone-ID constructors (replacing raw string literals like `"action:new"`, `"tab:review"`, `"form:save"` and the four per-tab `fmt.Sprintf("<prefix>:%d", i)` helpers — Cluster D), plus one `Button(zm, zone, label)` helper (Cluster I). This is the proper resurrection of the deleted `mouse/zones.go`.

**3.5 Merge the two dropdown wrappers.** `entries/dropdown.go` and `form/dropdown.go` share byte-identical `filterKinds`/`displayKind` (~25 lines) and two incompatible overlay-positioning solutions (`entries/view_helpers.go:62-66` vs `form/dropdown.go:160-183` with hardcoded `panelTop/panelLeft`). Extract `internal/tui/components/kinddropdown` with config for the "All" prefix vs "+ New kind…" sentinel, and one positioning mechanism driven by layout constants from 3.3.

**3.6 Keymaps + help: `bubbles/key` + `bubbles/help`.** Replace stringly-typed `switch msg.String()` in all four tabs (`entries/model.go:193`, `review/model.go:99`, `plugins/model.go:179`, `form/model.go:275`) with `key.Binding` keymaps per tab, and replace hand-built action-bar strings (`entries/view_helpers.go:230-242`, `review/view_helpers.go:138-148`, `plugins/view_helpers.go:214-236`) with a `help.Model` footer fed from the active tab's keymap (keep bubblezone marks on the rendered hints for clickability). Single source of truth: keys, hints, and README keyboard reference all derive from the keymaps.

**3.7 Rebuild the form on `charm.land/huh/v2` + `bubbles/textarea`.** `form/model.go` is 456 lines of manual focus ring (:283-286, :333-423), tab-order index math (:79-86), and field-box sizing (`form/view_helpers.go:28-63`). huh provides focus management, validation, `Select` (kind), `Input` (name/tags), `Text` (body — fixing the real bug that body is a single-line `textinput` with CharLimit 5000, `form/model.go:106-108`). Embed `huh.Form` as a Bubble Tea model inside the form tab; watch for `huh.StateCompleted`; theme it from 3.4. The dynamic key/value field rows are the one part huh handles awkwardly — keep a small custom add/remove-row group if a form rebuild per row is too clunky, but on the shared components. Add validation: non-empty name, name-conflict check (typed error from 2.3).

**3.8 Spinner + async states.** Plugins tab shows static "Fetching registry…"/"Working…" strings during HTTP (`plugins/view_helpers.go:73,127`). Use `bubbles/spinner`. Apply to any store op that could be slow (approve-all on big queues).

**3.9 Review tab: remove double-rendered buttons.** Approve/reject/edit render twice — as plain text inside `formatProposalDetail` (`review/model.go:290`) and as zone-marked buttons (`review/view_helpers.go:126-131`). Keep only the zone-marked ones (via 3.4's Button helper).

**Gate:** `make test && make test-integration` green; `go vet ./...`; grep-verify: no `msg.String()` key switches remain in tabs, `"205"` appears only in theme, `func truncate` appears once, `listW :=` split math appears once. Line-count sanity: `internal/tui` non-test LOC should drop noticeably despite new packages. Regenerate screenshots.

---

## Phase 4 — MCP server refactor + protocol features (1–2 days, partly folded into 1.2)

**4.1 Shared proposal handler.** `add_entry` (:117-178) and `update_entry` (:181-236) are near-identical including a byte-identical result block (:170-176 vs :228-233). Extract one `proposeEntry(st, spec, targetID)` returning the queued-proposal result.

**4.2 Shared executable-tool scan.** The `st.List("tool", nil)` → `tools.IsExecutable` → `tools.ParseDef` loop is repeated **four times** (`server.go:243-259`, :494-503, :527-535; `setup/go.go:37-40`). Add `tools.ListDefs(st) ([]tools.Def, error)` in `internal/tools` and use it everywhere.

**4.3 Shared output rendering.** Each tool hand-rolls `strings.Builder` formatting (`formatEntries` :611, `formatRegistry` :626, plus inline builders in five handlers). Consolidate into a small `internal/mcp/render.go`. While there: return **structured content** alongside text where the official SDK supports it (typed result structs) so clients get machine-readable results.

**4.4 Expose entries as MCP resources.** Register resources `naitv://entry/{id}` (and a `naitv://bundle` resource for the init document) alongside the tools. Tools remain the compatibility floor (universal client support); resources add @-mention/browsing surfaces in Claude Desktop/Code, VS Code, Cursor. Emit `list_changed` notifications on approval/rejection.

**4.5 Add MCP prompts.** One or two prompts, e.g. `load-context` ("call initialize, then fetch on-demand entries relevant to: {task}") and `propose-learning` ("summarize what you learned this session and propose entries"). Cheap, supported by Claude Desktop/Code and Cursor as slash commands.

**4.6 Dynamic tool hot-reload.** Today dynamic tools are registered once at startup; `install_plugin` output literally tells users to restart the server (`server.go:331-334`). With the official SDK, re-register tools when a `kind=tool` proposal is approved or a plugin is installed/uninstalled, and send `tools/list_changed`. Requires a store-change signal: simplest is a `Store.OnChange(func())` hook invoked by Approve/Delete/plugin ops (the TUI and server run in separate processes — so also add a lightweight refresh: re-scan defs on `tools/list` requests or on a debounced file-mtime check of the DB).

**4.7 Executable-tool safety polish.** `tools.Run` shells out `sh -c` with full env (`executor.go:39-40`), gated only by human approval. Add: per-tool `env_allowlist` field (default: minimal PATH/HOME), and include the exact command line in the review-tab detail so the human sees precisely what they're approving (the executable-tool warning already exists at `review/model.go:234-239` — extend it).

**4.8 Consolidate path/URL helpers.** Three `~`-expansion implementations (`tools/executor.go:125`, `plugin/loader.go:61`, `setup/go.go:144`) → one `internal/xpath` (or `internal/fsutil`) helper. Two URL-detection implementations (`plugin/loader.go:26` vs `plugin/registry.go:62`) → one.

**4.9 Plugin robustness.** `installer.go` links plugin↔entries via a comma-joined `entry_names` string (:103) and uninstalls by name match (:127-158) — breaks on commas/renames. Store the linkage as a JSON array of entry **IDs** in the tracker entry's fields; migrate existing trackers opportunistically on read. Add manifest validation (require version, validate entry specs) in `loader.go`.

**Gate:** new `internal/mcp/server_test.go` using the SDK's in-memory transport: initialize handshake, each tool happy-path + error-path, resource read, prompt get, tool list_changed after simulated approval. `make test` green.

---

## Phase 5 — Features & QoL (prioritized backlog, ~1 week)

Ordered by leverage. Each is a self-contained work item.

**5.1 Markdown rendering for entry bodies (`charm.land/glamour/v2`).** Render entry body markdown in the entries detail pane and review detail pane, cached per entry (glamour is not per-frame cheap). Toggle raw/rendered with a key.

**5.2 `export` / `import` CLI + tools.** Wire 2.8: `naitv-mcp export [--out file.json]`, `naitv-mcp import file.json [--replace]`; plus an `export_entries` MCP tool. This is backup, sync-between-machines, and sharing in one feature.

**5.3 Real diffs in Review (`github.com/aymanbagabas/go-udiff`).** The review tab shows update proposals; render a proper unified diff of body/fields between target entry and proposal, styled with theme colors (green/red backgrounds). go-udiff produces hunks you style with lipgloss.

**5.4 Edit body in `$EDITOR`.** From form/review, `ctrl+e` opens the body in `$EDITOR` via `tea.ExecProcess` (v2), reloads on exit. Removes the biggest friction of long bodies in a TUI.

**5.5 Undo + history view.** Building on 2.7: `u` in the entries tab undoes the last destructive action (delete/approve/toggle); an entry-detail keybinding opens a history list with restore. Status-bar hint after each destructive action ("deleted X — u to undo").

**5.6 Archive tab / filter.** Soft-deleted entries (2.7) visible under a status filter with restore/purge actions.

**5.7 `doctor` command.** `naitv-mcp doctor`: checks DB path/permissions, runs `PRAGMA integrity_check`, verifies FTS index sync (rebuild option), counts orphaned proposals (`target_id` pointing at missing entries), validates executable-tool defs, prints MCP client config snippets for Cursor/Claude Code/Claude Desktop.

**5.8 Streamable HTTP transport (optional).** `naitv-mcp serve --http :8321` using the official SDK's streamable HTTP handler, for clients/machines where stdio is awkward. Localhost-only default, bearer-token flag.

**5.9 Usage telemetry for entries (local only).** Record `last_accessed_at`/`access_count` on `get_entry`/`search_entries` hits and `initialize` inclusion. Surface in entries detail ("never fetched — consider on-demand?"). Data-driven curation of init-bundle size — this is the core problem of the tool (context budget) and no similar tool does it well.

**5.10 Stale-entry nudges.** In the TUI status bar or a `review`-adjacent section: entries not updated in N days and never accessed → suggest archive. Pairs with 5.9.

**5.11 Kind-scoped init bundle.** `initialize` tool arg `kinds: []string` and `naitv-mcp init --kinds rule,tool` — lets users emit slimmer per-project bundles. `instructions.FilterInit` already groups by kind; add the filter.

**5.12 Shell completions + man pages.** Free with fang (1.3); wire into GoReleaser archives and the Homebrew cask.

---

## Phase 6 — Test & CI hardening (1–2 days, interleave with Phases 3–5)

**6.1 Plugins tab tests — currently zero.** Cover: registry fetch states, install/uninstall flows, custom-install input mode (`plugins/model.go:152-177`), root `handlePluginsRequest` (`model.go:492-512`).

**6.2 Real end-to-end form save.** Every CRUD/review test seeds the store directly and injects `SaveMsg` (`crud_test.go:22,69`). Add a journey that types into the form and saves via `ctrl+s`, asserting the store row. Fix `TestJourney_FormFieldAddRemove` (`crud_test.go:117-135`) — it never adds/removes a field despite its name; make it exercise add/remove and assert the parsed `Fields`.

**6.3 Meaningful mouse tests.** `mouse_test.go` fires clicks at guessed coordinates and asserts only `view != ""`. Rewrite using bubblezone properly: render once, `Scan`, resolve a zone's actual bounds, click inside it, assert the behavioral outcome (selection changed, tab switched).

**6.4 Coverage for known-untested behavior:** group collapse (`buildFlatItems`), delivery toggle (`entries/actions.go:37-48`), copy action, search keymode incl. the possibly-dead `SearchCmd` wiring (`entries/actions.go:59-64` — appears never invoked from the model; fix or delete), pending-count badge, status-message expiry, height-fit regression tests for entries and plugins panes (mirror `review/view_test.go:11-22`).

**6.5 CI upgrades.** Add `go test -race`, a coverage report with a soft floor (e.g. 60%, ratchet up), and `govulncheck`. Keep golangci-lint current.

**Gate:** all green in CI; coverage report generated.

---

## Appendix A — Duplication cluster index (verified file:line, pre-refactor)

| # | Cluster | Locations | Fixed by |
|---|---------|-----------|----------|
| A | 35/65 split math ×5 | entries/model.go:319, entries/view_helpers.go:80, review/model.go:181, review/view_helpers.go:33, plugins/model.go:393 | 3.3 |
| B | j/k navigation ×3 | entries/model.go:194-203, review/model.go:100-109, plugins/model.go:180-189 | 3.2 |
| C | mouse release-gate + row hit-test ×3 | entries/model.go:251-307, review/model.go:130-170, plugins/model.go:202-238 | 1.1 + 3.2 |
| D | zone-ID sprintf helpers ×4 | entries/model.go:512, review/model.go:295, plugins/view_helpers.go:24, form/model.go:454 | 3.4 |
| E | truncate-with-ellipsis ×4 | plugins/view_helpers.go:238, entries/view_helpers.go:157+187, review/view_helpers.go:71 | 3.3 |
| F | filterKinds/displayKind byte-identical ×2 | entries/dropdown.go:20-38, form/dropdown.go:26-43 | 3.5 |
| G | per-package style vars, hardcoded "205" ×12 | entries/view_helpers.go:13-27, review/view_helpers.go:9-17, plugins/view_helpers.go:26-39, model.go:28-30, form/model.go:11-15 | 3.4 |
| H | hand-rolled joinHorizontal | plugins/model.go:408-436 | 3.3 |
| I | inline button rendering | entries/view_helpers.go:231-237, review/view_helpers.go:126-143, plugins/view_helpers.go:222-227 | 3.4 |
| J | root Update 6-line ritual ×10 | model.go:127-244, 283-305 | 3.1 |
| K | add/update_entry handlers | server.go:117-236 | 4.1 |
| L | tool-scan loop ×4 | server.go:243-259, 494-503, 527-535; setup/go.go:37-40 | 4.2 |
| M | Create/CreatePending INSERT ×2 | store.go:331, 416 | 2.6 |
| N | ~-expansion ×3, URL-detect ×2 | executor.go:125, loader.go:61, setup/go.go:144; loader.go:26, registry.go:62 | 4.8 |

## Appendix B — Target dependency set

| Dependency | From | To |
|---|---|---|
| bubbletea/bubbles/lipgloss | v1 | `charm.land/{bubbletea,bubbles,lipgloss}/v2` |
| bubblezone | v1 | `github.com/lrstanley/bubblezone/v2` |
| mark3labs/mcp-go v0.9.0 | — | `github.com/modelcontextprotocol/go-sdk` v1.6.x |
| atotto/clipboard | — | removed (OSC52 via tea.SetClipboard) |
| modernc.org/sqlite v1.34.5 | — | latest v1.5x |
| (new) | — | `charm.land/huh/v2`, `charm.land/glamour/v2`, `charm.land/fang/v2` + `spf13/cobra`, `github.com/aymanbagabas/go-udiff` |
| madicen/bubble-dropdown, bubble-overlay | — | keep; verify/publish v2-compatible versions before 1.1 (they consume bubbletea/lipgloss — this is a **blocking prerequisite** for Phase 1.1) |

## Appendix C — Suggested execution order & sizing

Phase 0 (0.5d) → Phase 1 (2–4d; 1.1 blocked on bubble-dropdown/overlay v2 compat) → Phase 2 (1–2d, parallelizable with Phase 1.1) → Phase 3 (3–5d) → Phase 4 (1–2d, fold 4.1–4.3 into 1.2) → Phase 5 items by priority (5.1–5.5 first) → Phase 6 interleaved throughout, final pass at the end.
