package fat

import (
	"math/big"
	"math/bits"
)

// Int256 is a signed 256-bit integer in two's complement, L0 is the low limb
type Int256 struct{ L0, L1, L2, L3 uint64 }

// --- constructors ---------------------------------------

// NewInt256 returns v as an Int256
func NewInt256(v int64) Int256 {
	ext := uint64(v >> 63)
	return Int256{uint64(v), ext, ext, ext}
}

// Int256FromBig returns v, or zero and ErrRange when v does not fit
func Int256FromBig(v *big.Int) (Int256, error) {
	x, ok := int256FromBig(v)
	if !ok {
		return Int256{}, ErrRange("Int256FromBig")
	}
	return x, nil
}

// int256FromBig is Int256FromBig with ok in place of the error
func int256FromBig(v *big.Int) (Int256, bool) {
	words := v.Bits()
	n := len(words)
	if bits.UintSize == 64 && uint(n-1) < 4 {
		// no bounds checks up to the top word k, missing words are masked off
		k := uint(n - 1)
		w3 := uint64(words[k]) & uint64((3-n)>>(bits.UintSize-1))
		w2 := uint64(words[k&^(k>>1)]) & uint64((2-n)>>(bits.UintSize-1))
		w1 := uint64(words[k>>(k>>1)]) & uint64((1-n)>>(bits.UintSize-1))
		w0 := uint64(words[0])
		// all ones for a negative v
		s := bigNeg(v, w0)
		// (w ^ s) - s negates when s is all ones
		r0, b := bits.Sub64(w0^s, s, 0)
		r1, b := bits.Sub64(w1^s, s, b)
		r2, b := bits.Sub64(w2^s, s, b)
		r3, _ := bits.Sub64(w3^s, s, b)
		// all ones when v fits: past the range the sign flips, not at MinInt256
		ok := uint64(int64(^(r3 ^ s)) >> 63)
		return Int256{r0 & ok, r1 & ok, r2 & ok, r3 & ok}, ok != 0
	}
	// zero, past 256 bits, or 32-bit words
	u, left := uint256FromWords(words)
	m := uint64(v.Sign() >> 1)
	r0, b := bits.Sub64(u.L0^m, m, 0)
	r1, b := bits.Sub64(u.L1^m, m, b)
	r2, b := bits.Sub64(u.L2^m, m, b)
	r3, _ := bits.Sub64(u.L3^m, m, b)
	// all ones when v fits: no words left and the sign kept
	ok := uint64(int64(^(r3^m)&^uint64(-left)) >> 63)
	return Int256{r0 & ok, r1 & ok, r2 & ok, r3 & ok}, ok != 0
}

// MustInt256FromBig is Int256FromBig that panics when v does not fit
func MustInt256FromBig(v *big.Int) Int256 {
	x, ok := int256FromBig(v)
	if !ok {
		panic("fat.MustInt256FromBig: out of range")
	}
	return x
}

// --- getters --------------------------------------------

// IsZero reports whether x == 0
func (x Int256) IsZero() bool {
	return x.L0|x.L1|x.L2|x.L3 == 0
}

// Sign returns -1 for x < 0, 0 for x == 0 and 1 for x > 0
func (x Int256) Sign() int {
	// -1 from the sign bit, or 1 when nonzero
	n := x.L0 | x.L1 | x.L2 | x.L3
	return int(int64(x.L3)>>63) | int((n|-n)>>63)
}

// IsNegative reports whether x < 0
func (x Int256) IsNegative() bool {
	return x.L3>>63 == 1
}

// IsPositive reports whether x > 0
func (x Int256) IsPositive() bool {
	// the top bit of n|-n is set when n != 0
	n := x.L0 | x.L1 | x.L2 | x.L3
	return int64((n|-n)&^x.L3) < 0
}

// IsInt64 reports whether x fits in an int64
func (x Int256) IsInt64() bool {
	// every upper limb is the sign extension of L0
	m := uint64(int64(x.L0) >> 63)
	return (x.L1^m)|(x.L2^m)|(x.L3^m) == 0
}

// Int64 returns x as an int64, it panics when x does not fit
func (x Int256) Int64() int64 {
	if !x.IsInt64() {
		panic("fat.Int256.Int64: out of range")
	}
	return int64(x.L0)
}

