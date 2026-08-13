# 01 — Walking skeleton: binary, install, empty board, `--help`

## What to build

The first end-to-end path: a Go module producing a single static binary called `todo`,
installable with `make install`, that launches a Bubble Tea TUI showing an empty board and
quits on `q`.

The board renders the three section headings in fixed TODO / DOING / DONE order with zero
counts. Nothing is read from or written to disk yet — launching and quitting in any
directory must leave that directory untouched.

`todo --help` prints a short usage line plus the keybinding table and exits 0 without
starting the TUI.

This slice also establishes the test seam that every later slice uses: a model constructed
against a `t.TempDir()`, a helper that feeds a sequence of `tea.KeyMsg` values through
`Update`, and assertions on the string returned by `View()` and on the bytes of
`todo-database.md`. No `tea.Program`, no terminal, no goroutines.

## Acceptance criteria

- [ ] `make build` produces a static binary; `make install` places it at `/usr/local/bin/todo`
- [ ] `make test` runs the test suite; `make clean` removes build output
- [ ] `todo` launches into the TUI and starts instantly
- [ ] `View()` on a fresh model contains `TODO (0)`, `DOING (0)` and `DONE (0)` in that order
- [ ] Empty sections still render their headers
- [ ] `q` in normal mode exits immediately
- [ ] Running and quitting in a directory with no todo file creates no files at all
- [ ] `todo --help` prints usage plus the keybinding table and exits 0
- [ ] A `send(m, keys...)` test helper exists and drives `Update` directly

## Blocked by

None — can start immediately.
