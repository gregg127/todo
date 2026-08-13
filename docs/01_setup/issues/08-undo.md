# 08 — Undo

## What to build

`u` undoes the last change. This is what makes `dd` safe without a confirmation dialog.

Before every mutation the current task slice is deep-copied and pushed onto an in-memory
stack; `u` pops and restores. Undo covers every mutation: `1`/`2`/`3`, `J`/`K`, `o`/`O`,
`cc` and `dd`. Repeated `u` walks back through several changes. `u` on an empty stack does
nothing visible, so hammering it is harmless.

There is no redo, no persistence and no depth cap — the data is a few dozen short strings
and the process is restarted often. The undo history is discarded on quit; nothing extra is
stored on disk.

Undo restores tasks only. The cursor is clamped into the restored list and the filter query,
once it exists, is left as-is. An undo is written to disk like any other change, so the file
always reflects what is on screen.

## Acceptance criteria

- [ ] `u` reverses a `1`/`2`/`3` status change
- [ ] `u` reverses `J`/`K`, `o`, `O`, `cc` and `dd`
- [ ] Repeated `u` walks back through several consecutive changes
- [ ] `u` on an empty undo stack changes nothing on screen or on disk
- [ ] A no-op key press (e.g. `2` on a task already in DOING) pushes no snapshot
- [ ] Every undo is written to disk immediately
- [ ] The cursor is clamped into the restored list
- [ ] Nothing about the undo stack is written to disk

## Blocked by

- 06 — Reordering within a section
- 07 — Adding, editing and deleting tasks
