# todo-cli — Design session & specification

Result of a design grilling session on 2026-08-13, before any code was written.
Part 1 is the spec to implement against; part 2 is the decision log explaining why each
choice was made and what cost it accepts.

Context: empty repo (`git init`, no commits). The app targets one pane of a 4-pane tmux
setup; measured pane size is **113x37** characters, which removes any need for
responsive layout. The user works in vim, so vim motions and vim conventions win ties.

---

# Part 1 — Specification (v1)

A minimal, interactive TUI todo app that lives permanently in one tmux pane and is
driven entirely by vim motions. Tasks move between TODO / DOING / DONE and are
persisted as plain Markdown.

## Stack

- **Language:** Go
- **TUI framework:** Bubble Tea (+ lipgloss for styling)
- **Install:** `Makefile` with `make install`, copying the binary to `/usr/local/bin/todo`

Rationale: a single static binary starts instantly and cannot break when Homebrew
updates a runtime — important for a process that stays open for weeks in a tmux pane.

## Data

**Location:** `./todo-database.md` in the current working directory (per project).

**Format:**

```markdown
## TODO
- [ ] fix bug

## DOING
- [ ] refactor

## DONE
- [x] deploy
```

- Line order in the file is the display order — reordering with `J`/`K` is persisted directly.
- Tasks in DONE are written as `[x]`; tasks in TODO and DOING as `[ ]`.
- **Missing file:** the app starts with an empty board and creates the file only when the
  first task is added. Merely opening the app in a directory never writes anything.

**Persistence:**

- Written after every mutation (add / edit / delete / status change / reorder).
- Writes are atomic: temp file + rename.
- A ~1s ticker checks the file's mtime; if it changed underneath, the app reloads it.
  This supports editing `todo-database.md` by hand in vim in another pane.
- Known race: saving in vim in the same second as a keypress in the app — last write wins.

## Layout

Single column, three sections stacked vertically. Target pane size is 113x37.

```
TODO (3)
 ○ napisac apke
▸○ fix bug

DOING (1)
 ◐ refactor

DONE (1)
 ● deploy
```

- Section headers carry a count: `TODO (5)`.
- The dot carries the status, so state is readable even without color:
  `○` TODO, `◐` DOING, `●` DONE.
- Cursor marker: `▸` in the left gutter.
- **Colors:** ANSI 16-color palette only (indices 1–15), so the app inherits the
  terminal/tmux colorscheme and stays visually consistent with vim next to it.
  TODO grey, DOING yellow, DONE dimmed green, cursor line inverted background.
- A dimmed key-hint bar occupies the bottom row.
- DONE is never folded or truncated: it grows, the list scrolls, and the user prunes
  it manually with `dd`.

## Keybindings

| Key | Action |
|---|---|
| `j` / `k` | move cursor, flowing across section boundaries |
| `gg` / `G` | jump to first / last task |
| `1` `2` `3` | move the current task to TODO / DOING / DONE |
| `J` / `K` | move the current task up / down within its section |
| `o` / `O` | new task below / above the cursor |
| `cc` | edit the current task |
| `dd` | delete the current task |
| `u` | undo the last change |
| `/` | filter the list |
| `q` | quit |

Convention: no shift = the cursor moves; shift = the task moves.

**Undo:** a stack of previous list states kept in memory, snapshotted before every
mutation. Covers `dd` as well as accidental `1`/`2`/`3` and `J`/`K`. The stack is
discarded on exit.

**Insert mode:** single-line input. Enter confirms, Esc cancels, empty text creates
no task.

**Search:** `/` filters the list incrementally as you type; Esc clears the filter.
No `n`/`N` — with a filtered list they would duplicate `j`/`k`.

## v1 scope

In scope:

- The layout, keybindings, dots and colors described above
- Bottom key-hint bar
- Per-section counters
- `/` filtering
- Round-trip tests for the parser/serializer (Markdown → model → Markdown), including
  odd lines and an empty file — the only place where a bug costs real data

Out of scope for v1:

- Folding / archiving DONE
- Configuration files, CLI flags other than `--help`
- Multi-line task descriptions, due dates, priorities, tags
- Searching with `n`/`N`

## Assumptions

1. `/` filters incrementally while typing; Esc clears. No `n`/`N`.
2. Insert mode is one line; Enter confirms, Esc cancels, empty input adds nothing.
3. The viewport scrolls to keep ~3 rows of context around the cursor (scrolloff).
4. DONE tasks serialize as `[x]`, TODO and DOING as `[ ]`.
5. Lines the parser does not understand (e.g. hand-written notes in the file) are
   **dropped on save** — the file belongs to the app.
6. No configuration and no CLI flags beyond `--help`.

## Known risks

- **Write race:** vim saving in the same second as a keypress in the app — last write
  wins, no merge.
- **Scrolling is load-bearing:** with no folding for DONE, a growing DONE section will
  push TODO off screen, so viewport scrolling must work well from the first version.
- **Per-directory data:** the pane shows whatever list belongs to the directory it was
  launched from; parallel per-project files are possible.

---

# Part 2 — Decision log

Each entry records the question, the option chosen, the alternatives rejected, and the
cost accepted. Written so a later reader can tell which decisions are load-bearing and
which are cheap to revisit.

## 1. Stack — Go + Bubble Tea

