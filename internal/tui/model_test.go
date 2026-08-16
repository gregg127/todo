package tui

import (
	"todo/internal/board"

	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	b, err := os.ReadFile(filepath.Join(dir, board.DefaultFile))
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestEmptyBoardShowsAllThreeSectionsInOrder(t *testing.T) {
	v := open(t, t.TempDir()).View()

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
	_, cmd := sendCmd(open(t, t.TempDir()), "q")
	if cmd == nil {
		t.Fatal("q produced no command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("q did not quit, got %T", cmd())
	}
}

func TestLaunchCreatesTheFileAndQuittingChangesNothingElse(t *testing.T) {
	dir := t.TempDir()

	send(open(t, dir), "j", "k", "G", "q")

	if got := file(t, dir); got != emptyBoard {
		t.Fatalf("file = %q, want the empty board %q", got, emptyBoard)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("directory holds more than the board: %v", entries)
	}
}

// emptyBoard is the file the app writes for a board with no tasks.
const emptyBoard = "## TODO\n\n## DOING\n\n## DONE\n"

// open builds a model over dir's default todo file, failing the test if the
// file cannot be read.
func open(t *testing.T, dir string) Model {
	t.Helper()
	m, err := New(filepath.Join(dir, board.DefaultFile))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return m
}

func TestAMissingFileIsCreatedEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fresh.md")

	m, err := New(path)
	if err != nil {
		t.Fatalf("New on a missing file: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("New did not create the file: %v", err)
	}
	if string(got) != emptyBoard {
		t.Fatalf("created file = %q, want %q", got, emptyBoard)
	}
	if !strings.Contains(m.View(), "TODO (0)") {
		t.Fatalf("a created board is not empty:\n%s", m.View())
	}
}

func TestANamedFileIsReadAndWrittenInsteadOfTheDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "other.md")
	if err := os.WriteFile(path, []byte("## TODO\n- [ ] from the named file\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if !strings.Contains(m.View(), "○ from the named file") {
		t.Fatalf("the named file was not loaded:\n%s", m.View())
	}

	send(m, "o", "second", "enter")

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "- [ ] second") {
		t.Fatalf("the named file was not written: %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, board.DefaultFile)); !os.IsNotExist(err) {
		t.Fatalf("the default file was touched as well")
	}
}

func TestAFileTheBoardCannotReadIsRefusedAndLeftAlone(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.md")
	const notes = "# my notes\n\n## TODO\n- [ ] a task\n\nsome prose the board would drop\n"
	if err := os.WriteFile(path, []byte(notes), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := New(path); err == nil {
		t.Fatal("a file with unreadable lines opened anyway")
	} else if !strings.Contains(err.Error(), "line 1") {
		t.Fatalf("error does not point at the offending line: %v", err)
	}

	if got, _ := os.ReadFile(path); string(got) != notes {
		t.Fatalf("the refused file was rewritten: %q", got)
	}
}

// write puts contents into dir's todo file.
func write(t *testing.T, dir, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, board.DefaultFile), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestExistingFileIsLoadedOntoTheBoard(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] write tests\n- [ ] ship it\n\n## DOING\n- [ ] refactor\n\n## DONE\n- [x] design\n")

	v := send(open(t, dir), "C").View()

	for _, want := range []string{"TODO (2)", "DOING (1)", "DONE (1)", "○ write tests", "○ ship it", "◐ refactor", "● design"} {
		if !strings.Contains(v, want) {
			t.Fatalf("view missing %q:\n%s", want, v)
		}
	}
}

func TestTickedBoxUnderTodoLoadsAsTodo(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [x] hand ticked\n")

	v := open(t, dir).View()

	if !strings.Contains(v, "TODO (1)") || !strings.Contains(v, "○ hand ticked") {
		t.Fatalf("view did not treat the ticked item as TODO:\n%s", v)
	}
}

func TestEmptyFileIsAnEmptyBoard(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "")

	v := open(t, dir).View()

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

// newBoard3 expands DONE, which starts collapsed, so the cursor tests can walk
// the whole board.
func newBoard3(t *testing.T) (Model, string) {
	t.Helper()
	dir := t.TempDir()
	write(t, dir, board3)
	return send(open(t, dir), "C"), dir
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
	m := send(open(t, dir), "j", "k", "G", "gg")

	if got := cursorLine(t, m); got != "" {
		t.Fatalf("empty board rendered a cursor line %q", got)
	}
	if !strings.Contains(m.View(), "TODO (0)") {
		t.Fatalf("empty board view broke:\n%s", m.View())
	}
}

// resize sends a window size to the model.
func resize(m Model, w, h int) Model {
	tm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return tm.(Model)
}

// longBoard is a TODO section with n tasks named task-0…task-(n-1).
func longBoard(n int) string {
	var b strings.Builder
	b.WriteString("## TODO\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "- [ ] task-%d\n", i)
	}
	return b.String()
}

func viewLines(m Model) []string {
	return strings.Split(strings.TrimRight(m.View(), "\n"), "\n")
}

func TestViewportFitsTheWindowAndKeepsTheCursorOnScreen(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, longBoard(40))
	m := resize(open(t, dir), 60, 12)

	for i := 0; i < 40; i++ {
		if got := len(viewLines(m)); got > 12 {
			t.Fatalf("view is %d rows, window is 12", got)
		}
		if cursorLine(t, m) == "" {
			t.Fatalf("cursor scrolled off screen at task-%d:\n%s", i, m.View())
		}
		m = send(m, "j")
	}
}

