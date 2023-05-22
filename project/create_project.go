package project

import (
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/viper"
)

//go:embed xsd
var xsdData []byte

// Creates a new project in the current working directory.
func CreateProject(name, template string) error {

	/* Get template data */

	assetDir := viper.GetString("asset_directory")
	tmplPath := filepath.Join(assetDir, "templates", template+".template")

	data, err := os.ReadFile(tmplPath)
	if err != nil {
		return err
	}

	/* Create project directory */

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	projDir := filepath.Join(wd, name)

	if err = os.MkdirAll(projDir, 0777); err != nil {
		return err
	}

	/* Create .xsd file */

	xsdPath := filepath.Join(projDir, ".xsd")

	if err = os.WriteFile(xsdPath, xsdData, 0777); err != nil {
		return err
	}

	/* Create .rlml file */

	xmlPath := filepath.Join(projDir, ".rlml")
	tmplXml := string(data)

	if err = os.WriteFile(xmlPath, []byte(tmplXml), 0777); err != nil {
		os.Remove(projDir)
		return err
	}

	/* Initialize git repository */

	cmd := exec.Command("git", "init", projDir)
	_, err = cmd.Output()
	if err == nil {

		/* Create .gitignore file */

		path := filepath.Join(projDir, ".gitignore")
		content := "/.extensions"

		os.WriteFile(path, []byte(content), 0777)
	}

	return nil
}
