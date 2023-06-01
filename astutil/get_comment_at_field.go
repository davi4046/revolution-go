package astutil

import (
	"go/ast"
	"go/token"
)

func GetCommentAtField(fset *token.FileSet, astFile *ast.File, object *ast.Field) (ast.Comment, bool) {
	line := fset.Position(object.Pos()).Line
	comment, ok := GetCommentAtLine(fset, astFile, line)
	return comment, ok
}
