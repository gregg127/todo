package main

import (
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// dbFile is the per-directory data file.
const dbFile = "todo-database.md"

// Modes are exclusive: in insert mode every printable key is text and only
// Enter and Esc are commands.
const (
	normalMode = iota
	insertMode
)

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
	// mode is normalMode or insertMode; input is the text being typed.
	mode  int
	input string
	// editing is the index of the task being edited, or -1 when the input
	// will create a new task at insertAt with status insertStatus.
	editing      int
	insertAt     int
	insertStatus Status
	// undo is a stack of task snapshots taken before each mutation. It is
	// in-memory only and is never written to disk.
	undo []Board
	// saved is the mtime the app itself last wrote, so the watcher can tell
	// its own writes from a hand-edit.
	saved time.Time
}

// push snapshots the current tasks so the mutation about to happen can be
// undone. No-op key presses must not call it.
func (m Model) push() Model {
	m.undo = append(append([]Board{}, m.undo...), m.tasks.clone())
	return m
}

// pop restores the last snapshot. On an empty stack nothing happens at all,
// on screen or on disk.
func (m Model) pop() Model {
	if len(m.undo) == 0 {
		return m
	}
	m.tasks = m.undo[len(m.undo)-1]
	m.undo = m.undo[:len(m.undo)-1]
	if n := len(m.visible()); m.cursor > n-1 {
		m.cursor = n - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	return m.save().scroll()
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

	m = m.push()
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
	m = m.push()
	tasks := m.tasks.clone()
	tasks[i], tasks[j] = tasks[j], tasks[i]
	m.tasks = tasks
	m.cursor = to
	return m.save().scroll()
}

// newTask opens the input for a task placed after (offset 1) or before
// (offset 0) the cursor, in the cursor's section. On an empty board it creates
// the first TODO task.
func (m Model) newTask(offset int) Model {
	visible := m.visible()
	if len(visible) == 0 {
		return m.insert(0, Todo)
	}
	i := visible[m.cursor]
	return m.insert(i+offset, m.tasks[i].Status)
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

// insert opens the one-line input to create a task at slice index at with
// status s.
func (m Model) insert(at int, s Status) Model {
	m.mode, m.input, m.editing = insertMode, "", -1
	m.insertAt, m.insertStatus = at, s
	return m
}

// edit opens the one-line input prefilled with the current task's text.
func (m Model) edit() Model {
	visible := m.visible()
	if m.cursor >= len(visible) {
		return m
	}
	m.mode, m.editing = insertMode, visible[m.cursor]
	m.input = m.tasks[m.editing].Title
	return m
}

// confirm applies the input. Empty or whitespace-only text creates nothing.
func (m Model) confirm() Model {
	title := strings.TrimSpace(m.input)
	m.mode, m.input = normalMode, ""
	if title == "" {
		return m
	}
	m = m.push()
	tasks := m.tasks.clone()
	if m.editing >= 0 {
		tasks[m.editing].Title = title
		m.tasks = tasks
		return m.save().scroll()
	}
	tasks = append(tasks, Task{})
	copy(tasks[m.insertAt+1:], tasks[m.insertAt:])
	tasks[m.insertAt] = Task{Title: title, Status: m.insertStatus}
	m.tasks = tasks
	return m.cursorTo(m.insertAt).save().scroll()
}

// remove deletes the task under the cursor. The cursor index is kept and
// clamped to the last visible task.
func (m Model) remove() Model {
	visible := m.visible()
	if m.cursor >= len(visible) {
		return m
	}
	m = m.push()
	i := visible[m.cursor]
	m.tasks = append(m.tasks.clone()[:i], m.tasks[i+1:]...)
	if m.cursor > len(m.tasks)-1 {
		m.cursor = len(m.tasks) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	return m.save().scroll()
}

// insertKey handles a key press while the one-line input is open.
func (m Model) insertKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "enter":
		return m.confirm(), nil
	case "esc":
		m.mode, m.input = normalMode, ""
		return m, nil
	case "backspace":
		if r := []rune(m.input); len(r) > 0 {
			m.input = string(r[:len(r)-1])
		}
		return m, nil
	case " ", "space":
		m.input += " "
		return m, nil
	}
	// Every other printable key is text, so a task can be called "quit the job".
	if r := []rune(k); len(r) == 1 {
		m.input += k
	}
	return m, nil
}

func (m Model) key(k string) (tea.Model, tea.Cmd) {
	if m.mode == insertMode {
		return m.insertKey(k)
	}

	if m.pending != "" {
		pending := m.pending
		m.pending = ""
		// A key that does not complete the sequence cancels it and is
		// swallowed, not re-dispatched.
		switch {
		case pending == "g" && k == "g":
			m.cursor = 0
		case pending == "d" && k == "d":
			return m.remove(), nil
		case pending == "c" && k == "c":
			return m.edit(), nil
		}
		return m.scroll(), nil
	}

	n := len(m.visible())
	switch k {
	case "q":
		return m, tea.Quit
	case "g", "d", "c":
		m.pending = k
	case "o":
		return m.newTask(1), nil
	case "O":
		return m.newTask(0), nil
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
	case "u":
		return m.pop(), nil
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
