package main

import "github.com/davi4046/revoutil"

type modifier struct{}

func newModifier() modifier {
	return modifier{}
}

func (m modifier) modify(note revoutil.Note) []revoutil.Note {
	return []revoutil.Note{note}
}