func TestScrolloffKeepsThreeRowsOfContext(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, longBoard(40))
	m := resize(open(t, dir), 60, 14)
	m = send(m, "G")
	for i := 0; i < 20; i++ {
		m = send(m, "k")
	}

	lines := viewLines(m)
	cursor := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "▸") {
			cursor = i
		}
	}
	if cursor < scrolloff {
		t.Fatalf("only %d rows above the cursor:\n%s", cursor, m.View())
	}
	// The last line is the hint bar, so board rows below the cursor are
	// len(lines)-1-cursor-1.
	if below := len(lines) - 2 - cursor; below < scrolloff {
		t.Fatalf("only %d rows below the cursor:\n%s", below, m.View())
	}
}

func TestLongListScrollsAndNothingIsFolded(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, longBoard(40)+"\n## DONE\n- [x] finished\n")
	m := resize(open(t, dir), 60, 12)
	m = send(m, "C", "G")

	v := m.View()
	if !strings.Contains(v, "● finished") {
		t.Fatalf("the bottom of the board is not reachable:\n%s", v)
	}
	if strings.Contains(v, "task-0\n") {
		t.Fatalf("the top of the board did not scroll away:\n%s", v)
	}
}

func TestLongTitleFoldsOntoTheBoard(t *testing.T) {
	dir := t.TempDir()
	title := strings.TrimSpace(strings.Repeat("alpha bravo charlie ", 5))
	write(t, dir, "## TODO\n- [ ] "+title+"\n")
	m := resize(open(t, dir), 40, 20)

	lines := viewLines(m)
	for _, line := range lines {
		if len([]rune(line)) > 40 {
			t.Fatalf("row is %d cells wide, window is 40: %q", len([]rune(line)), line)
		}
	}
	if got := strings.Join(strings.Fields(strings.Join(lines, " ")), " "); !strings.Contains(got, title) {
		t.Fatalf("title was cut, not folded:\n%s", m.View())
	}
}

func TestAWordWiderThanThePaneIsBrokenNotDropped(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] "+strings.Repeat("x", 200)+"\n")
	m := resize(open(t, dir), 40, 20)

	lines := viewLines(m)
	n := 0
	for _, line := range lines {
		if len([]rune(line)) > 40 {
			t.Fatalf("row is %d cells wide, window is 40: %q", len([]rune(line)), line)
		}
		n += strings.Count(line, "x")
	}
	if n != 200 {
		t.Fatalf("%d of 200 x's on screen:\n%s", n, m.View())
	}
}

func TestLongInputFoldsOntoTheWindowAndStaysWhole(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, longBoard(40))
	m := resize(open(t, dir), 30, 12)

	text := strings.TrimSpace(strings.Repeat("alpha bravo charlie ", 6))
	m = send(m, "o", text)

	lines := viewLines(m)
	if got := len(lines); got > 12 {
		t.Fatalf("view is %d rows, window is 12:\n%s", got, m.View())
	}
	for _, line := range lines {
		if len([]rune(line)) > 30 {
			t.Fatalf("row is %d cells wide, window is 30: %q", len([]rune(line)), line)
		}
	}
	// Every word typed is still on screen, folded across rows rather than cut.
	onScreen := strings.Join(strings.Fields(strings.Join(lines, " ")), " ")
	if !strings.Contains(onScreen, text) {
		t.Fatalf("input was cut, not folded:\n%s", m.View())
	}
}

func TestHintBarIsTheBottomRow(t *testing.T) {
	m, _ := newBoard3(t)
	lines := viewLines(m)

	last := lines[len(lines)-1]
	for _, want := range []string{"j/k", "q quit"} {
		if !strings.Contains(last, want) {
			t.Fatalf("hint bar %q missing %q", last, want)
		}
	}
}

func TestNoHexColorsInTheCodebase(t *testing.T) {
	entries, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range entries {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if i := strings.Index(string(src), `Color("#`); i >= 0 {
			t.Fatalf("%s uses a hex color", f)
		}
	}
}

func TestStatusChangeMovesTheTaskToTheTopOfTheTargetSection(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n- [ ] two\n\n## DONE\n- [x] old\n")
	m := send(open(t, dir), "C", "3")

	if got, want := file(t, dir), "## TODO\n\n- [ ] two\n\n## DOING\n\n## DONE\n\n- [x] one\n- [x] old\n"; got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
	if !strings.Contains(m.View(), "TODO (1)") || !strings.Contains(m.View(), "DONE (2)") {
		t.Fatalf("counts did not update:\n%s", m.View())
	}
	if got := cursorLine(t, m); got != "● one" {
		t.Fatalf("cursor did not follow the moved task, it is on %q", got)
	}
}

