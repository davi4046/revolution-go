package interpret

import "golang.org/x/exp/slices"

type Note struct {
	Value    int
	Start    float64
	Duration float64
	IsPause  bool

	Channel int
	Track   int
}

func binarySearchNote(slice []Note, start float64) (int, bool) {
	return slices.BinarySearchFunc(slice, Note{Start: start}, func(element Note, target Note) int {
		if target.Start > element.Start {
			return -1
		}
		if target.Start < element.Start {
			return 1
		}
		return 0
	})
}

func getFromTo(slice []Note, from float64, to float64) []Note {

	i, isNoteOnFrom := binarySearchNote(slice, from)
	j, _ := binarySearchNote(slice, to)

	if !isNoteOnFrom {
		i -= 1
	}

	slice = slice[i:j]

	if len(slice) == 0 {
		return slice
	}

	slice[0].Start = from

	return slice
}
