package interpret

type modItem struct {
	// Start point of the item on the track in bars.
	barStart float64
	// End point of the item on the track in bars.
	barEnd float64

	target string
}