// Int64o returns Int64u and whether x does not fit
func (x Int256) Int64o() (r int64, overflow bool) {
	return x.Int64u(), !x.IsInt64()
}

// Int64u returns the low 64 bits of x as a Go conversion does
func (x Int256) Int64u() int64 {
	return int64(x.L0)
}

// IsInt128 reports whether x fits in an Int128
func (x Int256) IsInt128() bool {
	m := uint64(int64(x.L1) >> 63)
	return (x.L2^m)|(x.L3^m) == 0
}

// Int128 returns x as an Int128, it panics when x does not fit
func (x Int256) Int128() Int128 {
	if !x.IsInt128() {
		panic("fat.Int256.Int128: out of range")
	}
	return Int128{x.L0, x.L1}
}

// Int128o returns Int128u and whether x does not fit
func (x Int256) Int128o() (r Int128, overflow bool) {
	return x.Int128u(), !x.IsInt128()
}

// Int128u returns the low 128 bits of x as a Go conversion does
func (x Int256) Int128u() Int128 {
	return Int128{x.L0, x.L1}
}

// IsUint64 reports whether x fits in a uint64
func (x Int256) IsUint64() bool {
	return x.L1|x.L2|x.L3 == 0
}

// Uint64 returns x as a uint64, it panics when x does not fit
func (x Int256) Uint64() uint64 {
	if !x.IsUint64() {
		if x.IsNegative() {
			panic("fat.Int256.Uint64: negative value")
		}
		panic("fat.Int256.Uint64: out of range")
	}
	return x.L0
}

// Uint64o returns Uint64u and whether x does not fit
func (x Int256) Uint64o() (r uint64, overflow bool) {
	return x.Uint64u(), !x.IsUint64()
}

// Uint64u returns the low 64 bits of x as a Go conversion does
func (x Int256) Uint64u() uint64 {
	return x.L0
}

// IsUint128 reports whether x fits in a Uint128
func (x Int256) IsUint128() bool {
	return x.L2|x.L3 == 0
}

// Uint128 returns x as a Uint128, it panics when x does not fit
func (x Int256) Uint128() Uint128 {
	if !x.IsUint128() {
		if x.IsNegative() {
			panic("fat.Int256.Uint128: negative value")
		}
		panic("fat.Int256.Uint128: out of range")
	}
	return Uint128{x.L0, x.L1}
}

// Uint128o returns Uint128u and whether x does not fit
func (x Int256) Uint128o() (r Uint128, overflow bool) {
	return x.Uint128u(), !x.IsUint128()
}

// Uint128u returns the low 128 bits of x as a Go conversion does
func (x Int256) Uint128u() Uint128 {
	return Uint128{x.L0, x.L1}
}

// IsUint256 reports whether x fits in a Uint256, that is x >= 0
func (x Int256) IsUint256() bool {
	return int64(x.L3) >= 0
}

// Uint256 returns x as a Uint256, it panics for x < 0
func (x Int256) Uint256() Uint256 {
	if !x.IsUint256() {
		panic("fat.Int256.Uint256: negative value")
	}
	return Uint256(x)
}

// Uint256o returns Uint256u and whether x < 0
func (x Int256) Uint256o() (r Uint256, overflow bool) {
	return x.Uint256u(), !x.IsUint256()
}

// Uint256u returns the bits of x as a Go conversion does
func (x Int256) Uint256u() Uint256 {
	return Uint256(x)
}

// Big returns x as a new big.Int
func (x Int256) Big() *big.Int {
	// the Int and its words in one allocation
	p := new(struct {
		b big.Int
		w [4 * limbWords]big.Word
	})
	// PutBig inlined by hand, the words need no cap check
	u := x.uabs()
	u.putWords(p.w[:])
	nz := (u.L0|-u.L0)>>63 | (u.L1|-u.L1)>>63<<1 | (u.L2|-u.L2)>>63<<2 | (u.L3|-u.L3)>>63<<3
	p.b.SetBits(p.w[:bits.Len64(nz)*limbWords])
	if x.L3>>63 != 0 {
		p.b.Neg(&p.b)
	}
	return &p.b
}

