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

// Divisors64 are edge divisors and a random one of every bit length
var Divisors64 = func() []uint64 {
	d := []uint64{1, 2, 3, 7, 10, 1e19, 1<<32 - 1, 1 << 32, 1<<32 + 1, math.MaxUint64 / 3,
		1<<63 - 1, 1 << 63, 1<<63 + 1, math.MaxUint64 - 1, math.MaxUint64,
		// the reciprocal is one short before its last step, as 1 in 40 are
		0x8000800080000000, 0xc03fc03f00000000, 0xff0000ff00000000, 0xa1df3fa6fffffffe, 0x81c48fce6fe09024,
		0xed409baf80000001, 0xa6a794bcffffffff, 0xc03fc03f, 0x4000400040000000}
	r := rand.New(rand.NewSource(4))
	for n := 1; n <= 64; n++ {
		d = append(d, RandBits(r, n)[0])
	}
	return d
}()

// SeedDivisors64 hit both ends of every class of the top 9 bits.
// The bits are those of the normalized divisor, which comes at many shifts
var SeedDivisors64 = func() []uint64 {
	var d []uint64
	for k := uint64(256); k < 512; k++ {
		d = append(d, k<<55>>(k%56), k<<55|(1<<55-1))
	}
	return d
}()
