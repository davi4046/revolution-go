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

func CreateModifier(name, outDir string) error {

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

	// Initial modifier code
	var data = `package components

import "components/types"

type Modifier struct {
	/*
		Any variables declared here
		will automatically be exposed
		as parameters of Modifier.
	*/
}

func (m Modifier) Modify(in [][]types.Note) [][]types.Note {

	return in
}
`
	data = strings.ReplaceAll(data, "Modifier", strcase.ToCamel(name))

	fileName := fmt.Sprintf("mod.%s.go", strcase.ToSnake(name))
	filePath := filepath.Join(outDir, fileName)

	if err := os.WriteFile(filePath, []byte(data), 0777); err != nil {
		return err
	}

	return nil
}
