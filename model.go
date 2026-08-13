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
	filterMode
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
	// mode is normalMode, insertMode or filterMode; input is the text being
	// typed in insert mode and filter is the live filter query.
	mode   int
	input  string
	filter string
	// collapsed hides the contents of the DONE section. It starts on: finished
	// work is context, not the working list.
	collapsed bool
	// editing is the index of the task being edited, or -1 when the input
	// will create a new task at insertAt with status insertStatus.
	editing      int
	insertAt     int
	insertStatus Status
	// undo is a stack of task snapshots taken before each mutation, redo the
	// stack of states undone away from. Both are in-memory only and are never
	// written to disk.
	undo []Board
	redo []Board
	// saved is the mtime the app itself last wrote, so the watcher can tell
	// its own writes from a hand-edit.
	saved time.Time
}

// push snapshots the current tasks so the mutation about to happen can be
// undone. No-op key presses must not call it. A fresh mutation forks the
// history, so whatever was undone away from is no longer reachable.
func (m Model) push() Model {
	m.undo = append(m.undo, m.tasks.clone())
	m.redo = nil
	return m
}

// pop restores the last snapshot, keeping the state it left behind for redo.
// On an empty stack nothing happens at all, on screen or on disk.
func (m Model) pop() Model {
	if len(m.undo) == 0 {
		return m
	}
	m.redo = append(m.redo, m.tasks.clone())
	m.tasks = m.undo[len(m.undo)-1]
	m.undo = m.undo[:len(m.undo)-1]
	return m.clampCursor().save().scroll()
}

// unpop replays the last undone state, keeping the state it left behind for
// undo. On an empty stack nothing happens at all.
func (m Model) unpop() Model {
	if len(m.redo) == 0 {
		return m
	}
	m.undo = append(m.undo, m.tasks.clone())
	m.tasks = m.redo[len(m.redo)-1]
	m.redo = m.redo[:len(m.redo)-1]
	return m.clampCursor().save().scroll()
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
			return m
		}
	}
	// The task is off the board — collapsed away, say. Keep the cursor legal.
	return m.clampCursor()
}

// New builds a model rooted at dir, loading ./todo-database.md if it exists.
func New(dir string) Model {
	path := filepath.Join(dir, dbFile)
	tasks, _ := Load(path)
	saved, _ := mtime(path)
	return Model{path: path, tasks: tasks, saved: saved, collapsed: true}
}

// matches reports whether t survives the current filter.
func (m Model) matches(t Task) bool {
	q := strings.ToLower(m.filter)
	return q == "" || strings.Contains(strings.ToLower(t.Title), q)
}

// count is how many tasks a section holds under the current filter, whether or
// not the section is collapsed.
func (m Model) count(s Status) int {
	n := 0
	for _, t := range m.tasks {
		if t.Status == s && m.matches(t) {
			n++
		}
	}
	return n
}

// visible lists indexes into m.tasks in display order: TODO, then DOING, then
// DONE, keeping the slice order within each section. A collapsed DONE section
// contributes nothing: its tasks are off the board and out of reach of the
// cursor until it is expanded again.
func (m Model) visible() []int {
	var out []int
	for _, s := range []Status{Todo, Doing, Done} {
		if s == Done && m.collapsed {
			continue
		}
		for i, t := range m.tasks {
			if t.Status != s || !m.matches(t) {
				continue
			}
			out = append(out, i)
		}
	}
	return out
}

// jumpSection puts the cursor on the first task of the nearest section delta
// steps away that has any visible tasks, skipping empty ones. At the end of
// the board nothing moves.
func (m Model) jumpSection(delta int) Model {
	visible := m.visible()
	if m.cursor >= len(visible) {
		return m
	}
	sections := []Status{Todo, Doing, Done}
	at := 0
	for i, s := range sections {
		if s == m.tasks[visible[m.cursor]].Status {
			at = i
		}
	}
	for at += delta; at >= 0 && at < len(sections); at += delta {
		for row, i := range visible {
			if m.tasks[i].Status == sections[at] {
				m.cursor = row
				return m.scroll()
			}
		}
	}
	return m
}

// clampCursor keeps the cursor inside the visible list, which the filter can
// narrow at any keystroke.
func (m Model) clampCursor() Model {
	if n := len(m.visible()); m.cursor > n-1 {
		m.cursor = n - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	return m
}

func (m Model) Init() tea.Cmd { return tick() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		// The app's own writes update m.saved, so only somebody else's
		// write reloads the board.
		if m.changedOnDisk() {
			return m.reload(), tick()
		}
		return m, tick()
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
	return m.clampCursor().save().scroll()
}

// insertKey handles a key press while the one-line input is open.
func (m Model) insertKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "enter":
		return m.confirm(), nil
	case "esc":
		m.mode, m.input = normalMode, ""
		return m, nil
	}
	// Every other printable key is text, so a task can be called "quit the job".
	m.input = typed(m.input, k)
	return m, nil
}

// typed applies a key press to a line of text being typed: backspace deletes
// the last rune, any single-rune key appends itself and everything else — the
// arrow keys, ctrl chords — is ignored.
func typed(s, k string) string {
	if k == "backspace" {
		if r := []rune(s); len(r) > 0 {
			return string(r[:len(r)-1])
		}
		return s
	}
	if r := []rune(k); len(r) == 1 {
		return s + k
	}
	return s
}

// filterKey handles a key press while the filter is being typed. Only Enter
// and Esc are commands; every printable key narrows the list further.
func (m Model) filterKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "enter":
		m.mode = normalMode
		return m.clampCursor().scroll(), nil
	case "esc":
		m.mode, m.filter = normalMode, ""
		return m.clampCursor().scroll(), nil
	}
	m.filter = typed(m.filter, k)
	return m.clampCursor().scroll(), nil
}

func (m Model) key(k string) (tea.Model, tea.Cmd) {
	switch m.mode {
	case insertMode:
		return m.insertKey(k)
	case filterMode:
		return m.filterKey(k)
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
	case "j", "down":
		if m.cursor < n-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "C":
		m.collapsed = !m.collapsed
		return m.clampCursor().scroll(), nil
	case "{":
		return m.jumpSection(-1), nil
	case "}":
		return m.jumpSection(1), nil
	case "G":
		if n > 0 {
			m.cursor = n - 1
		}
	case "/":
		m.mode, m.filter, m.cursor = filterMode, "", 0
		return m.scroll(), nil
	case "esc":
		m.filter = ""
		return m.clampCursor().scroll(), nil
	case "u":
		return m.pop(), nil
	case "r":
		return m.unpop(), nil
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