func TestStatusChangeToTheCurrentSectionHoistsTheTaskToTheTop(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n- [ ] two\n- [ ] three\n")

	m := send(open(t, dir), "jj", "1")

	if got, want := file(t, dir), "## TODO\n\n- [ ] three\n- [ ] one\n- [ ] two\n\n## DOING\n\n## DONE\n"; got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
	if got := cursorLine(t, m); got != "○ three" {
		t.Fatalf("cursor did not follow the hoisted task, it is on %q", got)
	}
}

func TestStatusChangeOnATaskAlreadyAtTheTopIsANoOp(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n- [ ] two\n")
	before := file(t, dir)

	m := send(open(t, dir), "1")

	if got := file(t, dir); got != before {
		t.Fatalf("no-op status change rewrote the file: %q", got)
	}
	if got := cursorLine(t, m); got != "○ one" {
		t.Fatalf("no-op status change moved the cursor to %q", got)
	}
}

func TestEveryStatusChangeIsPersisted(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n- [ ] two\n")

	m := send(open(t, dir), "C", "2", "gg", "3")

	want := "## TODO\n\n## DOING\n\n- [ ] one\n\n## DONE\n\n- [x] two\n"
	if got := file(t, dir); got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
	if !strings.Contains(m.View(), "◐ one") || !strings.Contains(m.View(), "● two") {
		t.Fatalf("view and file disagree:\n%s", m.View())
	}
}

func TestFirstMutationCreatesTheFile(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n")
	m := open(t, dir)
	// The board is loaded; the file is then gone, as it would be in a fresh
	// directory.
	if err := os.Remove(filepath.Join(dir, board.DefaultFile)); err != nil {
		t.Fatal(err)
	}

	send(m, "3")

	if got, want := file(t, dir), "## TODO\n\n## DOING\n\n## DONE\n\n- [x] one\n"; got != want {
		t.Fatalf("the first mutation did not create the file: %q", got)
	}
}

func TestSaveLeavesNoTempFilesBehind(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n")

	send(open(t, dir), "2", "1", "3")

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != board.DefaultFile {
		t.Fatalf("directory contains more than the todo file: %v", entries)
	}
}

func TestTheAppRecordsTheMtimeOfItsOwnSave(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n")

	m := send(open(t, dir), "3")

	fi, err := os.Stat(filepath.Join(dir, board.DefaultFile))
	if err != nil {
		t.Fatal(err)
	}
	if !m.saved.Equal(fi.ModTime()) {
		t.Fatalf("recorded mtime %v, file mtime %v", m.saved, fi.ModTime())
	}
}

func TestJAndKReorderWithinASection(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n- [ ] two\n- [ ] three\n")
	m := open(t, dir)

	m = send(m, "J")
	if got, want := file(t, dir), "## TODO\n\n- [ ] two\n- [ ] one\n- [ ] three\n\n## DOING\n\n## DONE\n"; got != want {
		t.Fatalf("after J file = %q, want %q", got, want)
	}
	if got := cursorLine(t, m); got != "○ one" {
		t.Fatalf("cursor left the moved task, it is on %q", got)
	}

	m = send(m, "K")
	if got, want := file(t, dir), "## TODO\n\n- [ ] one\n- [ ] two\n- [ ] three\n\n## DOING\n\n## DONE\n"; got != want {
		t.Fatalf("after K file = %q, want %q", got, want)
	}
	if got := cursorLine(t, m); got != "○ one" {
		t.Fatalf("cursor left the moved task, it is on %q", got)
	}
}

func TestReorderingStopsAtSectionBoundaries(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n\n## DOING\n- [ ] two\n\n## DONE\n- [x] three\n")
	before := file(t, dir)

	// J on the only (and therefore last) TODO task, K on the first DOING task.
	m := send(open(t, dir), "J")
	if got := file(t, dir); got != before {
		t.Fatalf("J across a section boundary changed the file: %q", got)
	}
	m = send(m, "j", "K", "J")
	if got := file(t, dir); got != before {
		t.Fatalf("J/K across a section boundary changed the file: %q", got)
	}
	if !strings.Contains(m.View(), "◐ two") {
		t.Fatalf("reordering changed a status:\n%s", m.View())
	}
}

func TestReorderingOnAnEmptyBoardIsHarmless(t *testing.T) {
	dir := t.TempDir()
	send(open(t, dir), "J", "K")

	if got := file(t, dir); got != emptyBoard {
		t.Fatalf("reordering an empty board wrote %q", got)
	}
}

func TestOAddsATaskBelowTheCursorInTheCursorsSection(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n- [ ] two\n\n## DOING\n- [ ] running\n")

	m := send(open(t, dir), "o", "new", "enter")

	if got, want := file(t, dir), "## TODO\n\n- [ ] one\n- [ ] new\n- [ ] two\n\n## DOING\n\n- [ ] running\n\n## DONE\n"; got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
	if got := cursorLine(t, m); got != "○ new" {
		t.Fatalf("cursor on %q after adding, want the new task", got)
	}
}

