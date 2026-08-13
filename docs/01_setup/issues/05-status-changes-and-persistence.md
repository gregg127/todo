# 05 — Status changes and atomic persistence

## What to build

The first mutation, and with it the whole write path.

`1`, `2` and `3` move the task under the cursor to TODO, DOING and DONE. The moved task
lands at the **top** of the target section, so DONE leads with what was finished most
recently. The cursor follows the moved task to its new position, so the user can
immediately act on it again (`3` then later `dd`). Pressing the number of the section a task
is already in is a no-op: it must not re-insert the task at the top of its own section.
The status dot and the section counts update immediately.

Every mutation is written to disk immediately, using an atomic write: create a temp file in
the same directory, write, `fsync`, `rename` over the target. Same-directory temp guarantees
the rename is atomic. After each save the app records the resulting mtime, for the watcher
added later.

`Load` of a missing file already yields an empty board; the **first** `Save`, triggered by
the first mutation, creates the file. Launching and quitting without a mutation still writes
nothing. `q` quits immediately — everything is already on disk, so there is no unsaved-changes
concept.

Concurrent-write races with an editor are accepted and unmitigated: last write wins, no merge.

## Acceptance criteria

- [ ] `1`/`2`/`3` move the task under the cursor to TODO/DOING/DONE
- [ ] The moved task lands at the top of the target section
- [ ] The cursor follows the moved task
- [ ] Pressing the number of the task's current section changes nothing at all
- [ ] Status dot and both affected section counts update immediately
- [ ] The file on disk matches the board after every status change
- [ ] Writes go through a same-directory temp file, fsync and rename
- [ ] The first mutation in a directory with no todo file creates the file
- [ ] The app records the mtime resulting from each of its own saves
- [ ] `q` exits immediately with everything already persisted

## Blocked by

- 04 — Viewport and presentation
