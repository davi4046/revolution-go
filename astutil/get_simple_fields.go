package astutil

import "go/ast"

func GetSimpleFields(fields []*ast.Field) []SimpleField {

	var simpleFields []SimpleField

	for _, field := range fields {
		typeIdent, ok := field.Type.(*ast.Ident)
		if !ok {
			continue
		}
		if len(field.Names) == 0 {
			simpleFields = append(simpleFields, SimpleField{
				Name: "",
				Type: typeIdent.Name,
			})
		} else {
			for _, nameIdent := range field.Names {
				simpleFields = append(simpleFields, SimpleField{
					Name: nameIdent.Name,
					Type: typeIdent.Name,
				})
			}
		}
	}
	return simpleFields
}
