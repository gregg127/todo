package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"todo/internal/board"
)

const tickInterval = 500 * time.Millisecond

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) changedOnDisk() bool {
	return !board.ModTime(m.path).Equal(m.saved)
}

func (m Model) reload() Model {
	var title string
	if visible := m.visible(); m.cursor < len(visible) {
		title = m.tasks[visible[m.cursor]].Title
	}

	tasks, meta, err := board.Load(m.path)
	if err != nil {
		m.readErr = err.Error()
		return m
	}
	m.readErr = ""
	m.tasks = tasks
	m.collapsed = collapsedDone(meta, m.collapsed)
	m.meta = meta
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
