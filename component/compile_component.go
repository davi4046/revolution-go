package component

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"revolution/astutil"
	"revolution/randutil"
	"revolution/strtags"
	"strings"
	"text/template"
	"time"

	_ "embed"

	"github.com/beevik/etree"
	"github.com/iancoleman/strcase"
	"github.com/otiai10/copy"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

// CompileComponent compiles the current working directory as a Revolution component
// and outputs an executable in specified outDir.
func CompileComponent(outDir string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	if err := validateComponent(wd); err != nil {
		return err
	}

	// Create temporary directory
	tempDirName := fmt.Sprintf("revolution_compilation_%d", time.Now().Unix())
	tempDir := filepath.Join(os.TempDir(), tempDirName)

	copy.Copy(wd, tempDir)

	defer os.RemoveAll(tempDir)

	// Read component info
	xsdData, _ := os.ReadFile("revocomp.yaml")
	var info Info
	yaml.Unmarshal(xsdData, &info)

	srcCode, _ := os.ReadFile(filepath.Join(wd, "revocomp.go"))
	fset := token.NewFileSet()
	astFile, _ := parser.ParseFile(fset, "", srcCode, parser.ParseComments)

	var funcName string
	switch info.Type {
	case "generator":
		funcName = "newGenerator"
	case "modifier":
		funcName = "newModifier"
	}

	funcDecl, _ := astutil.FindFuncDeclByName(astFile, funcName)
	params := funcDecl.Type.Params.List

	// Create XSD file

	attributes, err := generateAttributesFromFields(fset, astFile, params)
	if err != nil {
		return err
	}

	element := etree.NewElement("xs:element")
	elementName := info.Name + "-" + info.Version
	element.CreateAttr("name", elementName)
	complexType := element.CreateElement("xs:complexType")

	for _, attribute := range attributes {
		complexType.AddChild(attribute.Copy())
	}

	doc := etree.NewDocument()
	doc.SetRoot(element)
	doc.IndentTabs()

	xsdData, err = doc.WriteToBytes()
	if err != nil {
		return err
	}

	xsdFileName := randutil.GetRandomString(20)
	xsdFilePath := filepath.Join(tempDir, xsdFileName)
	if err := os.WriteFile(xsdFilePath, xsdData, 0777); err != nil {
		return err
	}

	// Create main file

	simpleParams := astutil.GetSimpleFields(params)

	var conversions []string

	for i, param := range simpleParams {
		convCode, err := generateConversionCode(
			param.Name,
			param.Type,
			fmt.Sprintf("os.Args[%d]", i+1),
		)
		if err != nil {
			return err
		}

		conversions = append(conversions, convCode)
	}

	var paramNames []string

	for _, param := range simpleParams {
		paramNames = append(paramNames, param.Name)
	}

	data := struct {
		XSDFileName, Conversions, NArgs, Args string
	}{
		XSDFileName: xsdFileName,
		Conversions: strings.Join(conversions, "; "),
		Args:        strings.Join(paramNames, ", "),
		NArgs:       fmt.Sprint(len(paramNames) + 1),
	}

	mainTmpl := template.New("mainTmpl")

	switch info.Type {
	case "generator":
		tmplData, err := files.ReadFile("boilerplate/generator/main_func.tmpl")
		if err != nil {
			return err
		}

		if _, err := mainTmpl.Parse(string(tmplData)); err != nil {
			return err
		}
	case "modifier":
		tmplData, err := files.ReadFile("boilerplate/modifier/main_func.tmpl")
		if err != nil {
			return err
		}

		if _, err := mainTmpl.Parse(string(tmplData)); err != nil {
			return err
		}
	}

	var builder strings.Builder
	if err := mainTmpl.Execute(&builder, data); err != nil {
		return err
	}

	mainFileName := randutil.GetRandomString(20) + ".go"
	mainFilePath := filepath.Join(tempDir, mainFileName)
	mainFileData := []byte(builder.String())
	if err := os.WriteFile(mainFilePath, mainFileData, 0777); err != nil {
		return err
	}

	buildName := randutil.GetRandomString(20)

	cmd := exec.Command("go", "build", "-o", buildName)
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		return errors.New("build failed")
	}

	src := filepath.Join(tempDir, buildName)
	destName := strcase.ToCamel(info.Name) + "@" + info.Version + ".revocomp"
	dest := filepath.Join(outDir, destName)

	if err := copy.Copy(src, dest); err != nil {
		return err
	}

	return nil
}

