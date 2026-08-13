BINARY := todo
PREFIX := /usr/local/bin

.PHONY: build install test clean

build:
	CGO_ENABLED=0 go build -ldflags "-s -w" -o $(BINARY) ./cmd/todo

install: build
	install -m 0755 $(BINARY) $(PREFIX)/$(BINARY)

test:
	go test ./...

clean:
	rm -f $(BINARY)
