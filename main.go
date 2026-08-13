package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

const usage = `usage: todo [file]

An interactive task board stored in a Markdown file, ./todo-database.md
unless another one is named. The file is created if it is not there yet;
a file the board cannot read is reported and nothing starts.

  j / k        move the cursor down / up
  gg / G       jump to the first / last task
  1 / 2 / 3    move the task to TODO / DOING / DONE
  J / K        move the task down / up within its section
  o / O        add a task below / above the cursor
  cc           edit the task under the cursor
  dd           delete the task under the cursor
  u            undo the last change
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

	path := dbFile
	if len(args) == 1 {
		path = args[0]
	}
	m, err := New(path)
	if err != nil {
		die(err)
	}
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		die(err)
	}
}

func die(v any) {
	fmt.Fprintln(os.Stderr, "todo:", v)
	os.Exit(1)
}
