package component

import (
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"revolution/astutil"
	"revolution/randutil"
	"strings"
	"text/template"
	"time"

	_ "embed"

	"github.com/beevik/etree"
	"github.com/iancoleman/strcase"
	"github.com/otiai10/copy"
	"gopkg.in/yaml.v3"
)

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

	//defer os.RemoveAll(tempDir)

	fmt.Println(tempDirName)

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
		funcName = "NewGenerator"
	case "modifier":
		funcName = "NewModifier"
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
		convCode, err := generateStringConversion(
			fmt.Sprintf("os.Args[%d]", i+1),
			param.Name,
			param.Type,
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
		tmplData, err := files.ReadFile("boilerplate/generator/main.tmpl")
		if err != nil {
			return err
		}

		if _, err := mainTmpl.Parse(string(tmplData)); err != nil {
			return err
		}
	case "modifier":
		tmplData, err := files.ReadFile("boilerplate/modifier/main.tmpl")
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
