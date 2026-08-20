package board

import "slices"

type Status int

const (
	Todo Status = iota
	Doing
	Done
)

var Statuses = []Status{Todo, Doing, Done}

type Task struct {
	Title  string
	Status Status
}

type Board []Task

func (b Board) Clone() Board { return slices.Clone(b) }

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
