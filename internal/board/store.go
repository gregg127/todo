package board

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"
)

// DefaultFile is the per-directory data file.
const DefaultFile = "todo-database.md"

// Meta is the optional frontmatter at the top of a board file: `key: value`
// lines fenced by `---`. The app reads one key and writes back whatever else
// it finds there, so a file can carry settings this version does not know.
type Meta map[string]string

// CollapsedDone is the metadata key holding whether the DONE section is folded.
const CollapsedDone = "collapsed-done"

// splitMeta separates a leading frontmatter block from the board below it. A
// file without one is all board and has no metadata: absent metadata means
// defaults, never an error. A block that is unterminated, or that holds a line
// a save would not write back as it stands, is left in the body instead, where
// Validate refuses it rather than let the app reformat somebody's file.
func splitMeta(text string) (Meta, string) {
	lines := strings.Split(text, "\n")
	if strings.TrimSuffix(lines[0], "\r") != "---" {
		return nil, text
	}
	meta := Meta{}
	for i, line := range lines[1:] {
		line = strings.TrimSuffix(line, "\r")
		if line == "---" {
			return meta, strings.Join(lines[i+2:], "\n")
		}
		k, v, ok := strings.Cut(line, ": ")
		if !ok || k+": "+v != line {
			return nil, text
		}
		meta[k] = v
	}
	return nil, text
}

// Parse reads the recognised structure out of a Markdown board: the optional
// frontmatter block, the exact headings `## TODO`, `## DOING` and `## DONE`,
// and `- [ ]` / `- [x]` list items beneath them. Everything else — prose, blank lines, other headings,
// items before the first heading — is dropped. The section wins over the
// checkbox, so `- [x]` under `## TODO` is a TODO task.
func Parse(text string) (Board, Meta) {
	meta, body := splitMeta(text)
	var b Board
	section := Status(-1)
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSuffix(line, "\r")
		if s, ok := heading(line); ok {
			section = s
			continue
		}
		if section < 0 {
			continue
		}
		if title, ok := parseItem(line); ok {
			b = append(b, Task{Title: title, Status: section})
		}
	}
	return b, meta
}

// Validate reports the first line of a board file the app cannot read. Blank
// lines, the three headings and the task items beneath them are the whole
// format; Parse drops anything else, and the next save would then write the
// file back without it. Rather than eat somebody's notes the app refuses to
// open the file at all.
func Validate(text string) error {
	meta, body := splitMeta(text)
	// The metadata block is read whole or not at all, so the only lines left to
	// check are the board's — at their own line numbers in the file.
	offset := 0
	if meta != nil {
		offset = strings.Count(text, "\n") - strings.Count(body, "\n")
	}
	inSection := false
	for i, line := range strings.Split(body, "\n") {
		line = strings.TrimSuffix(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if _, ok := heading(line); ok {
			inSection = true
			continue
		}
		title, ok := parseItem(line)
		if !ok || !inSection {
			return fmt.Errorf("line %d: %q is not a task or a section heading", i+1+offset, line)
		}
		// A title is one line of display text and goes to the terminal as it is,
		// so a control character in it is either a mistake or an attempt to write
		// escape sequences to the screen of whoever opens the board. Refuse the
		// file rather than strip them, for the same reason the parser refuses
		// anything else it cannot represent: the next save would rewrite it.
		for _, r := range title {
			if unicode.IsControl(r) {
				return fmt.Errorf("line %d: control character %q in task title", i+1+offset, r)
			}
		}
	}
	return nil
}

func heading(line string) (Status, bool) {
	switch strings.TrimRight(line, " \t") {
	case "## TODO":
		return Todo, true
	case "## DOING":
		return Doing, true
	case "## DONE":
		return Done, true
	}
	return 0, false
}

func parseItem(line string) (string, bool) {
	for _, prefix := range []string{"- [ ] ", "- [x] ", "- [X] "} {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix)), true
		}
	}
	return "", false
}

// Render serializes a board back to Markdown: the metadata block if there is
// any, then all three headings in order, one blank line between a heading and
// its list and one between sections, trailing newline at end of file. An empty
// section is its heading alone.
func Render(b Board, meta Meta) string {
	var sb strings.Builder
	if len(meta) > 0 {
		sb.WriteString("---\n")
		// Sorted, so the file does not churn on a key the app never touched.
		for _, k := range slices.Sorted(maps.Keys(meta)) {
			sb.WriteString(k + ": " + meta[k] + "\n")
		}
		sb.WriteString("---\n\n")
	}
	for i, s := range Statuses {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("## " + SectionName(s) + "\n")
		box := "- [ ] "
		if s == Done {
			box = "- [x] "
		}
		first := true
		for _, t := range b {
			if t.Status != s {
				continue
			}
			if first {
				sb.WriteString("\n")
				first = false
			}
			sb.WriteString(box + t.Title + "\n")
		}
	}
	return sb.String()
}

// Load parses the board at path, refusing a file the parser cannot read whole:
// a save would rewrite it without the parts it did not understand. A missing
// file is an empty board, not an error.
func Load(path string) (Board, Meta, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if err := Validate(string(data)); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", path, err)
	}
	b, meta := Parse(string(data))
	return b, meta, nil
}

// Save writes the board to path atomically — temp file in the same directory,
// fsync, rename — and returns the mtime of the file it just wrote.
func Save(path string, b Board, meta Meta) (time.Time, error) {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".todo-*.md")
	if err != nil {
		return time.Time{}, err
	}
	tmp := f.Name()
	defer os.Remove(tmp)

	// CreateTemp makes the file 0600; keep the board's own permissions.
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode()
	}
	if err := f.Chmod(mode); err != nil {
		f.Close()
		return time.Time{}, err
	}
	if _, err := f.WriteString(Render(b, meta)); err != nil {
		f.Close()
		return time.Time{}, err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return time.Time{}, err
	}
	if err := f.Close(); err != nil {
		return time.Time{}, err
	}
	if err := os.Rename(tmp, path); err != nil {
		return time.Time{}, err
	}
	return ModTime(path), nil
}

// ModTime is the modification time of path, or the zero time if it is missing
// or unreadable.
func ModTime(path string) time.Time {
	fi, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return fi.ModTime()
}
