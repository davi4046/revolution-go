package filemanage

func CreateGenerator(name, outDir string) error {
	template := `package main

import "components/types"

type %[1]s struct {
	/*
		Any variables declared here
		will automatically be exposed
		as parameters of %[1]s.
	*/
}

func (g %[1]s) Generate(i int) types.Note {

	return types.Note{
		Midinote: 60,
		Duration: 1,
	}
}
`

	if err := createComponent(name, template, outDir); err != nil {
		return err
	}

	return nil
}
