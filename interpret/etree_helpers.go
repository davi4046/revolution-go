package interpret

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/beevik/etree"
	"github.com/davi4046/revoutil"
)

func extractKey(el *etree.Element) revoutil.Key {

	pitch := revoutil.PitchClassMap[el.SelectAttrValue("root", "")]

	scale, err := strconv.Atoi(el.SelectAttrValue("mode", ""))
	if err != nil {
		log.Fatalln(err)
	}
	return revoutil.NewKey(pitch, scale)
}

func extractMeter(el *etree.Element) (revoutil.Meter, error) {
	numeratorStr, denominatorStr, ok := strings.Cut(el.Text(), "/")
	if !ok {
		return revoutil.Meter{}, fmt.Errorf("'/' is missing")
	}

	numerator, err := strconv.ParseUint(numeratorStr, 10, 8)
	if err != nil {
		return revoutil.Meter{}, err
	}

	denominator, err := strconv.ParseUint(denominatorStr, 10, 8)
	if err != nil {
		return revoutil.Meter{}, err
	}

	return revoutil.Meter{
		Numerator:   uint8(numerator),
		Denominator: uint8(denominator),
	}, nil
}

func extractTempo(el *etree.Element) (float64, error) {
	tempo, err := strconv.ParseFloat(el.Text(), 64)
	if err != nil {
		return 0, err
	}
	return tempo, nil
}
