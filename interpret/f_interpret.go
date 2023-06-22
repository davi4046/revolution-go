package interpret

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"revolution/component"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beevik/etree"
	"github.com/davi4046/revoutil"
	"github.com/radovskyb/watcher"
	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/smf"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

// Begin interpreting the specified project directory.
func Interpret(dir string) error {
	w := watcher.New()

	w.FilterOps(watcher.Write)

	xsdFilePath := filepath.Join(dir, ".xsd")

	generations := make(map[string]*generationManager)

	var modifications []modification

	var player *exec.Cmd

	go func() {
		for {
			select {
			case event := <-w.Event:
				fmt.Println(event) // Print the event's info.

				type myKey struct{ channel, track int }

				start := time.Now()

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

				fmt.Println("hello 1")

				/* Generation */

				genChannels := xmlDoc.FindElements("//Channels/GenChannel")

				genItems := make(map[string][]genItem)

				for i, channel := range genChannels {
					tracks := channel.FindElements("Track")
					for j, track := range tracks {
						xmlItems := track.FindElements("Item")

						var currBar float64

						for _, xmlItem := range xmlItems {
							ref := xmlItem.SelectAttrValue("ref", "none")
							lengthStr := xmlItem.SelectAttrValue("length", "0")
							offsetStr := xmlItem.SelectAttrValue("offset", "0")
							addStr := xmlItem.SelectAttrValue("add", "0")
							subStr := xmlItem.SelectAttrValue("sub", "0")

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
									channel:  i,
									track:    j,
									barStart: start,
									barEnd:   end,
									offset:   offset,
									add:      add,
									sub:      sub,
								},
							)
						}
					}
				}

				var changes []change

				keyEl := xmlDoc.FindElement("//Key")
				if keyEl == nil {
					fmt.Println("Please specify key")
					continue
				}
				meterEl := xmlDoc.FindElement("//Meter")
				if meterEl == nil {
					fmt.Println("Please specify meter")
					continue
				}
				tempoEl := xmlDoc.FindElement("//Tempo")
				if tempoEl == nil {
					fmt.Println("Please specify tempo")
					continue
				}

				key := extractKey(keyEl)
				meter, err := extractMeter(meterEl)
				if err != nil {
					log.Fatalln("Invalid Meter:", meterEl.Text())
				}
				tempo, err := extractTempo(tempoEl)
				if err != nil {
					log.Fatalln("Invalid Tempo:", tempoEl.Text())
				}

				changes = []change{
					{
						barStart: 0,
						key:      key,
						meter:    meter,
						tempo:    tempo,
					},
				}

				for _, changeEl := range xmlDoc.FindElements("//Changes/Change") {

					var change change

					barStr := changeEl.SelectAttrValue("bar", "")
					bar, err := strconv.ParseFloat(barStr, 64)
					if err != nil {
						log.Fatalln("Invalid Bar:", barStr)
					}
					change.barStart = bar

					keyEl := changeEl.FindElement("Key")
					meterEl := changeEl.FindElement("Time")
					tempoEl := changeEl.FindElement("Tempo")

					if keyEl == nil {
						// Key remains the same
						change.key = changes[len(changes)-1].key
					} else {
						change.key = extractKey(keyEl)
					}
					if meterEl == nil {
						// Meter remains the same
						change.meter = changes[len(changes)-1].meter
					} else {
						meter, err := extractMeter(meterEl)
						if err != nil {
							log.Fatalln("Invalid Time:", meterEl.Text())
						}
						change.meter = meter
					}

					if tempoEl == nil {
						// Tempo remains the same
						change.tempo = changes[len(changes)-1].tempo
					} else {
						tempo, err := extractTempo(tempoEl)
						if err != nil {
							log.Fatalln("Invalid Tempo:", tempoEl.Text())
						}
						change.tempo = tempo
					}

					changes = append(changes, change)
				}
				for i := range changes {
					changes[i].noteStart = barToWholeNote(changes[i].barStart, changes)
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

						wholeNoteStart := barToWholeNote(genItem.barStart, changes)
						wholeNoteEnd := barToWholeNote(genItem.barEnd, changes)

						length := wholeNoteEnd - wholeNoteStart

						genItems[id][i].noteStart = wholeNoteStart
						genItems[id][i].noteEnd = wholeNoteEnd

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

					if newSettings[id] == nil {
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

					attributes := firstChild.Attr

					sort.Slice(attributes, func(i, j int) bool {
						return attributes[i].Key < attributes[j].Key
					})

					var args []string

					for _, attr := range attributes {
						args = append(args, attr.Value)
					}

					sort.Slice(args, func(i, j int) bool {
						return firstChild.Attr[i].Key < firstChild.Attr[j].Key
					})

					newSettings[id].path = path
					newSettings[id].args = args
				}

				var wg sync.WaitGroup

				wg.Add(len(newSettings))

				for id, settings := range newSettings {
					if _, ok := generations[id]; !ok {
						generations[id] = &generationManager{}
					}
					fmt.Println("updating:", id)
					go generations[id].update(*settings, &wg)
				}

				wg.Wait()

				for id, g := range generations {
					fmt.Printf("%s:\n%v\n", id, g.generation)
				}

				var allNotes []Note

				for genId, genItems := range genItems {
					if genId == "none" {
						continue
					}
					for _, genItem := range genItems {

						notes := getFromTo(generations[genId].generation, genItem.offset, genItem.offset+genItem.noteEnd-genItem.noteStart)

						copiedNotes := make([]Note, len(notes))
						copy(copiedNotes, notes)

						for i := range copiedNotes {
							copiedNotes[i].Start -= genItem.offset
							copiedNotes[i].Start += genItem.noteStart

							copiedNotes[i].Channel = genItem.channel
							copiedNotes[i].Track = genItem.track

							copiedNotes[i].Value += genItem.add - genItem.sub
						}

						allNotes = append(allNotes, copiedNotes...)
					}
				}

				sort.SliceStable(allNotes, func(i int, j int) bool {
					return allNotes[i].Start < allNotes[j].Start
				})

				func() {
					var changeIndex int

					for i := range allNotes {
						for changeIndex+1 < len(changes) {
							if allNotes[i].Start >= changes[changeIndex+1].noteStart {
								changeIndex++
							} else {
								break
							}
						}
						allNotes[i].Value = changes[changeIndex].key.DegreeToMIDI(allNotes[i].Value)
					}
				}()

				/* Modification */

				modChannels := xmlDoc.FindElements("//Channels/ModChannel")

				modItems := make(map[string][]modItem)

				for _, channel := range modChannels {
					tracks := channel.FindElements("Track")
					for _, track := range tracks {
						xmlItems := track.FindElements("Item")

						var currBar float64

						for _, xmlItem := range xmlItems {
							ref := xmlItem.SelectAttrValue("ref", "none")
							lengthStr := xmlItem.SelectAttrValue("length", "0")
							targetStr := xmlItem.SelectAttrValue("target", "")

							length, err := strconv.ParseFloat(lengthStr, 64)
							if err != nil {
								log.Fatalln(err)
							}

							start := currBar
							currBar += length
							end := currBar

							modItems[ref] = append(modItems[ref],
								modItem{
									barStart: start,
									barEnd:   end,
									target:   targetStr,
								},
							)
						}
					}
				}

				func() {
					var usedModifications []int

					for _, modDef := range modDefs {

						id := modDef.SelectAttrValue("id", "")

						if modItems[id] == nil {
							continue
						}

						childElements := modDef.ChildElements()

						if len(childElements) == 0 {
							continue
						}

						firstChild := childElements[0]

						appinfo := modDefChoice.FindElement(
							fmt.Sprintf("//xs:element[@ref='%s']/xs:annotation/xs:appinfo", firstChild.Tag),
						)

						path := appinfo.Text()

						attributes := firstChild.Attr

						sort.Slice(attributes, func(i, j int) bool {
							return attributes[i].Key < attributes[j].Key
						})

						var args []string

						for _, attr := range attributes {
							args = append(args, attr.Value)
						}

						for _, modItem := range modItems[id] {
							target, err := stringToTarget(modItem.target)
							if err != nil {
								log.Fatalln(err)
							}

							wholeNoteStart := barToWholeNote(modItem.barStart, changes)
							wholeNoteEnd := barToWholeNote(modItem.barEnd, changes)

							i, isNoteOnFrom := binarySearchNote(allNotes, wholeNoteStart)
							j, _ := binarySearchNote(allNotes, wholeNoteEnd)

							if !isNoteOnFrom {
								i -= 1
							}

							notesInRange := make([]Note, len(allNotes[i:j]))
							copy(notesInRange, allNotes[i:j])

							var targetNotes []Note

							for k := range notesInRange {
								if !slices.Contains(target.channels, notesInRange[k].Channel) {
									continue
								}
								if !slices.Contains(target.tracks, notesInRange[k].Track) {
									continue
								}

								allNotesIndex := i + k - len(targetNotes)
								allNotes = slices.Delete(allNotes, allNotesIndex, allNotesIndex+1)

								targetNotes = append(targetNotes, notesInRange[k])
							}

							// Convert notes to revoutil notes

							currTime := make(map[myKey]float64)

							var input []revoutil.Note

							for _, note := range targetNotes {
								input = append(input, revoutil.Note{
									Pitch:    note.Value,
									Duration: note.Duration,
									Channel:  note.Channel,
									Track:    note.Track,
								})

								if _, ok := currTime[myKey{note.Channel, note.Track}]; !ok {
									currTime[myKey{note.Channel, note.Track}] = note.Start
								}
							}

							sort.SliceStable(input, func(i, j int) bool {
								if input[i].Channel < input[j].Channel {
									return true
								}
								if input[i].Track < input[j].Track {
									return true
								}
								return false
							})

							result := func() []revoutil.Note {

								// Try to find a modification with the same path, args, and input.
								for i, modification := range modifications {
									if modification.path == path &&
										slices.Equal(modification.args, args) &&
										slices.Equal(modification.input, input) {
										usedModifications = append(usedModifications, i)
										return modification.output
									}
								}
								var wg sync.WaitGroup

								wg.Add(1)

								modification := newModification(path, args, input, &wg)

								wg.Wait()

								modifications = append(modifications, modification)
								usedModifications = append(usedModifications, len(modifications)-1)

								return modification.output
							}()

							for _, note := range result {
								allNotes = append(allNotes, Note{
									Value:    note.Pitch,
									Start:    currTime[myKey{note.Channel, note.Track}],
									Duration: note.Duration,
									Channel:  note.Channel,
									Track:    note.Track,
								})
								currTime[myKey{note.Channel, note.Track}] += note.Duration
							}

							sort.SliceStable(allNotes, func(i int, j int) bool {
								return allNotes[i].Start < allNotes[j].Start
							})
						}
					}

					for i := 0; i < len(modifications); i++ {
						if !slices.Contains(usedModifications, i) {
							modifications = slices.Delete(modifications, i, i+1)
						}
					}

					fmt.Println("saved modifications count:", len(modifications))
				}()

				func() {

					channels := make(map[int]channel)

					for _, note := range allNotes {

						ch := note.Channel
						tr := note.Track

						if channels[ch] == nil {
							channels[ch] = make(channel)
						}

						channels[ch][tr] = append(channels[ch][tr], note)
					}

					s := smf.New()
					clock := smf.MetricTicks(96)
					s.TimeFormat = clock

					func() {
						changesTrack := smf.Track{}

						addChange := func(deltaTicks uint32, change change) {
							changesTrack.Add(deltaTicks, smf.MetaMeter(
								change.meter.Numerator,
								change.meter.Denominator,
							))
							changesTrack.Add(deltaTicks, smf.MetaTempo(change.tempo))
						}

						for i, change := range changes {
							if i == 0 {
								addChange(0, change)
								continue
							}
							deltaNotes := change.noteStart - changes[i-1].noteStart
							deltaTicks := uint32(float64(clock.Ticks4th())*deltaNotes) * 4
							addChange(deltaTicks, change)
						}

						changesTrack.Close(0)
						if err := s.Add(changesTrack); err != nil {
							log.Fatalln(err)
						}
					}()

					var instrumentMap = map[string]uint8{
						"Acoustic Grand Piano":    0,
						"Bright Acoustic Piano":   1,
						"Electric Grand Piano":    2,
						"Honky-tonk Piano":        3,
						"Electric Piano 1":        4,
						"Electric Piano 2":        5,
						"Harpsichord":             6,
						"Clavinet":                7,
						"Celesta":                 8,
						"Glockenspiel":            9,
						"Music Box":               10,
						"Vibraphone":              11,
						"Marimba":                 12,
						"Xylophone":               13,
						"Tubular Bells":           14,
						"Dulcimer":                15,
						"Drawbar Organ":           16,
						"Percussive Organ":        17,
						"Rock Organ":              18,
						"Church Organ":            19,
						"Reed Organ":              20,
						"Accordion":               21,
						"Harmonica":               22,
						"Tango Accordion":         23,
						"Acoustic Guitar (nylon)": 24,
						"Acoustic Guitar (steel)": 25,
						"Electric Guitar (jazz)":  26,
						"Electric Guitar (clean)": 27,
						"Electric Guitar (muted)": 28,
						"Overdriven Guitar":       29,
						"Distortion Guitar":       30,
						"Guitar Harmonics":        31,
						"Acoustic Bass":           32,
						"Electric Bass (finger)":  33,
						"Electric Bass (pick)":    34,
						"Fretless Bass":           35,
						"Slap Bass 1":             36,
						"Slap Bass 2":             37,
						"Synth Bass 1":            38,
						"Synth Bass 2":            39,
						"Violin":                  40,
						"Viola":                   41,
						"Cello":                   42,
						"Contrabass":              43,
						"Tremolo Strings":         44,
						"Pizzicato Strings":       45,
						"Orchestral Harp":         46,
						"Timpani":                 47,
						"String Ensemble 1":       48,
						"String Ensemble 2":       49,
						"Synth Strings 1":         50,
						"Synth Strings 2":         51,
						"Choir Aahs":              52,
						"Voice Oohs":              53,
						"Synth Choir":             54,
						"Orchestra Hit":           55,
						"Trumpet":                 56,
						"Trombone":                57,
						"Tuba":                    58,
						"Muted Trumpet":           59,
						"French Horn":             60,
						"Brass Section":           61,
						"Synth Brass 1":           62,
						"Synth Brass 2":           63,
						"Soprano Sax":             64,
						"Alto Sax":                65,
						"Tenor Sax":               66,
						"Baritone Sax":            67,
						"Oboe":                    68,
						"English Horn":            69,
						"Bassoon":                 70,
						"Clarinet":                71,
						"Piccolo":                 72,
						"Flute":                   73,
						"Recorder":                74,
						"Pan Flute":               75,
						"Blown Bottle":            76,
						"Shakuhachi":              77,
						"Whistle":                 78,
						"Ocarina":                 79,
						"Lead 1 (square)":         80,
						"Lead 2 (sawtooth)":       81,
						"Lead 3 (calliope)":       82,
						"Lead 4 (chiff)":          83,
						"Lead 5 (charang)":        84,
						"Lead 6 (voice)":          85,
						"Lead 7 (fifths)":         86,
						"Lead 8 (bass + lead)":    87,
						"Pad 1 (new age)":         88,
						"Pad 2 (warm)":            89,
						"Pad 3 (polysynth)":       90,
						"Pad 4 (choir)":           91,
						"Pad 5 (bowed)":           92,
						"Pad 6 (metallic)":        93,
						"Pad 7 (halo)":            94,
						"Pad 8 (sweep)":           95,
						"FX 1 (rain)":             96,
						"FX 2 (soundtrack)":       97,
						"FX 3 (crystal)":          98,
						"FX 4 (atmosphere)":       99,
						"FX 5 (brightness)":       100,
						"FX 6 (goblins)":          101,
						"FX 7 (echoes)":           102,
						"FX 8 (sci-fi)":           103,
						"Sitar":                   104,
						"Banjo":                   105,
						"Shamisen":                106,
						"Koto":                    107,
						"Kalimba":                 108,
						"Bagpipe":                 109,
						"Fiddle":                  110,
						"Shanai":                  111,
						"Tinkle Bell":             112,
						"Agogo":                   113,
						"Steel Drums":             114,
						"Woodblock":               115,
						"Taiko Drum":              116,
						"Melodic Tom":             117,
						"Synth Drum":              118,
						"Reverse Cymbal":          119,
						"Guitar Fret Noise":       120,
						"Breath Noise":            121,
						"Seashore":                122,
						"Bird Tweet":              123,
						"Telephone Ring":          124,
						"Helicopter":              125,
						"Applause":                126,
						"Gunshot":                 127,
					}

					for i, ch := range channels {

						instrument := genChannels[i].SelectAttrValue("instrument", "Bright Acoustic Piano")
						program := instrumentMap[instrument]

						for _, tr := range ch {

							track := smf.Track{}

							track.Add(0, midi.ProgramChange(uint8(i), program))

							for _, note := range tr {
								deltaticks := uint32(float64(clock.Ticks4th())*note.Duration) * 4
								track.Add(0, midi.NoteOn(uint8(i), uint8(note.Value), 64))
								track.Add(deltaticks, midi.NoteOff(uint8(i), uint8(note.Value)))
							}

							track.Close(0)

							if err := s.Add(track); err != nil {
								log.Fatalln(err)
							}
						}
					}

					var buf bytes.Buffer
					s.WriteTo(&buf)
					os.WriteFile("output.midi", buf.Bytes(), 0777)
				}()

				end := time.Now()

				fmt.Println("execution time:", end.Sub(start))

				newPlayer := exec.Command(`C:\Program Files\MuseScore 4\bin\MuseScore4.exe`, `output.midi`)
				if err := newPlayer.Start(); err != nil {
					log.Fatalln(err)
				}

				time.Sleep(3 * time.Second)

				if player != nil {
					if err := player.Process.Kill(); err != nil {
						log.Fatalln(err)
					}
				}

				player = newPlayer

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