// PutBig sets b to x and returns it, reusing the words of b if they fit
func (x Int256) PutBig(b *big.Int) *big.Int {
	// magnitude bits are right even for MinInt256
	u := x.uabs()
	words := b.Bits()
	if cap(words) < 4*limbWords {
		words = make([]big.Word, 4*limbWords)
	}
	w := words[:4*limbWords]
	u.putWords(w)
	// the exact length, so SetBits has at most one zero 32-bit word to trim.
	// BitLen would branch here
	nz := (u.L0|-u.L0)>>63 | (u.L1|-u.L1)>>63<<1 | (u.L2|-u.L2)>>63<<2 | (u.L3|-u.L3)>>63<<3
	b.SetBits(w[:bits.Len64(nz)*limbWords])
	if x.L3>>63 != 0 {
		b.Neg(b)
	}
	return b
}

// --- sign -----------------------------------------------

// Neg returns -x, it panics for MinInt256
func (x Int256) Neg() Int256 {
	r0, c := bits.Sub64(0, x.L0, 0)
	r1, c := bits.Sub64(0, x.L1, c)
	r2, c := bits.Sub64(0, x.L2, c)
	r3, _ := bits.Sub64(0, x.L3, c)
	// only MinInt256 stays negative
	if int64(x.L3&r3) < 0 {
		panic("fat.Int256.Neg: overflow")
	}
	return Int256{r0, r1, r2, r3}
}

// Abs returns |x|, it panics for MinInt256
func (x Int256) Abs() Int256 {
	// UnsignedAbs written out keeps the test within the inline budget.
	// A call spills the caller's values, slower in our benchmarks on Graviton
	var r Int256
	var b uint64
	r.L0, b = bits.Sub64(0, x.L0, 0)
	r.L1, b = bits.Sub64(0, x.L1, b)
	r.L2, b = bits.Sub64(0, x.L2, b)
	r.L3, _ = bits.Sub64(0, x.L3, b)
	if int64(x.L3) >= 0 {
		r = x
	}
	if int64(r.L3) < 0 {
		panic("fat.Int256.Abs: overflow")
	}
	return r
}

// UnsignedAbs returns |x| as a Uint256, which holds 2^255 for MinInt256
func (x Int256) UnsignedAbs() Uint256 {
	// -x picked by conditional moves, faster than uabs by masks in our benchmarks
	r0, b := bits.Sub64(0, x.L0, 0)
	r1, b := bits.Sub64(0, x.L1, b)
	r2, b := bits.Sub64(0, x.L2, b)
	r3, _ := bits.Sub64(0, x.L3, b)
	if int64(x.L3) >= 0 {
		r0, r1, r2, r3 = x.L0, x.L1, x.L2, x.L3
	}
	return Uint256{r0, r1, r2, r3}
}

// uabs is UnsignedAbs by masks. Big and PutBig take it as |x| sets the length of
// the words there, and Go turns a select feeding a length or an address into a branch.
// div128, divMod128 and mod128 take it for x as the result reuses the sign mask,
// 1.2..1.6% faster there in our benchmarks
func (x Int256) uabs() Uint256 {
	// (x ^ m) - m, m as a 256-bit value is 0 or -1
	m := uint64(int64(x.L3) >> 63)
	r0, b := bits.Sub64(x.L0^m, m, 0)
	r1, b := bits.Sub64(x.L1^m, m, b)
	r2, b := bits.Sub64(x.L2^m, m, b)
	r3, _ := bits.Sub64(x.L3^m, m, b)
	return Uint256{r0, r1, r2, r3}
}

// --- comparison -----------------------------------------

// flipping the sign bits maps signed order onto unsigned order

// Cmp returns -1 for x < y, 0 for x = y and 1 for x > y
func (x Int256) Cmp(y Int256) int {
	// Uint256.Cmp with the sign bits flipped.
	// One borrow chain is faster than a test by XOR in our benchmarks
	d0, carry := bits.Sub64(x.L0, y.L0, 0)
	d1, carry := bits.Sub64(x.L1, y.L1, carry)
	d2, carry := bits.Sub64(x.L2, y.L2, carry)
	d3, carry := bits.Sub64(x.L3^(1<<63), y.L3^(1<<63), carry)
	// -1 on borrow, else 1 if any difference
	return int(-carry | min(d0|d1|d2|d3, 1))
}

