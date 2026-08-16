# Security review

A security review of `todo-cli` as it stands at commit `6421c38`, covering the
application source and all 19 modules in the dependency graph.

> **Status: findings 1–3 fixed, recommendations 1–4 applied.** See `e0fd71d`
> (control characters), `aebd1dc` (the click guard) and `9b058c4` (the
> dependency floor). `govulncheck` now reports nothing at any level. Still open:
> recommendation 5 (Bubble Tea v2 migration) and 6 (CI). The findings below are
> left as written, as the record of what was wrong.

**Method.** The whole source tree (~800 lines excluding tests) was read by hand.
Dependencies were audited by five parallel sub-reviews, one per library group,
against the Go vulnerability database, OSV, and GitHub advisories. Findings were
verified by running code, not by inspection alone — the probe used for each is
given under *Verification*. `govulncheck` was run over the module.

**Threat model.** The app is local, single-user, offline: no network, no server,
no authentication, no privilege boundary. That leaves exactly two untrusted
inputs, and both of them are text that ends up on your terminal:

1. **the board file** — `todo-database.md`, or any path passed as `todo <file>`;
2. **the clipboard** — text arriving through bracketed paste.

Everything else (argv, environment, `TERM`, terminfo) is trusted local
configuration. The review focused on those two paths.

---

## Summary

| # | Finding | Severity | Confidence |
|---|---|---|---|
| 1 | Terminal escape sequence injection via task titles | **Medium** | High — verified |
| 2 | Escape sequences survive the clipboard paste path | **Medium** | High — verified |
| — | Panic on a negative mouse row (robustness, not a vulnerability) | Low | High — verified |

No high-severity findings. No reachable vulnerability in any dependency:
`govulncheck ./...` reports **0 vulnerabilities called**.

The two findings are the same underlying defect — untrusted text reaching the
terminal unfiltered — reached by two different paths. One fix at the right
boundary closes both.

---

## Finding 1 — Terminal escape sequence injection via task titles

* **Location:** `internal/board/store.go:73` (`parseItem`), rendered at `internal/tui/view.go:68`
* **Severity:** Medium
* **Category:** `terminal_escape_injection` (CWE-150, *Improper Neutralization of Escape Sequences*)
* **Confidence:** High — reproduced

### Description

`parseItem` accepts everything after the `- [ ] ` prefix as a task title, with no
filtering of control characters. `Validate` likewise treats such a line as
well-formed, so the file opens without complaint. The title is then rendered
straight to the terminal.

Nothing on that path strips escapes. This was confirmed at both layers by the
dependency review: `lipgloss.Style.Width(n).Render()` delegates to
`cellbuf.Wrap`, which tokenises the string and writes every zero-width token —
i.e. every escape sequence — into the output buffer verbatim, interpreting only
SGR and OSC 8 so it can re-emit them across a line break. Bubble Tea's renderer
then writes each line as-is, its only transform being an escape-*aware*
`ansi.Truncate` that preserves sequences rather than removing them.

The sequences are zero-width by definition, so they do not disturb the layout
that would otherwise reveal them. A poisoned title looks entirely ordinary on
screen.

### Verification

A board file containing an OSC 52 clipboard-write payload was pushed through the
real code path:

```
input:      "- [ ] buy milk\x1b]52;c;aGVsbG8=\x07"
Validate:   accepted (no error)
Parse:      title = "buy milk\x1b]52;c;aGVsbG8=\a"
rows():     "▸ ○ buy milk\x1b]52;c;aGVsbG8=\a"   ← escape reaches the terminal
Render():   "- [ ] buy milk\x1b]52;c;aGVsbG8=\a" ← and round-trips back to disk
```

### Exploit scenario

The board is one file per directory and is meant to live alongside a project, so
a checked-in `todo-database.md` is the expected case, not an exotic one. This
repository's own `make run` target copies `docs/todo-database-example.md` into a
scratch board and opens it.

An attacker who can land a line in a board file — a pull request touching the
example board, a shared repository, a project template — gets arbitrary escape
sequences written to the terminal of anyone who runs `todo` in that directory:

* **OSC 52** silently overwrites the user's clipboard on terminals that honour it
  (iTerm2, kitty, tmux with `set-clipboard on`). The user's next paste into a
  shell is attacker-controlled text. With a trailing newline in the payload this
  is command execution, and it is the standard escalation for this bug class.
* **OSC 0/2** rewrites the window title.
* **DSR / DECRQSS** (`CSI 5n` and friends) make the terminal *reply*, and that
  reply arrives on stdin, where Bubble Tea reads it as input. This is a weaker
  vector — most replies decode to unknown sequences rather than useful
  keystrokes — but it is the reason escape filtering is normally done on input
  rather than trusted to the terminal.

