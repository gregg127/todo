# Security review

**Verdict: low risk. No open security finding.** The escape-injection findings
of the previous review are fixed and covered by tests, `govulncheck` reports
nothing, and no module in the graph carries a reachable advisory. What remains
are four low-severity robustness and integrity defects, listed below. Supply
chain is the residual risk, held by the SHA-256 hashes in `go.sum` that
`make build` and CI verify.

Reviewed at commit `9d755c2`: the whole source tree (~800 lines excluding
tests), the build and CI, and all 26 modules in the graph. Findings were
verified by running code through the public `Update` / `Load` / `Save` entry
points, not by inspection alone.

## Threat model

Local, single-user, offline: no network, no server, no auth, no privilege
boundary, no subprocess. Two untrusted inputs, both text that ends up on your
terminal:

1. **the board file** — `todo-database.md`, or any path passed as `todo <file>`,
   which may be checked into a shared repo or written by a project template;
2. **the clipboard** — text arriving through bracketed paste.

`argv`, the environment and `TERM` are trusted local configuration.

## Findings

| # | Finding | Severity | Status |
|---|---|---|---|
| 1 | Panic on confirm after the watcher reloads a shrunken file | Low (robustness) | verified |
| 2 | Control characters reach disk on the write path, locking the board | Low | verified |
| 3 | Unicode format characters pass both control-character filters | Low (spoofing) | verified |
| 4 | Saving a symlinked board replaces the symlink | Low (integrity) | verified |

None is a trust-boundary break. No high or medium severity finding.

### 1 — Panic on confirm after an external reload

`internal/tui/watch.go:35` replaces `m.tasks` but leaves `m.mode`, `m.editing`
and `m.insertAt` untouched, and `Update` reloads on every tick with no mode
guard. Open the editor on the last task with `G cc`, let the file shrink on disk
(a hand edit, a `git checkout`, a second instance writing the DONE fold), then
press enter:

```
editing=2, tasks=1  → panic: index out of range [2] with length 1   (model.go:305)
insertAt=3, tasks=1 → panic: slice bounds out of range [3:1]        (model.go:309)
```

File content is an untrusted input here, so this is a crash driven by untrusted
data — but it is a crash only. Saves are atomic, so the board is not damaged.
Fix: clamp or reset `mode`, `editing` and `insertAt` in `reload()`.

### 2 — Control characters reach disk on the write path

`Validate` guards the *read* path only; `Save` renders and writes without
validating. `pasted()` strips control characters, but `typed()`
(`internal/tui/model.go:373`) inserts any single-rune key unfiltered, and Bubble
Tea's rune scanner breaks only on `r <= 0x1F`, DEL and space — so a C1 control
decoded from UTF-8 (U+0080–U+009F, including CSI and OSC) arrives as a one-rune
key. Verified end to end:

```
typed "a", U+009B, "b"  → file: "- [ ] a\u009bb"  → next Load: REFUSED
                            "line 3: control character '\u009b' in task title"
```

The app writes a board it will then refuse to open. Reachability is narrow — it
needs a terminal delivering a lone C1 rune as a key press, which bracketed paste
prevents for clipboard text — but the asymmetry is the point. Related:
`hints()` at `internal/tui/view.go:209` renders the filter raw (`"/" + m.filter`)
while the normal-mode branch four lines down uses `%q`. Fix: the same
`unicode.IsControl` check in `typed()`, and treat `Validate` as a precondition of
`Save`.

### 3 — Unicode format characters pass both filters

`unicode.IsControl` covers category Cc in full, which is why every escape
sequence is caught. It does not cover Cf or Zl/Zp. Verified accepted by
`Validate` and written back by `Render`: U+202E RLO and the other bidi overrides,
U+200B ZWSP, U+00AD, U+FEFF, U+2028, U+2029.

This is Trojan-Source *spoofing* only — no compiler, no interpreter, no
execution downstream; the payload is a line in a task list. Worst realistic case
is a task in a shared board that reads differently than it is stored, or a
zero-width prefix that makes `/` filter miss it. Error messages are unaffected:
`%q` escapes these too. Fix if you care about display integrity: extend both
filters to `unicode.IsControl(r) || unicode.Is(unicode.Cf, r)` plus U+2028/29.