// LessThan reports whether x < y
func (x Int256) LessThan(y Int256) bool {
	_, carry := bits.Sub64(x.L0, y.L0, 0)
	_, carry = bits.Sub64(x.L1, y.L1, carry)
	_, carry = bits.Sub64(x.L2, y.L2, carry)
	_, carry = bits.Sub64(x.L3^(1<<63), y.L3^(1<<63), carry)
	return carry != 0
}

// LessOrEqual reports whether x <= y
func (x Int256) LessOrEqual(y Int256) bool {
	_, carry := bits.Sub64(y.L0, x.L0, 0)
	_, carry = bits.Sub64(y.L1, x.L1, carry)
	_, carry = bits.Sub64(y.L2, x.L2, carry)
	_, carry = bits.Sub64(y.L3^(1<<63), x.L3^(1<<63), carry)
	return carry == 0
}

// GreaterThan reports whether x > y
func (x Int256) GreaterThan(y Int256) bool {
	_, carry := bits.Sub64(y.L0, x.L0, 0)
	_, carry = bits.Sub64(y.L1, x.L1, carry)
	_, carry = bits.Sub64(y.L2, x.L2, carry)
	_, carry = bits.Sub64(y.L3^(1<<63), x.L3^(1<<63), carry)
	return carry != 0
}

// GreaterOrEqual reports whether x >= y
func (x Int256) GreaterOrEqual(y Int256) bool {
	_, carry := bits.Sub64(x.L0, y.L0, 0)
	_, carry = bits.Sub64(x.L1, y.L1, carry)
	_, carry = bits.Sub64(x.L2, y.L2, carry)
	_, carry = bits.Sub64(x.L3^(1<<63), y.L3^(1<<63), carry)
	return carry == 0
}

// Max returns the larger of x and y, as the builtin max
func (x Int256) Max(y Int256) Int256 {
	// the borrow chain of LessThan written out picks by CSEL,
	// a select by masks does not inline
	_, carry := bits.Sub64(x.L0, y.L0, 0)
	_, carry = bits.Sub64(x.L1, y.L1, carry)
	_, carry = bits.Sub64(x.L2, y.L2, carry)
	_, carry = bits.Sub64(x.L3^(1<<63), y.L3^(1<<63), carry)
	if carry != 0 {
		x = y
	}
	return x
}

// Min returns the smaller of x and y, as the builtin min
func (x Int256) Min(y Int256) Int256 {
	_, carry := bits.Sub64(x.L0, y.L0, 0)
	_, carry = bits.Sub64(x.L1, y.L1, carry)
	_, carry = bits.Sub64(x.L2, y.L2, carry)
	_, carry = bits.Sub64(x.L3^(1<<63), y.L3^(1<<63), carry)
	if carry == 0 {
		x = y
	}
	return x
}

// Cmp128 returns -1 for x < y, 0 for x = y and 1 for x > y
func (x Int256) Cmp128(y Int128) int {
	ext := -(y.L1 >> 63)
	// differences are folded into d to stay within the inline budget
	d, carry := bits.Sub64(x.L0, y.L0, 0)
	t, carry := bits.Sub64(x.L1, y.L1, carry)
	d |= t
	t, carry = bits.Sub64(x.L2, ext, carry)
	d |= t
	t, carry = bits.Sub64(x.L3^(1<<63), ext^(1<<63), carry)
	return int(-carry | min(d|t, 1))
}

// Cmp64 returns -1 for x < y, 0 for x = y and 1 for x > y
func (x Int256) Cmp64(y int64) int {
	ext := uint64(y >> 63)
	d0, carry := bits.Sub64(x.L0, uint64(y), 0)
	d1, carry := bits.Sub64(x.L1, ext, carry)
	d2, carry := bits.Sub64(x.L2, ext, carry)
	d3, carry := bits.Sub64(x.L3^(1<<63), ext^(1<<63), carry)
	return int(-carry | min(d0|d1|d2|d3, 1))
}

