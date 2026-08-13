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

const insertHints = "enter confirm · esc cancel"

const normalHints = "j/k or ↑/↓ move · {/} section · 1/2/3 change status · J/K reorder · o/O add · cc edit · dd delete · C toggle DONE · u undo · r redo · / filter · q quit"

// rows renders every board row — section headers and task lines — and reports
// which row the cursor is on (-1 when the board is empty).
func (m Model) rows() (rows []string, cursorRow int) {
	cursorRow = -1
	visible := m.visible()
	i := 0
	for _, s := range []Status{Todo, Doing, Done} {
		header := fmt.Sprintf("  %s (%d)", sectionName(s), m.count(s))
		if s == Done && m.collapsed {
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
	start := m.offset
	if start > len(rows) {
		start = len(rows)
	}
	body := strings.Join(rows[start:end], "\n")
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
// needs: a task title being typed is never cut off, the board gives up the
// rows instead. It is empty unless the input is open.
func (m Model) inputLines() []string {
	if m.mode != insertMode {
		return nil
	}
	line := "› " + m.input
	if m.width <= 0 {
		return []string{line}
	}
	return strings.Split(lipgloss.NewStyle().Width(m.width).Render(line), "\n")
}

// hintLines wraps the hint bar onto as many rows as it needs, breaking at the
// " · " separators so no hint is split in half.
func (m Model) hintLines() []string {
	hints := m.hints()
	if m.width <= 0 || len([]rune(hints)) <= m.width {
		return []string{hints}
	}
	var lines []string
	cur := ""
	for _, seg := range strings.Split(hints, " · ") {
		switch {
		case cur == "":
			cur = seg
		case len([]rune(cur))+3+len([]rune(seg)) <= m.width:
			cur += " · " + seg
		default:
			lines = append(lines, cur)
			cur = seg
		}
	}
	lines = append(lines, cur)
	// A single hint wider than the pane — a long filter query, say — is cut, not
	// wrapped: the hint bar must not grow past the rows listHeight budgeted it.
	for i, line := range lines {
		if r := []rune(line); len(r) > m.width {
			lines[i] = string(r[:m.width])
		}
	}
	return lines
}

func (m Model) hints() string {
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
