package tui

import (
	"todo/internal/board"

	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// pollInterval is how often the app checks the file's mtime for a hand-edit.
const pollInterval = time.Second

// tickMsg drives the mtime poll.
type tickMsg time.Time

// tick schedules the next mtime poll.
func tick() tea.Cmd {
	return tea.Tick(pollInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// changedOnDisk reports whether the file's mtime differs from the one the app
// recorded after its own last save, i.e. whether somebody else wrote the file.
func (m Model) changedOnDisk() bool {
	t, err := board.ModTime(m.path)
	if err != nil {
		return false
	}
	return !t.Equal(m.saved)
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
		return m
	}
	m.tasks = tasks
	m.undo, m.redo = nil, nil
	m.saved, _ = board.ModTime(m.path)

	for row, i := range m.visible() {
		if m.tasks[i].Title == title {
			m.cursor = row
			return m.scroll()
		}
	}
	return m.clampCursor().scroll()
}