func TestOAddsATaskAboveTheCursor(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n- [ ] two\n")

	send(open(t, dir), "j", "O", "new", "enter")

	if got, want := file(t, dir), "## TODO\n\n- [ ] one\n- [ ] new\n- [ ] two\n\n## DOING\n\n## DONE\n"; got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
}

func TestOInDoingStartsATaskThere(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n\n## DOING\n- [ ] running\n")

	send(open(t, dir), "j", "o", "another", "enter")

	if got, want := file(t, dir), "## TODO\n\n- [ ] one\n\n## DOING\n\n- [ ] running\n- [ ] another\n\n## DONE\n"; got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
}

func TestOOnAnEmptyBoardCreatesTheFirstTodoTask(t *testing.T) {
	for _, key := range []string{"o", "O"} {
		dir := t.TempDir()
		send(open(t, dir), key, "first", "enter")

		if got, want := file(t, dir), "## TODO\n\n- [ ] first\n\n## DOING\n\n## DONE\n"; got != want {
			t.Fatalf("%s on an empty board wrote %q, want %q", key, got, want)
		}
	}
}

func TestEscCancelsTheInput(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n")
	before := file(t, dir)

	m := send(open(t, dir), "o", "never", "esc")

	if got := file(t, dir); got != before {
		t.Fatalf("cancelled input changed the file: %q", got)
	}
	if strings.Contains(m.View(), "never") {
		t.Fatalf("cancelled input still on the board:\n%s", m.View())
	}
}

func TestBlankInputCreatesNoTask(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n")
	before := file(t, dir)

	send(open(t, dir), "o", "space", "space", "enter")

	if got := file(t, dir); got != before {
		t.Fatalf("whitespace-only input created a task: %q", got)
	}
}

func TestCCEditsTheCurrentTaskPrefilled(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] typo\n")

	m := send(open(t, dir), "cc")
	if !strings.Contains(m.View(), "typo") {
		t.Fatalf("cc did not prefill the input:\n%s", m.View())
	}

	m = send(m, "backspace", "backspace", "backspace", "backspace", "fixed", "enter")

	if got, want := file(t, dir), "## TODO\n\n- [ ] fixed\n\n## DOING\n\n## DONE\n"; got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
}

func TestDDDeletesImmediatelyAndKeepsTheRow(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n- [ ] two\n- [ ] three\n")

	m := send(open(t, dir), "j", "dd")

	if got, want := file(t, dir), "## TODO\n\n- [ ] one\n- [ ] three\n\n## DOING\n\n## DONE\n"; got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
	if got := cursorLine(t, m); got != "○ three" {
		t.Fatalf("cursor on %q after delete, want the task that took the row", got)
	}
}

func TestRepeatedDDPrunesARun(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n- [ ] two\n- [ ] three\n")

	m := send(open(t, dir), "dd", "dd", "dd")

	if got, want := file(t, dir), "## TODO\n\n## DOING\n\n## DONE\n"; got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
	if got := cursorLine(t, m); got != "" {
		t.Fatalf("empty board still shows a cursor on %q", got)
	}
	m = send(m, "o", "again", "enter")
	if got := cursorLine(t, m); got != "○ again" {
		t.Fatalf("cursor was not at index 0 on the empty board, it is on %q", got)
	}
}

func TestNormalModeKeysAreLiteralTextInTheInput(t *testing.T) {
	dir := t.TempDir()

	send(open(t, dir), "o", "quit the job 1 2 3 dd", "enter")

	if got, want := file(t, dir), "## TODO\n\n- [ ] quit the job 1 2 3 dd\n\n## DOING\n\n## DONE\n"; got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
}

func TestHintBarShowsInsertHintsWhileTheInputIsOpen(t *testing.T) {
	dir := t.TempDir()
	m := send(open(t, dir), "o")

	lines := viewLines(m)
	if got := lines[len(lines)-1]; !strings.Contains(got, "enter confirm") || !strings.Contains(got, "esc cancel") {
		t.Fatalf("hint bar %q is not the insert-mode hint bar", got)
	}

	m = send(m, "esc")
	lines = viewLines(m)
	if got := lines[len(lines)-1]; !strings.Contains(got, "q quit") {
		t.Fatalf("hint bar %q did not return to normal mode", got)
	}
}

func TestUndoReversesEveryMutation(t *testing.T) {
	start := "## TODO\n\n- [ ] one\n- [ ] two\n\n## DOING\n\n## DONE\n"
	cases := []struct {
		name string
		keys []string
	}{
		{"status change", []string{"3"}},
		{"reorder", []string{"J"}},
		{"add below", []string{"o", "new", "enter"}},
		{"add above", []string{"O", "new", "enter"}},
		{"edit", []string{"cc", "backspace", "x", "enter"}},
		{"delete", []string{"dd"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			write(t, dir, start)

			m := send(open(t, dir), tc.keys...)
			if got := file(t, dir); got == start {
				t.Fatalf("%v did not change anything", tc.keys)
			}

			m = send(m, "u")

			if got := file(t, dir); got != start {
				t.Fatalf("after undo file = %q, want %q", got, start)
			}
			if !strings.Contains(m.View(), "○ one") || !strings.Contains(m.View(), "○ two") {
				t.Fatalf("view does not match the restored file:\n%s", m.View())
			}
		})
	}
}

