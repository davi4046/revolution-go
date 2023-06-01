package component

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/iancoleman/strcase"
)

//go:embed boilerplate/generator/revocomp.go
//go:embed boilerplate/generator/revocomp.yaml
//go:embed boilerplate/generator/main_func.tmpl
//go:embed boilerplate/modifier/revocomp.go
//go:embed boilerplate/modifier/revocomp.yaml
//go:embed boilerplate/modifier/main_func.tmpl
var files embed.FS

// createComponent creates a new component in the current working directory
// with the specified name and component type.
func createComponent(name, compType string) error {

	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	if compType != "generator" && compType != "modifier" {
		return fmt.Errorf("'%s' is not a valid component type", compType)
	}

	snakeCaseName := strcase.ToSnake(name)

	compDir := filepath.Join(wd, strcase.ToSnake(name))

	if err := os.Mkdir(compDir, 0777); err != nil {
		return err
	}

	goFilePath := filepath.Join(compDir, "revocomp.go")
	yamlFilePath := filepath.Join(compDir, "revocomp.yaml")

	switch compType {
	case "generator":

		goData, err := files.ReadFile("boilerplate/generator/revocomp.go")
		if err != nil {
			return err
		}

		yamlData, err := files.ReadFile("boilerplate/generator/revocomp.yaml")
		if err != nil {
			return err
		}

		if err := os.WriteFile(goFilePath, goData, 0777); err != nil {
			return err
		}

		if err := os.WriteFile(yamlFilePath, yamlData, 0777); err != nil {
			return err
		}
	case "modifier":

		goData, err := files.ReadFile("boilerplate/modifier/revocomp.go")
		if err != nil {
			return err
		}

		yamlData, err := files.ReadFile("boilerplate/modifier/revocomp.yaml")
		if err != nil {
			return err
		}

		if err := os.WriteFile(goFilePath, goData, 0777); err != nil {
			return err
		}

		if err := os.WriteFile(yamlFilePath, yamlData, 0777); err != nil {
			return err
		}
	}

	if err := os.Chdir(compDir); err != nil {
		return err
	}

	cmd := exec.Command("go", "mod", "init", snakeCaseName)

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
