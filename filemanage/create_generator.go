package filemanage

import (
	"log"
	"os"
	"path/filepath"
)

func CreateGenerator(name, outDir string) error {

	var data = []byte(
		`package components

	import "components/types"

	type Generator struct {
		/*
			Any variables declared here
			will automatically be exposed
			as parameters of the generator.
		*/
	}

	func (g Generator) Generate(i int) types.Note {

		return types.Note{
			Midinote: 60,
			Duration: 1,
		}
	}
	`)

	fileName := filepath.Join(outDir, toKebabCase(name)+".gen.go")

	if _, err := os.Stat(fileName); !os.IsNotExist(err) {
		log.Fatalln("Failed to create generator: A generator with the same name already exists.")
	}

	if err := os.WriteFile(fileName, data, 0777); err != nil {
		return err
	}

	return nil
}
