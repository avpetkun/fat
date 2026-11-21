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

// IsNegative reports whether x < 0
func (x Int256) IsNegative() bool {
	return x.L3>>63 == 1
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