### 4 — Saving a symlinked board replaces the symlink

`os.Rename` at `internal/board/store.go:189` does not follow symlinks while the
`os.Stat` at line 171 does. With `link.md -> real.md`, the first save leaves
`real.md` holding its old content and turns `link.md` into a regular file.
Verified. This is the *safe* direction for a symlink attack — the app never
writes through a link — but it silently detaches a board symlinked into a
dotfiles repo or a shared location.

## Dependencies

`govulncheck ./...` (v1.7.0, DB 2026-08-19) reports **`No vulnerabilities found.`**
at symbol and package level and against the compiled binary. `go mod verify`:
all modules verified. Go 1.25.13 is the head of its patch line and clears every
stdlib advisory that touches the 1.25 branch.

Only one version in the whole graph matches an advisory: `golang.org/x/mod`
v0.38.0 (GO-2026-6179/6180). It is not a dependency — absent from `go.sum`,
never downloaded, never linked, a module-graph residue of `x/tools`. The two
advisories the previous review tracked are gone: `x/sys` is pinned at v0.44.0,
which *is* the fix for GO-2026-5024, and `x/text` v0.41.0 postdates GO-2026-5970.
Both, plus `go-localereader` and `coninput`, are Windows-only and absent from the
darwin build graph — confirmed by diffing `go list -deps` across `GOOS`.

Posture is good: no `replace`, `exclude` or `vendor/`; `GOSUMDB=sum.golang.org`;
CI asserts `GOFLAGS`, `GOPRIVATE`, `GONOSUMDB`, `GONOPROXY` and `GOINSECURE` are
empty before downloading, pins both actions by commit SHA, and runs on
`pull_request` (not `pull_request_target`) with a read-only token.

Seven pseudo-versions exist; none was chosen here — `go.mod` requires only
`bubbletea` and `lipgloss`, and the rest are inherited. `muesli/ansi` and
`coninput` have no tags at all, so a pseudo-version is the only addressable form.
The charm stack's upper layers have drifted (`colorprofile` v0.2.3-pre → v0.4.3,
`x/ansi` v0.10.1 → v0.11.8, `go-runewidth` v0.0.16 → v0.0.28) and Bubble Tea v1
is an EOL branch — staleness and maintenance facts, not security findings.

One thing to fix: CI installs `govulncheck@latest`
(`.github/workflows/ci.yml:36`), the only step running a version nobody chose.
It is checksum-verified, and the runner is ephemeral with no secrets and a
read-only token, so the blast radius is small. Pin it.

## Checked and clean

Recorded so they are not re-reviewed:

* **Escape injection is closed.** `Validate` rejects control characters in
  titles, `pasted()` strips them, and both have tests (OSC 52 and OSC 0 payloads
  in `store_test.go:129`, a paste in `model_test.go:1423`).
* **Metadata is unvalidated and it does not matter.** `splitMeta` accepts
  arbitrary keys and values, escape sequences included, and `Render` writes them
  back verbatim — but no `Meta` string reaches `View()`: it is only compared in
  `collapsedDone` or re-rendered. Nor can it corrupt the file, since a value
  cannot hold a newline (the split happens first) or be `---` (that line ends the
  block). Note that the `k+": "+v != line` guard at `store.go:32` is dead —
  `strings.Cut` splits on the first separator, so the rejoin always matches
  whenever `ok` is true. The real check is `!ok`. Harmless, but not a defense.
* **No parser differential.** `Parse` accepts a strict subset of `Validate`.
* **Error text is safe.** `%q` escapes Cc, Cf and Zl/Zp alike; the only other
  component is the trusted `path`.
* **Atomic save is correct.** Temp file in the same directory, `O_EXCL`, fsync,
  rename. setuid/setgid do not propagate — the write follows the `Chmod`, and
  POSIX clears those bits on an unprivileged write.
* **`click` is guarded** against the negative row Bubble Tea can report.
* **`xo/terminfo` panics on a malformed terminfo file** and is reachable via
  `colorprofile`, but `TERM` and `TERMINFO_DIRS` are trusted configuration.
* **No secrets, credentials, network, subprocess, deserialisation, template
  rendering or SQL** anywhere in the tree. Nothing in the Makefile fetches code.
