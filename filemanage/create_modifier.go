package filemanage

func CreateModifier(name, outDir string) error {
	template := `package main

import "components/types"

type %[1]s struct {
	/*
		Any variables declared here
		will automatically be exposed
		as parameters of %[1]s.
	*/
}

func (m %[1]s) Modify(in [][]types.Note) [][]types.Note {

	return in
}
`

	if err := createComponent(name, template, outDir); err != nil {
		return err
	}

	return nil
}
