package main

import (
	"os"
	"strings"
)

// Parse reads the recognised structure out of a Markdown board: the exact
// headings `## TODO`, `## DOING` and `## DONE`, and `- [ ]` / `- [x]` list
// items beneath them. Everything else — prose, blank lines, other headings,
// items before the first heading — is dropped. The section wins over the
// checkbox, so `- [x]` under `## TODO` is a TODO task.
func Parse(s string) Board {
	var b Board
	section := Status(-1)
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSuffix(line, "\r")
		switch strings.TrimRight(line, " \t") {
		case "## TODO":
			section = Todo
			continue
		case "## DOING":
			section = Doing
			continue
		case "## DONE":
			section = Done
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
		sb.WriteString("## " + sectionName(s) + "\n")
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
