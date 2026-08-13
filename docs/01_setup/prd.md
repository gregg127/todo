# PRD — todo-cli v1

Derived from [the design grilling session](./grilling.md) (2026-08-13). The grilling doc
is the *why*; this document is the *what to build*. Where the grilling left a question
open, this PRD closes it in **Implementation Decisions**.

Repo state at time of writing: empty (`docs/` only, no code).

## Problem Statement

I keep one tmux pane (113x37) permanently open next to vim, and I want my task list to
live there. Every existing option fails that pane in some way: a web app or an Electron
app cannot live in a terminal pane at all; `todo.txt`-style CLIs make me re-run a command
and re-read the whole list after every change; a plain Markdown file open in a second vim
instance means I am editing checkboxes by hand and manually moving lines between
sections. None of them are driven by the motions my hands already know, and none of them
survive weeks of being open — a Homebrew upgrade of a runtime can break them out from
under me.

I also want the list to belong to the project, not to my home directory: when I `cd` into
a project, the pane should show that project's tasks, and those tasks should be
committable alongside the code, readable in a Markdown preview, and editable by hand in
vim when that is faster than the app.

## Solution

`todo` — a single static Go binary that renders an interactive, always-on TUI board in
one tmux pane and stores its state in `./todo-database.md` in the directory it was
launched from.

The board is one column of three stacked sections (TODO / DOING / DONE), each task on one
line prefixed by a status dot (`○` `◐` `●`) so state is legible without color. `j`/`k`
flow continuously across section boundaries; `1`/`2`/`3` move the task under the cursor to
TODO/DOING/DONE in a single keypress; `J`/`K` reorder it within its section; `o`/`O`/`cc`/
`dd` add, edit and delete; `u` undoes; `/` filters. No shift means the cursor moves, shift
means the task moves.

Every mutation is written to disk immediately and atomically, and a ~1s mtime poll picks
up hand-edits made in vim in another pane, so the file and the app are never out of sync
for longer than a second. The file uses standard Markdown headings and checkboxes, so it
renders correctly anywhere and stays readable when the app is not running.

## User Stories

### Launching and installing

1. As a developer, I want to install the app with `make install`, so that `todo` is on my
   PATH without me having to add `~/go/bin` to it.
2. As a developer, I want the app to be a single static binary, so that a Homebrew runtime
   upgrade cannot break a pane that has been open for weeks.
3. As a developer, I want the app to start instantly, so that relaunching it in a new pane
   costs me nothing.
4. As a developer, I want `todo --help` to print the keybindings and exit, so that I can
   remind myself of a key without launching the TUI.
5. As a developer, I want `todo` to read `./todo-database.md` from the current working
   directory, so that each project has its own list.
6. As a developer, I want running `todo` in a directory with no todo file to show an empty
   board rather than an error, so that I can start typing tasks immediately.
7. As a developer, I want running `todo` in an unrelated directory to create no files at
   all unless I add a task, so that I never litter a directory by accident.
8. As a developer, I want to commit `todo-database.md` alongside my code, so that the task
   list is versioned with the project.

### Reading the board

9. As a developer, I want the three sections TODO, DOING and DONE always rendered in that
   order, so that the board's shape is constant and I never hunt for a section.
10. As a developer, I want an empty section to still show its header, so that the layout
    does not jump as tasks move.
11. As a developer, I want each section header to show a task count like `TODO (5)`, so
    that I can see my backlog size at a glance.
12. As a developer, I want each task prefixed by a status dot (`○` TODO, `◐` DOING, `●`
    DONE), so that I can read state on a monochrome terminal or in a screenshot.
13. As a developer, I want the cursor marked with `▸` in a left gutter, so that the cursor
    is visible even when the inverted background is hard to see.
14. As a developer, I want the app to use only the terminal's 16 ANSI colors, so that
    changing my terminal colorscheme restyles the app for free and it matches vim in the
    neighbouring pane.
15. As a developer, I want a dimmed key-hint bar on the bottom row, so that I can
    rediscover keys without leaving the app.
