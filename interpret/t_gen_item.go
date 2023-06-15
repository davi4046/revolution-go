package interpret

type genItem struct {
	channel int
	track   int
	// The index (i.e. the order) of the item on its track.
	index int
	// The start point on the track (in bars).
	start float64
	// The end point on the track (in bars).
	end float64
	// The offset of the generation (in whole notes).
	offset float64
	// The length of the item (in whole notes).
	length float64
	add    int
	sub    int
}
