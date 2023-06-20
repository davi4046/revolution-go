package interpret

type modItem struct {
	// Start point of the item on the track in bars.
	barStart float64
	// End point of the item on the track in bars.
	barEnd float64
	// Start point of the item on the track in whole notes.
	noteStart float64
	// End point of the item on the track in whole notes.
	noteEnd float64

	target string
}
