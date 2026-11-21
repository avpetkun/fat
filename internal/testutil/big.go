package testutil

import (
	"fmt"
	"math"
	"math/big"
	"slices"
	"strings"
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

// BigOnesCount counts the one bits of b >= 0
func BigOnesCount(b *big.Int) int { return strings.Count(b.Text(2), "1") }

// BigTrailingZeros is b.TrailingZeroBits, n for b == 0 as in math/bits
func BigTrailingZeros(b *big.Int, n int) int {
	if b.Sign() == 0 {
		return n
	}
	return int(b.TrailingZeroBits())
}

// Rotations are rotate counts around each limb and the width, both signs
var Rotations = []int{0, 1, 63, 64, 65, 127, 128, 129, 191, 192, 200, 255, 256, 257, 1000, -1, -63, -64, -65, -128, -129, -255, -256, -1000, math.MaxInt, math.MinInt, math.MinInt + 1}

// BigRotateLeft rotates b in [0, 2^n) left by k mod n bits
func BigRotateLeft(b *big.Int, k, n int) *big.Int {
	k = (k%n + n) % n
	r := new(big.Int).Lsh(b, uint(k))
	r.Or(r, new(big.Int).Rsh(b, uint(n-k)))
	return r.And(r, new(big.Int).Sub(new(big.Int).Lsh(Big1, uint(n)), Big1))
}

// BigReverse reverses the n bits of b in [0, 2^n)
func BigReverse(b *big.Int, n int) *big.Int {
	s := fmt.Appendf(nil, "%0*b", n, b)
	slices.Reverse(s)
	r, _ := new(big.Int).SetString(string(s), 2)
	return r
}

// BigReverseBytes reverses the n/8 bytes of b in [0, 2^n)
func BigReverseBytes(b *big.Int, n int) *big.Int {
	s := b.FillBytes(make([]byte, n/8))
	slices.Reverse(s)
	return new(big.Int).SetBytes(s)
}

// BigAdd returns x + y in a new big.Int
func BigAdd(x, y *big.Int) *big.Int { return new(big.Int).Add(x, y) }

// BigSub returns x - y in a new big.Int
func BigSub(x, y *big.Int) *big.Int { return new(big.Int).Sub(x, y) }

// BigMul returns x * y in a new big.Int
func BigMul(x, y *big.Int) *big.Int { return new(big.Int).Mul(x, y) }

// BigAnd returns x & y in a new big.Int
func BigAnd(x, y *big.Int) *big.Int { return new(big.Int).And(x, y) }

// BigOr returns x | y in a new big.Int
func BigOr(x, y *big.Int) *big.Int { return new(big.Int).Or(x, y) }

// BigXor returns x ^ y in a new big.Int
func BigXor(x, y *big.Int) *big.Int { return new(big.Int).Xor(x, y) }

// BigAndNot returns x &^ y in a new big.Int
func BigAndNot(x, y *big.Int) *big.Int { return new(big.Int).AndNot(x, y) }

// Plain adapts an operation that reports no overflow to Arith.Fn
func Plain[T any](f func(x, y T) T) func(x, y T) (T, bool) {
	return func(x, y T) (T, bool) { return f(x, y), false }
}
