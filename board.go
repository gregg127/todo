package main

// Status is a task's section on the board.
type Status int

const (
	Todo Status = iota
	Doing
	Done
)

// Task is a single line on the board.
type Task struct {
	Title  string
	Status Status
}

// Board is an ordered list of tasks. Section membership is derived by
// filtering on status; order within a section is the order in the slice.
type Board []Task

func (b Board) clone() Board {
	out := make(Board, len(b))
	copy(out, b)
	return out
}

func (b Board) count(s Status) int {
	n := 0
	for _, t := range b {
		if t.Status == s {
			n++
		}
	}
	return n
}
