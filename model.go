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
}

// New builds a model rooted at dir.
func New(dir string) Model {
	return Model{path: filepath.Join(dir, dbFile)}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder
	for _, s := range []Status{Todo, Doing, Done} {
		fmt.Fprintf(&b, "%s (%d)\n", sectionName(s), m.tasks.count(s))
	}
	return b.String()
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
