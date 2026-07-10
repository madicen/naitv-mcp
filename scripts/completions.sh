#!/usr/bin/env bash
set -euo pipefail
mkdir -p completions
go run ./cmd/naitv-mcp completion bash > completions/naitv-mcp.bash
go run ./cmd/naitv-mcp completion zsh > completions/naitv-mcp.zsh
go run ./cmd/naitv-mcp completion fish > completions/naitv-mcp.fish
