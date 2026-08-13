# 06 — Reordering within a section

## What to build

`J` and `K` move the current task down and up **within its section**, so priority is
expressed by position rather than by a priority field. The top of TODO is "what's next".

`J`/`K` swap the task with the adjacent task of the same status. At a section boundary they
do nothing — reordering must never change a task's status by accident. This preserves the
mnemonic established by the rest of the board: no shift means the cursor moves, shift means
the task moves.

The new order is written to the file in the same line order, so the file and the screen
always agree.

## Acceptance criteria

- [ ] `J` moves the current task one position down within its section
- [ ] `K` moves it one position up within its section
- [ ] The cursor stays on the moved task
- [ ] `J` on the last task of a section is a no-op; `K` on the first is a no-op
- [ ] No `J`/`K` press ever changes a task's status
- [ ] The file on disk reflects the new order immediately, in the same line order as the screen

## Blocked by

- 05 — Status changes and atomic persistence
