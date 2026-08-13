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
