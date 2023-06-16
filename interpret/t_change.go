package interpret

import "github.com/davi4046/revoutil"

type change struct {
	barStart  float64
	noteStart float64
	key       key
	time      revoutil.Time
	tempo     uint8
}
