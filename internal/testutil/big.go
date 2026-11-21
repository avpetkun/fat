package testutil

import (
	"math/big"
	"slices"
)

// Big1 is 1 as a big.Int, read only
var Big1 = big.NewInt(1)

// LimbsBig builds a big.Int from little-endian limbs, not by the code tested
func LimbsBig(limbs ...uint64) *big.Int {
	b := new(big.Int)
	for _, limb := range slices.Backward(limbs) {
		b.Lsh(b, 64).Or(b, new(big.Int).SetUint64(limb))
	}
	return b
}

// BigClip is b on its words without spare capacity, as SetBits leaves them
func BigClip(b *big.Int) *big.Int {
	c := new(big.Int).SetBits(slices.Clip(b.Bits()))
	if b.Sign() < 0 {
		c.Neg(c)
	}
	return c
}

// BigFits reports whether b is in the range of an n-bit integer
func BigFits(b *big.Int, n int, signed bool) bool {
	if signed {
		return b.BitLen() < n || b.Sign() < 0 && new(big.Int).Add(b, Big1).BitLen() < n
	}
	return b.Sign() >= 0 && b.BitLen() <= n
}

// BigAdd returns x + y in a new big.Int
func BigAdd(x, y *big.Int) *big.Int { return new(big.Int).Add(x, y) }

// BigSub returns x - y in a new big.Int
func BigSub(x, y *big.Int) *big.Int { return new(big.Int).Sub(x, y) }
