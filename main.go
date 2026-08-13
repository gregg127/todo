package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

const usage = `usage: todo

An interactive task board stored in ./todo-database.md.

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
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			fmt.Print(usage)
			return
		}
	}

	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "todo:", err)
		os.Exit(1)
	}
	if _, err := tea.NewProgram(New(dir), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "todo:", err)
		os.Exit(1)
	}
}
