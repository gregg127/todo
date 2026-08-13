# 02 — Markdown store: read an existing board

## What to build

Teach the app to read `./todo-database.md` from the directory it was launched in and render
its tasks on the board.

The store is two pure functions, `Parse(string) → Board` and `Render(Board) → string`, with
`Load(path)` wrapping `Parse`. The board itself is an ordered slice of tasks, each with a
title and a status; section membership is derived by filtering on status, and order within a
section is the order in the slice. No ids, no priorities, no timestamps.

Recognised structure is `## TODO`, `## DOING`, `## DONE` headings (exact, uppercase) and
list items matching `- [ ] text` or `- [x] text` beneath them. **The section wins over the
checkbox**: a `- [x]` under `## TODO` parses as a TODO task and re-serializes as `- [ ]`.
Items before any recognised heading, prose, blank lines and other headings are dropped on
the next save. Round-tripping is normalizing, not byte-preserving.

`Load` of a nonexistent path yields an empty board and no error. `Render` always emits all
three sections in order, one blank line between them, trailing newline at end of file.

Rendered tasks appear one per line, prefixed by a status dot (`○` TODO, `◐` DOING, `●`
DONE), and each section header shows its count like `TODO (5)`.

No writing yet — this slice only reads.

## Acceptance criteria

- [ ] A todo file in the working directory is loaded and its tasks appear on the board
- [ ] Each task line is prefixed by `○`, `◐` or `●` matching its section
- [ ] Section headers show live counts, e.g. `TODO (5)`
- [ ] A missing todo file yields an empty board with no error
- [ ] An empty file parses to an empty board
- [ ] `- [x]` under `## TODO` parses as a TODO task and renders as `- [ ]`
- [ ] Prose, blank lines, unknown headings and pre-heading items are dropped by `Render`
- [ ] `Render` emits all three headings in order with a trailing newline
- [ ] Direct round-trip tests assert `Render(Parse(s)) == Render(Parse(Render(Parse(s))))`
- [ ] Round-trip tests cover: `[x]` under `## TODO`, prose between sections, a truncated
      file, an empty file, CRLF line endings, and a task title containing `- [ ]`

## Blocked by

- 01 — Walking skeleton
