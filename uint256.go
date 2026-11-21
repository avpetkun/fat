package fat

import (
	"math/big"
	"math/bits"
)

// Uint256 is an unsigned 256-bit integer of 64-bit limbs, L0 is the lowest
type Uint256 struct{ L0, L1, L2, L3 uint64 }

// --- constructors ---------------------------------------

// NewUint256 returns v as a Uint256
func NewUint256(v uint64) Uint256 {
	return Uint256{v, 0, 0, 0}
}

// Uint256FromBig returns v, or zero and ErrNegative or ErrRange
func Uint256FromBig(v *big.Int) (Uint256, error) {
	// x is zero on an error
	x, e := uint256FromBig(v)
	return x, uint256FromBigErrs[e&3]
}

// MustUint256FromBig is Uint256FromBig that panics when v does not fit
func MustUint256FromBig(v *big.Int) Uint256 {
	x, err := uint256FromBig(v)
	if err != 0 {
		// a table lookup and not a call keeps it within the inline budget
		panic(mustUint256FromBigPanics[err&3])
	}
	return x
}

// uint256FromBigErrs are the errors of Uint256FromBig by error bits
var uint256FromBigErrs = [4]error{
	1: ErrNegative{"Uint256FromBig", ""},
	2: ErrRange("Uint256FromBig"),
	3: ErrNegative{"Uint256FromBig", ""},
}

// mustUint256FromBigPanics are the panics of MustUint256FromBig by error bits
var mustUint256FromBigPanics = [4]string{
	1: "fat.MustUint256FromBig: negative value",
	2: "fat.MustUint256FromBig: out of range",
	3: "fat.MustUint256FromBig: negative value",
}

// uint256FromBig is Uint256FromBig with error bits: 1 for a negative v,
// 2 for one past 256 bits
func uint256FromBig(v *big.Int) (Uint256, uint) {
	words := v.Bits()
	n := len(words)
	if bits.UintSize == 64 && cap(words) >= 4 {
		// words past len are read within cap and masked off, no length branch
		w := words[:4]
		w0 := uint64(w[0]) & uint64((0-n)>>(bits.UintSize-1))
		// all ones for a negative v and for one past 4 words
		neg := bigNeg(v, w0)
		over := uint64(int64(4-n) >> 63)
		ok := ^(over | neg)
		return Uint256{
			w0 & ok,
			uint64(w[1]) & uint64((1-n)>>(bits.UintSize-1)) & ok,
			uint64(w[2]) & uint64((2-n)>>(bits.UintSize-1)) & ok,
			uint64(w[3]) & uint64((3-n)>>(bits.UintSize-1)) & ok,
		}, uint(neg&1 | over&2)
	}
	// short words from SetBits, or 32-bit words
	u, left := uint256FromWords(words)
	err := uint(v.Sign())>>(bits.UintSize-1) | uint(-left)>>(bits.UintSize-1)<<1
	// all ones when v fits
	ok := uint64(int64(uint64(err)-1) >> 63)
	return Uint256{u.L0 & ok, u.L1 & ok, u.L2 & ok, u.L3 & ok}, err
}

// --- getters --------------------------------------------

// IsZero reports whether x == 0
func (x Uint256) IsZero() bool {
	return x.L0|x.L1|x.L2|x.L3 == 0
}

// Sign returns 0 for x == 0 and 1 otherwise
func (x Uint256) Sign() int {
	// the top bit of n|-n is set when n != 0
	n := x.L0 | x.L1 | x.L2 | x.L3
	return int((n | -n) >> 63)
}

// IsUint64 reports whether x fits in a uint64
func (x Uint256) IsUint64() bool {
	return x.L1|x.L2|x.L3 == 0
}

// Uint64 returns x as a uint64, it panics when x does not fit
func (x Uint256) Uint64() uint64 {
	if !x.IsUint64() {
		panic("fat.Uint256.Uint64: out of range")
	}
	return x.L0
}

// Uint64o returns Uint64u and whether x does not fit
func (x Uint256) Uint64o() (r uint64, overflow bool) {
	return x.Uint64u(), !x.IsUint64()
}

// Uint64u returns the low 64 bits of x as a Go conversion does
func (x Uint256) Uint64u() uint64 {
	return x.L0
}

