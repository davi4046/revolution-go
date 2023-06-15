package interpret

import "github.com/davi4046/revoutil"

type change struct {
	start float64
	key   key
	time  revoutil.Time
	tempo uint8
}
