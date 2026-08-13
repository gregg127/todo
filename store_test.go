package main

import (
	"os"
	"testing"
)

func TestRenderEmptyBoard(t *testing.T) {
	if got, want := Render(nil), "## TODO\n\n## DOING\n\n## DONE\n"; got != want {
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
			want:  "## TODO\n- [ ] hand ticked\n\n## DOING\n\n## DONE\n",
		},
		{
			name:  "prose between sections is dropped",
			input: "notes\n\n## TODO\nsome prose\n- [ ] a\n\n### other\n\n## DONE\n- [x] b\n",
			want:  "## TODO\n- [ ] a\n\n## DOING\n\n## DONE\n- [x] b\n",
		},
		{
			name:  "items before any heading are dropped",
			input: "- [ ] orphan\n## TODO\n- [ ] a\n",
			want:  "## TODO\n- [ ] a\n\n## DOING\n\n## DONE\n",
		},
		{
			name:  "truncated file",
			input: "## TODO\n- [ ] a\n- [ ",
			want:  "## TODO\n- [ ] a\n\n## DOING\n\n## DONE\n",
		},
		{
			name:  "CRLF line endings",
			input: "## TODO\r\n- [ ] a\r\n\r\n## DOING\r\n- [ ] b\r\n",
			want:  "## TODO\n- [ ] a\n\n## DOING\n- [ ] b\n\n## DONE\n",
		},
		{
			name:  "title containing a checkbox",
			input: "## TODO\n- [ ] write - [ ] in a title\n",
			want:  "## TODO\n- [ ] write - [ ] in a title\n\n## DOING\n\n## DONE\n",
		},
		{
			name:  "all three sections in order",
			input: "## DONE\n- [x] c\n\n## TODO\n- [ ] a\n\n## DOING\n- [ ] b\n",
			want:  "## TODO\n- [ ] a\n\n## DOING\n- [ ] b\n\n## DONE\n- [x] c\n",
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

// The example file is the format's documentation, so it has to be exactly what
// the app writes: a hand-edit of it must survive a save untouched.
func TestExampleFileIsWrittenInTheAppsOwnFormat(t *testing.T) {
	want, err := os.ReadFile("todo-database-example.md")
	if err != nil {
		t.Fatal(err)
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
	}
	for _, text := range invalid {
		if err := Validate(text); err == nil {
			t.Fatalf("Validate(%q) = nil, want an error", text)
		}
	}
}

func TestLoadMissingFileIsEmptyAndNoError(t *testing.T) {
	b, err := Load(t.TempDir() + "/nope.md")
	if err != nil {
		t.Fatalf("Load of missing file: %v", err)
	}
	if len(b) != 0 {
		t.Fatalf("Load of missing file = %v, want empty", b)
	}
}
