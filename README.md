# todo tui app

Vibe coded for my personal use.

An interactive task board in the terminal, stored as a Markdown file you can also
hand-edit. Vim-style keys, three sections — TODO, DOING, DONE.

## Install

Needs Go 1.25.13+ and `make`.

```
make build           # ./todo
sudo make install    # /usr/local/bin/todo

PREFIX=~/.local/bin make install    # no sudo, if it is on your PATH
```

Build and install are separate because `sudo make build` runs go as root: no go
on sudo's `PATH`, and root-owned files if there is. `install` copies `./todo` as
it stands, so build again before installing a change.

`make uninstall` removes it, and wants the same `PREFIX` — under sudo as an
argument, `sudo make uninstall PREFIX=…`, since sudo drops it from the
environment.

## Usage

```
todo            # ./todo-database.md, created if missing
todo board.md   # any other board file
todo --help     # the key bindings
```

`todo` opens the board in whatever directory you run it from, so each project
keeps its own.

A file the parser cannot read whole is refused at startup rather than silently
rewritten.

## Development

```
make run     # a throwaway copy of the example board
make test
```

`go.sum` pins a SHA-256 hash of every dependency's file tree, and `make build`
runs `go mod verify` to check the module cache against it. Treat `go.sum` diffs
as security-relevant: a changed hash for a version that did not change means the
bytes moved under you.

## Docs

- [overview.md](docs/overview.md) — the design and the file format
- [security-review.md](docs/security-review.md) — the threat model, the findings
  and the dependency audit