// IsUint128 reports whether x fits in a Uint128
func (x Uint256) IsUint128() bool {
	return x.L2|x.L3 == 0
}

// Uint128 returns x as a Uint128, it panics when x does not fit
func (x Uint256) Uint128() Uint128 {
	if !x.IsUint128() {
		panic("fat.Uint256.Uint128: out of range")
	}
	return Uint128{x.L0, x.L1}
}

// Uint128o returns Uint128u and whether x does not fit
func (x Uint256) Uint128o() (r Uint128, overflow bool) {
	return x.Uint128u(), !x.IsUint128()
}

// Uint128u returns the low 128 bits of x as a Go conversion does
func (x Uint256) Uint128u() Uint128 {
	return Uint128{x.L0, x.L1}
}

// IsInt64 reports whether x fits in an int64
func (x Uint256) IsInt64() bool {
	return x.L1|x.L2|x.L3|x.L0>>63 == 0
}

// Int64 returns x as an int64, it panics when x does not fit
func (x Uint256) Int64() int64 {
	if !x.IsInt64() {
		panic("fat.Uint256.Int64: out of range")
	}
	return int64(x.L0)
}

// Int64o returns Int64u and whether x does not fit
func (x Uint256) Int64o() (r int64, overflow bool) {
	return x.Int64u(), !x.IsInt64()
}

// Int64u returns the low 64 bits of x as a Go conversion does
func (x Uint256) Int64u() int64 {
	return int64(x.L0)
}

// IsInt128 reports whether x fits in an Int128
func (x Uint256) IsInt128() bool {
	return x.L2|x.L3|x.L1>>63 == 0
}

// Int128 returns x as an Int128, it panics when x does not fit
func (x Uint256) Int128() Int128 {
	if !x.IsInt128() {
		panic("fat.Uint256.Int128: out of range")
	}
	return Int128{x.L0, x.L1}
}

// Int128o returns Int128u and whether x does not fit
func (x Uint256) Int128o() (r Int128, overflow bool) {
	return x.Int128u(), !x.IsInt128()
}

// Int128u returns the low 128 bits of x as a Go conversion does
func (x Uint256) Int128u() Int128 {
	return Int128{x.L0, x.L1}
}

// IsInt256 reports whether x fits in an Int256
func (x Uint256) IsInt256() bool {
	return x.L3>>63 == 0
}

// Int256 returns x as an Int256, it panics when x does not fit
func (x Uint256) Int256() Int256 {
	if !x.IsInt256() {
		panic("fat.Uint256.Int256: out of range")
	}
	return Int256(x)
}

// Int256o returns Int256u and whether x does not fit
func (x Uint256) Int256o() (r Int256, overflow bool) {
	return x.Int256u(), !x.IsInt256()
}

// Int256u returns the bits of x as a Go conversion does
func (x Uint256) Int256u() Int256 {
	return Int256(x)
}

// Big returns x as a new big.Int
func (x Uint256) Big() *big.Int {
	// the Int and its words in one allocation, PutBig writes into the words
	p := new(struct {
		b big.Int
		w [4 * limbWords]big.Word
	})
	return x.PutBig(p.b.SetBits(p.w[:0]))
}

// PutBig sets b to x and returns it, reusing the words of b if they fit
func (x Uint256) PutBig(b *big.Int) *big.Int {
	words := b.Bits()
	if cap(words) < 4*limbWords {
		words = make([]big.Word, 4*limbWords)
	}
	w := words[:4*limbWords]
	x.putWords(w)
	// a limb per nonzero prefix L3, L3|L2, ...: the exact length,
	// so SetBits has at most one zero 32-bit word to trim
	m2 := x.L3 | x.L2
	m1 := m2 | x.L1
	n := 4 - bits.LeadingZeros64(x.L3)>>6 - bits.LeadingZeros64(m2)>>6 - bits.LeadingZeros64(m1)>>6 - bits.LeadingZeros64(m1|x.L0)>>6
	return b.SetBits(w[:n*limbWords])
}