16. As a developer, I want a task title longer than the pane to be truncated on one line
    rather than wrapped, so that every task stays exactly one row and `j`/`k` stay
    predictable.
17. As a developer, I want the list to scroll with a few rows of context kept around the
    cursor, so that I can see what is above and below where I am working.
18. As a developer, I want a growing DONE section to simply scroll rather than be folded
    or truncated, so that nothing is ever hidden from me without my asking.

### Moving the cursor

19. As a developer, I want `j` and `k` to move the cursor down and up, so that navigation
    uses the motions I already have in my fingers.
20. As a developer, I want `j`/`k` to flow across section boundaries, so that the whole
    board behaves like one continuous list.
21. As a developer, I want `j` on the last task and `k` on the first task to do nothing,
    so that the cursor never wraps around and surprises me.
22. As a developer, I want `gg` to jump to the first task and `G` to the last, so that I
    can reach either end of a long board in two keystrokes.
23. As a developer, I want the viewport to follow the cursor automatically, so that the
    cursor is never off screen.

### Changing task status

24. As a developer, I want `1`, `2` and `3` to move the task under the cursor to TODO,
    DOING and DONE, so that finishing a task is one keypress rather than two.
25. As a developer, I want a task moved to a new section to land at the top of that
    section, so that DONE leads with what I finished most recently.
26. As a developer, I want pressing the number of the section a task is already in to be a
    no-op, so that a mistyped key does not silently reorder my list.
27. As a developer, I want the cursor to follow the task I just moved, so that I can
    immediately act on it again (e.g. `3` then `dd`).
28. As a developer, I want the status dot and the section count to update immediately, so
    that I get instant feedback that the key registered.

### Reordering tasks

29. As a developer, I want `J` and `K` to move the current task down and up within its
    section, so that I can express priority without a priority field.
30. As a developer, I want the top of TODO to be "what's next", so that the board answers
    my main question by position alone.
31. As a developer, I want `J`/`K` to stop at the section boundary rather than push the
    task into the next section, so that reordering never changes status by accident.
32. As a developer, I want the reordering to be written to the file in the same line
    order, so that the file and the screen always agree.

### Adding, editing and deleting

33. As a developer, I want `o` to open a one-line input for a new task below the cursor,
    so that adding a task never leaves the pane.
34. As a developer, I want `O` to do the same above the cursor, so that I can put an
    urgent task at the top of TODO in one step.
35. As a developer, I want a new task to be created in the section the cursor is in, so
    that `o` in DOING starts something rather than queueing it.
36. As a developer, I want `o`/`O` on an empty board to create the first TODO task, so
    that a fresh project needs no special path.
37. As a developer, I want Enter to confirm the input, so that the flow matches every
    other prompt I use.
38. As a developer, I want Esc to cancel the input and leave the board untouched, so that
    I can back out of a typo.
39. As a developer, I want confirming empty or whitespace-only text to create no task, so
    that I do not end up with blank rows.
40. As a developer, I want `cc` to edit the current task with its existing text prefilled,
    so that fixing a typo does not mean retyping the line.
41. As a developer, I want `dd` to delete the current task without a confirmation prompt,
    so that pruning DONE in a burst is not death by a thousand `y`s.
