package filemanage

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/iancoleman/strcase"
)

// Creates a new component in the current working directory.
func createComponent(name, base string) error {

	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	camelCaseName := strcase.ToCamel(name)
	snakeCaseName := strcase.ToSnake(name)

	compData := fmt.Sprintf(base, camelCaseName) // Insert name into template

	compDir := filepath.Join(wd, snakeCaseName)

	if _, err := os.Stat(compDir); !os.IsNotExist(err) {
		log.Fatalln("Failed to create component: A component by the same name already exists.")
	}

	if err := os.Mkdir(compDir, 0777); err != nil {
		return err
	}

	path := filepath.Join(compDir, snakeCaseName+".go")

	if err := os.WriteFile(path, []byte(compData), 0777); err != nil {
		return err
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
