package randutil

import "math/rand"

func GetRandomInt(min, max int) int {
	return rand.Intn(max-min+1) + min
}
