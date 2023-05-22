package astutil

import "go/ast"

func FindTypeSpecByName(astFile *ast.File, name string) (ast.TypeSpec, bool) {
	for _, decl := range astFile.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if typeSpec.Name.Name == name {
						return *typeSpec, true
					}
				}
			}
		}
	}
	return ast.TypeSpec{}, false
}
