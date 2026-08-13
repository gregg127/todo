# 03 — Cursor movement

## What to build

A cursor that moves over the board with vim motions. The cursor is an index into the visible
task list and is marked with `▸` in a left gutter, in addition to whatever line styling the
board uses.

`j` and `k` move down and up and flow continuously across section boundaries, so the whole
board behaves like one list. `j` on the last task and `k` on the first task do nothing — the
cursor never wraps.

`gg` jumps to the first task and `G` to the last. `gg` is handled by a single pending-prefix
field on the model, with **no timeout**: a pending `g` stays pending until the next key,
which either completes the sequence or cancels it (the cancelling key is discarded, not
re-dispatched). This same field is reused by `dd` and `cc` in a later slice.

## Acceptance criteria

- [ ] `j` moves the cursor down, `k` up, across section boundaries
- [ ] `j` on the last task is a no-op; `k` on the first task is a no-op
- [ ] `gg` jumps to the first task, `G` to the last
- [ ] A pending `g` followed by an unrelated key cancels the sequence and swallows that key
- [ ] A pending prefix survives indefinitely with no timer
- [ ] The cursor line is marked with `▸` in a left gutter
- [ ] Cursor movement on an empty board is harmless

## Blocked by

- 02 — Markdown store: read an existing board
