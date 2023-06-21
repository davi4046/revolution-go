package main

import "github.com/davi4046/revoutil"

type Modifier struct{}

func NewModifier() Modifier {
	return Modifier{}
}

func (m Modifier) Modify(note revoutil.Note) []revoutil.Note {
	return []revoutil.Note{note}
}