func TestRepeatedUndoWalksBack(t *testing.T) {
	dir := t.TempDir()
	start := "## TODO\n\n- [ ] one\n\n## DOING\n\n## DONE\n"
	write(t, dir, start)

	m := send(open(t, dir), "o", "two", "enter", "o", "three", "enter", "3")
	m = send(m, "u", "u", "u")

	if got := file(t, dir); got != start {
		t.Fatalf("after three undos file = %q, want %q", got, start)
	}
}

func TestUndoOnAnEmptyStackChangesNothing(t *testing.T) {
	dir := t.TempDir()
	m := send(open(t, dir), "u", "u", "u")

	if got := file(t, dir); got != emptyBoard {
		t.Fatalf("undo on an empty stack wrote %q", got)
	}
	if !strings.Contains(m.View(), "TODO (0)") {
		t.Fatalf("undo on an empty stack broke the view:\n%s", m.View())
	}
}

func TestDoneStartsCollapsedAndCTogglesIt(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, board3)

	m := open(t, dir)
	v := m.View()
	if strings.Contains(v, "● four") {
		t.Fatalf("DONE did not start collapsed:\n%s", v)
	}
	// The count is still there, so nothing looks lost.
	if !strings.Contains(v, "DONE (1)") {
		t.Fatalf("collapsed DONE lost its count:\n%s", v)
	}

	m = send(m, "C")
	if !strings.Contains(m.View(), "● four") {
		t.Fatalf("C did not expand DONE:\n%s", m.View())
	}
	m = send(m, "C")
	if strings.Contains(m.View(), "● four") {
		t.Fatalf("C did not collapse DONE again:\n%s", m.View())
	}
}

func TestCollapsedDoneIsOutOfReachOfTheCursor(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, board3)

	// G is the last task on the board, which with DONE collapsed is the DOING
	// one, and a status change into DONE folds the task away without stranding
	// the cursor.
	m := send(open(t, dir), "G")
	if got := cursorLine(t, m); got != "◐ three" {
		t.Fatalf("G went to %q, want the last unfolded task", got)
	}

	m = send(m, "3")
	if got := cursorLine(t, m); got != "○ two" {
		t.Fatalf("cursor on %q after folding a task away, want the last visible task", got)
	}
	if !strings.Contains(m.View(), "DONE (2)") {
		t.Fatalf("the task did not land in DONE:\n%s", m.View())
	}
}

func TestSectionJumpsLandOnTheFirstTaskOfTheNextSection(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n- [ ] two\n\n## DOING\n- [ ] three\n\n## DONE\n- [x] four\n")

	m := send(open(t, dir), "C", "}")
	if got := cursorLine(t, m); got != "◐ three" {
		t.Fatalf("cursor on %q after }, want the first DOING task", got)
	}
	m = send(m, "}")
	if got := cursorLine(t, m); got != "● four" {
		t.Fatalf("cursor on %q after a second }, want the first DONE task", got)
	}
	m = send(m, "{", "{")
	if got := cursorLine(t, m); got != "○ one" {
		t.Fatalf("cursor on %q after two {, want the first TODO task", got)
	}
}

func TestSectionJumpsSkipEmptySectionsAndStopAtTheEnds(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n\n## DOING\n\n## DONE\n- [x] two\n")

	// DOING is empty, so } jumps straight past it, and again at the end of the
	// board nothing moves.
	m := send(open(t, dir), "C", "}", "}")
	if got := cursorLine(t, m); got != "● two" {
		t.Fatalf("cursor on %q, want the DONE task", got)
	}
	m = send(m, "{", "{")
	if got := cursorLine(t, m); got != "○ one" {
		t.Fatalf("cursor on %q, want the TODO task", got)
	}
}

func TestRedoReplaysEveryUndo(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n\n## DOING\n\n## DONE\n")

	m := send(open(t, dir), "C", "o", "two", "enter", "3")
	want := file(t, dir)

	m = send(m, "u", "u")
	m = send(m, "r", "r")

	if got := file(t, dir); got != want {
		t.Fatalf("after two redos file = %q, want %q", got, want)
	}
	if !strings.Contains(m.View(), "● two") {
		t.Fatalf("redo did not restore the board:\n%s", m.View())
	}
}

func TestRedoOnAnEmptyStackChangesNothing(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n\n## DOING\n\n## DONE\n")

	// Nothing undone yet, and a redo after a redo has caught up.
	m := send(open(t, dir), "C", "r", "3", "u", "r", "r")

	want := "## TODO\n\n## DOING\n\n## DONE\n\n- [x] one\n"
	if got := file(t, dir); got != want {
		t.Fatalf("after redo file = %q, want %q", got, want)
	}
	if !strings.Contains(m.View(), "● one") {
		t.Fatalf("redo on an empty stack broke the view:\n%s", m.View())
	}
}

