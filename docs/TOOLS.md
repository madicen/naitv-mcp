# Executable tools

naitv-mcp turns approved `kind=tool` entries with an `exec` field into live MCP
tools. Agents propose them; you approve them in the Review tab; the server
hot-reloads without a restart.

## Entry fields

| Field | Required | Meaning |
|-------|----------|---------|
| `exec` | yes | Shell command template (`sh -c`). Supports `{param}` placeholders. |
| `working_dir` | no | CWD for the process (`~` expanded). |
| `timeout` | no | Go duration string (default `30s`). |
| `io_mode` | no | `text` (default) or `json`. Alias: `ioMode`. |
| `params` | no | JSON array of `{name,description,required}` for MCP schema / `{param}` fill. |
| `disabled` | no | `true` / `1` / `yes` to keep registered but refuse runs. |
| `env_allowlist` | no | Comma-separated env keys passed to the child (default: `PATH,HOME`). |

Entry `name` → MCP tool name (sanitised). Entry `body` → MCP description.

### Text mode (`io_mode=text`, default)

Stdout/stderr are returned as formatted text. Non-zero exit is not treated as a
hard MCP error (so `go test` failures stay usable). Placeholders in `exec` are
filled from string params.

### JSON mode (`io_mode=json`)

For external scripts (Python, Go, Shell, Node) that speak a fixed I/O contract:

```
Stdin:  {"key":"value",...}\n     # one JSON object line
Stdout: {"result":...}\n          # one JSON value (object/array/…)
Stderr: free-form diagnostics
Exit:   0 = success, non-zero = error
```

On success the MCP tool returns **only** the JSON stdout (no timing footer).
On failure (timeout, spawn error, non-zero exit, or invalid JSON stdout) the
result is marked as an error and includes stderr when present.

Example entry fields:

```json
{
  "exec": "python3 ~/.config/naitv-mcp/tools/structural-anchor.py",
  "io_mode": "json",
  "timeout": "30s",
  "params": "[{\"name\":\"root_path\",\"description\":\"Path to analyze\",\"required\":true}]"
}
```

## Language templates

### Python

```python
#!/usr/bin/env python3
import json, sys

try:
    data = json.loads(input())
    # ... work ...
    print(json.dumps({"status": "ok", "data": data}))
except Exception as e:
    print(str(e), file=sys.stderr)
    sys.exit(1)
```

### Go

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"
)

func main() {
    var in map[string]any
    if err := json.NewDecoder(os.Stdin).Decode(&in); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
    _ = json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "ok", "data": in})
}
```

### Shell (needs `jq`)

```bash
#!/bin/bash
set -euo pipefail
read -r input_json
root=$(echo "$input_json" | jq -r '.root_path')
jq -n --arg root "$root" '{status:"ok", root:$root}'
```

## Testing a script locally

```bash
echo '{"root_path":"."}' | python3 tool.py | jq
```

## When to use which mode

- **text** — shell wrappers, compilers, test runners, anything that already
  prints human/log text.
- **json** — structured plugins that return objects for the model to parse
  (Phase 2+ ecosystem tools).

See also `PHASE1_CORRECTED.md` and `PLUGIN_TEMPLATE_CORRECTED.md`.
