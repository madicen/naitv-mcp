.PHONY: build install test test-store test-integration demo-fixtures demo-build vhs screenshot-delivery clean

build:
	go build -o bin/naitv-mcp ./cmd/naitv-mcp

install:
	go install ./cmd/naitv-mcp

test:
	go test ./...

test-store:
	go test -v ./internal/store/...

test-integration:
	go test -v ./integration_tests/...

demo-fixtures:
	mkdir -p tmp/demo
	NAITV_MCP_DEMO_DIR=tmp/demo go run ./cmd/naitv-mcp seed-demo

# The VHS tapes invoke ./naitv-mcp from the repo root, so build it there.
demo-build:
	go build -o naitv-mcp ./cmd/naitv-mcp

vhs: demo-build demo-fixtures
	for tape in vhs/*.tape; do vhs $$tape; done

vhs/%: demo-build demo-fixtures
	vhs vhs/$*.tape

# Regenerate only the delivery-toggle GIF (also included in `make vhs`).
screenshot-delivery: demo-build demo-fixtures
	vhs vhs/delivery-toggle.tape

clean:
	rm -rf bin/ tmp/ screenshots/*.gif naitv-mcp
