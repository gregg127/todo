BINARY := todo
PREFIX ?= /usr/local/bin
SCRATCH := scratch-board.md

.PHONY: build install uninstall run test clean verify provenance

# go.sum is only checked on download; this re-hashes the cache on disk.
verify:
	go mod verify

# -trimpath drops local paths, so the same commit builds byte-identically.
build: verify
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o $(BINARY) ./cmd/todo

# copies only: sudo's PATH has no go, and a root build leaves root-owned files.
install:
	@test -x $(BINARY) || { echo "run 'make build' first"; exit 1; }
	install -d $(PREFIX)
	install -m 0755 $(BINARY) $(PREFIX)/$(BINARY)

uninstall:
	rm -f $(PREFIX)/$(BINARY)

# toolchain, module versions and dependency hashes baked into the binary.
provenance: build
	@go version -m $(BINARY)

# throwaway copy, so a real board is never the thing you experiment on.
run:
	@test -f $(SCRATCH) || cp docs/todo-database-example.md $(SCRATCH)
	go run ./cmd/todo $(SCRATCH)

test:
	go test ./...

clean:
	rm -f $(BINARY) $(SCRATCH)