// putWords writes x into w for any word size, w must hold 4*limbWords words
func (x Uint256) putWords(w []big.Word) {
	// the high halves of 32-bit words,
	// for 64-bit ones the low stores below overwrite them
	w[4*limbWords-1] = big.Word(x.L3 >> (bits.UintSize % 64))
	w[3*limbWords-1] = big.Word(x.L2 >> (bits.UintSize % 64))
	w[2*limbWords-1] = big.Word(x.L1 >> (bits.UintSize % 64))
	w[limbWords-1] = big.Word(x.L0 >> (bits.UintSize % 64))
	// low words next to each other, they pair up
	w[0] = big.Word(x.L0)
	w[limbWords] = big.Word(x.L1)
	w[2*limbWords] = big.Word(x.L2)
	w[3*limbWords] = big.Word(x.L3)
}

// uint256FromWords reads the low 4 limbs of |v| from words of any size.
// left is the count of words past them
func uint256FromWords(words []big.Word) (x Uint256, left int) {
	n := min(len(words), 4*limbWords)
	// from the top word down in registers:
	// an array would cost the fast path a stack frame
	for i := n - 1; i >= 0; i-- {
		// a word shifts in, for 64-bit words the two half shifts move a limb
		const h = bits.UintSize / 2
		x.L3, x.L2, x.L1, x.L0 = x.L3<<h<<h|x.L2>>(64-bits.UintSize), x.L2<<h<<h|x.L1>>(64-bits.UintSize), x.L1<<h<<h|x.L0>>(64-bits.UintSize), x.L0<<h<<h|uint64(words[i])
	}
	return x, len(words) - n
}

// --- comparison -----------------------------------------

// Cmp returns -1 for x < y, 0 for x = y and 1 for x > y
func (x Uint256) Cmp(y Uint256) int {
	// x - y: a borrow out of the top limb tells x < y and the difference
	// x != y, one chain for both.
	// Faster than an XOR test next to the borrows in our benchmarks,
	// and two chains as in Uint128.Cmp do not inline
	d0, carry := bits.Sub64(x.L0, y.L0, 0)
	d1, carry := bits.Sub64(x.L1, y.L1, carry)
	d2, carry := bits.Sub64(x.L2, y.L2, carry)
	d3, carry := bits.Sub64(x.L3, y.L3, carry)
	// -1 on borrow, else 1 if any difference
	nz := 0
	if d0|d1|d2|d3 != 0 {
		nz = 1
	}
	return int(-carry) | nz
}

// LessThan reports whether x < y
func (x Uint256) LessThan(y Uint256) bool {
	_, carry := bits.Sub64(x.L0, y.L0, 0)
	_, carry = bits.Sub64(x.L1, y.L1, carry)
	_, carry = bits.Sub64(x.L2, y.L2, carry)
	_, carry = bits.Sub64(x.L3, y.L3, carry)
	return carry != 0
}

// LessOrEqual reports whether x <= y
func (x Uint256) LessOrEqual(y Uint256) bool {
	// x <= y iff y - x doesn't borrow
	_, carry := bits.Sub64(y.L0, x.L0, 0)
	_, carry = bits.Sub64(y.L1, x.L1, carry)
	_, carry = bits.Sub64(y.L2, x.L2, carry)
	_, carry = bits.Sub64(y.L3, x.L3, carry)
	return carry == 0
}

// GreaterThan reports whether x > y
func (x Uint256) GreaterThan(y Uint256) bool {
	_, carry := bits.Sub64(y.L0, x.L0, 0)
	_, carry = bits.Sub64(y.L1, x.L1, carry)
	_, carry = bits.Sub64(y.L2, x.L2, carry)
	_, carry = bits.Sub64(y.L3, x.L3, carry)
	return carry == 1
}

// GreaterOrEqual reports whether x >= y
func (x Uint256) GreaterOrEqual(y Uint256) bool {
	// x >= y iff x - y doesn't borrow
	_, carry := bits.Sub64(x.L0, y.L0, 0)
	_, carry = bits.Sub64(x.L1, y.L1, carry)
	_, carry = bits.Sub64(x.L2, y.L2, carry)
	_, carry = bits.Sub64(x.L3, y.L3, carry)
	return carry == 0
}

