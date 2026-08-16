BINARY := todo
PREFIX := /usr/local/bin
SCRATCH := scratch-board.md

.PHONY: build install run test clean verify provenance

# go.sum is checked when a module is downloaded, not when it is reused from the
# module cache. go mod verify re-hashes what is actually on disk, which is the
# only thing that catches a dependency edited in place after download.
verify:
	go mod verify

# -trimpath keeps local filesystem paths out of the binary, so the same commit
# builds byte-identically on another machine.
build: verify
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o $(BINARY) ./cmd/todo

install: build
	install -m 0755 $(BINARY) $(PREFIX)/$(BINARY)
	@go version -m $(PREFIX)/$(BINARY) | head -3

# what went into the binary: toolchain, module versions, every dependency hash
provenance: build
	@go version -m $(BINARY)

# run against a throwaway copy of the example board, so a real board is never
# the thing you experiment on. The copy survives between runs; clean drops it.
run:
	@test -f $(SCRATCH) || cp docs/todo-database-example.md $(SCRATCH)
	go run ./cmd/todo $(SCRATCH)

test:
	go test ./...

clean:
	rm -f $(BINARY) $(SCRATCH)
