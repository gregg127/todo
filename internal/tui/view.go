package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"todo/internal/board"
)

// scrolloff is the number of context rows kept above and below the cursor.
const scrolloff = 3

// wheelRows is how far one wheel notch moves the viewport.
const wheelRows = 3

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
// which visible task each row belongs to: an index into the visible list, or
// -1 for a section header. A folded title owns every row it spans.
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
				// Continuation rows line up under the first one's text.
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
	// The hint bar, its blank spacer, and the input when it is open, cost
	// their rows.
	h := m.height - len(m.hintLines()) - len(m.inputLines()) - 1
	if h < 1 {
		h = 1
	}
	return h
}

// scroll moves the viewport the minimum distance needed to keep the cursor on
// screen with scrolloff rows of context where the list allows it.
func (m Model) scroll() Model {
	rows, at := m.rows()
	h := m.listHeight(len(rows))
	// The cursor's first row, -1 when it is on no task — an empty board, or
	// everything filtered away.
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

// scrollBy moves the viewport delta rows, stopping at the ends of the board. It
// leaves the cursor alone: looking around is not selecting, and the next key
// press scrolls the view back to wherever the cursor was left.
func (m Model) scrollBy(delta int) Model {
	rows, _ := m.rows()
	m.offset = min(max(m.offset+delta, 0), max(len(rows)-m.listHeight(len(rows)), 0))
	return m
}

// click puts the cursor on the task at screen row y. A click on a section
// header, on a blank row, or anywhere below the board selects nothing: the
// cursor stays where it is.
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
	// A blank row sets the hint bar off from the board.
	return body + "\n\n" + strings.Join(hints, "\n")
}

// inputLines is the open one-line input, folded onto as many rows as the text
// needs: a title being typed is never cut off, the board gives up the rows
// instead. Empty unless the input is open.
func (m Model) inputLines() []string {
	if m.mode != insertMode {
		return nil
	}
	// A placeholder rides through the wrap in the caret's column: wrap trims
	// trailing spaces, so a caret added afterwards would land before the
	// spaces just typed, or be trimmed away with them.
	lines := []string{"› " + m.input + caretCell}
	if m.width > 0 {
		lines = wrap(lines[0], m.width)
	}
	last := len(lines) - 1
	lines[last] = strings.TrimSuffix(lines[last], caretCell) + m.caret()
	return lines
}

// caretCell holds the caret's column through the wrap. Any width-one non-space
// rune would do; this one is visible if it ever escapes.
const caretCell = "\u2588"

// caret is the block cursor marking where the next rune will land: a reversed
// space, blinking by falling back to a plain one, so the row is the same width
// either way and text never shifts.
func (m Model) caret() string {
	if !m.blink {
		return " "
	}
	return cursorStyle.Render(" ")
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
