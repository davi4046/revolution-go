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

type genItem struct {
	channel int
	track   int
	start   float64
	end     float64
	offset  float64
	add     int
	sub     int
}

type key struct {
	root string
	mode string
}

type timeSignature struct {
	numerator   uint8
	denominator uint8
}

func (t timeSignature) getWholeNotesPerBar() float64 {
	return 1 / float64(t.denominator) * float64(t.numerator)
}

func extractKey(el *etree.Element) key {
	root := el.SelectAttrValue("root", "")
	mode := el.SelectAttrValue("mode", "")
	return key{
		root: root,
		mode: mode,
	}
}

func extractTime(el *etree.Element) (timeSignature, error) {
	numeratorStr, denominatorStr, ok := strings.Cut(el.Text(), "/")
	if !ok {
		return timeSignature{}, fmt.Errorf("'/' is missing")
	}

	numerator, err := strconv.ParseUint(numeratorStr, 10, 8)
	if err != nil {
		return timeSignature{}, err
	}

	denominator, err := strconv.ParseUint(denominatorStr, 10, 8)
	if err != nil {
		return timeSignature{}, err
	}

	return timeSignature{
		numerator:   uint8(numerator),
		denominator: uint8(denominator),
	}, nil
}

func extractTempo(el *etree.Element) (uint8, error) {
	tempo, err := strconv.ParseUint(el.Text(), 10, 8)
	if err != nil {
		return 0, err
	}
	return uint8(tempo), nil
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

type change struct {
	key   key
	time  timeSignature
	tempo uint8
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
					tracks := channel.FindElements("Track")
					for j, track := range tracks {
						items := track.FindElements("Item")

						var currBar float64

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

							start := currBar
							currBar += length
							end := currBar

							genItems[ref] = append(genItems[ref],
								genItem{
									channel: i,
									track:   j,
									start:   start,
									end:     end,
									offset:  offset,
									add:     add,
									sub:     sub,
								},
							)
						}
					}
				}

				keyEl := xmlDoc.FindElement("//Key")
				timeEl := xmlDoc.FindElement("//Time")
				tempoEl := xmlDoc.FindElement("//Tempo")

				key := extractKey(keyEl)
				time, err := extractTime(timeEl)
				if err != nil {
					log.Fatalln("Invalid Time:", timeEl.Text())
				}
				tempo, err := extractTempo(tempoEl)
				if err != nil {
					log.Fatalln("Invalid Tempo:", tempoEl.Text())
				}

				changes := map[uint64]change{
					0: {
						key:   key,
						time:  time,
						tempo: tempo,
					},
				}

				for _, changeEl := range xmlDoc.FindElements("//Changes/Change") {

					barStr := changeEl.SelectAttrValue("bar", "")
					bar, err := strconv.ParseUint(barStr, 10, 64)
					if err != nil {
						log.Fatalln("Invalid Bar:", barStr)
					}

					keyEl := changeEl.FindElement("Key")
					timeEl := changeEl.FindElement("Time")
					tempoEl := changeEl.FindElement("Tempo")

					var change change

					if keyEl == nil {
						// Key remains the same
						change.key = maps.Values(changes)[len(changes)-1].key
					} else {
						change.key = extractKey(keyEl)
					}
					if timeEl == nil {
						// Time remains the same
						change.time = maps.Values(changes)[len(changes)-1].time
					} else {
						time, err := extractTime(timeEl)
						if err != nil {
							log.Fatalln("Invalid Time:", timeEl.Text())
						}
						change.time = time
					}

					if tempoEl == nil {
						// Tempo remains the same
						change.tempo = maps.Values(changes)[len(changes)-1].tempo
					} else {
						tempo, err := extractTempo(tempoEl)
						if err != nil {
							log.Fatalln("Invalid Tempo:", tempoEl.Text())
						}
						change.tempo = tempo
					}

					changes[bar] = change
				}

				fmt.Printf("changes:\n%v\n", changes)

				newSettings := make(map[string]*generationSettings)

				for _, id := range maps.Keys(genItems) {
					var start float64
					var end float64

					for i, genItem := range genItems[id] {

						var length float64

						barsWithChanges := maps.Keys(changes)
						slices.Sort(barsWithChanges)

						// Find the length of the item in whole notes with respect to time signatures
						for i, changeStart := range barsWithChanges {

							var changeEnd float64

							if len(barsWithChanges) > i+1 {
								changeEnd = float64(barsWithChanges[i+1]) // The start of the next change
							} else {
								changeEnd = 1000000000
							}

							if changeEnd < genItem.start {
								continue
							}
							if float64(changeStart) > genItem.end {
								break
							}
							wholeNotesPerBar := changes[changeStart].time.getWholeNotesPerBar()

							start := math.Max(genItem.start, float64(changeStart))
							end := math.Min(genItem.end, float64(changeEnd))

							length += (end - start) * wholeNotesPerBar
						}

						// TODO: Overvej om offset skal være i bars fremfor whole notes
						// og i såfald i hvilken time signature, det skal interpretes

						if i == 0 {
							start = genItem.offset
							end = genItem.offset + length
							continue
						}
						if genItem.offset < start {
							start = genItem.offset
						}
						if genItem.offset+length > end {
							end = genItem.offset + length
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