// validateComponent checks that the specified directory fulfills the necessary requirements for being a component.
func validateComponent(dir string) error {

	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		return err
	}

	yamlData, err := os.ReadFile("revocomp.yaml")
	if err != nil {
		return err
	}

	var info Info
	if err := yaml.Unmarshal(yamlData, &info); err != nil {
		return err
	}

	if emptyFields := getEmptyFields(info); len(emptyFields) != 0 {
		return fmt.Errorf(`%s is missing from revocomp.yaml`, strings.Join(emptyFields, ", "))
	}

	data, err := os.ReadFile("revocomp.go")
	if err != nil {
		return err
	}

	astFile, err := parser.ParseFile(token.NewFileSet(), "", data, 0)
	if err != nil {
		return err
	}

	// Validate type-specific functions
	switch info.Type {
	case "generator":
		// Find and validate 'generator' type.
		if decl, ok := astutil.FindTypeSpecByName(astFile, "generator"); ok {
			if _, ok := decl.Type.(*ast.StructType); !ok {
				return errors.New("type 'generator' must be a struct")
			}
		} else {
			return errors.New("type 'generator' is missing from revocomp.go")
		}
		// Find and validate 'newGenerator' function.
		if decl, ok := astutil.FindFuncDeclByName(astFile, "newGenerator"); ok {
			for _, param := range astutil.GetSimpleFields(decl.Type.Params.List) {
				if !slices.Contains(maps.Keys(typeMap), param.Type) {
					return fmt.Errorf("parameter %s of function 'newGenerator' is an unsupported type", param.Name)
				}
			}

			if !slices.Equal(
				astutil.GetSimpleFields(decl.Type.Results.List),
				[]astutil.SimpleField{{
					Name: "",
					Type: "generator",
				}},
			) {
				return errors.New("function 'newGenerator' must have exactly one unnamed result of type of 'generator'")
			}
		} else {
			return errors.New("function 'newGenerator' is missing from revocomp.go")
		}
		// Find and validate 'generate' function.
		if decl, ok := astutil.FindFuncDeclByName(astFile, "generate"); ok {
			if !slices.Equal(
				astutil.GetSimpleFields(decl.Type.Params.List),
				[]astutil.SimpleField{{
					Name: "i",
					Type: "int",
				}},
			) {
				return errors.New("function 'generate' must have exactly one parameter named 'i' of type 'int'")
			}

			if !slices.Equal(
				astutil.GetSimpleFields(decl.Type.Results.List),
				[]astutil.SimpleField{{
					Name: "degree",
					Type: "int",
				}, {
					Name: "duration",
					Type: "float64",
				}},
			) {
				return errors.New("function 'generate' must return exactly two results named 'degree' and 'duration' of type 'int' and 'float64'")
			}
		} else {
			return errors.New("function 'generate' is missing from revocomp.go")
		}
	case "modifier":
		// Find and validate 'modifier' type.
		if decl, ok := astutil.FindTypeSpecByName(astFile, "modifier"); ok {
			if _, ok := decl.Type.(*ast.StructType); !ok {
				return errors.New("type 'modifier' must be a struct")
			}
		} else {
			return errors.New("type 'modifier' is missing from revocomp.go")
		}
		// Find and validate 'newModifier' function.
		if decl, ok := astutil.FindFuncDeclByName(astFile, "newModifier"); ok {
			for _, param := range astutil.GetSimpleFields(decl.Type.Params.List) {
				baseType := strings.TrimPrefix(param.Type, "[]")
				if !slices.Contains(maps.Keys(typeMap), baseType) {
					return fmt.Errorf("parameter %s of function 'newModifier' is an unsupported type", param.Name)
				}
			}

			if !slices.Equal(
				astutil.GetSimpleFields(decl.Type.Results.List),
				[]astutil.SimpleField{{
					Name: "",
					Type: "modifier",
				}},
			) {
				return errors.New("function 'newModifier' must have exactly one unnamed result of type of 'modifier'")
			}
		} else {
			return errors.New("function 'newModifier' is missing from revocomp.go")
		}
		// Find and validate 'modify' function.
		if decl, ok := astutil.FindFuncDeclByName(astFile, "modify"); ok {

			fmt.Println(astutil.GetSimpleFields(decl.Type.Params.List))

			if !slices.Equal(
				astutil.GetSimpleFields(decl.Type.Params.List),
				[]astutil.SimpleField{{
					Name: "in",
					Type: "[]struct {degree int; duration float64}",
				}},
			) {
				return errors.New("function 'modify' must have exactly one parameter named 'in' of type '[]struct {degree int; duration float64}'")
			}

			if !slices.Equal(
				astutil.GetSimpleFields(decl.Type.Results.List),
				[]astutil.SimpleField{{
					Name: "",
					Type: "[]struct {degree int; duration float64}",
				}},
			) {
				return errors.New("function 'modify' must return exactly one unnamed result of type '[]struct {degree int; duration float64}'")
			}
		} else {
			return errors.New("function 'modify' is missing from revocomp.go")
		}
	default:
		return fmt.Errorf("component type is invalid")
	}

	return nil
}

