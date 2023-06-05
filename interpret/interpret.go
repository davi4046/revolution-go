package interpret

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"revolution/component"
	"strings"
	"time"

	"github.com/beevik/etree"
	"github.com/radovskyb/watcher"
	"golang.org/x/exp/slices"
)

// Begin interpreting the specified project directory.
func Interpret(dir string) error {
	w := watcher.New()

	w.FilterOps(watcher.Write)

	xsdFilePath := filepath.Join(dir, ".xsd")

	go func() {
		for {
			select {
			case event := <-w.Event:
				fmt.Println(event) // Print the event's info.

				var wantedComponents []string

				xmlDoc := etree.NewDocument()
				if err := xmlDoc.ReadFromFile(event.Path); err != nil {
					fmt.Println("Failed to read XML file")
					break
				}

				definitions := xmlDoc.Root().FindElement("Definitions")
				genDefs := definitions.FindElements("GenDef")

				for _, genDef := range genDefs {
					childElements := genDef.ChildElements()
					if len(childElements) == 0 {
						continue
					}
					firstChild := childElements[0]
					wantedComponents = append(wantedComponents, firstChild.Tag)
				}

				var addedComponents []string

				xsdDoc := etree.NewDocument()
				if err := xsdDoc.ReadFromFile(xsdFilePath); err != nil {
					log.Fatalln("Failed to read XSD file")
				}

				genDefChoice := xsdDoc.FindElement("//xs:element[@name='GenDef']/xs:complexType/xs:choice")
				if genDefChoice == nil {
					log.Fatalln("XSD file is invalid")
				}

				genDefChoices := genDefChoice.ChildElements()

				for _, el := range genDefChoices {
					refValue := el.SelectAttrValue("ref", "")
					addedComponents = append(addedComponents, refValue)
				}

				// Add wanted components that are not yet added
				for _, wantedComponent := range wantedComponents {
					if slices.Contains(addedComponents, wantedComponent) {
						// Component is already added
						continue
					}

					name, version, ok := strings.Cut(wantedComponent, "-")
					if !ok {
						fmt.Println("Please specify version for", wantedComponent)
						continue
					}

					path, found := component.FindComponent(name, version)
					if !found {
						fmt.Println("Failed to locate component", wantedComponent)
						continue
					}

					cmd := exec.Command(path, "xsd")
					output, err := cmd.Output()
					if err != nil {
						fmt.Println("Failed to get XSD for component", wantedComponent)
						continue
					}

					doc := etree.NewDocument()
					if err := doc.ReadFromBytes(output); err != nil {
						fmt.Println("Failed to parse XSD for component", wantedComponent)
					}

					xsdDoc.Root().AddChild(doc.Root())

					reference := etree.NewElement("xs:element")
					reference.CreateAttr("ref", wantedComponent)

					// Store path to component
					annotation := reference.CreateElement("xs:annotation")
					appinfo := annotation.CreateElement("xs:appinfo")
					appinfo.SetText(path)

					genDefChoice.AddChild(reference)
				}

				// Remove added components that are no longer wanted
				for _, addedComponent := range addedComponents {
					if slices.Contains(wantedComponents, addedComponent) {
						// Component is still wanted
						continue
					}

					fmt.Println("trying to remove", addedComponent)

					element := xsdDoc.FindElement(
						fmt.Sprintf("//xs:element[@name='%s']", addedComponent),
					)
					fmt.Println(
						xsdDoc.Root().RemoveChild(element),
					)

					referenceElement := genDefChoice.FindElement(
						fmt.Sprintf("//xs:element[@ref='%s']", addedComponent),
					)
					genDefChoice.RemoveChild(referenceElement)

					fmt.Println("Removed", addedComponent)
				}

				xsdDoc.IndentTabs()

				if err := xsdDoc.WriteToFile(xsdFilePath); err != nil {
					log.Fatalln("Failed to update project XSD")
				}

			case err := <-w.Error:
				log.Fatalln(err)
			case <-w.Closed:
				return
			}
		}
	}()

	filePath := filepath.Join(dir, "revoproj.xml")

	if err := w.Add(filePath); err != nil {
		return err
	}

	for path, f := range w.WatchedFiles() {
		fmt.Printf("%s: %s\n", path, f.Name())
	}

	if err := w.Start(100 * time.Millisecond); err != nil {
		return err
	}

	return nil
}
