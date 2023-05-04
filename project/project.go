package project

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Creates a new project in the current working directory.
func Create(title, template string) (err error) {

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

	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	dirPath := filepath.Join(dir, title)

	err = os.MkdirAll(dirPath, 7777)
	if err != nil {
		return err
	}

	xmlPath := filepath.Join(dirPath, title+".xml")
	tmplXml := tmpl["xml"].(string)

	err = os.WriteFile(xmlPath, []byte(tmplXml), 7777)
	if err != nil {
		return err
	}

	return nil
}
