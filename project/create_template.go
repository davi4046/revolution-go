package project

import (
	"os"
	"path/filepath"
)

// Creates a new template from the current project.
func CreateTemplate(name, outDir string) error {

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	xmlPath := filepath.Join(wd, ".rlml")

	data, err := os.ReadFile(xmlPath)
	if err != nil {
		return err
	}

	tmplPath := filepath.Join(outDir, name+".template")

	err = os.WriteFile(tmplPath, data, 0777)
	if err != nil {
		return err
	}

	return nil
}
