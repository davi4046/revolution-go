package filemanage

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/iancoleman/strcase"
)

func CreateGenerator(name, outDir string) error {

	var data = `package components

import "components/types"

type Generator struct {
	/*
		Any variables declared here
		will automatically be exposed
		as parameters of Generator.
	*/
}

func (g Generator) Generate(i int) types.Note {

	return types.Note{
		Midinote: 60,
		Duration: 1,
	}
}
`
	data = strings.ReplaceAll(data, "Generator", strcase.ToCamel(name))

	fileName := filepath.Join(outDir, strcase.ToSnake(name)+".gen.go")

	if _, err := os.Stat(fileName); !os.IsNotExist(err) {
		log.Fatalln("Failed to create generator: A generator by the same name already exists.")
	}

	if err := os.WriteFile(fileName, []byte(data), 0777); err != nil {
		return err
	}

	return nil
}