func TestNewMutationDropsTheRedoStack(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n\n## DOING\n\n## DONE\n")

	// Undo the status change, then fork the history with a new mutation: the
	// undone change must no longer be reachable.
	send(open(t, dir), "3", "u", "o", "two", "enter", "r")

	want := "## TODO\n\n- [ ] one\n- [ ] two\n\n## DOING\n\n## DONE\n"
	if got := file(t, dir); got != want {
		t.Fatalf("after redo on a forked history file = %q, want %q", got, want)
	}
}

func TestNoOpKeyPressPushesNoSnapshot(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n\n## DOING\n- [ ] two\n")

	// 2 on a DOING task and K on the first task are both no-ops; a following
	// u must undo the status change, not one of them.
	m := send(open(t, dir), "3")
	// gg lands on the DOING task; 2 there, and K/J on the only task in its
	// section, are all no-ops, as is a cancelled input.
	m = send(m, "gg", "2", "K", "J", "o", "esc")
	m = send(m, "u")

	want := "## TODO\n\n- [ ] one\n\n## DOING\n\n- [ ] two\n\n## DONE\n"
	if got := file(t, dir); got != want {
		t.Fatalf("after undo file = %q, want %q", got, want)
	}
}

func TestUndoClampsTheCursorIntoTheRestoredList(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n- [ ] two\n- [ ] three\n")

	// Add a task at the bottom, put the cursor on it, then undo it away.
	m := send(open(t, dir), "G", "o", "four", "enter", "u")

	if got := cursorLine(t, m); got != "○ three" {
		t.Fatalf("cursor on %q after undo, want the last restored task", got)
	}
	if strings.Contains(m.View(), "four") {
		t.Fatalf("undone task still on the board:\n%s", m.View())
	}
}

func TestUndoStackIsNeverWrittenToDisk(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n")

	send(open(t, dir), "3", "u", "3")

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != board.DefaultFile {
		t.Fatalf("undo left something on disk: %v", entries)
	}
}

const filterBoard = "## TODO\n- [ ] Write docs\n- [ ] ship it\n\n## DOING\n- [ ] write tests\n\n## DONE\n- [x] design\n"

func TestFilterNarrowsTheListAsYouType(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, filterBoard)

	m := send(open(t, dir), "/", "wri")

	v := m.View()
	for _, want := range []string{"○ Write docs", "◐ write tests"} {
		if !strings.Contains(v, want) {
			t.Fatalf("filtered view missing %q:\n%s", want, v)
		}
	}
	for _, unwanted := range []string{"ship it", "design"} {
		if strings.Contains(v, unwanted) {
			t.Fatalf("filtered view still shows %q:\n%s", unwanted, v)
		}
	}
	if !strings.Contains(v, "TODO (1)") || !strings.Contains(v, "DOING (1)") || !strings.Contains(v, "DONE (0)") {
		t.Fatalf("counts do not reflect the filtered view:\n%s", v)
	}
}

func TestFilterHintBarShowsTheLiveQuery(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, filterBoard)

	m := send(open(t, dir), "/", "wri")

	lines := viewLines(m)
	if got := lines[len(lines)-1]; !strings.Contains(got, "/wri") {
		t.Fatalf("hint bar %q does not show the query", got)
	}
}

func TestCursorMovesWithinTheFilteredListOnly(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, filterBoard)

	m := send(open(t, dir), "/", "wri", "enter")

	if got := cursorLine(t, m); got != "○ Write docs" {
		t.Fatalf("cursor on %q, want the first match", got)
	}
	m = send(m, "j")
	if got := cursorLine(t, m); got != "◐ write tests" {
		t.Fatalf("j went to %q, want the second match", got)
	}
	m = send(m, "j")
	if got := cursorLine(t, m); got != "◐ write tests" {
		t.Fatalf("j left the filtered list, cursor on %q", got)
	}
}

func TestActingOnAFilteredTaskAndTheFilterPersists(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, filterBoard)

	m := send(open(t, dir), "/", "wri", "enter", "3")

	want := "## TODO\n\n- [ ] ship it\n\n## DOING\n\n- [ ] write tests\n\n## DONE\n\n- [x] Write docs\n- [x] design\n"
	if got := file(t, dir); got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
	if strings.Contains(m.View(), "ship it") {
		t.Fatalf("the filter was dropped after acting on a match:\n%s", m.View())
	}

	// dd and cc act on the filtered task under the cursor too.
	m = send(m, "gg", "cc", "backspace", "backspace", "backspace", "backspace", "ing", "enter")
	m = send(m, "dd")

	if got := file(t, dir); strings.Contains(got, "write") {
		t.Fatalf("cc/dd did not act on the filtered task: %q", got)
	}
}

func TestEscClearsTheFilter(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, filterBoard)

	m := send(open(t, dir), "C", "/", "wri", "esc")

	v := m.View()
	for _, want := range []string{"ship it", "design", "Write docs", "write tests"} {
		if !strings.Contains(v, want) {
			t.Fatalf("esc did not restore the full board, %q missing:\n%s", want, v)
		}
	}
	if !strings.Contains(v, "q quit") {
		t.Fatalf("esc did not return to normal mode:\n%s", v)
	}
}

