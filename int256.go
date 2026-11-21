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

// withSign returns x for m = 0 and -x wrapped around for m = ^0.
// Any other m gives a wrong result
func (x Uint256) withSign(m uint64) Int256 {
	r0, c := bits.Add64(x.L0^m, m&1, 0)
	r1, c := bits.Add64(x.L1^m, 0, c)
	r2, c := bits.Add64(x.L2^m, 0, c)
	return Int256{r0, r1, r2, x.L3 ^ m + c}
}

// withSign returns x for m = 0 and -x wrapped around for m = ^0.
// Any other m gives a wrong result
func (x Uint128) withSign(m uint64) Int128 {
	r0, c := bits.Add64(x.L0^m, m&1, 0)
	return Int128{r0, x.L1 ^ m + c}
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

// --- bits -----------------------------------------------

// BitLen returns the length of |x| in bits, 0 for zero
func (x Int256) BitLen() int {
	// magnitude bits are right even for MinInt256
	return x.UnsignedAbs().BitLen()
}

// Lsh returns x << n, it panics for n < 0
func (x Int256) Lsh(n int) Int256 {
	if n < 0 {
		panic("fat.Int256.Lsh: negative shift")
	}
	// the plain lsh is too big to inline
	return x.lsh(uint64(n))
}

// lsh is a copy of Uint256.lsh, a left shift is the same for both signs
func (x Int256) lsh(n uint64) Int256 {
	// bits within limbs, then whole limbs by conditional moves, no branch on n
	s, t := n&63, ^n&63
	// >>1>>t is >>(64-s) that gives 0 at s = 0
	r3 := x.L3<<s | x.L2>>1>>t
	r2 := x.L2<<s | x.L1>>1>>t
	r1 := x.L1<<s | x.L0>>1>>t
	r0 := x.L0 << s
	if n&64 != 0 {
		r3, r2, r1, r0 = r2, r1, r0, 0
	}
	// n > 127 for n&128 saves an AND, a wider n zeros all the limbs below
	if n > 127 {
		r3, r2, r1, r0 = r1, r0, 0, 0
	}
	if n > 255 {
		r3, r2 = 0, 0
	}
	return Int256{r0, r1, r2, r3}
}

// Rsh returns x >> n rounded toward negative infinity, it panics for n < 0
func (x Int256) Rsh(n int) Int256 {
	if n < 0 {
		panic("fat.Int256.Rsh: negative shift")
	}
	// the plain rsh is too big to inline
	return x.rsh(uint64(n))
}

// rsh is Rsh for any n: Uint256.rsh with the sign as fill
func (x Int256) rsh(n uint64) Int256 {
	// past 255 bits only the sign is left
	m := uint64(int64(x.L3) >> 63)
	s, t := n&63, ^n&63
	r0 := x.L0>>s | x.L1<<1<<t
	r1 := x.L1>>s | x.L2<<1<<t
	r2 := x.L2>>s | x.L3<<1<<t
	r3 := uint64(int64(x.L3) >> s)
	if n&64 != 0 {
		r0, r1, r2, r3 = r1, r2, r3, m
	}
	// n > 127 for n&128 saves an AND, a wider n fills all the limbs below
	if n > 127 {
		r0, r1, r2, r3 = r2, r3, m, m
	}
	if n > 255 {
		r0, r1 = m, m
	}
	return Int256{r0, r1, r2, r3}
}

// Xor returns x ^ y
func (x Int256) Xor(y Int256) Int256 {
	return Int256(Uint256(x).Xor(Uint256(y)))
}

// And returns x & y
func (x Int256) And(y Int256) Int256 {
	return Int256(Uint256(x).And(Uint256(y)))
}

// AndNot returns x &^ y
func (x Int256) AndNot(y Int256) Int256 {
	return Int256(Uint256(x).AndNot(Uint256(y)))
}

// Or returns x | y
func (x Int256) Or(y Int256) Int256 {
	return Int256(Uint256(x).Or(Uint256(y)))
}

// Not returns ^x
func (x Int256) Not() Int256 {
	return Int256(Uint256(x).Not())
}

// Bit returns bit i of x in two's complement, the sign bit past 255.
// It panics for i < 0
func (x Int256) Bit(i int) uint {
	if i < 0 {
		panic("fat.Int256.Bit: negative index")
	}
	// past bit 255 every bit is the sign
	u := min(uint64(i), 255)
	// conditional moves keep x in registers
	lo, hi := x.L0, x.L2
	if u&64 != 0 {
		lo, hi = x.L1, x.L3
	}
	// u > 127 for u&128 saves an AND
	if u > 127 {
		lo = hi
	}
	return uint(lo >> (u & 63) & 1)
}

// WithBit returns x with bit i set to bit, a bit past 255 is dropped.
// It panics for i < 0 and for bit > 1
func (x Int256) WithBit(i int, bit uint) Int256 {
	if i < 0 {
		panic("fat.Int256.WithBit: negative index")
	}
	if bit > 1 {
		panic("fat.Int256.WithBit: bit not 0 or 1")
	}
	u := uint64(i)
	// a shift past 63 is 0: only the limb of bit i has a mask, none past 255
	b := -uint64(bit)
	x.L0 ^= (x.L0 ^ b) & (1 << u)
	x.L1 ^= (x.L1 ^ b) & (1 << (u - 64))
	x.L2 ^= (x.L2 ^ b) & (1 << (u - 128))
	x.L3 ^= (x.L3 ^ b) & (1 << (u - 192))
	return x
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

// --- mul ------------------------------------------------

// Mul returns x * y, it panics on overflow
func (x Int256) Mul(y Int256) Int256 {
	// 14 multiplies in place of the 20 of Mulo.
	// Faster in our benchmarks on Graviton, where the multiplier is the bound
	r, c := x.mulFit(y)
	if c {
		panic("fat.Int256.Mul: overflow")
	}
	return r
}

// mulFit is Mulo for a caller that drops r on overflow, r is wrong then.
// It is Uint256.mulFit on |x| and |y| written out
func (x Int256) mulFit(y Int256) (r Int256, overflow bool) {
	m := uint64(int64(x.L3^y.L3) >> 63)
	a, v := x.UnsignedAbs(), y.UnsignedAbs()
	ah, vh := a.L2|a.L3, v.L2|v.L3
	x0, x1, x2, x3, y0, y1 := a.L0, a.L1, a.L2, a.L3, v.L0, v.L1
	if vh != 0 {
		x0, x1, x2, x3, y0, y1 = v.L0, v.L1, v.L2, v.L3, a.L0, a.L1
	}
	h0, p0 := bits.Mul64(x0, y0)
	h1, p1 := bits.Mul64(x1, y0)
	h2, p2 := bits.Mul64(x2, y0)
	h3, p3 := bits.Mul64(x3, y0)
	var c uint64
	p1, c = bits.Add64(p1, h0, 0)
	p2, c = bits.Add64(p2, h1, c)
	p3, c = bits.Add64(p3, h2, c)
	p4, _ := bits.Add64(h3, 0, c)
	g0, q1 := bits.Mul64(x0, y1)
	g1, q2 := bits.Mul64(x1, y1)
	g2, q3 := bits.Mul64(x2, y1)
	q2, c = bits.Add64(q2, g0, 0)
	q3, c = bits.Add64(q3, g1, c)
	q4, _ := bits.Add64(g2, 0, c)
	r1, c := bits.Add64(p1, q1, 0)
	r2, c := bits.Add64(p2, q2, c)
	r3, c := bits.Add64(p3, q3, c)
	r4, _ := bits.Add64(p4, q4, c)
	over := r4 | min(x3, y1) | min(ah, vh)
	// the sign as in Mulo
	r0, b := bits.Sub64(p0^m, m, 0)
	r1, b = bits.Sub64(r1^m, m, b)
	r2, b = bits.Sub64(r2^m, m, b)
	r3, b = bits.Sub64(r3^m, m, b)
	over |= (r3 ^ m) >> 63 &^ (m &^ b)
	return Int256{r0, r1, r2, r3}, over != 0
}

// Mulo returns Mulu and whether x * y overflowed
func (x Int256) Mulo(y Int256) (r Int256, overflow bool) {
	m := uint64(int64(x.L3^y.L3) >> 63)
	a, v := x.UnsignedAbs(), y.UnsignedAbs()
	// Uint256.Mulo on |x| and |y|: rows cut at limb 4.
	// The products past limb 4 are caught by the min test below
	h0, p0 := bits.Mul64(a.L0, v.L0)
	h1, p1 := bits.Mul64(a.L1, v.L0)
	h2, p2 := bits.Mul64(a.L2, v.L0)
	h3, p3 := bits.Mul64(a.L3, v.L0)
	var c uint64
	p1, c = bits.Add64(p1, h0, 0)
	p2, c = bits.Add64(p2, h1, c)
	p3, c = bits.Add64(p3, h2, c)
	p4, _ := bits.Add64(h3, 0, c)
	g0, q1 := bits.Mul64(a.L0, v.L1)
	g1, q2 := bits.Mul64(a.L1, v.L1)
	g2, q3 := bits.Mul64(a.L2, v.L1)
	q2, c = bits.Add64(q2, g0, 0)
	q3, c = bits.Add64(q3, g1, c)
	q4, _ := bits.Add64(g2, 0, c)
	f0, s2 := bits.Mul64(a.L0, v.L2)
	f1, s3 := bits.Mul64(a.L1, v.L2)
	s3, c = bits.Add64(s3, f0, 0)
	s4, _ := bits.Add64(f1, 0, c)
	e0, t3 := bits.Mul64(a.L0, v.L3)
	r1, c := bits.Add64(p1, q1, 0)
	r2, c := bits.Add64(p2, q2, c)
	r3, c := bits.Add64(p3, q3, c)
	r4, _ := bits.Add64(p4, q4, c)
	r2, c = bits.Add64(r2, s2, 0)
	r3, c = bits.Add64(r3, s3, c)
	r4, _ = bits.Add64(r4, s4, c)
	r3, c = bits.Add64(r3, t3, 0)
	r4, _ = bits.Add64(r4, e0, c)
	// min != 0 is a nonzero product past limb 4, else r4 is exactly limb 4.
	// The carries ride ADC into r4, an OR of each would cost a CSET more
	v23 := v.L2 | v.L3
	over := r4 | min(a.L3, v.L1|v23) | min(a.L2, v23) | min(a.L1, v.L3)
	// (r ^ m) - m, m as a 256-bit value is 0 or -1
	r0, b := bits.Sub64(p0^m, m, 0)
	r1, b = bits.Sub64(r1^m, m, b)
	r2, b = bits.Sub64(r2^m, m, b)
	r3, b = bits.Sub64(r3^m, m, b)
	// it fits if the sign is m, or for r = 0: the only negation with no borrow
	over |= (r3 ^ m) >> 63 &^ (m &^ b)
	return Int256{r0, r1, r2, r3}, over != 0
}

// Mulu returns x * y wrapped around as on a Go integer
func (x Int256) Mulu(y Int256) Int256 {
	return Int256(Uint256(x).Mulu(Uint256(y)))
}

// Mule returns the 512-bit product x * y as hi * 2^256 + lo, lo unsigned
func (x Int256) Mule(y Int256) (hi Int256, lo Uint256) {
	// Uint256.Mule written out, faster than its call in our benchmarks
	// x * y.L0
	h0, r0 := bits.Mul64(x.L0, y.L0)
	h1, p1 := bits.Mul64(x.L1, y.L0)
	h2, p2 := bits.Mul64(x.L2, y.L0)
	h3, p3 := bits.Mul64(x.L3, y.L0)
	p1, c := bits.Add64(p1, h0, 0)
	p2, c = bits.Add64(p2, h1, c)
	p3, c = bits.Add64(p3, h2, c)
	p4, _ := bits.Add64(h3, 0, c)
	// x * y.L1
	h0, q1 := bits.Mul64(x.L0, y.L1)
	h1, q2 := bits.Mul64(x.L1, y.L1)
	h2, q3 := bits.Mul64(x.L2, y.L1)
	h3, q4 := bits.Mul64(x.L3, y.L1)
	q2, c = bits.Add64(q2, h0, 0)
	q3, c = bits.Add64(q3, h1, c)
	q4, c = bits.Add64(q4, h2, c)
	p5, _ := bits.Add64(h3, 0, c)
	r1, c := bits.Add64(p1, q1, 0)
	p2, c = bits.Add64(p2, q2, c)
	p3, c = bits.Add64(p3, q3, c)
	p4, c = bits.Add64(p4, q4, c)
	p5, _ = bits.Add64(p5, 0, c)
	// x * y.L2
	h0, q2 = bits.Mul64(x.L0, y.L2)
	h1, q3 = bits.Mul64(x.L1, y.L2)
	h2, q4 = bits.Mul64(x.L2, y.L2)
	h3, q5 := bits.Mul64(x.L3, y.L2)
	q3, c = bits.Add64(q3, h0, 0)
	q4, c = bits.Add64(q4, h1, c)
	q5, c = bits.Add64(q5, h2, c)
	p6, _ := bits.Add64(h3, 0, c)
	r2, c := bits.Add64(p2, q2, 0)
	p3, c = bits.Add64(p3, q3, c)
	p4, c = bits.Add64(p4, q4, c)
	p5, c = bits.Add64(p5, q5, c)
	p6, _ = bits.Add64(p6, 0, c)
	// x * y.L3
	h0, q3 = bits.Mul64(x.L0, y.L3)
	h1, q4 = bits.Mul64(x.L1, y.L3)
	h2, q5 = bits.Mul64(x.L2, y.L3)
	h3, q6 := bits.Mul64(x.L3, y.L3)
	q4, c = bits.Add64(q4, h0, 0)
	q5, c = bits.Add64(q5, h1, c)
	q6, c = bits.Add64(q6, h2, c)
	p7, _ := bits.Add64(h3, 0, c)
	r3, c := bits.Add64(p3, q3, 0)
	p4, c = bits.Add64(p4, q4, c)
	p5, c = bits.Add64(p5, q5, c)
	p6, c = bits.Add64(p6, q6, c)
	p7, _ = bits.Add64(p7, 0, c)
	// signed = unsigned - 2^256*y if x < 0 - 2^256*x if y < 0.
	// The corrections are summed off the path of the product
	mx := uint64(int64(x.L3) >> 63)
	my := uint64(int64(y.L3) >> 63)
	k0, c := bits.Add64(y.L0&mx, x.L0&my, 0)
	k1, c := bits.Add64(y.L1&mx, x.L1&my, c)
	k2, c := bits.Add64(y.L2&mx, x.L2&my, c)
	k3, _ := bits.Add64(y.L3&mx, x.L3&my, c)
	p4, c = bits.Sub64(p4, k0, 0)
	p5, c = bits.Sub64(p5, k1, c)
	p6, c = bits.Sub64(p6, k2, c)
	p7, _ = bits.Sub64(p7, k3, c)
	return Int256{p4, p5, p6, p7}, Uint256{r0, r1, r2, r3}
}

// Mul128 returns x * y, it panics on overflow
func (x Int256) Mul128(y Int128) Int256 {
	r, c := x.Mul128o(y)
	if c {
		panic("fat.Int256.Mul128: overflow")
	}
	return r
}

// Mul128o returns Mul128u and whether x * y overflowed
func (x Int256) Mul128o(y Int128) (r Int256, overflow bool) {
	// two's complement product, no abs: x * y as unsigned = p0..p5
	// x * y.L0
	h0, r0 := bits.Mul64(x.L0, y.L0)
	h1, p1 := bits.Mul64(x.L1, y.L0)
	h2, p2 := bits.Mul64(x.L2, y.L0)
	h3, p3 := bits.Mul64(x.L3, y.L0)
	p1, c := bits.Add64(p1, h0, 0)
	p2, c = bits.Add64(p2, h1, c)
	p3, c = bits.Add64(p3, h2, c)
	p4, _ := bits.Add64(h3, 0, c)
	// x * y.L1, one limb up
	g0, q1 := bits.Mul64(x.L0, y.L1)
	g1, q2 := bits.Mul64(x.L1, y.L1)
	g2, q3 := bits.Mul64(x.L2, y.L1)
	g3, q4 := bits.Mul64(x.L3, y.L1)
	q2, c = bits.Add64(q2, g0, 0)
	q3, c = bits.Add64(q3, g1, c)
	q4, c = bits.Add64(q4, g2, c)
	q5, _ := bits.Add64(g3, 0, c)
	r1, c := bits.Add64(p1, q1, 0)
	r2, c := bits.Add64(p2, q2, c)
	r3, c := bits.Add64(p3, q3, c)
	p4, c = bits.Add64(p4, q4, c)
	p5, _ := bits.Add64(q5, 0, c)
	// signed = unsigned - 2^128*x if y < 0 - 2^256*y if x < 0
	my := -(y.L1 >> 63)
	mx := uint64(int64(x.L3) >> 63)
	r2, c = bits.Sub64(r2, x.L0&my, 0)
	r3, c = bits.Sub64(r3, x.L1&my, c)
	p4, c = bits.Sub64(p4, x.L2&my, c)
	p5, _ = bits.Sub64(p5, x.L3&my, c)
	p4, c = bits.Sub64(p4, y.L0&mx, 0)
	p5, _ = bits.Sub64(p5, y.L1&mx, c)
	// fits if limbs 4 and 5 only extend the sign of r3
	s := uint64(int64(r3) >> 63)
	return Int256{r0, r1, r2, r3}, (p4^s)|(p5^s) != 0
}

// Mul128u returns x * y wrapped around as on a Go integer
func (x Int256) Mul128u(y Int128) Int256 {
	// the unsigned product written out: a nested call costs more than the copy
	// top limbs take carries by ADC, a plain + costs a CSET and an ADD more
	// x * y.L0
	h0, r0 := bits.Mul64(x.L0, y.L0)
	h1, a1 := bits.Mul64(x.L1, y.L0)
	h2, a2 := bits.Mul64(x.L2, y.L0)
	a1, c := bits.Add64(a1, h0, 0)
	a2, c = bits.Add64(a2, h1, c)
	a3, _ := bits.Add64(x.L3*y.L0, h2, c)
	// x * y.L1, one limb up
	g0, b1 := bits.Mul64(x.L0, y.L1)
	g1, b2 := bits.Mul64(x.L1, y.L1)
	b2, c = bits.Add64(b2, g0, 0)
	b3, _ := bits.Add64(x.L2*y.L1, g1, c)
	r1, c := bits.Add64(a1, b1, 0)
	r2, c := bits.Add64(a2, b2, c)
	r3, _ := bits.Add64(a3, b3, c)
	// a negative y adds -2^128 * x
	m := -(y.L1 >> 63)
	r2, c = bits.Sub64(r2, x.L0&m, 0)
	r3, _ = bits.Sub64(r3, x.L1&m, c)
	return Int256{r0, r1, r2, r3}
}

// Mul64 returns x * y, it panics on overflow
func (x Int256) Mul64(y int64) Int256 {
	r, c := x.Mul64o(y)
	if c {
		panic("fat.Int256.Mul64: overflow")
	}
	return r
}

// Mul64o returns Mul64u and whether x * y overflowed
func (x Int256) Mul64o(y int64) (r Int256, overflow bool) {
	// two's complement product, no abs: x * y as unsigned = r0..r3, p4
	v := uint64(y)
	h0, r0 := bits.Mul64(x.L0, v)
	h1, r1 := bits.Mul64(x.L1, v)
	h2, r2 := bits.Mul64(x.L2, v)
	h3, r3 := bits.Mul64(x.L3, v)
	r1, c := bits.Add64(r1, h0, 0)
	r2, c = bits.Add64(r2, h1, c)
	r3, c = bits.Add64(r3, h2, c)
	p4, _ := bits.Add64(h3, 0, c)
	// signed = unsigned - 2^64*x if y < 0 - 2^256*y if x < 0
	my := uint64(y >> 63)
	r1, c = bits.Sub64(r1, x.L0&my, 0)
	r2, c = bits.Sub64(r2, x.L1&my, c)
	r3, c = bits.Sub64(r3, x.L2&my, c)
	p4, _ = bits.Sub64(p4, x.L3&my, c)
	p4 -= v & uint64(int64(x.L3)>>63)
	// fits if limb 4 only extends the sign of r3
	return Int256{r0, r1, r2, r3}, p4 != uint64(int64(r3)>>63)
}

// Mul64u returns x * y wrapped around as on a Go integer
func (x Int256) Mul64u(y int64) Int256 {
	// the unsigned product is unrolled here, limbs stay in locals
	v := uint64(y)
	h0, r0 := bits.Mul64(x.L0, v)
	h1, r1 := bits.Mul64(x.L1, v)
	h2, r2 := bits.Mul64(x.L2, v)
	r1, c := bits.Add64(r1, h0, 0)
	r2, c = bits.Add64(r2, h1, c)
	r3, _ := bits.Add64(x.L3*v, h2, c)
	// a negative y adds -2^64 * x
	m := uint64(y >> 63)
	r1, c = bits.Sub64(r1, x.L0&m, 0)
	r2, c = bits.Sub64(r2, x.L1&m, c)
	r3, _ = bits.Sub64(r3, x.L2&m, c)
	return Int256{r0, r1, r2, r3}
}
