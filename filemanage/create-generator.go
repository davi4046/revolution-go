package filemanage

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var data = []byte("blah blah")

func CreateGenerator(name, outDir string) error {

	fileName := filepath.Join(outDir, toKebabCase(name)+".gen.go")

	if _, err := os.Stat(fileName); !os.IsNotExist(err) {
		log.Fatalln("Failed to create generator: A generator with the same name already exists.")
	}

	if err := os.WriteFile(fileName, data, 0777); err != nil {
		return err
	}

	return nil
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toKebabCase(s string) string {
	snake := matchFirstCap.ReplaceAllString(s, "${1}-${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}-${2}")
	return strings.ToLower(snake)
}
