package interpret

import "golang.org/x/exp/slices"

type note struct {
	start  float64
	degree int
}

func binarySearchNote(slice []note, start float64) (int, bool) {
	return slices.BinarySearchFunc(slice, note{start: start}, func(element note, target note) int {
		if target.start > element.start {
			return -1
		}
		if target.start < element.start {
			return 1
		}
		return 0
	})
}

func getFromTo(slice []note, from float64, to float64) []note {

	i, isNoteOnFrom := binarySearchNote(slice, from)
	j, _ := binarySearchNote(slice, to)

	if !isNoteOnFrom {
		i -= 1
	}

	slice = slice[i:j]

	if len(slice) == 0 {
		return slice
	}

	slice[0].start = from

	return slice
}
