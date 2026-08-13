# 07 — Adding, editing and deleting tasks

## What to build

Task authoring, without ever leaving the pane.

`o` opens a one-line input for a new task below the cursor; `O` does the same above it, so an
urgent task reaches the top of TODO in one step. A new task is created in the section the
cursor is in, so `o` in DOING starts something rather than queueing it. On an empty board,
`o`/`O` create the first TODO task — no special path needed for a fresh project.

Enter confirms the input, Esc cancels it and leaves the board untouched. Confirming empty or
whitespace-only text creates no task.

`cc` edits the current task with its existing text prefilled. `dd` deletes the current task
with no confirmation prompt — undo (a later slice) is what makes that safe. After a delete
the index is kept and clamped to the last visible task, so the cursor lands on whatever took
the deleted task's row and `dd dd dd` prunes a run; deleting the last task on the board
leaves the cursor at index 0 on an empty board.

Modes are exclusive: normal and insert. In insert mode every printable key is text — only
Enter and Esc are commands — so a task can be called "quit the job". `dd` and `cc` reuse the
pending-prefix field introduced with `gg`. The hint bar in insert mode reads
"enter confirm · esc cancel".

Every one of these mutations is saved immediately through the existing atomic write.

## Acceptance criteria

- [ ] `o` opens an input and creates the task below the cursor, in the cursor's section
- [ ] `O` creates it above the cursor, in the cursor's section
- [ ] `o`/`O` on an empty board create the first TODO task
- [ ] Enter confirms; Esc cancels and leaves the board and file untouched
- [ ] Empty or whitespace-only input creates no task
- [ ] `cc` opens the input prefilled with the current task's text and replaces it on confirm
- [ ] `dd` deletes immediately with no confirmation
- [ ] After a delete the cursor stays on the same row index, clamped to the last task
- [ ] Deleting the last task leaves an empty board with the cursor at index 0
- [ ] In insert mode `q`, `j`, `1` etc. are typed as literal characters
- [ ] The hint bar changes to the insert-mode hints while the input is open
- [ ] Every add, edit and delete is on disk immediately

## Blocked by

- 05 — Status changes and atomic persistence
