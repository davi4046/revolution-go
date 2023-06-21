package astutil

import (
	"go/ast"
	"sort"
)

// alphabetical sort
func SortParameters(fn *ast.FuncDecl) {
	sort.Slice(fn.Type.Params.List, func(i, j int) bool {
		name1 := fn.Type.Params.List[i].Names[0].Name
		name2 := fn.Type.Params.List[j].Names[0].Name
		return name1 < name2
	})
}
