package filemanage

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/iancoleman/strcase"
)

func CreateGenerator(name, outDir string) error {

	/* Check that a component by the same name doesn't exist */

	files, err := os.ReadDir(outDir)
	if err != nil {
		return err
	}

	r := regexp.MustCompile(fmt.Sprintf(`(gen|mod).%s.go`, name))

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if r.MatchString(file.Name()) {
			log.Fatalln("Failed to create generator: A component by the specified name already exists.")
		}
	}

	// Initial generator code
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

	fileName := fmt.Sprintf("gen.%s.go", strcase.ToSnake(name))
	filePath := filepath.Join(outDir, fileName)

	if err := os.WriteFile(filePath, []byte(data), 0777); err != nil {
		return err
	}

	return nil
}