func TestNAndNAreNotBound(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, filterBoard)
	before := file(t, dir)

	m := send(open(t, dir), "n", "N")

	if got := file(t, dir); got != before {
		t.Fatalf("n/N changed the file: %q", got)
	}
	if got := cursorLine(t, m); got != "○ Write docs" {
		t.Fatalf("n/N moved the cursor to %q", got)
	}
}

// poll sends one watcher tick.
func poll(m Model) Model {
	tm, _ := m.Update(tickMsg(time.Now()))
	return tm.(Model)
}

func TestTickerFiresAboutOncePerSecond(t *testing.T) {
	if pollInterval < 500*time.Millisecond || pollInterval > 2*time.Second {
		t.Fatalf("poll interval is %v, want about a second", pollInterval)
	}
	if open(t, t.TempDir()).Init() == nil {
		t.Fatal("the model starts no ticker")
	}
	tm, cmd := open(t, t.TempDir()).Update(tickMsg(time.Now()))
	if cmd == nil {
		t.Fatal("a tick does not schedule the next tick")
	}
	_ = tm
}

func TestExternalEditIsPickedUpOnTheNextTick(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n")
	m := open(t, dir)

	write(t, dir, "## TODO\n- [ ] one\n- [ ] added in vim\n")
	m = poll(m)

	if !strings.Contains(m.View(), "○ added in vim") || !strings.Contains(m.View(), "TODO (2)") {
		t.Fatalf("the hand-edit was not picked up:\n%s", m.View())
	}
}

func TestOwnWritesNeverTriggerAReload(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n- [ ] two\n- [ ] three\n")
	m := open(t, dir)

	m = send(m, "j")
	for i := 0; i < 5; i++ {
		m = send(m, "J", "K")
		m = poll(m)
		if got := cursorLine(t, m); got != "○ two" {
			t.Fatalf("a poll after the app's own write moved the cursor to %q", got)
		}
	}
	// The undo stack survives, so no reload happened.
	m = send(m, "u")
	if !strings.Contains(m.View(), "○ two") {
		t.Fatalf("the undo stack was cleared by the app's own writes:\n%s", m.View())
	}
}

// A hand-edit can leave the file unreadable. The app must not save over it:
// the next save would rewrite the file without the lines the parser dropped.
func TestAHandEditTheParserCannotReadIsNeverSavedOver(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n")
	m := open(t, dir)

	broken := "## TODO\n- [ ] one\nnotes I typed straight into the file\n"
	write(t, dir, broken)
	m = poll(m)

	m = send(m, "o", "two", "enter", "dd")
	if got := file(t, dir); got != broken {
		t.Fatalf("the app wrote over a file it could not read:\ngot  %q\nwant %q", got, broken)
	}
	if !strings.Contains(m.View(), "not saving") {
		t.Fatalf("nothing on screen says saving is off:\n%s", m.View())
	}

	// Fixing the file by hand puts the board back in charge of it.
	write(t, dir, "## TODO\n- [ ] one\n- [ ] two\n")
	m = poll(m)
	m = send(m, "G", "dd")
	if got := file(t, dir); !strings.Contains(got, "- [ ] one") || strings.Contains(got, "two") {
		t.Fatalf("saving did not resume after the file was fixed: %q", got)
	}
}

func TestReloadKeepsTheCursorOnTheSameTask(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n- [ ] two\n- [ ] three\n")
	m := send(open(t, dir), "j")

	write(t, dir, "## TODO\n- [ ] zero\n- [ ] one\n- [ ] two\n- [ ] three\n")
	m = poll(m)

	if got := cursorLine(t, m); got != "○ two" {
		t.Fatalf("cursor on %q after reload, want the task it was on", got)
	}
}

func TestReloadClampsTheIndexWhenTheTaskIsGone(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n- [ ] two\n- [ ] three\n")
	m := send(open(t, dir), "G")

	write(t, dir, "## TODO\n- [ ] one\n")
	m = poll(m)

	if got := cursorLine(t, m); got != "○ one" {
		t.Fatalf("cursor on %q after reload, want it clamped into the new list", got)
	}
}

func TestReloadEmptiesTheUndoStack(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n")
	m := send(open(t, dir), "3")

	edited := "## TODO\n- [ ] hand written\n"
	write(t, dir, edited)
	m = poll(m)
	m = send(m, "u")

	if !strings.Contains(m.View(), "○ hand written") {
		t.Fatalf("undo after a reload clobbered the hand-edit:\n%s", m.View())
	}
	if got := file(t, dir); got != edited {
		t.Fatalf("undo after a reload rewrote the file: %q", got)
	}
}

func TestHandTickedBoxReloadsAsTodoAndIsNormalizedOnTheNextSave(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n")
	m := open(t, dir)

	write(t, dir, "## TODO\n- [x] one\n- [ ] two\n")
	m = poll(m)

	if !strings.Contains(m.View(), "○ one") || !strings.Contains(m.View(), "TODO (2)") {
		t.Fatalf("the hand-ticked box did not reload as a TODO task:\n%s", m.View())
	}

	m = send(m, "G", "3")

	want := "## TODO\n\n- [ ] one\n\n## DOING\n\n## DONE\n\n- [x] two\n"
	if got := file(t, dir); got != want {
		t.Fatalf("the next save did not normalize the file: %q, want %q", got, want)
	}
}

