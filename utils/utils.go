package utils

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
)

// Creates a new project in the current working directory.
func CreateProject(title, template string) (err error) {

	/* Get template properties */

	tmplPath := filepath.Join("templates", template+".template")

	data, err := os.ReadFile(tmplPath)
	if err != nil {
		return err
	}

	var tmpl map[string]interface{}

	err = json.Unmarshal(data, &tmpl)
	if err != nil {
		return err
	}

	/* Create project directory */

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	projDir := filepath.Join(wd, title)

	err = os.MkdirAll(projDir, 7777)
	if err != nil {
		return err
	}

	/* Create .xml file */

	xmlPath := filepath.Join(projDir, title+".xml")
	tmplXml := tmpl["xml"].(string)

	err = os.WriteFile(xmlPath, []byte(tmplXml), 7777)
	if err != nil {
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

		os.WriteFile(path, []byte(content), 7777)
	}

	return nil
}
