BINARY := todo
PREFIX := /usr/local/bin
SCRATCH := scratch-board.md

.PHONY: build install run test clean

build:
	CGO_ENABLED=0 go build -ldflags "-s -w" -o $(BINARY) ./cmd/todo

install: build
	install -m 0755 $(BINARY) $(PREFIX)/$(BINARY)

# run against a throwaway copy of the example board, so a real board is never
# the thing you experiment on. The copy survives between runs; clean drops it.
run:
	@test -f $(SCRATCH) || cp docs/todo-database-example.md $(SCRATCH)
	go run ./cmd/todo $(SCRATCH)

test:
	go test ./...

clean:
	rm -f $(BINARY) $(SCRATCH)
