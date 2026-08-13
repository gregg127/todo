package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// send feeds keys through Update. Each argument is either a special key name
// ("enter", "esc", "space", "backspace") or literal text, in which case every
// rune is sent as its own key press.
func send(m Model, keys ...string) Model {
	m, _ = sendCmd(m, keys...)
	return m
}

// sendCmd is send, also returning the command from the last key press.
func sendCmd(m Model, keys ...string) (Model, tea.Cmd) {
	var cmd tea.Cmd
	for _, k := range keys {
		for _, msg := range keyMsgs(k) {
			var tm tea.Model
			tm, cmd = m.Update(msg)
			m = tm.(Model)
		}
	}
	return m, cmd
}

func keyMsgs(k string) []tea.KeyMsg {
	switch k {
	case "enter":
		return []tea.KeyMsg{{Type: tea.KeyEnter}}
	case "esc":
		return []tea.KeyMsg{{Type: tea.KeyEsc}}
	case "space":
		return []tea.KeyMsg{{Type: tea.KeySpace, Runes: []rune{' '}}}
	case "backspace":
		return []tea.KeyMsg{{Type: tea.KeyBackspace}}
	}
	msgs := make([]tea.KeyMsg, 0, len(k))
	for _, r := range k {
		msgs = append(msgs, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return msgs
}

// file returns the contents of the todo file in dir, or "" if there is none.
func file(t *testing.T, dir string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, dbFile))
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestEmptyBoardShowsAllThreeSectionsInOrder(t *testing.T) {
	v := New(t.TempDir()).View()

	todo := strings.Index(v, "TODO (0)")
	doing := strings.Index(v, "DOING (0)")
	done := strings.Index(v, "DONE (0)")
	if todo < 0 || doing < 0 || done < 0 {
		t.Fatalf("missing section headers in view:\n%s", v)
	}
	if !(todo < doing && doing < done) {
		t.Fatalf("sections out of order in view:\n%s", v)
	}
}

func TestQuitsOnQ(t *testing.T) {
	_, cmd := sendCmd(New(t.TempDir()), "q")
	if cmd == nil {
		t.Fatal("q produced no command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("q did not quit, got %T", cmd())
	}
}

func TestLaunchAndQuitWritesNothing(t *testing.T) {
	dir := t.TempDir()

	send(New(dir), "j", "k", "G", "q")

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("directory not left untouched: %v", entries)
	}
}

// write puts contents into dir's todo file.
func write(t *testing.T, dir, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, dbFile), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestExistingFileIsLoadedOntoTheBoard(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] write tests\n- [ ] ship it\n\n## DOING\n- [ ] refactor\n\n## DONE\n- [x] design\n")

	v := New(dir).View()

	for _, want := range []string{"TODO (2)", "DOING (1)", "DONE (1)", "○ write tests", "○ ship it", "◐ refactor", "● design"} {
		if !strings.Contains(v, want) {
			t.Fatalf("view missing %q:\n%s", want, v)
		}
	}
}

func TestTickedBoxUnderTodoLoadsAsTodo(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [x] hand ticked\n")

	v := New(dir).View()

	if !strings.Contains(v, "TODO (1)") || !strings.Contains(v, "○ hand ticked") {
		t.Fatalf("view did not treat the ticked item as TODO:\n%s", v)
	}
}

func TestEmptyFileIsAnEmptyBoard(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "")

	v := New(dir).View()

	if !strings.Contains(v, "TODO (0)") || !strings.Contains(v, "DONE (0)") {
		t.Fatalf("empty file did not yield an empty board:\n%s", v)
	}
}

// cursorLine returns the task line marked with the cursor gutter, or "" if
// nothing is marked.
func cursorLine(t *testing.T, m Model) string {
	t.Helper()
	for _, line := range strings.Split(m.View(), "\n") {
		if strings.HasPrefix(line, "▸") {
			return strings.TrimSpace(strings.TrimPrefix(line, "▸"))
		}
	}
	return ""
}

// board3 is a board with tasks in all three sections.
const board3 = "## TODO\n- [ ] one\n- [ ] two\n\n## DOING\n- [ ] three\n\n## DONE\n- [x] four\n"

func newBoard3(t *testing.T) (Model, string) {
	t.Helper()
	dir := t.TempDir()
	write(t, dir, board3)
	return New(dir), dir
}

func TestCursorStartsOnTheFirstTask(t *testing.T) {
	m, _ := newBoard3(t)
	if got := cursorLine(t, m); got != "○ one" {
		t.Fatalf("cursor on %q, want %q", got, "○ one")
	}
}

func TestCursorMovesAcrossSectionBoundaries(t *testing.T) {
	m, _ := newBoard3(t)

	for _, want := range []string{"○ two", "◐ three", "● four"} {
		m = send(m, "j")
		if got := cursorLine(t, m); got != want {
			t.Fatalf("after j cursor on %q, want %q", got, want)
		}
	}
	for _, want := range []string{"◐ three", "○ two", "○ one"} {
		m = send(m, "k")
		if got := cursorLine(t, m); got != want {
			t.Fatalf("after k cursor on %q, want %q", got, want)
		}
	}
}

func TestCursorDoesNotWrap(t *testing.T) {
	m, _ := newBoard3(t)

	m = send(m, "k", "k")
	if got := cursorLine(t, m); got != "○ one" {
		t.Fatalf("k on the first task moved to %q", got)
	}

	m = send(m, "G", "j", "j")
	if got := cursorLine(t, m); got != "● four" {
		t.Fatalf("j on the last task moved to %q", got)
	}
}

func TestGGAndG(t *testing.T) {
	m, _ := newBoard3(t)

	m = send(m, "G")
	if got := cursorLine(t, m); got != "● four" {
		t.Fatalf("G went to %q, want the last task", got)
	}
	m = send(m, "gg")
	if got := cursorLine(t, m); got != "○ one" {
		t.Fatalf("gg went to %q, want the first task", got)
	}
}

func TestPendingGIsCancelledAndSwallowsTheNextKey(t *testing.T) {
	m, _ := newBoard3(t)

	// g then j: the sequence is cancelled and the j is discarded.
	m = send(m, "g", "j")
	if got := cursorLine(t, m); got != "○ one" {
		t.Fatalf("cancelling key was re-dispatched, cursor on %q", got)
	}
	// The pending prefix is gone, so the next j moves normally.
	m = send(m, "j")
	if got := cursorLine(t, m); got != "○ two" {
		t.Fatalf("after a cancelled sequence j moved to %q", got)
	}
}

func TestPendingPrefixSurvivesUnrelatedMessages(t *testing.T) {
	m, _ := newBoard3(t)
	m = send(m, "G", "g")

	// No timer: any number of non-key messages leave the g pending.
	tm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tm.(Model)

	m = send(m, "g")
	if got := cursorLine(t, m); got != "○ one" {
		t.Fatalf("pending g did not survive, cursor on %q", got)
	}
}

func TestCursorMovementOnAnEmptyBoardIsHarmless(t *testing.T) {
	dir := t.TempDir()
	m := send(New(dir), "j", "k", "G", "gg")

	if got := cursorLine(t, m); got != "" {
		t.Fatalf("empty board rendered a cursor line %q", got)
	}
	if !strings.Contains(m.View(), "TODO (0)") {
		t.Fatalf("empty board view broke:\n%s", m.View())
	}
}
