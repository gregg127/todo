# 04 — Viewport and presentation

## What to build

Make the board survive a list longer than the pane. This is the riskiest area in the PRD:
because DONE is never folded, a long-lived project pushes TODO off the top within weeks, so
viewport behaviour needs to be right in v1.

The app reads its size from `tea.WindowSizeMsg` and keeps a viewport offset. The viewport
follows the cursor automatically so the cursor is never off screen, keeping a 3-row
scrolloff above and below wherever the list is long enough to allow it. A growing DONE
section simply scrolls — nothing is folded, collapsed or truncated away.

Every task occupies exactly one row: a title longer than the available width is truncated
with a trailing `…` rather than wrapped, so `j`/`k` stay predictable.

Styling uses only ANSI indices 1–15 — TODO grey, DOING yellow, DONE dimmed green, cursor
line inverted — so changing the terminal colorscheme restyles the app for free. No truecolor
hex anywhere.

The bottom row is a dimmed hint bar listing the normal-mode keys. Its content becomes
mode-dependent in later slices.

## Acceptance criteria

- [ ] The model reacts to `tea.WindowSizeMsg` and uses the size for width and viewport height
- [ ] The cursor is always within the rendered viewport
- [ ] Three rows of context are kept above and below the cursor when the list allows it
- [ ] A long list scrolls; no section is folded or truncated away
- [ ] A title wider than the pane renders on one row, truncated with `…`
- [ ] Only ANSI colors 1–15 are used; no hex colors appear in the codebase
- [ ] The bottom row is a dimmed hint bar showing the normal-mode keys

## Blocked by

- 03 — Cursor movement