**Rejected:** Python + stdlib curses (zero deps, but painful colors and unicode width,
and a brew Python upgrade can break the shebang); Python + Textual (heaviest dependency
for a "very simple" app); Node + Ink (slowest startup, most boilerplate).

**Cost accepted:** a `go build` step after every code change.

**Why it mattered:** the pane runs for weeks, so startup speed and immunity to runtime
upgrades outweigh edit-and-rerun convenience.

## 2. Layout — one column, three stacked sections

**Rejected:** a 3-column kanban board (fits comfortably at 113 wide, keeps the whole
state in one glance, and makes left/right movement literal); columns with only the
active one expanded.

**Cost accepted:** with >30 tasks the list scrolls and the whole board is no longer
visible at once; "move left/right" loses its literal meaning.

**Gain:** full 113-char width for task titles and one continuous `j`/`k` motion through
everything.

## 3. Status keys — `1` / `2` / `3` jump straight to a section

**Rejected:** `H`/`L` to push a task forward/back through the pipeline (keeps the kanban
mnemonic and pairs with `J`/`K` under one rule: shift = move the task); Space/Backspace.

**Cost accepted:** the numbers have to be memorized, and they carry no directional
meaning. In exchange, moving TODO → DONE is one keypress instead of two.

## 4. Ordering within a section — `J` / `K`

**Rejected:** insertion order only; LIFO (newest on top).

**Why:** gives priorities for ~15 lines of code without introducing a `priority` field
that would then need sorting and rendering. Top of TODO = what's next.

## 5. File format — three headers plus standard checkboxes

**Rejected:** a single list with custom markers (`- [~]` for doing) — non-standard,
renders wrong in Markdown previews, and one typo during hand-editing moves a task to the
wrong section; checkboxes with inline metadata comments (`<!-- id:7 created:… -->`) —
destroys the readability that motivated using Markdown in the first place.

**Consequence:** file line order equals screen order, so `J`/`K` persists naturally.

## 6. File location — `./todo-database.md` in the current directory

**Rejected:** a fixed `~/todo-database.md` (one global list, always the same regardless
of where the app is launched — the obvious fit for a permanently-open pane);
`$TODO_FILE`.

**Cost accepted:** what the pane shows depends on the directory it was launched from, so
parallel per-project files are possible and a task can be "lost" by being in another
project's file.

**Gain:** the list lives with the project and can be committed alongside the code.

## 7. Saving — write on every change, auto-reload on mtime change

**Rejected:** manual reload via `r` (simpler, no ticker, but forgetting `r` after editing
in vim means the app overwrites those edits with its stale in-memory state); save on
quit (a killed pane loses the whole day).

**Cost accepted:** ~25 lines and a theoretical race if vim writes in the same second as a
keypress — last write wins.

## 8. Input — insert mode inside the app

**Rejected:** launching `$EDITOR` on the file (zero input code, full vim power, but
adding one task becomes five steps and blanks the pane); a separate `todo add "…"` CLI
(two interfaces for one app — more code, not less).

**Cost accepted:** tasks are a single line; no multi-line descriptions.

## 9. Dots — the dot is the status

`○` TODO, `◐` DOING, `●` DONE, per task.

**Rejected:** dots only as a per-section progress row (cleaner list, but during scroll
you lose track of which section you're in); dots plus a global progress bar at the
bottom (costs a row and more render code).

**Why:** the dot carries information rather than being decoration, and it survives a
monochrome terminal.

## 10. Colors — the terminal's 16 ANSI colors

**Rejected:** a fixed truecolor hex palette (identical everywhere, but clashes with the
surrounding panes and can be unreadable on a light theme); no color at all.

**Why:** changing the terminal colorscheme restyles the app for free, and it stays
consistent with vim in the neighbouring pane.

## 11. Growing DONE — do nothing

**Rejected:** a foldable DONE section collapsed to `DONE (12) ▸` with `za` (~15 lines,
keeps TODO/DOING always on screen); showing only the last N.

**Cost accepted:** periodic manual cleanup with `dd`, and scrolling has to work well
because DONE will push TODO off screen.

## 12. Delete protection — `u` undo

**Rejected:** a `y/n` confirmation on `dd` (friction on exactly the operation done in
bursts while pruning DONE); no protection at all.

**Design:** snapshot the task slice before every mutation onto an in-memory stack. At
this scale copying the whole slice is free, and it also covers accidental `1`/`2`/`3`
and `J`/`K`. The stack is not persisted.

## 13. Missing file — start empty, create on first task

**Rejected:** searching up the directory tree like `.git` (same list from any
subdirectory, but you stop knowing which file you're editing); erroring out.

**Why:** running `todo` in an unrelated directory must not litter it with a file.

## 14. Installation — `Makefile` with `make install`

**Rejected:** `go install` into `~/go/bin` (the Go standard, no extra file, but depends
on `~/go/bin` being in PATH); `go run .` from the repo (recompiles on every start and
breaks the per-cwd data file).

**Cost accepted:** an extra file, and `sudo` when installing to `/usr/local/bin`.

## 15. v1 extras

**In:** bottom key-hint bar, per-section counters, `/` filtering, parser round-trip tests.

**Search, revised:** `n`/`N` were requested, then dropped. With `/` filtering the list
down to matches, `n` would move the cursor by exactly one row — i.e. duplicate `j`. The
coherent alternative (vim-style: `/` jumps and highlights without filtering, `n`/`N`
walk the matches) was considered and rejected in favour of keeping plain filtering.
