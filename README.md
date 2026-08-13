# todo

Vibe coded for my personal use.

An interactive task board in the terminal, stored as a Markdown file you can also
hand-edit. Vim-style keys, three sections — TODO, DOING, DONE.

```
make build      # ./todo
make install    # /usr/local/bin/todo
make test
```

```
todo            # ./todo-database.md, created if missing
todo board.md   # any other board file
todo --help     # the key bindings
```

The board file is the whole data model: `internal/board` knows how to read and write it
and nothing else, `internal/tui` drives it. A file the parser cannot read whole is
refused at startup rather than silently rewritten.
