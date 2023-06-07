package interpret

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path/filepath"
	"revolution/component"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beevik/etree"
	"github.com/radovskyb/watcher"
	"golang.org/x/exp/slices"
)

type note struct {
	degree   int
	duration float64
}

type generationSettings struct {
	path   string
	args   []string
	length float64
}

type generationManager struct {
	settings   generationSettings
	command    *exec.Cmd
	stdin      *io.WriteCloser
	stdout     *io.ReadCloser
	generation []note
}

func (g *generationManager) update(settings generationSettings, wg *sync.WaitGroup) {
	hasArgsChanged := !slices.Equal(settings.args, g.settings.args)
	hasPathChanged := settings.path != g.settings.path
	hasLengthChanged := settings.length != g.settings.length

	if hasArgsChanged || hasPathChanged {
		g.settings = settings
		g.regenerate(wg)
	} else if hasLengthChanged {
		diff := settings.length - g.settings.length
		if diff > 0 {
			g.extentGeneration(diff, wg)
			g.settings.length = settings.length
		}
	} else {
		wg.Done()
	}
}

func (g *generationManager) regenerate(wg *sync.WaitGroup) {
	g.command = exec.Command(g.settings.path, g.settings.args...)

	stdin, err := g.command.StdinPipe()
	if err != nil {
		log.Fatalln(err)
	}

	stdout, err := g.command.StdoutPipe()
	if err != nil {
		log.Fatalln(err)
	}

	g.stdin = &stdin
	g.stdout = &stdout

	if err := g.command.Start(); err != nil {
		log.Fatalln(err)
	}

	generation := g.generate(0, g.settings.length, wg)

	g.generation = generation
}

func (g *generationManager) extentGeneration(length float64, wg *sync.WaitGroup) {
	startIndex := len(g.generation)

	generation := g.generate(startIndex, length, wg)

	g.generation = append(g.generation, generation...)
}

func (g generationManager) generate(startIndex int, length float64, wg *sync.WaitGroup) []note {
	defer wg.Done()

	var generation []note
	var currIndex int
	var currLength float64

	writeIndex := func() {
		_, err := io.WriteString(*g.stdin, fmt.Sprintf("%d\n", currIndex))
		if err != nil {
			log.Fatalln(err)
		}
		currIndex++
	}

	scanner := bufio.NewScanner(*g.stdout)

	writeIndex()

	for scanner.Scan() {
		line := scanner.Text()

		// Process line
		degreeStr, durationStr, ok := strings.Cut(line, " ")
		if !ok {
			log.Fatalln("Invalid generator output:", line)
		}
		degree, err := strconv.Atoi(degreeStr)
		if err != nil {
			log.Fatalln(err)
		}
		duration, err := strconv.ParseFloat(durationStr, 64)
		if err != nil {
			log.Fatalln(err)
		}
		generation = append(generation, note{
			degree:   degree,
			duration: duration,
		})

		currLength += duration

		if currLength > length {
			break
		}

		writeIndex()
	}

	return generation
}

// Begin interpreting the specified project directory.
func Interpret(dir string) error {
	w := watcher.New()

	w.FilterOps(watcher.Write)

	xsdFilePath := filepath.Join(dir, ".xsd")

	generators := make(map[string]*generationManager)

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

				genDefs := xmlDoc.FindElements("//Definitions/GenDef")

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

				/* Generation */

				var usedGenDefIDs []string

				items := xmlDoc.FindElements("//Channels/Channel/Track/Item")

				for _, item := range items {
					id := item.SelectAttrValue("ref", "")
					if id == "" {
						continue
					}
					if slices.Contains(usedGenDefIDs, id) {
						continue
					}
					usedGenDefIDs = append(usedGenDefIDs, id)
				}

				newSettings := make(map[string]*generationSettings)

				for _, genDef := range genDefs {
					id := genDef.SelectAttrValue("id", "")

					if !slices.Contains(usedGenDefIDs, id) {
						delete(generators, id)
						continue
					}

					childElements := genDef.ChildElements()
					if len(childElements) == 0 {
						continue
					}
					firstChild := childElements[0]

					appinfo := genDefChoice.FindElement(
						fmt.Sprintf("//xs:element[@ref='%s']/xs:annotation/xs:appinfo", firstChild.Tag),
					)

					path := appinfo.Text()

					var args []string

					for _, attr := range firstChild.Attr {
						args = append(args, attr.Value)
					}

					newSettings[id] = &generationSettings{
						path: path,
						args: args,
					}
				}

				for _, item := range items {
					id := item.SelectAttrValue("ref", "")
					if !slices.Contains(usedGenDefIDs, id) {
						continue
					}

					if _, ok := newSettings[id]; !ok {
						continue
					}

					length, _ := strconv.ParseFloat(item.SelectAttrValue("length", ""), 64)
					offset, _ := strconv.ParseFloat(item.SelectAttrValue("offset", ""), 64)
					end := length + offset

					if end > newSettings[id].length {
						newSettings[id].length = end
					}
				}

				var wg sync.WaitGroup

				wg.Add(len(newSettings))

				for id, settings := range newSettings {
					if _, ok := generators[id]; !ok {
						generators[id] = &generationManager{}
					}
					go generators[id].update(*settings, &wg)
				}

				wg.Wait()

				for id, g := range generators {
					fmt.Printf("%s:\n%v\n", id, g.generation)
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
