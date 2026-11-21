package testutil

import (
	"math"
	"math/big"
	"math/rand"
)

// Shape256 is the limb layout of the 256-bit types, both signs
type Shape256 interface {
	~struct{ L0, L1, L2, L3 uint64 }
}

// Powers of two and the int256 range as big.Int, read only
var (
	Two255       = new(big.Int).Lsh(Big1, 255)
	Two256       = new(big.Int).Lsh(Big1, 256)
	Two512       = new(big.Int).Lsh(Big1, 512)
	BigMinInt256 = new(big.Int).Neg(Two255)
	BigMaxInt256 = new(big.Int).Sub(Two255, Big1)
)

// RefU256 returns x as an unsigned big.Int
func RefU256[T Shape256](x T) *big.Int {
	v := struct{ L0, L1, L2, L3 uint64 }(x)
	return LimbsBig(v.L0, v.L1, v.L2, v.L3)
}

// RefI256 returns x as a two's complement big.Int
func RefI256[T Shape256](x T) *big.Int {
	v := struct{ L0, L1, L2, L3 uint64 }(x)
	b := LimbsBig(v.L0, v.L1, v.L2, v.L3)
	if v.L3>>63 != 0 {
		b.Sub(b, Two256)
	}
	return b
}

// Edge256 are limb patterns at every carry, width and sign boundary
var Edge256 = [][4]uint64{
	{},
	{1},
	{2},
	{math.MaxUint64},
	{0, 1},
	{math.MaxUint64, math.MaxUint64},
	{0, 0, 1},
	{0, 0, 0, 1},
	{10000000000000000000},
	{0, 0, 0, 1 << 63}, // min int256
	{1, 0, 0, 1 << 63}, // min int256 + 1
	{math.MaxUint64, math.MaxUint64, math.MaxUint64, 1<<63 - 1},          // max int256
	{math.MaxUint64 - 1, math.MaxUint64, math.MaxUint64, 1<<63 - 1},      // max int256 - 1
	{math.MaxUint64 - 1, math.MaxUint64, math.MaxUint64, math.MaxUint64}, // -2
	{math.MaxUint64, math.MaxUint64, math.MaxUint64, math.MaxUint64},     // -1, max uint256
	{1 << 63, math.MaxUint64, math.MaxUint64, math.MaxUint64},            // min int64
	{0, math.MaxUint64, math.MaxUint64, math.MaxUint64},                  // -2^64
	{0, 1 << 63, math.MaxUint64, math.MaxUint64},                         // min int128
	{math.MaxUint64, 1<<63 - 1},                                          // max int128
	{0x9c3f5e2b7a1d4086, 0x51e7b20f9d3ac648, 0x2d8f61a4c7e3b905, 0xe6b03d9f2a74c851},
}

// Rand256 returns limbs of any width, upper limbs often sign-extended
func Rand256(r *rand.Rand) [4]uint64 {
	x := [4]uint64{RandLimb(r), RandLimb(r), RandLimb(r), RandLimb(r)}
	if n := r.Intn(4); n > 0 {
		ext := uint64(0)
		if r.Intn(2) == 0 {
			ext = math.MaxUint64
		}
		for i := 4 - n; i < 4; i++ {
			x[i] = ext
		}
	}
	return x
}

// Limbs256 packs limbs into a 256-bit type
func Limbs256[T Shape256](l [4]uint64) T {
	return T(struct{ L0, L1, L2, L3 uint64 }{l[0], l[1], l[2], l[3]})
}

// Values256 calls fn with every edge and many random values
func Values256[T Shape256](fn func(x T)) {
	for _, x := range Edge256 {
		fn(Limbs256[T](x))
	}
	r := rand.New(rand.NewSource(1))
	for range RandomCount {
		fn(Limbs256[T](Rand256(r)))
	}
}