// typeMap maps go types to their XSD equivelants.
var typeMap = map[string]string{
	"int":     "xs:integer",
	"int8":    "xs:byte",
	"int16":   "xs:short",
	"int32":   "xs:int",
	"int64":   "xs:long",
	"uint":    "xs:nonNegativeInteger",
	"uint8":   "xs:unsignedByte",
	"uint16":  "xs:unsignedShort",
	"uint32":  "xs:unsignedInt",
	"uint64":  "xs:unsignedLong",
	"float32": "xs:float",
	"float64": "xs:double",
	"bool":    "xs:boolean",
	"string":  "xs:string",
}

// typeConvMethods maps go types to their respective methods for converting a string to that type.
var typeConvMethods = map[string]string{
	"int":     "strconv.ParseInt(%s, 10, 0)",
	"int8":    "strconv.ParseInt(%s, 10, 8)",
	"int16":   "strconv.ParseInt(%s, 10, 16)",
	"int32":   "strconv.ParseInt(%s, 10, 32)",
	"int64":   "strconv.ParseInt(%s, 10, 64)",
	"uint":    "strconv.ParseUint(%s, 10, 0)",
	"uint8":   "strconv.ParseUint(%s, 10, 8)",
	"uint16":  "strconv.ParseUint(%s, 10, 16)",
	"uint32":  "strconv.ParseUint(%s, 10, 32)",
	"uint64":  "strconv.ParseUint(%s, 10, 64)",
	"float32": "strconv.ParseFloat(%s, 32)",
	"float64": "strconv.ParseFloat(%s, 64)",
	"bool":    "strconv.ParseBool(%s)",
}

// convString is a template for converting a string to another type.
var convString = "{{.TmpVarName}}, err := {{.ConvMethod}}; if err != nil { log.Fatalln(err) }; {{.VarName}} := {{.Type}}({{.TmpVarName}})"

// listConvString is a template for converting a slice of strings to a slice of another type.
var listConvString = "var {{.VarName}} []{{.Type}}; for _, s := range strings.Split({{.Source}}, ',') { {{.LoopBody}} }"

