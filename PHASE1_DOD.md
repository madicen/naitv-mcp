# Phase 1 Definition of Done (mapped to Approach A)

The original prompt DoD assumed a `plugins/` folder scanner.  
`PHASE1_CORRECTED.md` / `00_READ_ME_FIRST_ARCHITECTURE_CORRECTION.md` replace that with
**approved `kind=tool` entries** + optional `io_mode=json`. This checklist maps each
original item to what “done” means under that correction.

| Original criterion | Approach A meaning | Status |
|--------------------|--------------------|--------|
| New plugin can be placed in `plugins/` folder | Script lives anywhere; you add a `kind=tool` entry with `exec` (+ `io_mode=json`). Folder auto-scan deferred (Phase 4+). | ✅ by design (entry-based) |
| Plugin is immediately discoverable without code changes | After Review approval (or `store.Create` in tests), hot-reload registers the MCP tool — no core code change. | ✅ |
| Tool calls route correctly to subprocess | `tools.Run` / `RunJSON` via `sh -c` | ✅ |
| JSON I/O works end-to-end | Unit + MCP `TestServer_JSONIOToolEndToEnd` | ✅ |
| No breaking changes to existing plugins | Text mode default; existing `exec` tools unchanged | ✅ |
| Tests pass | `go test ./...` | ✅ |
| Documentation complete | `docs/TOOLS.md`, README pointer, templates in docs | ✅ |
| Commit pushed to `main` | Your land/merge of `feat/phase1-json-tool-io` | ⬜ you own this |

## Notes (original) → status

| Note | Status |
|------|--------|
| Keep naitv-mcp generic (routing only) | ✅ |
| Plugin logic outside core | ✅ (scripts + entries; no domain plugins in core) |
| Model-friendly errors (JSON when possible) | ✅ JSON-mode failures return `{"ok":false,...}` |
| `< 200ms` overhead per tool call | ✅ `TestRunJSON_OverheadUnder200ms` (warmed trivial script) |
| Start simple | ✅ |

## Explicitly out of corrected Phase 1

- Folder scanner under `plugins/` or `~/.config/naitv-mcp/tools/`
- `plugin-manager` bootstrap plugin (Phase 1 matrix leftover → first plugins-repo task)
- Push/merge to `main` (operator step)
