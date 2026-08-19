# What this is

A personal task board in the terminal, kept in a Markdown file you can also
hand-edit. Three fixed sections — TODO, DOING, DONE — vim keys, no config, no
database, no sync. Written from the code as it stands; the README has the
usage and `todo --help` has the key bindings.

## The idea

A todo list is worth having only if editing it is cheaper than ignoring it. So
the board is one file per directory, plain Markdown, and every keystroke that
changes something saves it — there is no save key and no session to lose. The
file is as good a UI as the app: open it in an editor and the board reloads
within a second.

Everything else follows from that. No priorities, no due dates, no tags, no
projects — a task is a line of text in one of three sections, and where it sits
in that section is where you put it.

## The file

`## TODO` / `## DOING` / `## DONE` headings, and `- [ ] ` / `- [x] ` items
beneath them. That is the whole grammar. The section wins over the checkbox, so
a `- [x]` under `## TODO` is a TODO task and gets its box rewritten on the next
save. Order within a section is the order of the lines.

A file may also open with a `---` fenced block of `key: value` lines. One key
is read, `collapsed-done`, which is where the DONE section's fold outlives the
session; the block is written only once you have toggled the fold, so a board
nobody has folded stays plain Markdown, and a key the app does not know is
written back untouched.

Because a save rewrites the file from the parsed board, anything the parser
does not understand would be dropped. Rather than eat your notes, the app
refuses to open a file it cannot read whole and names the first bad line, and
stops saving if a later hand-edit makes it unreadable. A control character in a
title counts as unreadable — a title goes to the terminal as it is — though a
paste is stripped of them rather than refused, since nothing is on disk to
damage yet. A missing file is not an error — it is created empty. Saves are
atomic (temp file, fsync, rename) and the app tracks the mtime of its own last
write, so it can tell your hand-edit from its own and reload only for yours.

## The code

- `cmd/todo` — argument handling and `--help`.
- `internal/board` — the format. `Parse`, `Render`, `Validate`, atomic `Save`.
  No terminal, no state; this is where the tests are cheapest.
- `internal/tui` — a single `Model` and one reducer. Every key is a pure
  `Model → Model` step, which is what makes the behaviour testable without a
  terminal. `watch.go` is the single half-second tick, which both polls the
  file's mtime and flips the input caret's blink. `view.go` owns everything
  screen-shaped: wrapping, scrolling, and the row-to-task map a click reads.

Two things about the model are worth knowing before changing it: the cursor
indexes the *visible* list, not the task slice, so a filter or a collapsed DONE
section changes what it means; and undo/redo are in-memory snapshots of the
whole board, dropped on an external reload because they describe a file that no
longer exists.

## Deliberately missing

Priorities, due dates, recurring tasks, sub-tasks, tags, multiple boards at
once, colour configuration, sync. The mouse does two things — a left click puts
the cursor on the task you clicked, the wheel scrolls the board without moving
the cursor — and no more: no drag to reorder, no click to tick off. Styling
uses ANSI colours 1–15 only, so the terminal's own colourscheme restyles the
app for free.