// CmpBig returns -1 for x < y, 0 for x = y and 1 for x > y
func (x Int256) CmpBig(y *big.Int) int {
	w := y.Bits()
	n := len(w)
	if bits.UintSize == 64 && uint(n-1) < 4 {
		l0 := uint64(w[0])
		// m>>(m>>1) is min(1, m), m&^(m>>1) is min(2, m): no bounds checks.
		// Masks zero the words past n
		m := uint(n - 1)
		l1 := uint64(w[m>>(m>>1)]) & uint64(int64(1-n)>>63)
		l2 := uint64(w[m&^(m>>1)]) & uint64(int64(2-n)>>63)
		l3 := uint64(w[m]) & uint64(int64(3-n)>>63)
		// all ones for a negative y, Sign would branch
		sm := bigNeg(y, l0)
		// x - y over 5 limbs as x - (l ^ sm) - -sm, no range check of y
		d0, b := bits.Sub64(x.L0, l0^sm, -sm)
		d1, b := bits.Sub64(x.L1, l1^sm, b)
		d2, b := bits.Sub64(x.L2, l2^sm, b)
		d3, b := bits.Sub64(x.L3, l3^sm, b)
		d4, _ := bits.Sub64(uint64(int64(x.L3)>>63), sm, b)
		// -1 from the sign, else 1 if any difference
		return int(int64(d4)>>63) | int(min(d0|d1|d2|d3|d4, 1))
	}
	// zero, past 256 bits, or 32-bit words: shifted in from the top word.
	// Written out, as a call would cost a stack frame
	var l0, l1, l2, l3 uint64
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
	// all ones for a negative y, Sign would branch
	sm := bigNeg(y, l0)
	// x - y over 5 limbs as x - (l ^ sm) - -sm, no range check of y
	d0, b := bits.Sub64(x.L0, l0^sm, -sm)
	d1, b := bits.Sub64(x.L1, l1^sm, b)
	d2, b := bits.Sub64(x.L2, l2^sm, b)
	d3, b := bits.Sub64(x.L3, l3^sm, b)
	d4, _ := bits.Sub64(uint64(int64(x.L3)>>63), sm, b)
	// -1 from the sign, else 1 if any difference
	return int(int64(d4)>>63) | int(min(d0|d1|d2|d3|d4, 1))
}

// --- add ------------------------------------------------

// signed overflow: the operands share a sign that the result lacks.
// The top limb takes its carry by ADC, a plain + adds a CSET and an ADD.
// The sign bit is tested as int64 < 0, a TST where >= 1<<63 loads a constant

// Add returns x + y, it panics on overflow
func (x Int256) Add(y Int256) Int256 {
	r0, c := bits.Add64(x.L0, y.L0, 0)
	r1, c := bits.Add64(x.L1, y.L1, c)
	r2, c := bits.Add64(x.L2, y.L2, c)
	r3, _ := bits.Add64(x.L3, y.L3, c)
	if int64((x.L3^r3)&(y.L3^r3)) < 0 {
		panic("fat.Int256.Add: overflow")
	}
	return Int256{r0, r1, r2, r3}
}

// Addo returns Addu and whether x + y overflowed
func (x Int256) Addo(y Int256) (r Int256, overflow bool) {
	r0, c := bits.Add64(x.L0, y.L0, 0)
	r1, c := bits.Add64(x.L1, y.L1, c)
	r2, c := bits.Add64(x.L2, y.L2, c)
	r3, _ := bits.Add64(x.L3, y.L3, c)
	return Int256{r0, r1, r2, r3}, int64((x.L3^r3)&(y.L3^r3)) < 0
}

// Addu returns x + y wrapped around as on a Go integer
func (x Int256) Addu(y Int256) Int256 {
	var carry uint64
	x.L0, carry = bits.Add64(x.L0, y.L0, 0)
	x.L1, carry = bits.Add64(x.L1, y.L1, carry)
	x.L2, carry = bits.Add64(x.L2, y.L2, carry)
	x.L3, _ = bits.Add64(x.L3, y.L3, carry)
	return x
}

