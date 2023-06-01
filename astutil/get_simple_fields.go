package astutil

import (
	"go/ast"
	"go/types"
)

func GetSimpleFields(fields []*ast.Field) []SimpleField {

	var simpleFields []SimpleField

	for _, field := range fields {
		fieldType := types.ExprString(field.Type)
		if len(field.Names) == 0 {
			simpleFields = append(simpleFields, SimpleField{
				Name: "",
				Type: fieldType,
			})
		} else {
			for _, nameIdent := range field.Names {
				simpleFields = append(simpleFields, SimpleField{
					Name: nameIdent.Name,
					Type: fieldType,
				})
			}
		}
	}
	return simpleFields
}