42. As a developer, I want the cursor to stay in place after a delete (landing on the task
    that took the deleted one's row), so that `dd dd dd` prunes a run of tasks.
43. As a developer, I want normal-mode keys like `q` and `j` to be typed as literal
    characters while I am in the input, so that a task can be called "quit the job".

### Undo

44. As a developer, I want `u` to undo the last change, so that `dd` needs no confirmation
    dialog.
45. As a developer, I want undo to also cover an accidental `1`/`2`/`3`, so that a mistyped
    status is one keypress to fix.
46. As a developer, I want undo to also cover `J`/`K`, `o`/`O` and `cc`, so that every
    mutation is reversible by the same key.
47. As a developer, I want repeated `u` to walk back through several changes, so that I can
    recover from a run of mistakes.
48. As a developer, I want `u` on an empty undo stack to do nothing visible, so that
    hammering it is harmless.
49. As a developer, I want an undo to be written to disk like any other change, so that the
    file reflects what I see.
50. As a developer, I accept that the undo history is discarded when I quit, so that
    nothing extra is stored on disk.

### Filtering

51. As a developer, I want `/` to start a filter and narrow the list as I type, so that I
    can find a task in a long DONE section.
52. As a developer, I want the filter to match case-insensitively on the task text, so that
    I do not have to remember how I capitalised it.
53. As a developer, I want section headers and counts to reflect the filtered view, so that
    I can see how many matches are where.
54. As a developer, I want Esc to clear the filter and return to the full board, so that
    exiting a search is one key.
55. As a developer, I want to act on a filtered task with `1`/`2`/`3`, `cc` and `dd`, so
    that filtering is a way to reach a task, not just to look at it.
56. As a developer, I want the filter to persist while I act on matches, so that I can
    clean up several matching tasks in a row.
57. As a developer, I do not want `n`/`N`, because with a filtered list they would just
    duplicate `j`/`k`.

### Persistence and hand-editing

58. As a developer, I want every change written to disk immediately, so that a killed tmux
    pane never loses a day of work.
59. As a developer, I want writes to be atomic (temp file plus rename), so that a crash
    mid-write cannot leave me with a truncated list.
60. As a developer, I want the file to use standard `## TODO` headings and `- [ ]` /
    `- [x]` checkboxes, so that it renders correctly in any Markdown preview.
61. As a developer, I want tasks in DONE written as `[x]` and tasks in TODO and DOING as
    `[ ]`, so that the checkbox and the section never contradict each other.
62. As a developer, I want to edit `todo-database.md` by hand in vim in another pane, so
    that bulk edits use the tool that is best at them.
63. As a developer, I want the app to notice the file changing underneath it within about
    a second and reload, so that I never overwrite my own vim edits with stale state.
64. As a developer, I want the app's own writes not to trigger a reload, so that the cursor
    does not jump every time I press a key.
65. As a developer, I want the cursor to stay on a sensible task after an external reload,
    so that a reload does not lose my place.
66. As a developer, I want lines the parser does not understand to be dropped when the app
    next saves, so that the file has one owner and one format.
67. As a developer, I want `q` to quit immediately with everything already on disk, so that
    there is no "unsaved changes" concept at all.

## Implementation Decisions

### Modules

Four units, in dependency order. Only the last one talks to Bubble Tea.

- **Board model** — the in-memory state: an ordered slice of tasks, each with a title and
  a status (TODO / DOING / DONE). Section membership is derived by filtering on status;
  order within a section is the order in the slice. There is no `id`, no `priority` and no
  timestamps.
- **Markdown store** — `Parse(string) → Board` and `Render(Board) → string`, plus
  `Load(path)` / `Save(path)` wrapping them. Pure functions in the middle so the round-trip
  is testable without a filesystem.
- **File watcher** — a Bubble Tea ticker command firing about every second, emitting a
  message when the file's mtime differs from the last mtime the app itself wrote.
- **TUI** — the Bubble Tea `Model` holding the board, cursor index, viewport offset, input
  mode, filter query, pending key prefix and undo stack; `Update` as the single reducer,
  `View` as the renderer (lipgloss).

### Parsing and serializing

- Recognised structure: `## TODO`, `## DOING`, `## DONE` headings (exact, uppercase), and
  under them list items matching `- [ ] text` or `- [x] text`.
- **The section wins over the checkbox.** A `- [x]` under `## TODO` parses as a TODO task
  and is re-serialized as `- [ ]`. This is the sane resolution when someone ticks a box by
  hand in vim without moving the line, and it keeps `Render` total.
- Items appearing before any recognised heading, and any other line (prose, blank lines,
  other headings), are dropped on the next save. Round-tripping is therefore
  *normalizing*, not byte-preserving: `Render(Parse(Render(b))) == Render(b)` is the
  invariant, not `Parse` of arbitrary input being byte-identical.
- Empty input parses to an empty board. `Render` of an empty board emits all three
  headings with no items — but see below: it is never written unprompted.
- `Render` always emits all three sections in TODO / DOING / DONE order, one blank line
  between sections, trailing newline at end of file.

### Persistence

- Save after every mutation: add, edit, delete, status change, reorder, undo.
- Atomic write: create a temp file in the same directory, write, `fsync`, `rename` over the
  target. Same-directory temp guarantees the rename is atomic (same filesystem).
- After each save, the app records the resulting mtime; the watcher only fires a reload
  when the observed mtime differs from that recorded value.
- **Missing file:** `Load` of a nonexistent path yields an empty board and no error. The
  first `Save` — triggered by the first mutation — creates the file. Launching and quitting
  without a mutation writes nothing.
- Concurrent-write race with vim is accepted, unmitigated: last write wins, no merge.

### Key handling

- `gg`, `dd` and `cc` are handled with a single pending-prefix field on the model. There is
  **no timeout**: a pending `g`/`d`/`c` stays pending until the next key, which either
  completes the sequence or cancels it (and is then discarded, not re-dispatched). This
  costs one field and no timers.
- `1`/`2`/`3` insert at the start of the target section. Pressing the number of the task's
  current section is a no-op — it must not re-insert the task at the top of its own
  section, and it must not push an undo snapshot.
- `J`/`K` swap with the adjacent task **of the same status**; at a section boundary they do
  nothing.
- Modes are exclusive: normal, insert (add/edit) and filter. In insert and filter modes,
  every printable key is text; only Enter and Esc are commands.
- On quit (`q` in normal mode) the app exits immediately — there is nothing to flush.

### Cursor semantics

- The cursor is an index into the *visible* task list (the board, or the filtered subset).
- After a status change, the cursor follows the moved task to its new position.
- After a delete, the index is kept and clamped to the last visible task; deleting the last
  task on the board leaves the cursor at index 0 on an empty board.
- After an external reload, the app tries to keep the cursor on the task with the same
  title; failing that, it clamps the old index into the new list.
- The viewport keeps a 3-row scrolloff above and below the cursor where the list is long
  enough to allow it.

### Undo

- Before every mutation, the current task slice is deep-copied and pushed onto an in-memory
  stack. `u` pops and restores. No redo, no persistence, no depth cap — the data is a few
  dozen short strings and the process is restarted often enough that unbounded growth is
  not a real concern.
- Undo restores tasks only. The cursor is clamped into the restored list; the filter query
  is left as-is.
- An external reload clears the undo stack: the snapshots describe a state the file no
  longer has, and restoring one would silently clobber the vim edit that just arrived.

### Rendering

- Fixed single-column layout. The pane is known to be 113x37; the app still reads the size
  from `tea.WindowSizeMsg` and uses it for truncation and viewport height, but no
  alternative layout exists for other sizes.
- Titles longer than the available width are truncated with a trailing `…`.
- Styling uses ANSI indices 1–15 only: TODO grey, DOING yellow, DONE dimmed green, cursor
  line inverted. No truecolor hex anywhere.
- The bottom row is a dimmed hint bar; its content changes with mode (normal keys vs.
  "enter confirm · esc cancel" in insert, vs. the live filter query in filter mode).

### CLI

- No flags other than `--help`, which prints a short usage line plus the keybinding table
  and exits 0. No config file, no environment variables.

### Install

- `Makefile` with `build`, `install` (copy binary to `/usr/local/bin/todo`), `test` and
  `clean`. `install` may require `sudo`; that is accepted.

## Testing Decisions

### What makes a good test here

A test drives the app the way the user does — keys in, screen and file out — and asserts
only what the user could observe. It must not reach into the model's private fields, assert
on cursor indices or viewport offsets directly, or call internal helpers. If a test would
survive a rewrite of the reducer that preserves behaviour, it is a good test; if renaming a
field breaks it, it is not.

### The seam

**One seam: `Update` + `View` + the file on disk.** Bubble Tea's `Model` is already a pure
reducer, so tests construct a model rooted at a `t.TempDir()`, feed a sequence of
`tea.KeyMsg` values through `Update`, and assert on two observable outputs: the string
returned by `View()` and the bytes of `todo-database.md`. No `tea.Program`, no terminal, no
goroutines — the ticker is driven by sending the watcher message explicitly.

```
m := New(tmpdir)
m = send(m, "o", "fix bug", enter, "2")
assert View(m)   contains "DOING (1)" and "◐ fix bug"
assert file(dir) == "## TODO\n\n## DOING\n- [ ] fix bug\n\n## DONE\n"
```

This one seam covers keybindings, cursor movement, section counts, status dots, undo,
filtering and persistence. External hand-edits are tested through it too: write the file
directly, send the reload message, assert the new `View()`.

### The one exception

`Parse` / `Render` also get direct round-trip tests, because that is the only place where a
bug destroys data the user typed, and because the interesting inputs (a `[x]` under
`## TODO`, prose between sections, a truncated file, an empty file, CRLF, a task title
containing `- [ ]`) are awkward to express as keystrokes. Assert the normalizing invariant
`Render(Parse(s)) == Render(Parse(Render(Parse(s))))`, plus explicit expected output for
each odd input.

### Modules tested

| Module | How |
|---|---|
| TUI (`Update`/`View`) | the main seam; the large majority of tests |
| Markdown store | via the main seam, plus direct round-trip tests |
| Board model | only through the two above — no direct tests |
| File watcher | the mtime comparison is tested by sending reload messages; the ticker itself is not tested |

### Prior art

None — this is the first code in the repo. This PRD's seam definition *is* the prior art
for everything added later; new features are expected to be tested through `Update`/`View`
rather than by introducing a new seam.

## Out of Scope

- Folding, collapsing, archiving or auto-pruning DONE. It grows; the user prunes with `dd`.
- `n` / `N` search navigation, and vim-style `/` that jumps-and-highlights instead of
  filtering.
- Multi-line task descriptions, due dates, priorities, tags, subtasks, assignees.
- Configuration files, colour themes, remappable keys, environment variables, and any CLI
  flag other than `--help`.
- A non-interactive CLI (`todo add "…"`, `todo list`) — one interface only.
- Searching up the directory tree for a todo file, a global `~/todo-database.md`, or
  `$TODO_FILE`.
- Merging concurrent edits, file locking, or any conflict resolution beyond last-write-wins.
- Mouse support, resizing beyond truncation, and any layout for pane sizes other than
  roughly 113x37.
- Redo, and persisting the undo stack across runs.
- Byte-preserving round-trips of hand-written prose in `todo-database.md`.
- Distribution beyond `make install`: no Homebrew formula, no release binaries, no CI.

## Further Notes

- **The riskiest area is scrolling**, not parsing. Because DONE is never folded, a
  long-lived project will push TODO off the top of the pane within weeks. Viewport and
  scrolloff behaviour needs to be right in v1 — it is the decision most likely to send us
  back to the folding question that was rejected in the grilling.
- **The second riskiest is the reload path**, specifically the interaction between the
  mtime check, the app's own writes, and the undo stack. Getting the "don't reload my own
  write" guard wrong produces a cursor that jumps on every keypress; getting it wrong the
  other way silently eats vim edits.
- `1`/`2`/`3` were chosen over `H`/`L` knowing they break the "shift = the task moves"
  mnemonic that `J`/`K` establish. If the numbers turn out not to stick in muscle memory,
  adding `H`/`L` as aliases is a few lines and does not change the data model.
- The per-directory data file means a task can be "lost" in another project's file. There
  is deliberately no cross-project view; if that becomes painful, the fix is a separate
  tool that greps for `todo-database.md`, not a feature inside this app.
- No issue tracker is configured for this repo, so this PRD lives as a file rather than an
  issue. If a tracker is added later, this document is the content of the first ticket.
