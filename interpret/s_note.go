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
