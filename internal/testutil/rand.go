package testutil

import (
	"math"
	"math/rand"
)

// RandomCount is the number of random values or pairs on top of the edges.
// It covers the limb patterns RandLimb mixes, more only slows the tests
const RandomCount = 2000

// RandLimb favours values around limb and sign boundaries
func RandLimb(r *rand.Rand) uint64 {
	switch r.Intn(8) {
	case 0:
		return 0
	case 1:
		return math.MaxUint64
	case 2:
		return 1 << 63
	case 3:
		return 1<<63 - 1
	case 4:
		return uint64(r.Intn(3))
	default:
		return r.Uint64() >> r.Intn(64)
	}
}
