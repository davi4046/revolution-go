package component

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"revolution/astutil"
	"strings"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

func validateComponent(dir string) error {

	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		return err
	}

	yamlData, err := os.ReadFile("revocomp.yaml")
	if err != nil {
		return err
	}

	var info Info
	if err := yaml.Unmarshal(yamlData, &info); err != nil {
		return err
	}

	if emptyFields := getEmptyFields(info); len(emptyFields) != 0 {
		return fmt.Errorf(`%s is missing from revocomp.yaml`, strings.Join(emptyFields, ", "))
	}

	data, err := os.ReadFile("revocomp.go")
	if err != nil {
		return err
	}

	astFile, err := parser.ParseFile(token.NewFileSet(), "", data, 0)
	if err != nil {
		return err
	}

	// Validate type-specific functions
	switch info.Type {
	case "generator":
		// Find and validate 'generator' type.
		if decl, ok := astutil.FindTypeSpecByName(astFile, "Generator"); ok {
			if _, ok := decl.Type.(*ast.StructType); !ok {
				return errors.New("type 'Generator' must be a struct")
			}
		} else {
			return errors.New("type 'Generator' is missing from revocomp.go")
		}
		// Find and validate 'newGenerator' function.
		if decl, ok := astutil.FindFuncDeclByName(astFile, "NewGenerator"); ok {
			for _, param := range astutil.GetSimpleFields(decl.Type.Params.List) {
				if !slices.Contains(maps.Keys(typeMap), param.Type) {
					return fmt.Errorf("parameter %s of function 'NewGenerator' is an unsupported type", param.Name)
				}
			}

			if !slices.Equal(
				astutil.GetSimpleFields(decl.Type.Results.List),
				[]astutil.SimpleField{{
					Name: "",
					Type: "Generator",
				}},
			) {
				return errors.New("function 'NewGenerator' must have exactly one unnamed result of type 'Generator'")
			}
		} else {
			return errors.New("function 'NewGenerator' is missing from revocomp.go")
		}
		// Find and validate 'generate' function.
		if decl, ok := astutil.FindFuncDeclByName(astFile, "Generate"); ok {
			if !slices.Equal(
				astutil.GetSimpleFields(decl.Type.Params.List),
				[]astutil.SimpleField{{
					Name: "i",
					Type: "int",
				}},
			) {
				return errors.New("function 'Generate' must have exactly one parameter named 'i' of type 'int'")
			}

			if !slices.Equal(
				astutil.GetSimpleFields(decl.Type.Results.List),
				[]astutil.SimpleField{{
					Name: "degree",
					Type: "int",
				}, {
					Name: "duration",
					Type: "float64",
				}},
			) {
				return errors.New("function 'Generate' must return exactly two results named 'degree' and 'duration' of type 'int' and 'float64'")
			}
		} else {
			return errors.New("function 'Generate' is missing from revocomp.go")
		}
	case "modifier":
		// Find and validate 'modifier' type.
		if decl, ok := astutil.FindTypeSpecByName(astFile, "Modifier"); ok {
			if _, ok := decl.Type.(*ast.StructType); !ok {
				return errors.New("type 'Modifier' must be a struct")
			}
		} else {
			return errors.New("type 'Modifier' is missing from revocomp.go")
		}
		// Find and validate 'newModifier' function.
		if decl, ok := astutil.FindFuncDeclByName(astFile, "NewModifier"); ok {
			for _, param := range astutil.GetSimpleFields(decl.Type.Params.List) {
				baseType := strings.TrimPrefix(param.Type, "[]")
				if !slices.Contains(maps.Keys(typeMap), baseType) {
					return fmt.Errorf("parameter %s of function 'NewModifier' is an unsupported type", param.Name)
				}
			}

			if !slices.Equal(
				astutil.GetSimpleFields(decl.Type.Results.List),
				[]astutil.SimpleField{{
					Name: "",
					Type: "Modifier",
				}},
			) {
				return errors.New("function 'NewModifier' must have exactly one unnamed result of type of 'Modifier'")
			}
		} else {
			return errors.New("function 'NewModifier' is missing from revocomp.go")
		}
		// Find and validate 'modify' function.
		if decl, ok := astutil.FindFuncDeclByName(astFile, "Modify"); ok {

			fmt.Println(astutil.GetSimpleFields(decl.Type.Params.List))

			if !slices.Equal(
				astutil.GetSimpleFields(decl.Type.Params.List),
				[]astutil.SimpleField{{
					Name: "note",
					Type: "revoutil.Note",
				}},
			) {
				return errors.New("function 'Modify' must have exactly one parameter named 'note' of type 'revoutil.Note'")
			}

			if !slices.Equal(
				astutil.GetSimpleFields(decl.Type.Results.List),
				[]astutil.SimpleField{{
					Name: "",
					Type: "[]revoutil.Note",
				}},
			) {
				return errors.New("function 'Modify' must return exactly one unnamed result of type 'revoutil.Note'")
			}
		} else {
			return errors.New("function 'Modify' is missing from revocomp.go")
		}
	default:
		return fmt.Errorf("component type is invalid")
	}

	return nil
}
