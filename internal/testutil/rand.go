package testutil

import (
	"math"
	"math/bits"
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

// RandBits returns the limbs of a random n-bit value, 1 <= n <= 256
func RandBits(r *rand.Rand, n int) [4]uint64 {
	var l [4]uint64
	for i := range l {
		switch lo := i * 64; {
		case n <= lo:
		case n < lo+64:
			l[i] = r.Uint64()&(1<<(n-lo)-1) | 1<<(n-lo-1)
		default:
			l[i] = r.Uint64()
		}
	}
	return l
}

// RandSigned256 returns the limbs of a random n-bit magnitude, signed
func RandSigned256(r *rand.Rand, n int) [4]uint64 {
	l := RandBits(r, n)
	if r.Intn(2) == 0 {
		var c uint64
		l[0], c = bits.Add64(^l[0], 1, 0)
		l[1], c = bits.Add64(^l[1], 0, c)
		l[2], c = bits.Add64(^l[2], 0, c)
		l[3], _ = bits.Add64(^l[3], 0, c)
	}
	return l
}