// Add128 returns x + y, it panics on overflow
func (x Int256) Add128(y Int128) Int256 {
	// r declared ahead keeps the sum within the inline budget.
	// A call spills the caller's values, slower in our benchmarks on Graviton
	ext := -(y.L1 >> 63)
	var r Int256
	var c uint64
	r.L0, c = bits.Add64(x.L0, y.L0, 0)
	r.L1, c = bits.Add64(x.L1, y.L1, c)
	r.L2, c = bits.Add64(x.L2, ext, c)
	r.L3, _ = bits.Add64(x.L3, ext, c)
	if int64((x.L3^r.L3)&(ext^r.L3)) < 0 {
		panic("fat.Int256.Add128: overflow")
	}
	return r
}

// Add128o returns Add128u and whether x + y overflowed
func (x Int256) Add128o(y Int128) (r Int256, overflow bool) {
	ext := -(y.L1 >> 63)
	r0, c := bits.Add64(x.L0, y.L0, 0)
	r1, c := bits.Add64(x.L1, y.L1, c)
	r2, c := bits.Add64(x.L2, ext, c)
	r3, _ := bits.Add64(x.L3, ext, c)
	return Int256{r0, r1, r2, r3}, int64((x.L3^r3)&(ext^r3)) < 0
}

// Add128u returns x + y wrapped around as on a Go integer
func (x Int256) Add128u(y Int128) Int256 {
	ext := uint64(int64(y.L1) >> 63)
	var carry uint64
	x.L0, carry = bits.Add64(x.L0, y.L0, 0)
	x.L1, carry = bits.Add64(x.L1, y.L1, carry)
	x.L2, carry = bits.Add64(x.L2, ext, carry)
	x.L3, _ = bits.Add64(x.L3, ext, carry)
	return x
}

// Add64 returns x + y, it panics on overflow
func (x Int256) Add64(y int64) Int256 {
	// no int64 copy of y, it would cost the inline budget
	ext := uint64(y >> 63)
	r0, c := bits.Add64(x.L0, uint64(y), 0)
	r1, c := bits.Add64(x.L1, ext, c)
	r2, c := bits.Add64(x.L2, ext, c)
	r3, _ := bits.Add64(x.L3, ext, c)
	if int64((x.L3^r3)&(ext^r3)) < 0 {
		panic("fat.Int256.Add64: overflow")
	}
	return Int256{r0, r1, r2, r3}
}

// Add64o returns Add64u and whether x + y overflowed
func (x Int256) Add64o(y int64) (r Int256, overflow bool) {
	// no int64 copy of y, it would cost the inline budget
	ext := uint64(y >> 63)
	r0, c := bits.Add64(x.L0, uint64(y), 0)
	r1, c := bits.Add64(x.L1, ext, c)
	r2, c := bits.Add64(x.L2, ext, c)
	r3, _ := bits.Add64(x.L3, ext, c)
	return Int256{r0, r1, r2, r3}, int64((x.L3^r3)&(ext^r3)) < 0
}

// Add64u returns x + y wrapped around as on a Go integer
func (x Int256) Add64u(y int64) Int256 {
	ext := uint64(y >> 63)
	var carry uint64
	x.L0, carry = bits.Add64(x.L0, uint64(y), 0)
	x.L1, carry = bits.Add64(x.L1, ext, carry)
	x.L2, carry = bits.Add64(x.L2, ext, carry)
	x.L3, _ = bits.Add64(x.L3, ext, carry)
	return x
}

// --- sub ------------------------------------------------

// signed overflow: the operands differ in sign and the result took that of y

// Sub returns x - y, it panics on overflow
func (x Int256) Sub(y Int256) Int256 {
	r0, b := bits.Sub64(x.L0, y.L0, 0)
	r1, b := bits.Sub64(x.L1, y.L1, b)
	r2, b := bits.Sub64(x.L2, y.L2, b)
	r3, _ := bits.Sub64(x.L3, y.L3, b)
	if int64((x.L3^y.L3)&^(r3^y.L3)) < 0 {
		panic("fat.Int256.Sub: overflow")
	}
	return Int256{r0, r1, r2, r3}
}

// Subo returns Subu and whether x - y overflowed
func (x Int256) Subo(y Int256) (r Int256, overflow bool) {
	r0, b := bits.Sub64(x.L0, y.L0, 0)
	r1, b := bits.Sub64(x.L1, y.L1, b)
	r2, b := bits.Sub64(x.L2, y.L2, b)
	r3, _ := bits.Sub64(x.L3, y.L3, b)
	return Int256{r0, r1, r2, r3}, int64((x.L3^y.L3)&^(r3^y.L3)) < 0
}