// generateConversionCode generates the necessary code for converting a string to the specified go type.
// Returns an error if the type is unsupported.
func generateConversionCode(varName, gotype, source string) (string, error) {

	base := strings.TrimPrefix(gotype, "[]")

	if base == "string" {
		if strings.HasPrefix(gotype, "[]") {
			return fmt.Sprintf("%s := %s", varName, source), nil
		} else {
			return fmt.Sprintf("%s := strings.Split(%s, ',')", varName, source), nil
		}
	} else {

		convMethod := typeConvMethods[base]
		tmpVarName := randutil.GetRandomString(20)

		convTmpl, err := template.New("convTmpl").Parse(convString)
		if err != nil {
			return "", err
		}

		listConvTmpl, err := template.New("listConvTmpl").Parse(listConvString)
		if err != nil {
			return "", err
		}

		if strings.HasPrefix(gotype, "[]") {

			convMethod = fmt.Sprintf(convMethod, "s")

			convData := struct {
				VarName, TmpVarName, ConvMethod, Type string
			}{
				VarName:    "val",
				TmpVarName: tmpVarName,
				ConvMethod: convMethod,
				Type:       base,
			}

			var builder strings.Builder
			if err := convTmpl.Execute(&builder, convData); err != nil {
				return "", err
			}

			convBody := builder.String()
			loopBody := fmt.Sprintf("%[1]s; %[2]s = append(%[2]s, %[3]s)",
				convBody,
				varName,
				"val",
			)

			listConvData := struct {
				VarName, Type, Source, LoopBody string
			}{
				VarName:  varName,
				Type:     base,
				Source:   source,
				LoopBody: loopBody,
			}

			builder.Reset()
			if err := listConvTmpl.Execute(&builder, listConvData); err != nil {
				return "", err
			}

			return builder.String(), nil

		} else {

			convMethod = fmt.Sprintf(convMethod, source)

			data := struct {
				VarName, TmpVarName, ConvMethod, Type string
			}{
				VarName:    varName,
				TmpVarName: tmpVarName,
				ConvMethod: convMethod,
				Type:       base,
			}

			var builder strings.Builder
			if err := convTmpl.Execute(&builder, data); err != nil {
				return "", err
			}

			return builder.String(), nil
		}
	}
}

// getEmptyFields returns a slice of all empty fields in the specified interface.
func getEmptyFields(s interface{}) []string {
	emptyFields := []string{}

	value := reflect.ValueOf(s)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	for i := 0; i < value.NumField(); i++ {
		fieldValue := value.Field(i)
		fieldType := value.Type().Field(i)

		// Check if the field is a string and empty
		if fieldType.Type.Kind() == reflect.String && fieldValue.String() == "" {
			emptyFields = append(emptyFields, fieldType.Name)
		}

		// Check if the field is a zero value
		if reflect.DeepEqual(fieldValue.Interface(), reflect.Zero(fieldType.Type).Interface()) {
			emptyFields = append(emptyFields, fieldType.Name)
		}
	}

	return emptyFields
}

// getRestrictions gets the content of the @restrict tag in the specified string.
// The result is a map of restriction name to restriction value.
func getRestrictions(comment string) (map[string]string, error) {
	tags := strtags.Extract(comment)

	for _, tag := range tags {
		if tag.Name == "restrict" {

			restrictions := make(map[string]string)

			for _, option := range tag.Options {
				option = strings.TrimSpace(option)
				key, value, ok := strings.Cut(option, "=")
				if !ok {
					return nil, fmt.Errorf("invalid restriction '%s' on a parameter of function 'newGenerator'", option)
				}
				restrictions[key] = value
			}

			return restrictions, nil
		}
	}

	return make(map[string]string), nil
}

// getDocumentation gets the content of the @doc tag in the specified string.
func getDocumentation(comment string) string {
	tags := strtags.Extract(comment)

	for _, tag := range tags {
		if tag.Name == "doc" {
			return strings.Join(tag.Options, ",")
		}
	}

	return ""
}

// generateAttributesFromFields generates XSD attribute tags from AST fields.
func generateAttributesFromFields(fset *token.FileSet, astFile *ast.File, fields []*ast.Field) ([]etree.Element, error) {
	var attributes []etree.Element

	for _, field := range fields {

		var restrictions map[string]string
		var documentation string

		comment, ok := astutil.GetCommentAtField(fset, astFile, field)
		if ok {
			var err error
			restrictions, err = getRestrictions(comment.Text)
			if err != nil {
				return nil, err
			}
			documentation = getDocumentation(comment.Text)
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

// generateAttribute generates an XSD attribute tag with the specified name, documentation, and restrictions,
// and the XSD equivelant of the specified go type.
// Returns an error if there is no XSD equivelant to the go type.
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
