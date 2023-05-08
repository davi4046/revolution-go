package FileManagement

import (
	"os"
	"path/filepath"
)

// Create a new template from the current project.
func CreateTemplate(title, outDir string) error {

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	xmlPath := filepath.Join(wd, ".rlml")

	data, err := os.ReadFile(xmlPath)
	if err != nil {
		return err
	}

	tmplPath := filepath.Join(outDir, title+".template")

	err = os.WriteFile(tmplPath, data, 0777)
	if err != nil {
		return err
	}

	return nil
}
