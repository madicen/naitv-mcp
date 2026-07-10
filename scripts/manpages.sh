#!/usr/bin/env bash
set -euo pipefail
mkdir -p manpages
go run ./cmd/naitv-mcp man > manpages/naitv-mcp.1
gzip -nf manpages/naitv-mcp.1
