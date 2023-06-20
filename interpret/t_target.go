package interpret

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type target struct {
	channels []int
	tracks   []int
}

func stringToTarget(s string) (target, error) {

	var target target

	r := regexp.MustCompile(`(ch\(\d+(-\d+)?(,\s*\d+(-\d+)?)*\))\/(tr\(\d+(-\d+)?(,\s*\d+(-\d+)?)*\))`)

	if !r.MatchString(s) {
		return target, fmt.Errorf("invalid target string:", s)
	}

	s = strings.ReplaceAll(s, " ", "")

	ch, tr, _ := strings.Cut(s, "/")

	chStrings := strings.Split(strings.Trim(ch, "ch()"), ",")
	trStrings := strings.Split(strings.Trim(tr, "tr()"), ",")

	for _, s := range chStrings {
		fromStr, toStr, ok := strings.Cut(s, "-")
		if !ok {
			i, _ := strconv.Atoi(s)
			target.channels = append(target.channels, i)
			continue
		}
		from, _ := strconv.Atoi(fromStr)
		to, _ := strconv.Atoi(toStr)
		for i := from; i <= to; i++ {
			target.channels = append(target.channels, i)
		}
	}

	for _, s := range trStrings {
		fromStr, toStr, ok := strings.Cut(s, "-")
		if !ok {
			i, _ := strconv.Atoi(s)
			target.tracks = append(target.tracks, i)
			continue
		}
		from, _ := strconv.Atoi(fromStr)
		to, _ := strconv.Atoi(toStr)
		for i := from; i <= to; i++ {
			target.tracks = append(target.tracks, i)
		}
	}

	sort.Ints(target.channels)
	sort.Ints(target.tracks)

	return target, nil
}
