package main

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// dbFile is the per-directory data file.
const dbFile = "todo-database.md"

// Model is the whole application state; Update is the single reducer.
type Model struct {
	path  string
	tasks Board
	// cursor indexes the visible task list, not the task slice.
	cursor int
	// pending holds an incomplete key sequence (`g`, `d`, `c`). It has no
	// timeout: it stays pending until the next key completes or cancels it.
	pending string
}

// visible lists indexes into m.tasks in display order: TODO, then DOING, then
// DONE, keeping the slice order within each section.
func (m Model) visible() []int {
	var out []int
	for _, s := range []Status{Todo, Doing, Done} {
		for i, t := range m.tasks {
			if t.Status == s {
				out = append(out, i)
			}
		}
	}
	return out
}

// New builds a model rooted at dir, loading ./todo-database.md if it exists.
func New(dir string) Model {
	path := filepath.Join(dir, dbFile)
	tasks, _ := Load(path)
	return Model{path: path, tasks: tasks}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.key(msg.String())
	}
	return m, nil
}

func (m Model) key(k string) (tea.Model, tea.Cmd) {
	if m.pending != "" {
		pending := m.pending
		m.pending = ""
		// A key that does not complete the sequence cancels it and is
		// swallowed, not re-dispatched.
		if pending == "g" && k == "g" {
			m.cursor = 0
		}
		return m, nil
	}

	n := len(m.visible())
	switch k {
	case "q":
		return m, tea.Quit
	case "g":
		m.pending = "g"
	case "j":
		if m.cursor < n-1 {
			m.cursor++
		}
	case "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "G":
		if n > 0 {
			m.cursor = n - 1
		}
	}
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder
	row := 0
	for _, s := range []Status{Todo, Doing, Done} {
		fmt.Fprintf(&b, "  %s (%d)\n", sectionName(s), m.tasks.count(s))
		for _, i := range m.visible() {
			t := m.tasks[i]
			if t.Status != s {
				continue
			}
			gutter := "  "
			if row == m.cursor {
				gutter = "▸ "
			}
			fmt.Fprintf(&b, "%s%s %s\n", gutter, statusDot(s), t.Title)
			row++
		}
	}
	return b.String()
}

func statusDot(s Status) string {
	switch s {
	case Doing:
		return "◐"
	case Done:
		return "●"
	default:
		return "○"
	}
}

func sectionName(s Status) string {
	switch s {
	case Doing:
		return "DOING"
	case Done:
		return "DONE"
	default:
		return "TODO"
	}
}