Because `Render` writes the title back unchanged, a poisoned entry also persists
across the app's own saves.

### Precedent

This is not theoretical for this stack. **CVE-2025-64494** (GHSA-fv2r-r8mp-pg48,
CWE-150) is exactly this bug in Charm's own `soft-serve`: ANSI sequences not
stripped from user-supplied names and messages. The libraries are consistent and
deliberate about passing escapes through; neutralising them is the consumer's
job.

### Recommendation

Reject control characters in `Validate`, rather than silently stripping them.
Refusing to open a file it cannot represent is already this app's documented
posture for unparseable content ("rather than eat somebody's notes the app
refuses to open the file at all"), and stripping would quietly rewrite the user's
file on the next save — the very thing that posture exists to prevent.

```go
// in internal/board/store.go

// controlChar reports the first C0/C1 control character or DEL in s. A title is
// one line of display text; a control character in it is either a mistake or an
// attempt to write escape sequences to the terminal of whoever opens the board.
func controlChar(s string) (rune, bool) {
	for _, r := range s {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			return r, true
		}
	}
	return 0, false
}
```

Wire it into `Validate` alongside the existing line check:

```go
if title, ok := parseItem(line); !ok || !inSection {
	return fmt.Errorf("line %d: %q is not a task or a section heading", i+1, line)
} else if r, bad := controlChar(title); bad {
	return fmt.Errorf("line %d: control character %q in task title", i+1, r)
}
```

`%q` already renders the offending rune escaped, so the error message itself is
safe to print — as is the existing one.

Note that `ansi.Strip` from `github.com/charmbracelet/x/ansi` is **not sufficient
on its own** here: it removes CSI/OSC/DCS sequences but preserves raw C0 control
bytes (BEL, backspace, CR), so BEL- and CR-based tricks survive it. The rune
filter above is the complete check.

---

## Finding 2 — Escape sequences survive the clipboard paste path

* **Location:** `internal/tui/model.go:371` (`pasted`)
* **Severity:** Medium
* **Category:** `terminal_escape_injection` (CWE-150)
* **Confidence:** High

### Description

```go
text = strings.Join(strings.Fields(text), " ")
```

`pasted` flattens a multi-line paste to spaces, which handles newlines and tabs
but nothing else: `strings.Fields` splits on `unicode.IsSpace`, and **ESC (0x1b)
is not whitespace**. Escape sequences in pasted text pass through into `m.input`
or `m.filter` unchanged.

Bubble Tea does not filter them earlier either — `detectBracketedPaste` keeps
every rune between `ESC[200~` and `ESC[201~` except `RuneError`, ESC and C0
controls included.

From there the text becomes a task title, is written to disk by `confirm()`, and
is re-emitted on every subsequent render — arriving at Finding 1 from the other
direction, and persisting.

### Exploit scenario

Weaker than Finding 1, since it needs the user to copy attacker-controlled text —
from a web page, a chat message, a terminal — and paste it into the task input.
That is an ordinary thing to do with a todo app. The result is a self-inflicted
persistent escape sequence in the user's own board file.

### Recommendation

Strip rather than reject here: there is no file to damage yet, and refusing a
paste outright would be a poor interaction.

```go
func (m Model) pasted(text string) Model {
	text = strings.Join(strings.Fields(text), " ")
	text = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			return -1
		}
		return r
	}, text)
	// ...
}
```

`typed()` needs no change: it appends only single-rune keys, and Bubble Tea
reports control keys as multi-rune names (`"ctrl+a"`, `"esc"`), which it already
ignores.

---

## Not a vulnerability, but worth fixing — panic on a negative mouse row

* **Location:** `internal/tui/view.go:146` (`click`)
* **Severity:** Low (availability only — outside the scope of this review, recorded because it was found in passing)

Commit `6421c38` removed the `y < 0` guard on the reasoning that "a press
coordinate is never negative". That does not hold: both of Bubble Tea's mouse
parsers normalise 1-based terminal coordinates by subtracting one, without
clamping.

```go
// bubbletea@v1.3.10/mouse.go:198  (SGR)
m.Y = y - 1                                  // y = Atoi("0") → -1
// bubbletea@v1.3.10/mouse.go:220  (X10)
m.Y = int(v[2]) - x10MouseByteOffset - 1     // byte 32 → -1
```

A row-0 mouse report therefore reaches `click(-1)`, and the remaining guards do
not catch it: `-1 >= listHeight` is false, `-1 >= len(at)` is false, and `at[-1]`
panics.

**Verified:** `click(-1)` on a one-task board panics with
`runtime error: index out of range [-1]`.

Impact is limited to a crash — saves are atomic, so the board file is not left
damaged — which is why this is not filed as a security finding. Restoring the
`y < 0` guard is a one-word fix.

---

## Dependency audit

`govulncheck ./...` (Go 1.24.4, darwin/arm64):

```
=== Symbol Results ===
No vulnerabilities found.
Your code is affected by 0 vulnerabilities.
```

Two advisories exist in the module graph but are **unreachable**, confirmed
against the real build graph rather than `go.mod` alone:

| Advisory | Module | Fixed in | Reachable? |
|---|---|---|---|
| GO-2026-5024 / CVE-2026-39824 — integer overflow in `NewNTUnicodeString` | `golang.org/x/sys` v0.36.0 | v0.44.0 | **No.** Windows-only symbol; `go list -deps ./cmd/todo` links only `x/sys/unix` on this platform, and nothing in the tree calls it on any platform. |
| GO-2026-5970 / CVE-2026-56852 — infinite loop in `unicode/norm` on invalid UTF-8 | `golang.org/x/text` v0.3.8 | v0.39.0 | **No.** x/text enters only via `go-localereader`, whose Unix build is a two-line passthrough. `unicode/norm` and `language` are never in the build graph. |

The x/text finding deserves a note: an invalid-UTF-8 infinite loop is precisely
the bug class that *would* matter for a program rendering arbitrary Markdown, and
`v0.3.8` is roughly four years and 33 minor versions behind current. It is safe
today only by accident of what is imported. Bump it.

### Per-library results

No CVEs or advisories were found for any Charm-stack module. All modules are MIT
or BSD-3 licensed. `go mod verify` passes against `sum.golang.org`.

| Module | Pinned | Latest | Notes | Risk |
|---|---|---|---|---|
| `charmbracelet/bubbletea` | v1.3.10 | v2.0.8 | Clean. **v1 is an EOL branch** — upstream moved to `/v2` (Feb 2026). v2 fixes not backported include an `onMouse` data race and a wide-character infinite loop. | Low–Medium |
| `charmbracelet/lipgloss` | v1.1.0 | v1.1.0 | Clean, current for v1. Is the ANSI passthrough path for Finding 1. | Medium |
| `charmbracelet/colorprofile` | pseudo `0.2.3-2025031…` | v0.4.3 | Untagged commit, but **imposed upstream** by lipgloss and bubbletea — not a local pin. Commit verified genuine and reachable on the default branch. | Low–Medium |
| `charmbracelet/x/ansi` | v0.10.1 | v0.11.8 | Clean. Provides `ansi.Strip` (see caveat in Finding 1). | Low |
| `charmbracelet/x/cellbuf` | pseudo `0.0.13-2025031…` | v0.0.15 | Untagged commit, ~1 year stale. Auditability concern only; `go.sum` covers tamper risk. | Medium (supply chain) |
| `charmbracelet/x/term` | v0.2.1 | v0.2.2 | Clean. | Low |
| `muesli/termenv` | v0.16.0 | v0.16.0 | Clean, current. Its `Output.Copy` (OSC 52 write) is never called by this app. | Low |
| `aymanbagabas/go-osc52/v2` | v2.0.1 | v2.0.1 | Clean, current. Emit-only and **inert** — nothing in this app invokes it. Upstream dormant since 2023, but ~150 lines of stateless string building. | Low |
| `lucasb-eyer/go-colorful` | v1.2.0 | v1.4.1 | Clean. Pure colour maths, no I/O. | Low |
| `mattn/go-runewidth` | v0.0.16 | v0.0.27 | Clean. | Low |
| `rivo/uniseg` | v0.4.7 | v0.4.7 | Clean, current. | Low |
| `mattn/go-localereader` | v0.0.1 | v0.0.1 | Clean. Dormant since 2022, sole release, 12 stars. No-op on Unix. Ships no LICENSE file in the module zip — license scanners will flag it as unknown. | Low |
| `golang.org/x/text` | v0.3.8 | v0.41.0 | GO-2026-5970, unreachable. Badly stale. | Low (Medium if `norm`/`language` are ever imported) |
| `golang.org/x/sys` | v0.36.0 | v0.47.0 | GO-2026-5024, unreachable (Windows-only). | Low |

| `muesli/cancelreader` | v0.2.2 | v0.2.2 | Clean, at latest tag. Dormant since Mar 2023 but complete. | Low |
| `muesli/ansi` | pseudo `2023-03-16` | none tagged | Clean. Untagged, but the pinned hash **is** the current `main` HEAD. | Low |
| `erikgeiser/coninput` | pseudo `2021-10-04` | none tagged | Clean. Abandoned since 2021, but Windows-only and **not linked into the darwin binary**. | Low |
| `mattn/go-isatty` | v0.0.20 | v0.0.24 | Clean. | Low |
| `xo/terminfo` | pseudo `2022-09-10` | v1.0.0 | Clean of advisories, but panics on malformed terminfo files — see below. | Low–Medium |

### `xo/terminfo` — panic on a malformed terminfo file

The sub-review fuzzed `terminfo.Decode` and got **12,473 panics in 200,000**
inputs with valid magic (~6%), from two bugs in `dec.go`:

* header fields decode as signed `int16` and `hasInvalidCaps` bounds-checks only
  the upper end, so a negative `nameSize` reaches `readBytes(n)` — whose guard
  `d.n < d.pos+n` passes for negative `n` — giving `slice bounds out of range`;
* `canonicalizeAscChars` reads `z[i+1]` with no odd-length guard, giving
  `index out of range` on an odd-length `acsc` capability.

**Both are still present in v1.0.0**, so upgrading does not fix them. The path is
reachable rather than theoretical: `colorprofile/env.go:200` calls
`terminfo.Load(term)`, pulled in via `internal/tui` → lipgloss → cellbuf →
colorprofile, so a crafted terminfo file crashes the TUI at startup.

Rated **Low** and filed as robustness rather than security: `TERM`, `TERMINFO`
and `TERMINFO_DIRS` are trusted local configuration, and anyone able to set them
can already run code as the user. Go panics safely, so there is no memory-safety
consequence. Recorded here so it is not mistaken for a fresh finding later.

Three of these five are unreleased pseudo-versions. All pinned hashes resolve —
no deleted or force-pushed commits — and `go.sum` pins content hashes, so the
concern is auditability rather than tamper risk.

### Toolchain

The module builds on **Go 1.24.4**. `govulncheck` reports 8 advisories in
packages it imports and 40 in modules it requires; every one of those except the
`x/sys` entry above is stdlib (`net/url`, `net/mail`, `os`, `os/exec`), and none
are reachable from this code. Go **1.25.13** clears all of them.

The toolchain is both the cheapest item on this list to keep current and the one
carrying by far the most advisories.

---

## Recommendations

In priority order.

**1. Neutralise control characters in untrusted text.** Findings 1 and 2. Reject
in `Validate`, strip in `pasted`. This is the only change here with a real
security payoff, and it is perhaps 15 lines including tests. Test cases worth
having: OSC 52 in a file, ESC in a paste, BEL and CR (which `ansi.Strip` would
miss), and a legitimate title containing emoji and em-dashes, which must still
pass.

**2. Restore the `y < 0` guard in `click`.** A crash reachable from terminal
input, and a one-word revert of `6421c38`.

**3. Refresh the dependency floor.** Nothing here is exploitable, so this is
hygiene rather than remediation — but it clears every scanner finding at once:

```sh
go get -u golang.org/x/text golang.org/x/sys
go get -u github.com/charmbracelet/x/ansi github.com/charmbracelet/x/term
go get -u github.com/mattn/go-runewidth github.com/lucasb-eyer/go-colorful
go mod tidy && go test ./...
```

Upgrading `lipgloss`/`bubbletea` is what would move `colorprofile` and `cellbuf`
off their untagged pseudo-versions; they cannot be moved directly.

**4. Update the Go toolchain** past the advisories above.

**5. Plan the Bubble Tea v2 migration.** v1.3.10 is the last v1 release and the
branch is closed. Nothing is broken today, but security fixes will not arrive
here. This is a roadmap item, not an urgent one.

**6. Add CI — there is none.** A `.github/workflows` running `go test ./...`,
`go vet ./...` and `govulncheck ./...` would have caught the advisories in this
report automatically, and `govulncheck`'s call-graph analysis correctly reports
the x/text and x/sys findings as unreachable rather than raising noise.

### Things that were checked and are fine

Worth recording so they are not re-reviewed:

* **Atomic save** (`store.go:130`) is correct — temp file in the same directory,
  `fsync`, `rename`. `os.CreateTemp` uses `O_EXCL`, so there is no temp-file race,
  and because `Save` renames over the path it never follows a symlink to write
  through it.
* **Error messages** interpolate the offending line with `%q`, which escapes
  control characters — so the parse-error path is not itself an injection vector.
  The same is true of the filter display in `hints()`.
* **The read-error latch** (`readErr`, `model.go:94`) correctly stops saving when
  a hand-edit makes the file unparseable, which prevents the app from destroying
  content it could not read.
* **`argv` and `TERM`** are trusted local configuration; no finding is filed
  against them.
* **No secrets, no credentials, no network, no subprocess execution, no
  deserialisation, no template rendering, no SQL** anywhere in the tree.
