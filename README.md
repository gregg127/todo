# todo tui app

Vibe coded for my personal use.

An interactive task board in the terminal, stored as a Markdown file you can also
hand-edit. Vim-style keys, three sections — TODO, DOING, DONE.

```
make build      # ./todo
make install    # /usr/local/bin/todo
make run        # a throwaway copy of the example board
make test
```

```
todo            # ./todo-database.md, created if missing
todo board.md   # any other board file
todo --help     # the key bindings
```

`make install` puts `todo` on your PATH. It opens the board in whatever
directory you run it from, so each project keeps its own.

A file the parser cannot read whole is refused at startup rather than silently
rewritten.

`go.sum` pins a SHA-256 hash of every dependency's file tree, and `make build`
runs `go mod verify` to check the module cache against it. Treat `go.sum` diffs
as security-relevant: a changed hash for a version that did not change means the
bytes moved under you.
