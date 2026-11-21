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
