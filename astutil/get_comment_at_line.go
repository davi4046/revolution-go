package astutil

import (
	"go/ast"
	"go/token"
)

func GetCommentAtLine(fset *token.FileSet, astFile *ast.File, line int) (ast.Comment, bool) {
	for _, commentGroup := range astFile.Comments {
		for _, comment := range commentGroup.List {
			commentLine := fset.Position(comment.Pos()).Line
			if commentLine == line {
				return *comment, true
			}
		}
	}
	return ast.Comment{}, false
}
