package main

import "math/rand"

type Generator struct {
	seed   int64
	minDur float64
	maxDur float64
	minDeg int
	maxDeg int
}

func NewGenerator(

	/************* Parameters **************

		Any parameters declared here
		will automatically be exposed
		as parameters of the generator.

	***************************************/

	seed int64,
	minDur float64, // @restrict minInclusive=0.25, maxInclusive=10 @doc This is the documentation.
	maxDur float64,
	minDeg int,
	maxDeg int,

) Generator {

	/*********** Initialization ************

		Perform any one-time operations
		here and save the result in the
		generator struct.

		In this case there are no one-
		time operations and we simply
		parse the parameters as is.

	***************************************/

	return Generator{
		seed:   seed,
		minDur: minDur,
		maxDur: maxDur,
		minDeg: minDeg,
		maxDeg: maxDeg,
	}
}

func (g Generator) Generate(i int) (degree int, duration float64) {

	/************* Generation **************

		Implement the per-note generation
		logic here. Try to keep it fairly
		lightweight to ensure performance.

	***************************************/

	// Custom seed for this particular generation.
	// Ensures that the result is always the same.
	var seed = g.seed + int64(i)

	r := rand.New(rand.NewSource(seed))

	// Random integer value between minDeg and maxDeg.
	degree = r.Intn(g.maxDeg-g.minDeg) + g.minDeg

	// Random float value between minDur and maxDur.
	duration = r.Float64()*(g.maxDur-g.minDur) + g.minDur

	return
}
