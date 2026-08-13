# 10 — External reload after a hand-edit

## What to build

Let the user bulk-edit `todo-database.md` by hand in vim in another pane and have the app
pick the change up within about a second.

A Bubble Tea ticker command fires roughly every second and emits a message when the file's
mtime differs from the last mtime the app itself recorded after its own save. The app's own
writes must therefore never trigger a reload — getting that guard wrong in one direction
makes the cursor jump on every keypress, and in the other direction silently eats vim edits.
This is the second riskiest area in the PRD after scrolling.

On reload the board is re-parsed from disk. The app tries to keep the cursor on the task with
the same title; failing that, it clamps the old index into the new list. An external reload
**clears the undo stack**: those snapshots describe a state the file no longer has, and
restoring one would silently clobber the vim edit that just arrived.

Tests drive this through the same seam as everything else: write the file directly, send the
reload message, assert the new `View()`. The ticker itself is not tested.

## Acceptance criteria

- [ ] A ticker command fires about once per second
- [ ] A reload message is emitted only when the observed mtime differs from the recorded one
- [ ] A sequence of ordinary key presses never triggers a reload and never moves the cursor
      unexpectedly
- [ ] Editing the file externally and sending the reload message updates the board within one tick
- [ ] After a reload the cursor stays on the task with the same title
- [ ] If that task is gone, the old index is clamped into the new list
- [ ] A reload empties the undo stack, so `u` immediately afterwards does nothing
- [ ] A hand-ticked `- [x]` under `## TODO` reloads as a TODO task and is normalized on the
      next save

## Blocked by

- 08 — Undo
- 09 — Filtering
