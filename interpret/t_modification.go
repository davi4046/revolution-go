package interpret

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/davi4046/revoutil"
)

type modification struct {
	path   string
	args   []string
	input  []revoutil.Note
	output []revoutil.Note
}

func newModification(path string, args []string, input []revoutil.Note, wg *sync.WaitGroup) modification {
	defer wg.Done()

	fmt.Println("newModification called")

	var output []revoutil.Note

	command := exec.Command(path, args...)

	stdin, err := command.StdinPipe()
	if err != nil {
		log.Fatalln(err)
	}

	stdout, err := command.StdoutPipe()
	if err != nil {
		log.Fatalln(err)
	}

	if err := command.Start(); err != nil {
		log.Fatalln(err)
	}

	scanner := bufio.NewScanner(stdout)

	var currIndex int

	writeIndex := func() {
		fmt.Println("writeIndex called:", fmt.Sprintf("%v\n", input[currIndex]))
		_, err := io.WriteString(stdin, fmt.Sprintf("%v\n", input[currIndex]))
		if err != nil {
			log.Fatalln(err)
		}
	}

	writeIndex()

	for scanner.Scan() {
		line := scanner.Text()

		fmt.Println("scan:", line)

		line = strings.Trim(line, "[{}]")

		if line != "" {
			parts := strings.Split(line, "} {")

			for _, s := range parts {
				parts := strings.Split(s, " ")

				pitch, err := strconv.Atoi(parts[0])
				if err != nil {
					log.Fatalln(err)
				}

				duration, err := strconv.ParseFloat(parts[1], 64)
				if err != nil {
					log.Fatalln(err)
				}

				channel, err := strconv.Atoi(parts[2])
				if err != nil {
					log.Fatalln(err)
				}

				track, err := strconv.Atoi(parts[3])
				if err != nil {
					log.Fatalln(err)
				}

				output = append(output, revoutil.Note{
					Pitch:    pitch,
					Duration: duration,
					Channel:  channel,
					Track:    track,
				})
			}
		}

		currIndex++

		if currIndex == len(input) {
			break
		}

		writeIndex()
	}
	return modification{
		path:   path,
		args:   args,
		input:  input,
		output: output,
	}
}
