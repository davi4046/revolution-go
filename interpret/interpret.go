package interpreter

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/radovskyb/watcher"
)

// Begin interpreting the specified project directory.
func Watch(filePath string) error {
	w := watcher.New()

	w.FilterOps(watcher.Write)

	go func() {
		for {
			select {
			case event := <-w.Event:
				fmt.Println(event) // Print the event's info.

				data, err := os.ReadFile(event.Path)
				if err != nil {
					log.Fatalln(err)
				}

				var comp Composition

				if err := xml.Unmarshal(data, &comp); err != nil {
					log.Fatalln(err)
				}

				data, err = xml.MarshalIndent(comp, " ", " ")
				if err != nil {
					log.Fatal(err)
				}

				fmt.Println(string(data))

			case err := <-w.Error:
				log.Fatalln(err)
			case <-w.Closed:
				return
			}
		}
	}()

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
