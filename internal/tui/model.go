package tui

import (
	"os"
	"slices"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"todo/internal/board"
)

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
	tasks board.Board
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
	// blink is the caret's phase: on for one tick, off for the next. The
	// terminal's own cursor sits wherever the last rendered row ends, so the
	// input draws its own.
	blink bool
	// collapsed hides the contents of the DONE section. It starts on: finished
	// work is context, not the working list.
	collapsed bool
	// editing is the index of the task being edited, or -1 when the input
	// will create a new task at insertAt with status insertStatus.
	editing      int
	insertAt     int
	insertStatus board.Status
	// undo is a stack of task snapshots taken before each mutation, redo the
	// stack of states undone away from. Both are in-memory only and are never
	// written to disk.
	undo []board.Board
	redo []board.Board
	// saved is the mtime the app itself last wrote, so the watcher can tell
	// its own writes from a hand-edit.
	saved time.Time
	// readErr is set when a hand-edit left the file unreadable. Saving is off
	// while it is: the board on screen no longer describes the file, and
	// writing it would delete whatever the parser could not read.
	readErr string
}

// push snapshots the current tasks so the mutation about to happen can be
// undone. No-op key presses must not call it. A fresh mutation forks the
// history: whatever was undone away from is no longer reachable.
func (m Model) push() Model {
	m.undo = append(m.undo, m.tasks.Clone())
	m.redo = nil
	return m
}

// pop restores the last snapshot, keeping the state it left behind for redo.
// On an empty stack nothing happens, on screen or on disk.
func (m Model) pop() Model {
	if len(m.undo) == 0 {
		return m
	}
	m.redo = append(m.redo, m.tasks.Clone())
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
	m.undo = append(m.undo, m.tasks.Clone())
	m.tasks = m.redo[len(m.redo)-1]
	m.redo = m.redo[:len(m.redo)-1]
	return m.clampCursor().save().scroll()
}

func (m Model) save() Model {
	if m.readErr != "" {
		return m
	}
	if t, err := board.Save(m.path, m.tasks); err == nil {
		m.saved = t
	}
	return m
}

// selected is the index into m.tasks of the task under the cursor, false when
// the cursor is on no task — an empty board, or everything filtered away.
func (m Model) selected() (int, bool) {
	visible := m.visible()
	if m.cursor < 0 || m.cursor >= len(visible) {
		return 0, false
	}
	return visible[m.cursor], true
}

// setStatus moves the task under the cursor to s, landing it at the top of
// the target section, and keeps the cursor on it. Pressing the key for the
// section the task is already in hoists it to the top of that section, so the
// same key both moves a task across and to the front. A task already at the
// top of its own section has nowhere to go.
func (m Model) setStatus(s board.Status) Model {
	i, ok := m.selected()
	if !ok {
		return m
	}
	// The first task of the section, filter ignored: a hidden task still holds
	// the top of its section.
	first := slices.IndexFunc(m.tasks, func(t board.Task) bool { return t.Status == s })
	if m.tasks[i].Status == s && first == i {
		return m
	}

	m = m.push()
	t := m.tasks[i]
	t.Status = s
	// First in the slice is first within its section.
	m.tasks = append(board.Board{t}, slices.Delete(m.tasks.Clone(), i, i+1)...)
	m = m.cursorTo(0)
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
	tasks := m.tasks.Clone()
	tasks[i], tasks[j] = tasks[j], tasks[i]
	m.tasks = tasks
	m.cursor = to
	return m.save().scroll()
}

