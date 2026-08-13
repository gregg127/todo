package board

import "slices"

// Status is a task's section on the board.
type Status int

const (
	Todo Status = iota
	Doing
	Done
)

// Statuses is every section, in board order.
var Statuses = []Status{Todo, Doing, Done}

// Task is a single line on the board.
type Task struct {
	Title  string
	Status Status
}

// Board is an ordered list of tasks. Section membership is derived by
// filtering on status; order within a section is the order in the slice.
type Board []Task

func (b Board) Clone() Board { return slices.Clone(b) }

// SectionName is the heading text for a status, as written to the file.
func SectionName(s Status) string {
	switch s {
	case Doing:
		return "DOING"
	case Done:
		return "DONE"
	default:
		return "TODO"
	}
}