// Max returns the larger of x and y, as the builtin max
func (x Uint256) Max(y Uint256) Uint256 {
	// the borrow chain of LessThan written out picks by CSEL,
	// a select by masks does not inline
	_, carry := bits.Sub64(x.L0, y.L0, 0)
	_, carry = bits.Sub64(x.L1, y.L1, carry)
	_, carry = bits.Sub64(x.L2, y.L2, carry)
	_, carry = bits.Sub64(x.L3, y.L3, carry)
	if carry != 0 {
		x = y
	}
	return x
}

// Min returns the smaller of x and y, as the builtin min
func (x Uint256) Min(y Uint256) Uint256 {
	_, carry := bits.Sub64(x.L0, y.L0, 0)
	_, carry = bits.Sub64(x.L1, y.L1, carry)
	_, carry = bits.Sub64(x.L2, y.L2, carry)
	_, carry = bits.Sub64(x.L3, y.L3, carry)
	if carry == 0 {
		x = y
	}
	return x
}

// Cmp128 returns -1 for x < y, 0 for x = y and 1 for x > y
func (x Uint256) Cmp128(y Uint128) int {
	d0, carry := bits.Sub64(x.L0, y.L0, 0)
	d1, carry := bits.Sub64(x.L1, y.L1, carry)
	hi := x.L2 | x.L3
	// borrows only if hi is zero and low limbs borrowed
	_, carry = bits.Sub64(hi, 0, carry)
	nz := 0
	if d0|d1|hi != 0 {
		nz = 1
	}
	return int(-carry) | nz
}

// Cmp64 returns -1 for x < y, 0 for x = y and 1 for x > y
func (x Uint256) Cmp64(y uint64) int {
	hi := x.L1 | x.L2 | x.L3
	d, b := bits.Sub64(x.L0, y, 0)
	// borrows only if hi is zero and x.L0 < y
	_, b = bits.Sub64(hi, 0, b)
	nz := 0
	if d|hi != 0 {
		nz = 1
	}
	return int(-b) | nz
}

// CmpBig returns -1 for x < y, 0 for x = y and 1 for x > y
func (x Uint256) CmpBig(y *big.Int) int {
	w := y.Bits()
	n := len(w)
	var l0, l1, l2, l3 uint64
	if bits.UintSize == 64 && uint(n-1) < 4 {
		l0 = uint64(w[0])
		// m>>(m>>1) is min(1, m), m&^(m>>1) is min(2, m): no bounds checks,
		// and masks zero the words past n
		m := uint(n - 1)
		l1 = uint64(w[m>>(m>>1)]) & uint64(int64(1-n)>>63)
		l2 = uint64(w[m&^(m>>1)]) & uint64(int64(2-n)>>63)
		l3 = uint64(w[m]) & uint64(int64(3-n)>>63)
	} else {
		// zero, past 256 bits, or 32-bit words: the low 256 bits shift in
		// from the top word, inline as a call would cost a stack frame
		sh := uint(bits.UintSize)
		for i := min(n, 256/bits.UintSize) - 1; i >= 0; i-- {
			l3 = l3<<sh | l2>>(64-sh)
			l2 = l2<<sh | l1>>(64-sh)
			l1 = l1<<sh | l0>>(64-sh)
			l0 = l0<<sh | uint64(w[i])
		}
		if n > 256/bits.UintSize {
			// |y| past 256 bits is below or above any x by its sign
			return -y.Sign()
		}
	}
	c := x.Cmp(Uint256{l0, l1, l2, l3})
	// negative y is below any x
	if bigIsNeg(y, l0) {
		c = 1
	}
	return c
}

// --- add ------------------------------------------------

// Add returns x + y, it panics on overflow
func (x Uint256) Add(y Uint256) Uint256 {
	// written out: a call of Addo leaves the NOP of its inline mark in a loop
	var carry uint64
	x.L0, carry = bits.Add64(x.L0, y.L0, 0)
	x.L1, carry = bits.Add64(x.L1, y.L1, carry)
	x.L2, carry = bits.Add64(x.L2, y.L2, carry)
	x.L3, carry = bits.Add64(x.L3, y.L3, carry)
	if carry != 0 {
		panic("fat.Uint256.Add: overflow")
	}
	return x
}

