package interpret

import (
	"math"
)

func barToWholeNote(bar float64, changes []change) float64 {
	var wholeNote float64

	for i, change := range changes {

		if change.barStart > bar {
			break
		}

		var end float64

		if len(changes) > i+1 {
			changeEnd := changes[i+1].barStart
			end = math.Min(bar, changeEnd)
		} else {
			end = bar
		}

		wholeNotesPerBar := change.time.GetWholeNotesPerBar()

		wholeNote += (end - change.barStart) * wholeNotesPerBar
	}

	return wholeNote
}
