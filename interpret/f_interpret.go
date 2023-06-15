package interpret

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os/exec"
	"path/filepath"
	"revolution/component"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beevik/etree"
	"github.com/radovskyb/watcher"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

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

				wantedComponents := make(map[string]string)

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
					wantedComponents[firstChild.Tag] = "generator"
				}

				modDefs := xmlDoc.FindElements("//Definitions/ModDef")

				for _, modDef := range modDefs {
					childElements := modDef.ChildElements()
					if len(childElements) == 0 {
						continue
					}
					firstChild := childElements[0]
					wantedComponents[firstChild.Tag] = "modifier"
				}

				addedComponents := make(map[string]string)

				xsdDoc := etree.NewDocument()
				if err := xsdDoc.ReadFromFile(xsdFilePath); err != nil {
					log.Fatalln("Failed to read XSD file")
				}

				genDefChoice := xsdDoc.FindElement("//xs:element[@name='GenDef']/xs:complexType/xs:choice")
				if genDefChoice == nil {
					log.Fatalln("XSD file is invalid")
				}

				for _, el := range genDefChoice.ChildElements() {
					refValue := el.SelectAttrValue("ref", "")
					addedComponents[refValue] = "generator"
				}

				modDefChoice := xsdDoc.FindElement("//xs:element[@name='ModDef']/xs:complexType/xs:choice")
				if modDefChoice == nil {
					log.Fatalln("XSD file is invalid")
				}

				for _, el := range modDefChoice.ChildElements() {
					refValue := el.SelectAttrValue("ref", "")
					addedComponents[refValue] = "modifier"
				}

				fmt.Println("Wanted Components:", wantedComponents)
				fmt.Println("Added Components:", addedComponents)

				// Add wanted components that are not yet added
				for tag, kind := range wantedComponents {
					if slices.Contains(maps.Keys(addedComponents), tag) {
						// Component is already added
						continue
					}

					name, version, ok := strings.Cut(tag, "-")
					if !ok {
						fmt.Println("Please specify version for", tag)
						continue
					}

					path, found := component.FindComponent(name, kind, version)
					if !found {
						fmt.Println("Failed to locate component", tag)
						continue
					}

					cmd := exec.Command(path, "xsd")
					output, err := cmd.Output()
					if err != nil {
						fmt.Println("Failed to get XSD for component", tag)
						continue
					}

					doc := etree.NewDocument()
					if err := doc.ReadFromBytes(output); err != nil {
						fmt.Println("Failed to parse XSD for component", tag)
					}

					docRoot := doc.Root()
					if docRoot == nil {
						log.Fatalln("Invalid XSD for component", tag)
					}

					xsdDoc.Root().AddChild(docRoot)

					reference := etree.NewElement("xs:element")
					reference.CreateAttr("ref", tag)

					// Store path to component
					annotation := reference.CreateElement("xs:annotation")
					appinfo := annotation.CreateElement("xs:appinfo")
					appinfo.SetText(path)

					if kind == "generator" {
						genDefChoice.AddChild(reference)
					} else {
						modDefChoice.AddChild(reference)
					}
				}

				// Remove added components that are no longer wanted
				for tag, kind := range addedComponents {
					if slices.Contains(maps.Keys(wantedComponents), tag) {
						// Component is still wanted
						continue
					}

					element := xsdDoc.FindElement(
						fmt.Sprintf("//xs:element[@name='%s']", tag),
					)
					xsdDoc.Root().RemoveChild(element)

					if kind == "generator" {
						referenceElement := genDefChoice.FindElement(
							fmt.Sprintf("//xs:element[@ref='%s']", tag),
						)
						genDefChoice.RemoveChild(referenceElement)
					} else {
						referenceElement := modDefChoice.FindElement(
							fmt.Sprintf("//xs:element[@ref='%s']", tag),
						)
						modDefChoice.RemoveChild(referenceElement)
					}

					fmt.Println("Removed", tag)
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
							ref := item.SelectAttrValue("ref", "none")
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
					if id == "none" {
						continue
					}

					var generationStart float64
					var generationEnd float64

					for i, genItem := range genItems[id] {

						var length float64

						barsWithChanges := maps.Keys(changes)
						slices.Sort(barsWithChanges)

						// Find the length of the item in whole notes with respect to time signatures
						for i, changeStart := range barsWithChanges {

							if float64(changeStart) > genItem.end {
								// The change starts after the item ends
								break
							}

							var end float64

							if len(barsWithChanges) > i+1 {
								changeEnd := float64(barsWithChanges[i+1]) // The start of the next change
								if genItem.start > changeEnd {
									// The item starts after the change ends
									continue
								}
								end = math.Min(genItem.end, float64(changeEnd))
							} else {
								end = genItem.end
							}

							start := math.Max(genItem.start, float64(changeStart))

							wholeNotesPerBar := changes[changeStart].time.GetWholeNotesPerBar()

							length += (end - start) * wholeNotesPerBar
						}

						// TODO: Overvej om offset skal være i bars fremfor whole notes
						// og i såfald i hvilken time signature, det skal interpretes

						genItems[id][i].length = length

						if i == 0 {
							generationStart = genItem.offset
							generationEnd = genItem.offset + length
							continue
						}
						if genItem.offset < generationStart {
							generationStart = genItem.offset
						}
						if genItem.offset+length > generationEnd {
							generationEnd = genItem.offset + length
						}
					}

					newSettings[id] = &generationSettings{
						start: generationStart,
						end:   generationEnd,
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

				channels := make(map[int]channel)

				for id, genItems := range genItems {
					if id == "none" {
						continue
					}
					for _, genItem := range genItems {

						notes := getFromTo(generators[id].generation, genItem.offset, genItem.offset+genItem.length)
						copiedNotes := make([]Note, len(notes))
						copy(copiedNotes, notes)

						for i := range copiedNotes {
							copiedNotes[i].Start -= genItem.offset
							copiedNotes[i].Start += genItem.start
						}

						ch := genItem.channel
						tr := genItem.track

						if channels[ch] == nil {
							channels[ch] = make(channel)
						}

						channels[ch][tr] = append(channels[ch][tr], copiedNotes...)
					}
				}

				for _, ch := range channels {
					for _, tr := range ch {
						sort.Slice(tr, func(i, j int) bool {
							return tr[i].Start < tr[j].Start
						})
					}
				}

				jsonData, err := json.MarshalIndent(channels, "", "  ")
				if err != nil {
					log.Fatalln(err)
				}
				fmt.Println(string(jsonData))

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

type channel map[int]track

type track []Note
