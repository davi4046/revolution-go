package interpret

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/exp/slices"
)

type generationManager struct {
	settings   generationSettings
	command    *exec.Cmd
	stdin      *io.WriteCloser
	stdout     *io.ReadCloser
	generation []Note
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

func (g *generationManager) generateFromTo(from float64, to float64, wg *sync.WaitGroup) []Note {
	wg.Add(1)

	negativeGen := g.generate(-1, math.Min(from, 0), wg)
	positiveGen := g.generate(0, math.Max(to, 0), wg)

	generation := append(negativeGen, positiveGen...)

	return getFromTo(generation, from, to)
}

func (g *generationManager) generate(startIndex int, length float64, wg *sync.WaitGroup) []Note {
	defer wg.Done()

	var generation []Note

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

		generation = append(generation, Note{
			Value:    degree,
			Start:    start,
			Duration: duration,
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
