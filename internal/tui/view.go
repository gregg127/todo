package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"todo/internal/board"
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

const insertHints = "enter confirm · esc cancel"

const normalHints = "j/k or ↑/↓ move · {/} section · 1/2/3 status or top · J/K reorder · o/O add · cc edit · dd delete · C toggle DONE · u undo · r redo · / filter · q quit"

// rows renders every board row — section headers and task lines — and reports
// which row the cursor is on (-1 when the board is empty).
func (m Model) rows() (rows []string, cursorRow int) {
	cursorRow = -1
	visible := m.visible()
	i := 0
	for _, s := range board.Statuses {
		header := fmt.Sprintf("  %s (%d)", board.SectionName(s), m.count(s))
		if s == board.Done && m.collapsed {
			header += " ▸"
		}
		rows = append(rows, headerStyle.Render(header))
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
			style := todoStyle
			switch s {
			case board.Doing:
				style = doingStyle
			case board.Done:
				style = doneStyle
			}
			if i == m.cursor {
				style = cursorStyle
			}
			for j, part := range m.fold(t.Title) {
				// Continuation rows line up under the first one's text.
				prefix := "    "
				if j == 0 {
					prefix = gutter + statusDot(s) + " "
				}
				rows = append(rows, style.Render(prefix+part))
			}
			i++
		}
	}
	return rows, cursorRow
}

// fold breaks a title onto as many rows as it needs, so a long task is read in
// full instead of being cut off. The width left for text is the pane minus the
// gutter (2) and the dot and its space (2).
func (m Model) fold(title string) []string {
	if m.width <= 4 {
		return []string{title}
	}
	return wrap(title, m.width-4)
}

// wrap breaks s onto rows at most width runes wide, at spaces where it can and
// mid-word where a single word is wider than the pane. Width pads every row out
// to the full width; the padding is trimmed, or it would show as a block under
// the cursor's reverse video.
func wrap(s string, width int) []string {
	lines := strings.Split(lipgloss.NewStyle().Width(width).Render(s), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return lines
}

// listHeight is the number of rows available to the board, the pane minus the
// hint bar. A height of zero (no tea.WindowSizeMsg yet) means unlimited.
func (m Model) listHeight(total int) int {
	if m.height <= 0 {
		return total
	}
	// The hint bar, and the input when it is open, cost their rows.
	h := m.height - len(m.hintLines()) - len(m.inputLines())
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
	body := strings.Join(rows[m.offset:end], "\n")
	for _, line := range m.inputLines() {
		body += "\n" + line
	}
	var hints []string
	for _, line := range m.hintLines() {
		hints = append(hints, hintStyle.Render(line))
	}
	return body + "\n" + strings.Join(hints, "\n")
}

// inputLines is the open one-line input, folded onto as many rows as the text
// needs: a title being typed is never cut off, the board gives up the rows
// instead. Empty unless the input is open.
func (m Model) inputLines() []string {
	if m.mode != insertMode {
		return nil
	}
	line := "› " + m.input
	if m.width <= 0 {
		return []string{line}
	}
	return wrap(line, m.width)
}

// hintLines wraps the hint bar onto as many rows as it needs.
func (m Model) hintLines() []string {
	if m.width <= 0 {
		return []string{m.hints()}
	}
	return wrap(m.hints(), m.width)
}

func (m Model) hints() string {
	if m.readErr != "" {
		return "not saving — " + m.readErr
	}
	switch m.mode {
	case insertMode:
		return insertHints
	case filterMode:
		return "/" + m.filter
	}
	if m.filter != "" {
		return fmt.Sprintf("filter: %q · esc clear · %s", m.filter, normalHints)
	}
	return normalHints
}

func statusDot(s board.Status) string {
	switch s {
	case board.Doing:
		return "◐"
	case board.Done:
		return "●"
	default:
		return "○"
	}
}
