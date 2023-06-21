package component

import (
	"fmt"
	"go/ast"
	"go/token"
	"revolution/astutil"
	"revolution/strtags"
	"strings"

	"github.com/beevik/etree"
)

func generateAttributesFromFields(fset *token.FileSet, astFile *ast.File, fields []*ast.Field) ([]etree.Element, error) {
	var attributes []etree.Element

	for _, field := range fields {

		restrictions := make(map[string]string)
		var documentation string

		comment, ok := astutil.GetCommentAtField(fset, astFile, field)
		if ok {
			tags := strtags.Extract(comment.Text)
			for _, tag := range tags {
				if tag.Name == "restrict" {
					for _, option := range tag.Options {
						option = strings.TrimSpace(option)
						key, value, ok := strings.Cut(option, "=")
						if !ok {
							return nil, fmt.Errorf("invalid restriction '%s' on a parameter of function 'NewGenerator'", option)
						}
						restrictions[key] = value
					}
				}
				if tag.Name == "doc" {
					documentation = strings.Join(tag.Options, ",")
				}
			}
		}

		typeIdent, ok := field.Type.(*ast.Ident)
		if !ok {
			continue
		}

		for _, nameIdent := range field.Names {

			attribute, err := generateAttribute(
				nameIdent.Name,
				typeIdent.Name,
				documentation,
				restrictions,
			)
			if err != nil {
				return nil, err
			}
			attributes = append(attributes, attribute)
		}
	}

	return attributes, nil
}
