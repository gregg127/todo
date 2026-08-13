# 09 — Filtering

## What to build

`/` starts a filter that narrows the list as the user types, so a task can be found in a long
DONE section. Matching is case-insensitive on the task text.

The filter is a way to *reach* a task, not just to look at it: section headers and counts
reflect the filtered view, and `1`/`2`/`3`, `cc` and `dd` all act on the filtered task under
the cursor. The filter persists while the user acts on matches, so several matching tasks can
be cleaned up in a row. Esc clears the filter and returns to the full board.

Filter is a third exclusive mode alongside normal and insert: every printable key is text,
only Enter and Esc are commands. The hint bar shows the live filter query while filtering.

The cursor is an index into the *visible* task list, so with a filter active it indexes the
filtered subset.

There is deliberately no `n`/`N` — with a filtered list they would just duplicate `j`/`k`.

## Acceptance criteria

- [ ] `/` enters filter mode and the list narrows on every keystroke
- [ ] Matching is case-insensitive against the task text
- [ ] Section headers and counts show the number of matches in each section
- [ ] `j`/`k` move within the filtered list only
- [ ] `1`/`2`/`3`, `cc` and `dd` act on the filtered task under the cursor
- [ ] The filter stays active after acting on a match
- [ ] Esc clears the filter and restores the full board
- [ ] The hint bar shows the live filter query while in filter mode
- [ ] `n` and `N` are not bound

## Blocked by

- 07 — Adding, editing and deleting tasks
