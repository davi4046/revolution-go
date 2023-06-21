package astutil

import "go/ast"

func FindFuncDeclByName(astFile *ast.File, name string) *ast.FuncDecl {
	for _, decl := range astFile.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if funcDecl.Name.Name == name {
				return funcDecl
			}
		}
	}
	return nil
}
