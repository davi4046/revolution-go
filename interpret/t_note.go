package interpret

import (
	"github.com/davi4046/revoutil"
	"golang.org/x/exp/slices"
)

type Note struct {
	Value int

	Start    float64
	Duration float64

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

func replace(s []Note, i int, j int, e []revoutil.Note) []Note {
	var replacement []Note

	currTime := s[i].Start

	for _, note := range e {
		replacement = append(replacement, Note{
			Value:    note.Pitch,
			Start:    currTime,
			Duration: note.Duration,
			Channel:  note.Channel,
			Track:    note.Track,
		})
		currTime += note.Duration
	}

	return slices.Replace(s, i, j, replacement...)
}
