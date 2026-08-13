package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// scrolloff is the number of context rows kept above and below the cursor.
const scrolloff = 3

// Styles use ANSI indices 1–15 only, so the terminal colorscheme restyles the
// app for free.
var (
	todoStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	doingStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	doneStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Faint(true)
	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	cursorStyle = lipgloss.NewStyle().Reverse(true)
	hintStyle   = lipgloss.NewStyle().Faint(true)
)

const normalHints = "j/k move · 1/2/3 status · J/K reorder · o/O add · cc edit · dd delete · u undo · / filter · q quit"

// rows renders every board row — section headers and task lines — and reports
// which row the cursor is on (-1 when the board is empty).
func (m Model) rows() (rows []string, cursorRow int) {
	cursorRow = -1
	visible := m.visible()
	i := 0
	for _, s := range []Status{Todo, Doing, Done} {
		n := 0
		for _, idx := range visible {
			if m.tasks[idx].Status == s {
				n++
			}
		}
		rows = append(rows, headerStyle.Render(fmt.Sprintf("  %s (%d)", sectionName(s), n)))
		for _, idx := range visible {
			t := m.tasks[idx]
			if t.Status != s {
				continue
			}
			gutter := "  "
			if i == m.cursor {
				gutter = "▸ "
				cursorRow = len(rows)
			}
			line := gutter + statusDot(s) + " " + m.truncate(t.Title)
			style := todoStyle
			switch s {
			case Doing:
				style = doingStyle
			case Done:
				style = doneStyle
			}
			if i == m.cursor {
				style = cursorStyle
			}
			rows = append(rows, style.Render(line))
			i++
		}
	}
	return rows, cursorRow
}

// truncate keeps every task on exactly one row.
func (m Model) truncate(title string) string {
	if m.width <= 0 {
		return title
	}
	// gutter (2) + dot and space (2)
	max := m.width - 4
	if max < 1 {
		max = 1
	}
	r := []rune(title)
	if len(r) <= max {
		return title
	}
	return string(r[:max-1]) + "…"
}

// listHeight is the number of rows available to the board, the pane minus the
// hint bar. A height of zero (no tea.WindowSizeMsg yet) means unlimited.
func (m Model) listHeight(total int) int {
	if m.height <= 0 {
		return total
	}
	h := m.height - 1
	if h < 1 {
		h = 1
	}
	return h
}

// scroll moves the viewport the minimum distance needed to keep the cursor on
// screen with scrolloff rows of context where the list allows it.
func (m Model) scroll() Model {
	rows, cursorRow := m.rows()
	h := m.listHeight(len(rows))
	if cursorRow < 0 || len(rows) <= h {
		m.offset = 0
		return m
	}
	if top := cursorRow - scrolloff; m.offset > top {
		m.offset = top
	}
	if bottom := cursorRow + scrolloff - h + 1; m.offset < bottom {
		m.offset = bottom
	}
	if m.offset > len(rows)-h {
		m.offset = len(rows) - h
	}
	if m.offset < 0 {
		m.offset = 0
	}
	return m
}

func (m Model) View() string {
	rows, _ := m.rows()
	h := m.listHeight(len(rows))
	end := m.offset + h
	if end > len(rows) {
		end = len(rows)
	}
	start := m.offset
	if start > len(rows) {
		start = len(rows)
	}
	body := strings.Join(rows[start:end], "\n")
	hints := m.hints()
	if m.width > 0 && len([]rune(hints)) > m.width {
		hints = string([]rune(hints)[:m.width-1]) + "…"
	}
	return body + "\n" + hintStyle.Render(hints)
}

func (m Model) hints() string {
	return normalHints
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
