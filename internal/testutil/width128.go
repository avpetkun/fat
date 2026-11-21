package testutil

import (
	"math"
	"math/big"
	"math/rand"
)

// Shape128 is the limb layout of the 128-bit types, both signs
type Shape128 interface{ ~struct{ L0, L1 uint64 } }

// Powers of two and the int128 range as big.Int, read only
var (
	Two127       = new(big.Int).Lsh(Big1, 127)
	Two128       = new(big.Int).Lsh(Big1, 128)
	BigMinInt128 = new(big.Int).Neg(Two127)
	BigMaxInt128 = new(big.Int).Sub(Two127, Big1)
)

// RefU128 returns x as an unsigned big.Int
func RefU128[T Shape128](x T) *big.Int {
	v := struct{ L0, L1 uint64 }(x)
	return LimbsBig(v.L0, v.L1)
}

// RefI128 returns x as a two's complement big.Int
func RefI128[T Shape128](x T) *big.Int {
	v := struct{ L0, L1 uint64 }(x)
	b := LimbsBig(v.L0, v.L1)
	if v.L1>>63 != 0 {
		b.Sub(b, Two128)
	}
	return b
}

// WrapU128 reduces b modulo 2^128
func WrapU128(b *big.Int) *big.Int { return new(big.Int).Mod(b, Two128) }

// WrapI128 reduces b into the int128 range
func WrapI128(b *big.Int) *big.Int {
	r := WrapU128(b)
	if r.Cmp(Two127) >= 0 {
		r.Sub(r, Two128)
	}
	return r
}

// FitsU128 reports whether b is in the uint128 range
func FitsU128(b *big.Int) bool { return b.Sign() >= 0 && b.BitLen() <= 128 }

// FitsI128 reports whether b is in the int128 range
func FitsI128(b *big.Int) bool { return b.Cmp(BigMinInt128) >= 0 && b.Cmp(BigMaxInt128) <= 0 }

// Edge128 are limb patterns at every carry, width and sign boundary
var Edge128 = [][2]uint64{
	{},
	{1},
	{2},
	{math.MaxUint64},
	{0, 1},
	{10000000000000000000},
	{0, 1 << 63},                         // min int128
	{1, 1 << 63},                         // min int128 + 1
	{math.MaxUint64, 1<<63 - 1},          // max int128
	{math.MaxUint64 - 1, 1<<63 - 1},      // max int128 - 1
	{math.MaxUint64 - 1, math.MaxUint64}, // -2
	{math.MaxUint64, math.MaxUint64},     // -1, max uint128
	{1 << 63, math.MaxUint64},            // min int64
	{0, math.MaxUint64},                  // -2^64
	{0x9c3f5e2b7a1d4086, 0xe6b03d9f2a74c851},
}

// Rand128 returns limbs of any width, the upper limb often sign-extended
func Rand128(r *rand.Rand) [2]uint64 {
	x := [2]uint64{RandLimb(r), RandLimb(r)}
	switch r.Intn(4) {
	case 0:
		x[1] = 0
	case 1:
		x[1] = math.MaxUint64
	}
	return x
}

// Limbs128 packs limbs into a 128-bit type
func Limbs128[T Shape128](l [2]uint64) T { return T(struct{ L0, L1 uint64 }{l[0], l[1]}) }

// Values128 calls fn with every edge and many random values
func Values128[T Shape128](fn func(x T)) {
	for _, x := range Edge128 {
		fn(Limbs128[T](x))
	}
	r := rand.New(rand.NewSource(1))
	for range RandomCount {
		fn(Limbs128[T](Rand128(r)))
	}
}

// Pairs128 calls fn with every edge pair and many random pairs.
// Some random pairs are equal, or y is x with one limb plus 1
func Pairs128[T Shape128](fn func(x, y T)) {
	for _, x := range Edge128 {
		for _, y := range Edge128 {
			fn(Limbs128[T](x), Limbs128[T](y))
		}
	}
	r := rand.New(rand.NewSource(2))
	for range RandomCount {
		x, y := Rand128(r), Rand128(r)
		switch r.Intn(8) {
		case 0:
			y = x
		case 1:
			y = x
			y[r.Intn(2)]++
		}
		fn(Limbs128[T](x), Limbs128[T](y))
	}
}
