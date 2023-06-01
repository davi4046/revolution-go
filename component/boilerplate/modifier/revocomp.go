package main

type modifier struct {
}

func newModifier(

/************* Parameters **************

	Any parameters declared here
	will automatically be exposed
	as parameters of the generator.

***************************************/

) modifier {

	/*********** Initialization ************

		Perform any one-time operations
		here and save the result in the
		generator struct.

		In this case there are no one-
		time operations and we simply
		parse the parameters as is.

	***************************************/

	return modifier{}
}

func (m modifier) modify(in []struct {
	pitch    int
	duration float64
}) []struct {
	pitch    int
	duration float64
} {

	/************* Generation **************

		Implement the per-note generation
		logic here. Try to keep it fairly
		lightweight to ensure performance.

	***************************************/
	return in
}