// Addo returns Addu and whether x + y overflowed
func (x Uint256) Addo(y Uint256) (r Uint256, overflow bool) {
	var carry uint64
	x.L0, carry = bits.Add64(x.L0, y.L0, 0)
	x.L1, carry = bits.Add64(x.L1, y.L1, carry)
	x.L2, carry = bits.Add64(x.L2, y.L2, carry)
	x.L3, carry = bits.Add64(x.L3, y.L3, carry)
	return x, carry != 0
}

// Addu returns x + y wrapped around as on a Go integer
func (x Uint256) Addu(y Uint256) Uint256 {
	var carry uint64
	x.L0, carry = bits.Add64(x.L0, y.L0, 0)
	x.L1, carry = bits.Add64(x.L1, y.L1, carry)
	x.L2, carry = bits.Add64(x.L2, y.L2, carry)
	x.L3, _ = bits.Add64(x.L3, y.L3, carry)
	return x
}

// Add128 returns x + y, it panics on overflow
func (x Uint256) Add128(y Uint128) Uint256 {
	// written out: a call of Add128o leaves the NOP of its inline mark in a loop
	var carry uint64
	x.L0, carry = bits.Add64(x.L0, y.L0, 0)
	x.L1, carry = bits.Add64(x.L1, y.L1, carry)
	x.L2, carry = bits.Add64(x.L2, 0, carry)
	x.L3, carry = bits.Add64(x.L3, 0, carry)
	if carry != 0 {
		panic("fat.Uint256.Add128: overflow")
	}
	return x
}

// Add128o returns Add128u and whether x + y overflowed
func (x Uint256) Add128o(y Uint128) (r Uint256, overflow bool) {
	var carry uint64
	x.L0, carry = bits.Add64(x.L0, y.L0, 0)
	x.L1, carry = bits.Add64(x.L1, y.L1, carry)
	x.L2, carry = bits.Add64(x.L2, 0, carry)
	x.L3, carry = bits.Add64(x.L3, 0, carry)
	return x, carry != 0
}

// Add128u returns x + y wrapped around as on a Go integer
func (x Uint256) Add128u(y Uint128) Uint256 {
	var carry uint64
	x.L0, carry = bits.Add64(x.L0, y.L0, 0)
	x.L1, carry = bits.Add64(x.L1, y.L1, carry)
	x.L2, carry = bits.Add64(x.L2, 0, carry)
	x.L3, _ = bits.Add64(x.L3, 0, carry)
	return x
}

// Add64 returns x + y, it panics on overflow
func (x Uint256) Add64(y uint64) Uint256 {
	// written out: a call of Add64o leaves the NOP of its inline mark in a loop
	var c uint64
	x.L0, c = bits.Add64(x.L0, y, 0)
	x.L1, c = bits.Add64(x.L1, 0, c)
	x.L2, c = bits.Add64(x.L2, 0, c)
	x.L3, c = bits.Add64(x.L3, 0, c)
	if c != 0 {
		panic("fat.Uint256.Add64: overflow")
	}
	return x
}

// Add64o returns Add64u and whether x + y overflowed
func (x Uint256) Add64o(y uint64) (r Uint256, overflow bool) {
	var c uint64
	x.L0, c = bits.Add64(x.L0, y, 0)
	x.L1, c = bits.Add64(x.L1, 0, c)
	x.L2, c = bits.Add64(x.L2, 0, c)
	x.L3, c = bits.Add64(x.L3, 0, c)
	return x, c != 0
}

// Add64u returns x + y wrapped around as on a Go integer
func (x Uint256) Add64u(y uint64) Uint256 {
	var c uint64
	x.L0, c = bits.Add64(x.L0, y, 0)
	x.L1, c = bits.Add64(x.L1, 0, c)
	x.L2, c = bits.Add64(x.L2, 0, c)
	x.L3, _ = bits.Add64(x.L3, 0, c)
	return x
}

// --- sub ------------------------------------------------

// Sub returns x - y, it panics when y > x
func (x Uint256) Sub(y Uint256) Uint256 {
	// written out: a call of Subo leaves the NOP of its inline mark in a loop
	var carry uint64
	x.L0, carry = bits.Sub64(x.L0, y.L0, 0)
	x.L1, carry = bits.Sub64(x.L1, y.L1, carry)
	x.L2, carry = bits.Sub64(x.L2, y.L2, carry)
	x.L3, carry = bits.Sub64(x.L3, y.L3, carry)
	if carry != 0 {
		panic("fat.Uint256.Sub: negative value")
	}
	return x
}

