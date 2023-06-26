package project

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Creates a new template from the current project.
func CreateTemplate(name string) error {

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	xmlPath := filepath.Join(wd, "revoproj.xml")

	data, err := os.ReadFile(xmlPath)
	if err != nil {
		return err
	}

	resDir := viper.GetString("resource_directory")
	if resDir == "" {
		return errors.New("resource directory is unspecified")
	}

	tmplPath := filepath.Join(resDir, "templates", name+".template")

	err = os.WriteFile(tmplPath, data, 0777)
	if err != nil {
		return err
	}

	return nil
}
