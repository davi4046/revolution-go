package interpret

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/beevik/etree"
	"github.com/davi4046/revoutil"
)

func extractKey(el *etree.Element) key {
	root := el.SelectAttrValue("root", "")
	mode := el.SelectAttrValue("mode", "")
	return key{
		root: root,
		mode: mode,
	}
}

func extractTime(el *etree.Element) (revoutil.Time, error) {
	numeratorStr, denominatorStr, ok := strings.Cut(el.Text(), "/")
	if !ok {
		return revoutil.Time{}, fmt.Errorf("'/' is missing")
	}

	numerator, err := strconv.ParseUint(numeratorStr, 10, 8)
	if err != nil {
		return revoutil.Time{}, err
	}

	denominator, err := strconv.ParseUint(denominatorStr, 10, 8)
	if err != nil {
		return revoutil.Time{}, err
	}

	return revoutil.Time{
		Numerator:   uint8(numerator),
		Denominator: uint8(denominator),
	}, nil
}

func extractTempo(el *etree.Element) (uint8, error) {
	tempo, err := strconv.ParseUint(el.Text(), 10, 8)
	if err != nil {
		return 0, err
	}
	return uint8(tempo), nil
}
