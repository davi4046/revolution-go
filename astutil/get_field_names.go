package astutil

import "go/ast"

func GetFieldNames(field *ast.Field) []string {

	var names []string

	for _, nameIdent := range field.Names {
		names = append(names, nameIdent.Name)
	}

	return names
}
