package astutil

import "go/ast"

func FindFuncDeclByName(astFile *ast.File, name string) (ast.FuncDecl, bool) {
	for _, decl := range astFile.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if funcDecl.Name.Name == name {
				return *funcDecl, true
			}
		}
	}
	return ast.FuncDecl{}, false
}
