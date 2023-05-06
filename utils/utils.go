package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Creates a new project in the current working directory.
func CreateProject(title, template string) error {

	/* Get template properties */

	tmplPath := filepath.Join("templates", template+".template")

	data, err := os.ReadFile(tmplPath)
	if err != nil {
		return err
	}

	/* Create project directory */

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	projDir := filepath.Join(wd, title)

	err = os.MkdirAll(projDir, 0777)
	if err != nil {
		return err
	}

	/* Create .xsd file */

	xsdPath := filepath.Join(projDir, ".xsd")

	err = CopyFile(".xsd", xsdPath)
	if err != nil {
		fmt.Println(err)
		return err
	}

	/* Create .rlml file */

	xmlPath := filepath.Join(projDir, ".rlml")
	tmplXml := string(data)

	err = os.WriteFile(xmlPath, []byte(tmplXml), 0777)
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

		os.WriteFile(path, []byte(content), 0777)
	}

	return nil
}

// Create a new template from the current project.
func SaveAsTemplate(title, outDir string) error {

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

func CopyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	err = os.WriteFile(dst, data, 0777)
	if err != nil {
		return err
	}

	return nil
}
