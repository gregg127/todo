package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"todo/internal/board"
	"todo/internal/tui"
)

const usage = `usage: todo [file]

An interactive task board stored in a Markdown file, ./todo-database.md
unless another one is named. The file is created if it is not there yet;
a file the board cannot read is reported and nothing starts.

  j / k        move the cursor down / up (also ↓ / ↑)
  click        put the cursor on a task
  wheel        scroll the board
  gg / G       jump to the first / last task
  { / }        jump to the previous / next section
  1 / 2 / 3    move the task to TODO / DOING / DONE
  J / K        move the task down / up within its section
  o / O        add a task below / above the cursor
  cc           edit the task under the cursor
  dd           delete the task under the cursor
  C            fold / unfold the DONE section
  u / r        undo / redo the last change
  /            filter tasks, esc clears
  q            quit
`

func main() {
	args := os.Args[1:]
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			fmt.Print(usage)
			return
		}
	}
	if len(args) > 1 {
		die("one board file at a time: todo [file]")
	}

	path := board.DefaultFile
	if len(args) == 1 {
		path = args[0]
	}
	m, err := tui.New(path)
	if err != nil {
		die(err)
	}
	if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		die(err)
	}
}

func die(v any) {
	fmt.Fprintln(os.Stderr, "todo:", v)
	os.Exit(1)
}
