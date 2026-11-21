package fat

import (
	"math/big"
	"math/bits"
)

// --- big.Int --------------------------------------------

// limbWords is the number of big.Word in a limb, 1 or 2
const limbWords = 64 / bits.UintSize

// bigNeg returns all ones for a negative v, else 0.
// w0 must be the low word of |v|; Sign would branch on the sign
func bigNeg(v *big.Int, w0 uint64) uint64 {
	if w0<<1 == 0 {
		// 0 and 1<<63 are their own negation, rare enough for Sign
		return uint64(v.Sign() >> 1)
	}
	// Int64 negates by a select, so it differs from w0 only for v < 0
	d := uint64(v.Int64()) ^ w0
	return -((d | -d) >> 63)
}

// bigIsNeg reports whether v < 0.
// w0 must be the low word of |v|; Sign would branch on the sign
func bigIsNeg(v *big.Int, w0 uint64) bool {
	// Int64 negates by a select, so it differs from w0 only for v < 0
	neg := uint64(v.Int64()) != w0
	if w0<<1 == 0 {
		// 0 and 1<<63 are their own negation, rare enough for Sign
		neg = v.Sign() < 0
	}
	return neg
}

// div2by1 returns the quotient and remainder of <u1, u0> / d by v,
// the reciprocal of d: DIV2BY1 of "Improved division by invariant integers".
// It needs u1 < d and the top bit of d set, other inputs give garbage
func div2by1(u1, u0, d, v uint64) (uint64, uint64) {
	q1, q0 := bits.Mul64(v, u1)
	q0, c := bits.Add64(q0, u0, 0)
	q1, _ = bits.Add64(q1, u1, c)
	// the candidate quotient is q1+1, u0-d is ready before the product
	r := u0 - d - q1*d

	// one if over both values: amd64 keeps an if-else as a branch,
	// and the branch follows no pattern
	q := q1 + 1
	if r > q0 {
		q = q1
		r += d
	}
	q1 = q

	if r >= d {
		q1++
		r -= d
	}

	return q1, r
}

// quo2by1 is div2by1 without the remainder, with the same preconditions
func quo2by1(u1, u0, d, v uint64) uint64 {
	q1, q0 := bits.Mul64(v, u1)
	q0, c := bits.Add64(q0, u0, 0)
	q1, _ = bits.Add64(q1, u1, c)
	r := u0 - d - q1*d
	q := q1 + 1
	if r > q0 {
		q = q1
		r += d
	}
	q1 = q
	if r >= d {
		q1++
	}
	return q1
}

// reciprocal2by1 returns (2^128-1)/d - 2^64; d must have its top bit set.
// A hardware division gives 2^87/d less than 3 short.
// One Newton step with its second order term gets within 1 of the result,
// and the last step of Algorithm 3 of the paper above makes it exact
func reciprocal2by1(d uint64) uint64 {
	// a 24-bit seed and not 32: a divider that takes a cycle per 4 bits
	// of the quotient is 2 cycles shorter
	v := (1<<47 - 1) / (d>>40 + 1)
	// e = (2^87 - v*d) / 2 < 2^64, halved through d/2 rounded up
	d0 := d & 1
	e := (v>>1)&-d0 - v*(d>>1+d0)
	// 2^128/d = 2^41*v + v*E/2^46 + v*E^2/2^133 + less than 0.22
	// for E = 2^87 - v*d < 1.5*2^64 + 2^40
	hi, _ := bits.Mul64(e, e)
	e += hi >> 22
	hi, _ = bits.Mul64(v<<19, e)
	v = v<<41 + hi
	// one more when (v + 2^64 + 1) * d still fits 2^128
	hi, lo := bits.Mul64(v, d)
	_, b := bits.Sub64(^d, lo, 0)
	v, _ = bits.Sub64(v, hi+d, b)
	return v
}
