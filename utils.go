package crawler

import (
	"math/rand"
	"time"
)

func randIn(max int) int {
	if max == 0 {
		return max
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return r.Intn(max)
}
