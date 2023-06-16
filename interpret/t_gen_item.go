package interpret

type genItem struct {
	channel int
	track   int
	// Start point of the item on the track in bars.
	barStart float64
	// End point of the item on the track in bars.
	barEnd float64
	// Start point of the item on the track in whole notes.
	noteStart float64
	// End point of the item on the track in whole notes.
	noteEnd float64
	// Offset of the generation in whole notes.
	noteOffset float64
	add        int
	sub        int
}
