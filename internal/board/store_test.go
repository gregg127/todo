package board

import (
	"os"
	"testing"
)

const emptyBoard = "## TODO\n\n## DOING\n\n## DONE\n"

const board3 = "## TODO\n- [ ] one\n- [ ] two\n\n## DOING\n- [ ] three\n\n## DONE\n- [x] four\n"

func TestRenderEmptyBoard(t *testing.T) {
	if got, want := Render(nil, nil), "## TODO\n\n## DOING\n\n## DONE\n"; got != want {
		t.Fatalf("Render(empty) = %q, want %q", got, want)
	}
}

func TestParseAndRender(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty file",
			input: "",
			want:  "## TODO\n\n## DOING\n\n## DONE\n",
		},
		{
			name:  "ticked box under TODO stays a TODO task",
			input: "## TODO\n- [x] hand ticked\n",
			want:  "## TODO\n\n- [ ] hand ticked\n\n## DOING\n\n## DONE\n",
		},
		{
			name:  "prose between sections is dropped",
			input: "notes\n\n## TODO\nsome prose\n- [ ] a\n\n### other\n\n## DONE\n- [x] b\n",
			want:  "## TODO\n\n- [ ] a\n\n## DOING\n\n## DONE\n\n- [x] b\n",
		},
		{
			name:  "items before any heading are dropped",
			input: "- [ ] orphan\n## TODO\n- [ ] a\n",
			want:  "## TODO\n\n- [ ] a\n\n## DOING\n\n## DONE\n",
		},
		{
			name:  "truncated file",
			input: "## TODO\n- [ ] a\n- [ ",
			want:  "## TODO\n\n- [ ] a\n\n## DOING\n\n## DONE\n",
		},
		{
			name:  "CRLF line endings",
			input: "## TODO\r\n- [ ] a\r\n\r\n## DOING\r\n- [ ] b\r\n",
			want:  "## TODO\n\n- [ ] a\n\n## DOING\n\n- [ ] b\n\n## DONE\n",
		},
		{
			name:  "title containing a checkbox",
			input: "## TODO\n- [ ] write - [ ] in a title\n",
			want:  "## TODO\n\n- [ ] write - [ ] in a title\n\n## DOING\n\n## DONE\n",
		},
		{
			name:  "frontmatter is kept and the board below it read",
			input: "---\ncollapsed-done: true\n---\n\n## TODO\n- [ ] a\n",
			want:  "---\ncollapsed-done: true\n---\n\n## TODO\n\n- [ ] a\n\n## DOING\n\n## DONE\n",
		},
		{
			name:  "an unknown metadata key is written back untouched",
			input: "---\nzz: 1\ncollapsed-done: false\n---\n## TODO\n",
			want:  "---\ncollapsed-done: false\nzz: 1\n---\n\n## TODO\n\n## DOING\n\n## DONE\n",
		},
		{
			name:  "all three sections in order",
			input: "## DONE\n- [x] c\n\n## TODO\n- [ ] a\n\n## DOING\n- [ ] b\n",
			want:  "## TODO\n\n- [ ] a\n\n## DOING\n\n- [ ] b\n\n## DONE\n\n- [x] c\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Render(Parse(tc.input))
			if got != tc.want {
				t.Fatalf("Render(Parse(%q)) = %q, want %q", tc.input, got, tc.want)
			}
			if round := Render(Parse(got)); round != got {
				t.Fatalf("round-trip not stable: %q then %q", got, round)
			}
		})
	}
}

func TestExampleFileIsWrittenInTheAppsOwnFormat(t *testing.T) {
	want, err := os.ReadFile("../../docs/todo-database-example.md")
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(string(want)); err != nil {
		t.Fatalf("the app would refuse to open its own example: %v", err)
	}
	if got := Render(Parse(string(want))); got != string(want) {
		t.Fatalf("example file is not what Render writes:\ngot  %q\nwant %q", got, want)
	}
}

func TestValidateAcceptsWhatTheParserKeepsAndRejectsWhatItDrops(t *testing.T) {
	valid := []string{
		"",
		emptyBoard,
		board3,
		"## DONE\r\n- [X] windows line endings and an uppercase box\r\n",
		"## DOING\n- [ ] a\n   \n## TODO\n- [ ] b\n",
		"## TODO\n- [ ] punctuation, dashes — and emoji 🎉 are all just text\n",
		"---\ncollapsed-done: true\n---\n\n" + board3,
		"---\n---\n" + board3,
	}
	for _, text := range valid {
		if err := Validate(text); err != nil {
			t.Fatalf("Validate(%q) = %v, want no error", text, err)
		}
	}

	invalid := []string{
		"# my notes\n",
		"## Todo\n",
		"## TODO\n* a bullet the parser does not read\n",
		"- [ ] an item before any heading\n",
		"## TODO\n- [ ] a\nprose after a task\n",
		"## TODO\n  - [ ] an indented item\n",
		"---\ncollapsed-done: true\n" + board3, // no closing fence
		"---\ncollapsed-done:true\n---\n" + board3,
		"---\njust a note\n---\n" + board3,
		"## TODO\n- [ ] buy milk\x1b]52;c;aGVsbG8=\x07\n", // OSC 52 clipboard write
		"## TODO\n- [ ] buy milk\x1b]0;pwned\x07\n",       // OSC 0 window title
		"## TODO\n- [ ] buy milk\x07\n",                   // bare BEL, which ansi.Strip keeps
		"## TODO\n- [ ] buy\rmilk\n",                      // bare CR, ditto
	}
	for _, text := range invalid {
		if err := Validate(text); err == nil {
			t.Fatalf("Validate(%q) = nil, want an error", text)
		}
	}
}

func TestLoadMissingFileIsEmptyAndNoError(t *testing.T) {
	b, _, err := Load(t.TempDir() + "/nope.md")
	if err != nil {
		t.Fatalf("Load of missing file: %v", err)
	}
	if len(b) != 0 {
		t.Fatalf("Load of missing file = %v, want empty", b)
	}
}
