package tui

import (
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"todo/internal/board"
)

const (
	normalMode = iota
	insertMode
	filterMode
)

type Model struct {
	path  string
	tasks board.Board
	// cursor indexes the visible task list, not the task slice.
	cursor                int
	pending               string
	width, height, offset int
	mode                  int
	input                 string
	pos                   int
	filter                string
	blink                 bool
	typing                bool
	collapsed             bool
	meta                  board.Meta
	// editing is the index of the task being edited, or -1 when the input
	// will create a new task at insertAt with status insertStatus.
	editing      int
	insertAt     int
	insertStatus board.Status
	undo         []board.Board
	redo         []board.Board
	// The mtime the app itself last wrote, so the watcher can tell its own
	// writes from a hand-edit.
	saved   time.Time
	readErr string
}

// No-op key presses must not call this, or undo has nothing to do.
func (m Model) push() Model {
	m.undo = append(m.undo, m.tasks.Clone())
	m.redo = nil
	return m
}

func (m Model) pop() Model {
	if len(m.undo) == 0 {
		return m
	}
	m.redo = append(m.redo, m.tasks.Clone())
	m.tasks = m.undo[len(m.undo)-1]
	m.undo = m.undo[:len(m.undo)-1]
	return m.clampCursor().save().scroll()
}

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
	if t, err := board.Save(m.path, m.tasks, m.meta); err == nil {
		m.saved = t
	}
	return m
}

func (m Model) selected() (int, bool) {
	visible := m.visible()
	if m.cursor < 0 || m.cursor >= len(visible) {
		return 0, false
	}
	return visible[m.cursor], true
}

func (m Model) setStatus(s board.Status) Model {
	i, ok := m.selected()
	if !ok {
		return m
	}
	// Filter ignored: a hidden task still holds the top of its section.
	first := slices.IndexFunc(m.tasks, func(t board.Task) bool { return t.Status == s })
	if m.tasks[i].Status == s && first == i {
		return m
	}

	m = m.push()
	t := m.tasks[i]
	t.Status = s
	m.tasks = append(board.Board{t}, slices.Delete(m.tasks.Clone(), i, i+1)...)
	m = m.cursorTo(0)
	return m.save().scroll()
}

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

func (m Model) newTask(offset int) Model {
	i, ok := m.selected()
	if !ok {
		return m.insert(0, board.Todo)
	}
	return m.insert(i+offset, m.tasks[i].Status)
}

func (m Model) cursorTo(i int) Model {
	for row, idx := range m.visible() {
		if idx == i {
			m.cursor = row
			return m
		}
	}
	return m.clampCursor()
}

func New(path string) (Model, error) {
	tasks, meta, err := board.Load(path)
	if err != nil {
		return Model{}, err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if _, err := board.Save(path, nil, nil); err != nil {
			return Model{}, err
		}
	}
	m := Model{path: path, tasks: tasks, meta: meta, saved: board.ModTime(path)}
	m.collapsed = collapsedDone(meta, true)
	return m, nil
}

func collapsedDone(meta board.Meta, fallback bool) bool {
	if v, ok := meta[board.CollapsedDone]; ok {
		return v == "true"
	}
	return fallback
}

func (m Model) matches(t board.Task) bool {
	q := strings.ToLower(m.filter)
	return q == "" || strings.Contains(strings.ToLower(t.Title), q)
}

func (m Model) count(s board.Status) int {
	n := 0
	for _, t := range m.tasks {
		if t.Status == s && m.matches(t) {
			n++
		}
	}
	return n
}

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
		m.blink, m.typing = m.typing || !m.blink, false
		if m.changedOnDisk() {
			return m.reload(), tick()
		}
		return m, tick()
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.scroll(), nil
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			return m.scrollBy(-wheelRows), nil
		case tea.MouseButtonWheelDown:
			return m.scrollBy(wheelRows), nil
		case tea.MouseButtonLeft:
			if m.mode == insertMode {
				return m, nil
			}
			return m.click(msg.Y), nil
		}
		return m, nil
	case tea.KeyMsg:
		// A terminal paste arrives as one key event holding the whole clipboard,
		// not as key presses, so it never reaches typed().
		if msg.Paste {
			return m.pasted(string(msg.Runes)), nil
		}
		return m.key(msg.String())
	}
	return m, nil
}

func (m Model) insert(at int, s board.Status) Model {
	m.mode, m.input, m.pos, m.editing = insertMode, "", 0, -1
	m.insertAt, m.insertStatus = at, s
	m.blink = true
	return m
}

func (m Model) edit() Model {
	i, ok := m.selected()
	if !ok {
		return m
	}
	m.mode, m.editing = insertMode, i
	m.input = m.tasks[m.editing].Title
	m.pos = len([]rune(m.input))
	m.blink = true
	return m
}

func (m Model) confirm() Model {
	title := strings.TrimSpace(m.input)
	m.mode, m.input, m.pos = normalMode, "", 0
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
		m.mode, m.input, m.pos = normalMode, "", 0
		return m, nil
	}
	m.input, m.pos = typed(m.input, k, m.pos)
	m.blink, m.typing = true, true
	return m, nil
}

func (m Model) pasted(text string) Model {
	text = strings.Join(strings.Fields(text), " ")
	// Fields deals with the whitespace controls; drop the rest, or a paste
	// smuggles escape sequences into a title and from there into the file.
	text = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, text)
	switch m.mode {
	case insertMode:
		r := []rune(m.input)
		m.input = string(r[:m.pos]) + text + string(r[m.pos:])
		m.pos += len([]rune(text))
	case filterMode:
		m.filter += text
	}
	return m.clampCursor().scroll()
}

func typed(s, k string, pos int) (string, int) {
	r := []rune(s)
	switch k {
	case "left":
		if pos > 0 {
			pos--
		}
	case "right":
		if pos < len(r) {
			pos++
		}
	case "backspace":
		if pos > 0 {
			return string(slices.Delete(r, pos-1, pos)), pos - 1
		}
	default:
		if k := []rune(k); len(k) == 1 {
			return string(slices.Insert(r, pos, k[0])), pos + 1
		}
	}
	return s, pos
}

func (m Model) filterKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "enter":
		m.mode = normalMode
		return m.clampCursor().scroll(), nil
	case "esc":
		m.mode, m.filter = normalMode, ""
		return m.clampCursor().scroll(), nil
	}
	m.filter, _ = typed(m.filter, k, len([]rune(m.filter)))
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
		// A key that does not complete the sequence is swallowed, not
		// re-dispatched.
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
		// Written only here, so a board nobody has folded keeps no metadata.
		m.meta = maps.Clone(m.meta)
		if m.meta == nil {
			m.meta = board.Meta{}
		}
		m.meta[board.CollapsedDone] = strconv.FormatBool(m.collapsed)
		return m.save().clampCursor().scroll(), nil
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
