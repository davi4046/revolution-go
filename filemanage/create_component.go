package filemanage

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/iancoleman/strcase"
)

func createComponent(name, template, outDir string) error {
	camelCaseName := strcase.ToCamel(name)
	snakeCaseName := strcase.ToSnake(name)
	lowerCaseName := strings.ToLower(camelCaseName)

	data := fmt.Sprintf(template, camelCaseName) // Insert name into template

	dir := filepath.Join(outDir, snakeCaseName)

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		log.Fatalln("Failed to create component: A component by the same name already exists.")
	}

	if err := os.Mkdir(dir, 0777); err != nil {
		return err
	}

	path := filepath.Join(dir, snakeCaseName+".go")

	if err := os.WriteFile(path, []byte(data), 0777); err != nil {
		return err
	}

	if err := os.Chdir(dir); err != nil {
		return err
	}

	cmd := exec.Command("go", "mod", "init", lowerCaseName)

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
