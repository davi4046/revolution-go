package interpret

type genItem struct {
	// Start point of the item on the track in bars.
	barStart float64
	// End point of the item on the track in bars.
	barEnd float64
	// Start point of the item on the track in whole notes.
	noteStart float64
	// End point of the item on the track in whole notes.
	noteEnd float64
	// Offset of the generation in bars.
	barOffset float64
	// Offset of the generation in whole notes.
	noteOffset float64

	channel int
	track   int

	add int
	sub int
}
