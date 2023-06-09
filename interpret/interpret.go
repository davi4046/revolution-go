package interpret

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math"
	"os/exec"
	"path/filepath"
	"revolution/component"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beevik/etree"
	"github.com/radovskyb/watcher"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

type note struct {
	degree int
	start  float64
}

type generationSettings struct {
	path  string
	args  []string
	start float64
	end   float64
}

type generationManager struct {
	settings   generationSettings
	command    *exec.Cmd
	stdin      *io.WriteCloser
	stdout     *io.ReadCloser
	generation []note
}

type genItem struct {
	channel int
	track   int
	start   float64
	end     float64
	add     int
	sub     int
}

func (g *generationManager) update(settings generationSettings, wg *sync.WaitGroup) {
	hasPathChanged := settings.path != g.settings.path
	hasArgsChanged := !slices.Equal(settings.args, g.settings.args)
	hasStartChanged := settings.start != g.settings.start
	hasEndChanged := settings.end != g.settings.end

	g.settings = settings

	if hasPathChanged || hasArgsChanged {
		wg.Add(1)
		g.initialize(wg)
		g.regenerate(wg)
		return
	}
	if hasStartChanged || hasEndChanged {
		g.regenerate(wg)
		return
	}

	wg.Done()
}

func (g *generationManager) initialize(wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Println("init with command:", g.settings.path, g.settings.args)

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
}

func (g *generationManager) regenerate(wg *sync.WaitGroup) {
	g.generation = g.generateFromTo(g.settings.start, g.settings.end, wg)
}

func (g *generationManager) generateFromTo(from float64, to float64, wg *sync.WaitGroup) []note {
	wg.Add(1)

	negativeGen := g.generate(-1, math.Min(from, 0), wg)
	positiveGen := g.generate(0, math.Max(to, 0), wg)

	generation := append(negativeGen, positiveGen...)

	i, isNoteOnFrom := binarySearchNote(generation, from)
	j, _ := binarySearchNote(generation, to)

	if !isNoteOnFrom {
		i -= 1
	}

	generation = generation[i:j]

	generation[0].start = from

	return generation
}

func (g *generationManager) generate(startIndex int, length float64, wg *sync.WaitGroup) []note {
	defer wg.Done()

	var generation []note

	if length == 0 {
		return generation
	}

	var currLength float64

	currIndex := startIndex

	writeIndex := func() {
		_, err := io.WriteString(*g.stdin, fmt.Sprintf("%d\n", currIndex))
		if err != nil {
			log.Fatalln(err)
		}
		if length > 0 {
			currIndex++
		} else {
			currIndex--
		}
	}

	scanner := bufio.NewScanner(*g.stdout)

	writeIndex()

	for scanner.Scan() {
		line := scanner.Text()

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

		start := currLength
		if length < 0 {
			start -= duration
		}

		generation = append(generation, note{
			degree: degree,
			start:  start,
		})

		if length > 0 {
			currLength += duration
			if currLength >= length {
				break
			}
		} else {
			currLength -= duration
			if currLength <= length {
				generation = reverse(generation)
				break
			}
		}

		writeIndex()
	}

	return generation
}

func binarySearchNote(slice []note, start float64) (int, bool) {
	return slices.BinarySearchFunc(slice, note{start: start}, func(element note, target note) int {
		if target.start > element.start {
			return -1
		}
		if target.start < element.start {
			return 1
		}
		return 0
	})
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

				genChannels := xmlDoc.FindElements("//Channels/GenChannel")

				genItems := make(map[string][]genItem)

				for i, channel := range genChannels {
					tracks := channel.FindElements("//Track")
					for j, track := range tracks {
						items := track.FindElements("//Item")
						for _, item := range items {
							ref := item.SelectAttrValue("ref", "")
							lengthStr := item.SelectAttrValue("length", "0")
							offsetStr := item.SelectAttrValue("offset", "0")
							addStr := item.SelectAttrValue("add", "0")
							subStr := item.SelectAttrValue("sub", "0")

							length, err := strconv.ParseFloat(lengthStr, 64)
							if err != nil {
								log.Fatalln(err)
							}

							offset, err := strconv.ParseFloat(offsetStr, 64)
							if err != nil {
								log.Fatalln(err)
							}

							add, err := strconv.Atoi(addStr)
							if err != nil {
								log.Fatalln(err)
							}

							sub, err := strconv.Atoi(subStr)
							if err != nil {
								log.Fatalln(err)
							}

							genItems[ref] = append(genItems[ref],
								genItem{
									channel: i,
									track:   j,
									start:   offset,
									end:     offset + length,
									add:     add,
									sub:     sub,
								},
							)
						}
					}
				}

				fmt.Printf("genItems:\n%v\n", genItems)

				newSettings := make(map[string]*generationSettings)

				for _, id := range maps.Keys(genItems) {
					var start float64
					var end float64

					for i, genItem := range genItems[id] {
						if i == 0 {
							start = genItem.start
							end = genItem.end
							continue
						}
						if genItem.start < start {
							start = genItem.start
						}
						if genItem.end > end {
							end = genItem.end
						}
					}

					// TODO: konvertér start og end fra bars til whole notes ifølge time signatures

					newSettings[id] = &generationSettings{
						start: start,
						end:   end,
					}
				}

				for _, genDef := range genDefs {
					id := genDef.SelectAttrValue("id", "")

					if _, ok := newSettings[id]; !ok {
						// The GenDef is unused
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

					newSettings[id].path = path
					newSettings[id].args = args
				}

				var wg sync.WaitGroup

				wg.Add(len(newSettings))

				for id, settings := range newSettings {
					if _, ok := generators[id]; !ok {
						generators[id] = &generationManager{}
					}
					fmt.Println("updating:", id)
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

func reverse[T any](slice []T) []T {
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
	return slice
}