// paste feeds a terminal paste — the whole clipboard in one key event.
func paste(m Model, text string) Model {
	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text), Paste: true})
	return tm.(Model)
}

func TestPasteLandsInTheInputAndIsSaved(t *testing.T) {
	dir := t.TempDir()

	m := send(open(t, dir), "o", "call ")
	m = paste(m, "Ada Lovelace")
	m = send(m, " back", "enter")

	if got, want := file(t, dir), "## TODO\n\n- [ ] call Ada Lovelace back\n\n## DOING\n\n## DONE\n"; got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
}

func TestMultiLinePasteBecomesOneTitle(t *testing.T) {
	dir := t.TempDir()

	m := send(open(t, dir), "o")
	m = send(paste(m, "one\ntwo\n"), "enter")

	if got, want := file(t, dir), "## TODO\n\n- [ ] one two\n\n## DOING\n\n## DONE\n"; got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
}

func TestPasteDropsEscapeSequences(t *testing.T) {
	dir := t.TempDir()

	m := send(open(t, dir), "o")
	m = send(paste(m, "buy milk\x1b]52;c;aGVsbG8=\x07"), "enter")

	if got, want := file(t, dir), "## TODO\n\n- [ ] buy milk]52;c;aGVsbG8=\n\n## DOING\n\n## DONE\n"; got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
}

func TestPasteInNormalModeChangesNothing(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "## TODO\n- [ ] one\n")

	m := paste(open(t, dir), "junk")

	if m.input != "" || m.filter != "" || m.mode != normalMode {
		t.Fatalf("paste outside the input left input=%q filter=%q mode=%d", m.input, m.filter, m.mode)
	}
}

// click sends a left mouse press on screen row y.
func click(m Model, y int) Model {
	tm, _ := m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 4, Y: y})
	return tm.(Model)
}

func TestClickingATaskSelectsIt(t *testing.T) {
	m, _ := newBoard3(t)

	// Rows: TODO, one, two, DOING, three, DONE, four.
	for y, want := range map[int]string{2: "○ two", 4: "◐ three", 6: "● four", 1: "○ one"} {
		if got := cursorLine(t, click(m, y)); got != want {
			t.Fatalf("click on row %d selected %q, want %q", y, got, want)
		}
	}
}

func TestClickingASectionHeaderOrEmptySpaceSelectsNothing(t *testing.T) {
	m, _ := newBoard3(t)
	m = send(m, "j")

	// -1 is the row-0 mouse report Bubble Tea normalises past zero.
	for _, y := range []int{-1, 0, 3, 5, 7, 20} {
		if got := cursorLine(t, click(m, y)); got != "○ two" {
			t.Fatalf("click on row %d moved the cursor to %q", y, got)
		}
	}
}

func TestClickingAFoldedTitleSelectsThatTask(t *testing.T) {
	dir := t.TempDir()
	long := strings.Repeat("word ", 10)
	write(t, dir, "## TODO\n- [ ] one\n- [ ] "+strings.TrimSpace(long)+"\n- [ ] three\n")
	m := resize(open(t, dir), 20, 24)

	// Rows: TODO, one, then the folded task over several rows.
	if got := cursorLine(t, click(m, 3)); !strings.Contains(got, "word") {
		t.Fatalf("click on a continuation row selected %q", got)
	}
}

func TestClickingUsesTheScrolledViewport(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, longBoard(40))
	m := resize(open(t, dir), 60, 12)
	m = send(m, "G")

	want := strings.TrimSpace(strings.TrimPrefix(viewLines(m)[1], "▸"))
	if got := cursorLine(t, click(m, 1)); got != want {
		t.Fatalf("click on the second visible row selected %q, want %q", got, want)
	}
}

func TestAClickChangesNothingButTheCursor(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, board3)
	before := file(t, dir)

	m := send(open(t, dir), "C")
	m = click(m, 6)

	if got := file(t, dir); got != before {
		t.Fatalf("a click rewrote the file:\n%s", got)
	}
	if len(m.undo) != 0 {
		t.Fatalf("a click pushed %d undo snapshots", len(m.undo))
	}
}

func TestOnlyALeftPressSelects(t *testing.T) {
	m, _ := newBoard3(t)

	for _, msg := range []tea.MouseMsg{
		{Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft, Y: 2},
		{Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft, Y: 2},
		{Action: tea.MouseActionPress, Button: tea.MouseButtonRight, Y: 2},
		{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown, Y: 2},
	} {
		tm, _ := m.Update(msg)
		if got := cursorLine(t, tm.(Model)); got != "○ one" {
			t.Fatalf("%v moved the cursor to %q", msg, got)
		}
	}
}

func TestAClickWhileTheInputIsOpenChangesNothing(t *testing.T) {
	m, _ := newBoard3(t)

	m = send(m, "o", "new")
	m = click(m, 6)

	if m.cursor != 0 {
		t.Fatalf("a click during insert moved the cursor to %d", m.cursor)
	}
	if m.input != "new" {
		t.Fatalf("a click during insert left input %q", m.input)
	}
}
