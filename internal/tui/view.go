package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"todo/internal/board"
)

const scrolloff = 3

const wheelRows = 3

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

// at maps each row to its index in the visible list, or -1 for a section header.
func (m Model) rows() (rows []string, at []int) {
	visible := m.visible()
	i := 0
	for _, s := range board.Statuses {
		header := fmt.Sprintf("  %s (%d)", board.SectionName(s), m.count(s))
		if s == board.Done && m.collapsed {
			header += " ▸"
		}
		rows, at = append(rows, headerStyle.Render(header)), append(at, -1)
		for _, idx := range visible {
			t := m.tasks[idx]
			if t.Status != s {
				continue
			}
			gutter := "  "
			if i == m.cursor {
				gutter = "▸ "
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
				prefix := "    "
				if j == 0 {
					prefix = gutter + statusDot(s) + " "
				}
				rows, at = append(rows, style.Render(prefix+part)), append(at, i)
			}
			i++
		}
	}
	return rows, at
}

// Text gets the pane minus the gutter (2) and the dot and its space (2).
func (m Model) fold(title string) []string {
	if m.width <= 4 {
		return []string{title}
	}
	return wrap(title, m.width-4)
}

// Width pads rows out; the padding is trimmed, or it shows as a block under the
// cursor's reverse video.
func wrap(s string, width int) []string {
	lines := strings.Split(lipgloss.NewStyle().Width(width).Render(s), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return lines
}

func (m Model) listHeight(total int) int {
	if m.height <= 0 {
		return total
	}
	h := m.height - len(m.hintLines()) - len(m.inputLines()) - 1
	if h < 1 {
		h = 1
	}
	return h
}

func (m Model) scroll() Model {
	rows, at := m.rows()
	h := m.listHeight(len(rows))
	cursorRow := slices.Index(at, m.cursor)
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

func (m Model) scrollBy(delta int) Model {
	rows, _ := m.rows()
	m.offset = min(max(m.offset+delta, 0), max(len(rows)-m.listHeight(len(rows)), 0))
	return m
}

func (m Model) click(y int) Model {
	rows, at := m.rows()
	row := m.offset + y
	// Bubble Tea normalises 1-based mouse coordinates by subtracting one and
	// does not clamp, so a row-0 report arrives as y = -1.
	if y < 0 || y >= m.listHeight(len(rows)) || row >= len(at) || at[row] < 0 {
		return m
	}
	m.cursor = at[row]
	return m.scroll()
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
	return body + "\n\n" + strings.Join(hints, "\n")
}

func (m Model) inputLines() []string {
	if m.mode != insertMode {
		return nil
	}
	// A placeholder holds the caret's column through the wrap, which trims the
	// trailing spaces a caret added afterwards would land before.
	r := []rune(m.input)
	under, rest := " ", ""
	if m.pos < len(r) {
		under, rest = string(r[m.pos]), string(r[m.pos+1:])
	}
	lines := []string{"› " + string(r[:m.pos]) + caretCell + rest}
	if m.width > 0 {
		lines = wrap(lines[0], m.width)
	}
	for i, line := range lines {
		if strings.Contains(line, caretCell) {
			lines[i] = strings.Replace(line, caretCell, m.caret(under), 1)
			break
		}
	}
	return lines
}

// Any width-one non-space rune would do; this one is visible if it escapes.
const caretCell = "\u2588"

func (m Model) caret(under string) string {
	if !m.blink {
		return under
	}
	return cursorStyle.Render(under)
}

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
