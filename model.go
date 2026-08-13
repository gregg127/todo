package main

import (
	"path/filepath"
	"time"

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
	// width and height come from tea.WindowSizeMsg; offset is the first
	// board row on screen.
	width, height, offset int
	// saved is the mtime the app itself last wrote, so the watcher can tell
	// its own writes from a hand-edit.
	saved time.Time
}

// save writes the board and records the resulting mtime.
func (m Model) save() Model {
	if t, err := Save(m.path, m.tasks); err == nil {
		m.saved = t
	}
	return m
}

// setStatus moves the task under the cursor to s, landing it at the bottom of
// the target section, and keeps the cursor on it. Moving a task to the section
// it is already in changes nothing at all.
func (m Model) setStatus(s Status) Model {
	visible := m.visible()
	if m.cursor >= len(visible) {
		return m
	}
	i := visible[m.cursor]
	if m.tasks[i].Status == s {
		return m
	}

	t := m.tasks[i]
	t.Status = s
	tasks := append(m.tasks.clone()[:i], m.tasks[i+1:]...)
	// Last in the slice is last within its section.
	m.tasks = append(tasks, t)
	m = m.cursorTo(len(m.tasks) - 1)
	return m.save().scroll()
}

// move swaps the task under the cursor with its neighbour delta rows away,
// but only when that neighbour is in the same section: reordering must never
// change a task's status.
func (m Model) move(delta int) Model {
	visible := m.visible()
	to := m.cursor + delta
	if m.cursor >= len(visible) || to < 0 || to >= len(visible) {
		return m
	}
	i, j := visible[m.cursor], visible[to]
	if m.tasks[i].Status != m.tasks[j].Status {
		return m
	}
	tasks := m.tasks.clone()
	tasks[i], tasks[j] = tasks[j], tasks[i]
	m.tasks = tasks
	m.cursor = to
	return m.save().scroll()
}

// cursorTo puts the cursor on the task at index i in the task slice.
func (m Model) cursorTo(i int) Model {
	for row, idx := range m.visible() {
		if idx == i {
			m.cursor = row
			break
		}
	}
	return m
}

// New builds a model rooted at dir, loading ./todo-database.md if it exists.
func New(dir string) Model {
	path := filepath.Join(dir, dbFile)
	tasks, _ := Load(path)
	return Model{path: path, tasks: tasks}
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

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.scroll(), nil
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
		return m.scroll(), nil
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
	case "J":
		return m.move(1), nil
	case "K":
		return m.move(-1), nil
	case "1":
		return m.setStatus(Todo), nil
	case "2":
		return m.setStatus(Doing), nil
	case "3":
		return m.setStatus(Done), nil
	}
	return m.scroll(), nil
}