// Subu returns x - y wrapped around as on a Go integer
func (x Int256) Subu(y Int256) Int256 {
	var carry uint64
	x.L0, carry = bits.Sub64(x.L0, y.L0, 0)
	x.L1, carry = bits.Sub64(x.L1, y.L1, carry)
	x.L2, carry = bits.Sub64(x.L2, y.L2, carry)
	x.L3, _ = bits.Sub64(x.L3, y.L3, carry)
	return x
}

// Sub128 returns x - y, it panics on overflow
func (x Int256) Sub128(y Int128) Int256 {
	// r declared ahead keeps it within the inline budget, as in Add128
	ext := -(y.L1 >> 63)
	var r Int256
	var b uint64
	r.L0, b = bits.Sub64(x.L0, y.L0, 0)
	r.L1, b = bits.Sub64(x.L1, y.L1, b)
	r.L2, b = bits.Sub64(x.L2, ext, b)
	r.L3, _ = bits.Sub64(x.L3, ext, b)
	if int64((x.L3^ext)&^(r.L3^ext)) < 0 {
		panic("fat.Int256.Sub128: overflow")
	}
	return r
}

// Sub128o returns Sub128u and whether x - y overflowed
func (x Int256) Sub128o(y Int128) (r Int256, overflow bool) {
	ext := -(y.L1 >> 63)
	r0, b := bits.Sub64(x.L0, y.L0, 0)
	r1, b := bits.Sub64(x.L1, y.L1, b)
	r2, b := bits.Sub64(x.L2, ext, b)
	r3, _ := bits.Sub64(x.L3, ext, b)
	return Int256{r0, r1, r2, r3}, int64((x.L3^ext)&^(r3^ext)) < 0
}

// Sub128u returns x - y wrapped around as on a Go integer
func (x Int256) Sub128u(y Int128) Int256 {
	ext := uint64(int64(y.L1) >> 63)
	var carry uint64
	x.L0, carry = bits.Sub64(x.L0, y.L0, 0)
	x.L1, carry = bits.Sub64(x.L1, y.L1, carry)
	x.L2, carry = bits.Sub64(x.L2, ext, carry)
	x.L3, _ = bits.Sub64(x.L3, ext, carry)
	return x
}

// Sub64 returns x - y, it panics on overflow
func (x Int256) Sub64(y int64) Int256 {
	// no int64 copy of y, it would cost the inline budget
	ext := uint64(y >> 63)
	r0, b := bits.Sub64(x.L0, uint64(y), 0)
	r1, b := bits.Sub64(x.L1, ext, b)
	r2, b := bits.Sub64(x.L2, ext, b)
	r3, _ := bits.Sub64(x.L3, ext, b)
	if int64((x.L3^ext)&^(r3^ext)) < 0 {
		panic("fat.Int256.Sub64: overflow")
	}
	return Int256{r0, r1, r2, r3}
}

// Sub64o returns Sub64u and whether x - y overflowed
func (x Int256) Sub64o(y int64) (r Int256, overflow bool) {
	// no int64 copy of y, it would cost the inline budget
	ext := uint64(y >> 63)
	r0, b := bits.Sub64(x.L0, uint64(y), 0)
	r1, b := bits.Sub64(x.L1, ext, b)
	r2, b := bits.Sub64(x.L2, ext, b)
	r3, _ := bits.Sub64(x.L3, ext, b)
	return Int256{r0, r1, r2, r3}, int64((x.L3^ext)&^(r3^ext)) < 0
}

// Sub64u returns x - y wrapped around as on a Go integer
func (x Int256) Sub64u(y int64) Int256 {
	ext := uint64(y >> 63)
	var carry uint64
	x.L0, carry = bits.Sub64(x.L0, uint64(y), 0)
	x.L1, carry = bits.Sub64(x.L1, ext, carry)
	x.L2, carry = bits.Sub64(x.L2, ext, carry)
	x.L3, _ = bits.Sub64(x.L3, ext, carry)
	return x
}
