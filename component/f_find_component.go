package component

import (
	"errors"
	"io/fs"
	"os/exec"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

func FindComponent(name, kind, version string) (string, bool) {
	resourceDir := viper.GetString("resource_directory")
	if resourceDir == "" {
		return "", false
	}

	componentDir := filepath.Join(resourceDir, "components")

	var componentPath string

	filepath.Walk(componentDir, func(path string, info fs.FileInfo, err error) error {

		if err != nil {
			return nil // Continue
		}

		ext := filepath.Ext(info.Name())
		if ext != ".revocomp" {
			return nil // Continue
		}

		cmd := exec.Command(path, "info")
		output, err := cmd.Output()
		if err != nil {
			return nil // Continue
		}

		var componentInfo Info
		if err := yaml.Unmarshal(output, &componentInfo); err != nil {
			return nil // Continue
		}

		isSameName := componentInfo.Name == name
		isSameType := componentInfo.Type == kind
		isSameVersion := componentInfo.Version == version

		if !isSameName || !isSameType || !isSameVersion {
			return nil // Continue
		}

		componentPath = path

		return errors.New("") // Break

	})

	if componentPath == "" {
		return "", false
	}

	return componentPath, true
}
