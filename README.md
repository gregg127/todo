# todo tui app

An interactive task board in the terminal, stored as a Markdown file. 
Vim-style keys, three sections - **TODO**, **DOING**, **DONE**.

## Install / uninstall

Needs Go 1.25.13+ and `make`.


### With sudo

```
make build
sudo make install      # /usr/local/bin/todo
sudo make uninstall
```

### Without sudo

Use any directory you own that is on your `PATH`.

```
make build
PREFIX=~/.local/bin make install
PREFIX=~/.local/bin make uninstall
```


## Usage

```
todo                   # ./todo-database.md, created if missing
todo board.md     # any other board file
todo --help         # the key bindings
```

`todo` opens the board in whatever directory you run it from, so each project keeps its own.

A file the parser cannot read whole is refused at startup rather than silently rewritten.

## Development

```
make run     # a throwaway copy of the example board
make test
```

## Docs

- [overview.md](docs/overview.md) - the design and the file format
- [security-review.md](docs/security-review.md) - the threat model, the findings and the dependency audit

## License

[MIT](LICENSE) - use it, change it, ship it, sell it. Just keep the copyright notice.
