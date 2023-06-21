package component

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/iancoleman/strcase"
	"gopkg.in/yaml.v3"
)

//go:embed boilerplate/*
var files embed.FS

func CreateComponent(name, kind string) error {

	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	dir := filepath.Join(wd, strcase.ToCamel(name))

	if err := os.Mkdir(dir, 0777); err != nil {
		return err
	}

	goFilePath := filepath.Join(dir, "revocomp.go")
	yamlFilePath := filepath.Join(dir, "revocomp.yaml")

	var goData []byte
	var yamlData []byte

	switch kind {
	case "generator":

		goData, err = files.ReadFile("boilerplate/generator/revocomp.go")
		if err != nil {
			return err
		}

		yamlData, err = files.ReadFile("boilerplate/generator/revocomp.yaml")
		if err != nil {
			return err
		}

	case "modifier":

		goData, err = files.ReadFile("boilerplate/modifier/revocomp.go")
		if err != nil {
			return err
		}

		yamlData, err = files.ReadFile("boilerplate/modifier/revocomp.yaml")
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid component kind: %s", kind)
	}

	if err := os.WriteFile(goFilePath, goData, 0777); err != nil {
		return err
	}

	data := make(map[string]interface{})

	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return err
	}

	data["name"] = strcase.ToCamel(name)

	yamlData, err = yaml.Marshal(data)
	if err != nil {
		return err
	}

	if err := os.WriteFile(yamlFilePath, yamlData, 0777); err != nil {
		return err
	}

	if err := os.Chdir(dir); err != nil {
		return err
	}

	cmd := exec.Command("go", "mod", "init", strcase.ToSnake(name))

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
