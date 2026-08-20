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

const DefaultFile = "todo-database.md"

type Meta map[string]string

const CollapsedDone = "collapsed-done"

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

func Validate(text string) error {
	meta, body := splitMeta(text)
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
		// Titles reach the terminal as they are: refuse escape sequences rather
		// than strip them, which the next save would write back.
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

func Render(b Board, meta Meta) string {
	var sb strings.Builder
	if len(meta) > 0 {
		sb.WriteString("---\n")
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

func ModTime(path string) time.Time {
	fi, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return fi.ModTime()
}