// newTask opens the input for a task placed after (offset 1) or before
// (offset 0) the cursor, in the cursor's section. On an empty board it creates
// the first TODO task.
func (m Model) newTask(offset int) Model {
	i, ok := m.selected()
	if !ok {
		return m.insert(0, board.Todo)
	}
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

// New builds a model over the board file at path, creating an empty one if it
// is not there yet. A file the app cannot read is an error and no model: the
// board never opens on a file it would damage by saving.
func New(path string) (Model, error) {
	tasks, err := board.Load(path)
	if err != nil {
		return Model{}, err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if _, err := board.Save(path, nil); err != nil {
			return Model{}, err
		}
	}
	return Model{path: path, tasks: tasks, saved: board.ModTime(path), collapsed: true}, nil
}

func (m Model) matches(t board.Task) bool {
	q := strings.ToLower(m.filter)
	return q == "" || strings.Contains(strings.ToLower(t.Title), q)
}

// count is how many tasks a section holds under the current filter, whether or
// not the section is collapsed.
func (m Model) count(s board.Status) int {
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
	for _, s := range board.Statuses {
		if s == board.Done && m.collapsed {
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
	sections := board.Statuses
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
		m.blink = !m.blink
		// The app's own writes update m.saved, so only somebody else's
		// write reloads the board.
		if m.changedOnDisk() {
			return m.reload(), tick()
		}
		return m, tick()
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.scroll(), nil
	case tea.MouseMsg:
		// Only a left press selects; the input is a text field, and moving the
		// cursor out from under it mid-edit would be a surprise.
		if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft || m.mode == insertMode {
			return m, nil
		}
		return m.click(msg.Y), nil
	case tea.KeyMsg:
		// A terminal paste (⌘V, ctrl+shift+V) arrives as one key event holding
		// the whole clipboard, not as key presses, so it never reaches typed().
		if msg.Paste {
			return m.pasted(string(msg.Runes)), nil
		}
		return m.key(msg.String())
	}
	return m, nil
}

// insert opens the one-line input for a new task at slice index at.
func (m Model) insert(at int, s board.Status) Model {
	m.mode, m.input, m.editing = insertMode, "", -1
	m.insertAt, m.insertStatus = at, s
	// The input opens with the caret up, not half a tick of nothing.
	m.blink = true
	return m
}

// edit opens the one-line input prefilled with the current task's text.
func (m Model) edit() Model {
	i, ok := m.selected()
	if !ok {
		return m
	}
	m.mode, m.editing = insertMode, i
	m.input = m.tasks[m.editing].Title
	m.blink = true
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
	tasks := m.tasks.Clone()
	if m.editing >= 0 {
		tasks[m.editing].Title = title
		m.tasks = tasks
		return m.save().scroll()
	}
	m.tasks = slices.Insert(tasks, m.insertAt, board.Task{Title: title, Status: m.insertStatus})
	return m.cursorTo(m.insertAt).save().scroll()
}

// remove deletes the task under the cursor, keeping the cursor index and
// clamping it to the last visible task.
func (m Model) remove() Model {
	i, ok := m.selected()
	if !ok {
		return m
	}
	m = m.push()
	m.tasks = slices.Delete(m.tasks.Clone(), i, i+1)
	return m.clampCursor().save().scroll()
}

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

// pasted appends clipboard text to the line being typed. A task title is one
// line, so a multi-line paste is flattened to spaces instead of being refused.
// Outside the input and the filter there is no line to paste into.
func (m Model) pasted(text string) Model {
	text = strings.Join(strings.Fields(text), " ")
	// Fields deals with the whitespace controls; drop the rest, or a paste could
	// smuggle escape sequences into a title and from there into the file. Strip
	// rather than refuse: there is nothing on disk yet to damage.
	text = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, text)
	switch m.mode {
	case insertMode:
		m.input += text
	case filterMode:
		m.filter += text
	}
	return m.clampCursor().scroll()
}

// typed applies a key press to a line being typed: backspace deletes the last
// rune, any single-rune key appends itself, everything else is ignored.
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

// Only Enter and Esc are commands; every printable key narrows the list.
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
		return m.setStatus(board.Todo), nil
	case "2":
		return m.setStatus(board.Doing), nil
	case "3":
		return m.setStatus(board.Done), nil
	}
	return m.scroll(), nil
}
