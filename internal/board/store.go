package board

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultFile is the per-directory data file.
const DefaultFile = "todo-database.md"

// Parse reads the recognised structure out of a Markdown board: the exact
// headings `## TODO`, `## DOING` and `## DONE`, and `- [ ]` / `- [x]` list
// items beneath them. Everything else — prose, blank lines, other headings,
// items before the first heading — is dropped. The section wins over the
// checkbox, so `- [x]` under `## TODO` is a TODO task.
func Parse(text string) Board {
	var b Board
	section := Status(-1)
	for _, line := range strings.Split(text, "\n") {
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
	return b
}

// Validate reports the first line of a board file the app cannot read. Blank
// lines, the three headings and the task items beneath them are the whole
// format; Parse drops anything else, and the next save would then write the
// file back without it. Rather than eat somebody's notes the app refuses to
// open the file at all.
func Validate(text string) error {
	inSection := false
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSuffix(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if _, ok := heading(line); ok {
			inSection = true
			continue
		}
		if _, ok := parseItem(line); !ok || !inSection {
			return fmt.Errorf("line %d: %q is not a task or a section heading", i+1, line)
		}
	}
	return nil
}

// heading maps a section heading to its status.
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

// Render serializes a board back to Markdown: all three headings in order,
// one blank line between sections, trailing newline at end of file.
func Render(b Board) string {
	var sb strings.Builder
	for i, s := range []Status{Todo, Doing, Done} {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("## " + SectionName(s) + "\n")
		box := "- [ ] "
		if s == Done {
			box = "- [x] "
		}
		for _, t := range b {
			if t.Status == s {
				sb.WriteString(box + t.Title + "\n")
			}
		}
	}
	return sb.String()
}

// Load parses the board at path. A missing file yields an empty board and no
// error.
func Load(path string) (Board, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return Parse(string(data)), nil
}

// Save writes the board to path atomically: a temp file in the same directory,
// written, fsynced and renamed over the target. It returns the mtime of the
// file it just wrote.
func Save(path string, b Board) (time.Time, error) {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".todo-*.md")
	if err != nil {
		return time.Time{}, err
	}
	tmp := f.Name()
	defer os.Remove(tmp)

	if _, err := f.WriteString(Render(b)); err != nil {
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
	return ModTime(path)
}

// ModTime is the modification time of path, or the zero time if it is missing.
func ModTime(path string) (time.Time, error) {
	fi, err := os.Stat(path)
	if os.IsNotExist(err) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	return fi.ModTime(), nil
}
