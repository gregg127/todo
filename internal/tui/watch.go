package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"todo/internal/board"
)

// pollInterval is how often the app checks the file's mtime for a hand-edit.
const pollInterval = time.Second

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(pollInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// changedOnDisk reports whether the file's mtime differs from the one the app
// recorded after its own last save, i.e. whether somebody else wrote the file.
func (m Model) changedOnDisk() bool {
	return !board.ModTime(m.path).Equal(m.saved)
}

// reload re-parses the board from disk after a hand-edit. It keeps the cursor
// on the task with the same title where it can and clamps the old index into
// the new list otherwise, and clears the undo stack: those snapshots describe
// a state the file no longer has.
func (m Model) reload() Model {
	var title string
	if visible := m.visible(); m.cursor < len(visible) {
		title = m.tasks[visible[m.cursor]].Title
	}

	tasks, err := board.Load(m.path)
	if err != nil {
		// Hold the board as it is and stop saving until the file reads again,
		// rather than rewrite it without the lines the parser cannot read.
		m.readErr = err.Error()
		return m
	}
	m.readErr = ""
	m.tasks = tasks
	m.undo, m.redo = nil, nil
	m.saved = board.ModTime(m.path)

	for row, i := range m.visible() {
		if m.tasks[i].Title == title {
			m.cursor = row
			return m.scroll()
		}
	}
	return m.clampCursor().scroll()
}
