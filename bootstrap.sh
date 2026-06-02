#!/usr/bin/env bash
# Run this once from the repo root to pull all dependencies.
set -euo pipefail

go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/bubbles
go get github.com/charmbracelet/lipgloss
go get github.com/lrstanley/bubblezone
go get github.com/madicen/bubble-overlay
go get modernc.org/sqlite
go get github.com/oklog/ulid/v2
go get github.com/mark3labs/mcp-go
go get github.com/muesli/termenv

go mod tidy

echo "Building..."
go build ./...

echo "Testing store..."
go test -v ./internal/store/...

echo "Done."
