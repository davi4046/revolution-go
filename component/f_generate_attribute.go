package component

import (
	"fmt"
	"strings"

	"github.com/beevik/etree"
)

func generateAttribute(name, goType, doc string, restrictions map[string]string) (etree.Element, error) {

	base, ok := typeMap[strings.TrimPrefix(goType, "[]")]
	if !ok {
		return etree.Element{}, fmt.Errorf("type %s is not supported", goType)
	}

	attribute := etree.NewElement("xs:attribute")
	attribute.CreateAttr("name", name)
	attribute.CreateAttr("use", "required")

	if doc != "" {
		annotation := attribute.CreateElement("xs:annotation")
		documentation := annotation.CreateElement("xs:documentation")
		documentation.SetText(doc)
	}

	simpleType := attribute.CreateElement("xs:simpleType")

	restriction := etree.NewElement("xs:restriction")
	restriction.CreateAttr("base", base)

	if strings.HasPrefix(goType, "[]") {
		list := simpleType.CreateElement("xs:list")
		simpleType := list.CreateElement("xs:simpleType")
		simpleType.AddChild(restriction)
	} else {
		simpleType.AddChild(restriction)
	}

	for key, value := range restrictions {
		el := restriction.CreateElement("xs:" + key)
		el.CreateAttr("value", value)
	}

	return *attribute, nil
}
