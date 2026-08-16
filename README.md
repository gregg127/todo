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