// Subo returns Subu and whether y > x
func (x Uint256) Subo(y Uint256) (r Uint256, overflow bool) {
	var carry uint64
	x.L0, carry = bits.Sub64(x.L0, y.L0, 0)
	x.L1, carry = bits.Sub64(x.L1, y.L1, carry)
	x.L2, carry = bits.Sub64(x.L2, y.L2, carry)
	x.L3, carry = bits.Sub64(x.L3, y.L3, carry)
	return x, carry != 0
}

// Subu returns x - y wrapped around as on a Go integer
func (x Uint256) Subu(y Uint256) Uint256 {
	var carry uint64
	x.L0, carry = bits.Sub64(x.L0, y.L0, 0)
	x.L1, carry = bits.Sub64(x.L1, y.L1, carry)
	x.L2, carry = bits.Sub64(x.L2, y.L2, carry)
	x.L3, _ = bits.Sub64(x.L3, y.L3, carry)
	return x
}

// Sub128 returns x - y, it panics when y > x
func (x Uint256) Sub128(y Uint128) Uint256 {
	// written out: a call of Sub128o leaves the NOP of its inline mark in a loop
	var carry uint64
	x.L0, carry = bits.Sub64(x.L0, y.L0, 0)
	x.L1, carry = bits.Sub64(x.L1, y.L1, carry)
	x.L2, carry = bits.Sub64(x.L2, 0, carry)
	x.L3, carry = bits.Sub64(x.L3, 0, carry)
	if carry != 0 {
		panic("fat.Uint256.Sub128: negative value")
	}
	return x
}

// Sub128o returns Sub128u and whether y > x
func (x Uint256) Sub128o(y Uint128) (r Uint256, overflow bool) {
	var carry uint64
	x.L0, carry = bits.Sub64(x.L0, y.L0, 0)
	x.L1, carry = bits.Sub64(x.L1, y.L1, carry)
	x.L2, carry = bits.Sub64(x.L2, 0, carry)
	x.L3, carry = bits.Sub64(x.L3, 0, carry)
	return x, carry != 0
}

// Sub128u returns x - y wrapped around as on a Go integer
func (x Uint256) Sub128u(y Uint128) Uint256 {
	var carry uint64
	x.L0, carry = bits.Sub64(x.L0, y.L0, 0)
	x.L1, carry = bits.Sub64(x.L1, y.L1, carry)
	x.L2, carry = bits.Sub64(x.L2, 0, carry)
	x.L3, _ = bits.Sub64(x.L3, 0, carry)
	return x
}

// Sub64 returns x - y, it panics when y > x
func (x Uint256) Sub64(y uint64) Uint256 {
	// written out: a call of Sub64o leaves the NOP of its inline mark in a loop
	var c uint64
	x.L0, c = bits.Sub64(x.L0, y, 0)
	x.L1, c = bits.Sub64(x.L1, 0, c)
	x.L2, c = bits.Sub64(x.L2, 0, c)
	x.L3, c = bits.Sub64(x.L3, 0, c)
	if c != 0 {
		panic("fat.Uint256.Sub64: negative value")
	}
	return x
}

// Sub64o returns Sub64u and whether y > x
func (x Uint256) Sub64o(y uint64) (r Uint256, overflow bool) {
	var c uint64
	x.L0, c = bits.Sub64(x.L0, y, 0)
	x.L1, c = bits.Sub64(x.L1, 0, c)
	x.L2, c = bits.Sub64(x.L2, 0, c)
	x.L3, c = bits.Sub64(x.L3, 0, c)
	return x, c != 0
}

// Sub64u returns x - y wrapped around as on a Go integer
func (x Uint256) Sub64u(y uint64) Uint256 {
	var carry uint64
	x.L0, carry = bits.Sub64(x.L0, y, 0)
	x.L1, carry = bits.Sub64(x.L1, 0, carry)
	x.L2, carry = bits.Sub64(x.L2, 0, carry)
	x.L3, _ = bits.Sub64(x.L3, 0, carry)
	return x
}
