# todo tui app

Vibe coded for my personal use.

An interactive task board in the terminal, stored as a Markdown file you can also
hand-edit. Vim-style keys, three sections — TODO, DOING, DONE.

## Install

Needs Go 1.25+ and `make`.

```
make build           # ./todo
sudo make install    # /usr/local/bin/todo

PREFIX=~/.local/bin make install    # no sudo, if it is on your PATH
```

Build and install are separate because `sudo make build` runs go as root: no go
on sudo's `PATH`, and root-owned files if there is. `sudo make uninstall`
removes it again.

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

## Docs

- [overview.md](docs/overview.md) — the design and the file format
- [security-review.md](docs/security-review.md) — `go.sum` and dependency notes
