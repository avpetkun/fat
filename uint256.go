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

// --- bits -----------------------------------------------

// BitLen returns the length of x in bits, 0 for zero
func (x Uint256) BitLen() int {
	// conditional moves instead of a branch on the top limb
	n := bits.Len64(x.L0)
	if x.L1 != 0 {
		n = 64 + bits.Len64(x.L1)
	}
	if x.L2 != 0 {
		n = 128 + bits.Len64(x.L2)
	}
	if x.L3 != 0 {
		n = 192 + bits.Len64(x.L3)
	}
	return n
}

// LeadingZeros returns the number of leading zero bits in x, 256 for x == 0
func (x Uint256) LeadingZeros() int {
	// conditional moves on counts made before the ifs,
	// k - Len64 folds into CLZ and inlines cheaper than LeadingZeros64
	z0 := 256 - bits.Len64(x.L0)
	z1 := 192 - bits.Len64(x.L1)
	z2 := 128 - bits.Len64(x.L2)
	n := 64 - bits.Len64(x.L3)
	if x.L1 == 0 {
		z1 = z0
	}
	if x.L2 == 0 {
		z2 = z1
	}
	if x.L3 == 0 {
		n = z2
	}
	return n
}

// TrailingZeros returns the number of trailing zero bits in x, 256 for x == 0
func (x Uint256) TrailingZeros() int {
	// conditional moves, less latency than masks in our benchmarks
	n := 192 + bits.TrailingZeros64(x.L3)
	if x.L2 != 0 {
		n = 128 + bits.TrailingZeros64(x.L2)
	}
	if x.L1 != 0 {
		n = 64 + bits.TrailingZeros64(x.L1)
	}
	if x.L0 != 0 {
		n = bits.TrailingZeros64(x.L0)
	}
	return n
}

// OnesCount returns the number of one bits in x
func (x Uint256) OnesCount() int {
	return bits.OnesCount64(x.L0) + bits.OnesCount64(x.L1) + bits.OnesCount64(x.L2) + bits.OnesCount64(x.L3)
}

// Lsh returns x << n, it panics for n < 0
func (x Uint256) Lsh(n int) Uint256 {
	if n < 0 {
		panic("fat.Uint256.Lsh: negative shift")
	}
	// the plain lsh is too big to inline
	return x.lsh(uint64(n))
}

// lsh is Lsh for any n
func (x Uint256) lsh(n uint64) Uint256 {
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
	return Uint256{r0, r1, r2, r3}
}

// Rsh returns x >> n, it panics for n < 0
func (x Uint256) Rsh(n int) Uint256 {
	if n < 0 {
		panic("fat.Uint256.Rsh: negative shift")
	}
	// the plain rsh is too big to inline
	return x.rsh(uint64(n))
}

// rsh is Rsh for any n
func (x Uint256) rsh(n uint64) Uint256 {
	// bits within limbs, then whole limbs by conditional moves, no branch on n
	s, t := n&63, ^n&63
	// <<1<<t is <<(64-s) that gives 0 at s = 0
	r0 := x.L0>>s | x.L1<<1<<t
	r1 := x.L1>>s | x.L2<<1<<t
	r2 := x.L2>>s | x.L3<<1<<t
	r3 := x.L3 >> s
	if n&64 != 0 {
		r0, r1, r2, r3 = r1, r2, r3, 0
	}
	// n > 127 for n&128 saves an AND, a wider n zeros all the limbs below
	if n > 127 {
		r0, r1, r2, r3 = r2, r3, 0, 0
	}
	if n > 255 {
		r0, r1 = 0, 0
	}
	return Uint256{r0, r1, r2, r3}
}

// RotateLeft returns x rotated left by k mod 256 bits, right for k < 0
func (x Uint256) RotateLeft(k int) Uint256 {
	// limbs by conditional moves, then bits within limbs:
	// k&192 and k&63 are k mod 256 for any sign
	s, t := uint(k)&63, uint(^k)&63
	l0, l1, l2, l3 := x.L0, x.L1, x.L2, x.L3
	if k&64 != 0 {
		l0, l1, l2, l3 = l3, l0, l1, l2
	}
	if k&128 != 0 {
		l0, l1, l2, l3 = l2, l3, l0, l1
	}
	// >>1>>t is >>(64-s) that gives 0 at s = 0
	return Uint256{l0<<s | l3>>1>>t, l1<<s | l0>>1>>t, l2<<s | l1>>1>>t, l3<<s | l2>>1>>t}
}

// RotateRight returns x rotated right by k mod 256 bits
func (x Uint256) RotateRight(k int) Uint256 {
	// a copy of RotateLeft by -k, the call to it costs a stack frame
	k = -k
	s, t := uint(k)&63, uint(^k)&63
	l0, l1, l2, l3 := x.L0, x.L1, x.L2, x.L3
	if k&64 != 0 {
		l0, l1, l2, l3 = l3, l0, l1, l2
	}
	if k&128 != 0 {
		l0, l1, l2, l3 = l2, l3, l0, l1
	}
	return Uint256{l0<<s | l3>>1>>t, l1<<s | l0>>1>>t, l2<<s | l1>>1>>t, l3<<s | l2>>1>>t}
}

// Reverse returns x with its bits in reverse order
func (x Uint256) Reverse() Uint256 {
	return Uint256{bits.Reverse64(x.L3), bits.Reverse64(x.L2), bits.Reverse64(x.L1), bits.Reverse64(x.L0)}
}

// ReverseBytes returns x with its bytes in reverse order
func (x Uint256) ReverseBytes() Uint256 {
	return Uint256{bits.ReverseBytes64(x.L3), bits.ReverseBytes64(x.L2), bits.ReverseBytes64(x.L1), bits.ReverseBytes64(x.L0)}
}

// Xor returns x ^ y
func (x Uint256) Xor(y Uint256) Uint256 {
	x.L0 ^= y.L0
	x.L1 ^= y.L1
	x.L2 ^= y.L2
	x.L3 ^= y.L3
	return x
}

// And returns x & y
func (x Uint256) And(y Uint256) Uint256 {
	x.L0 &= y.L0
	x.L1 &= y.L1
	x.L2 &= y.L2
	x.L3 &= y.L3
	return x
}

// AndNot returns x &^ y
func (x Uint256) AndNot(y Uint256) Uint256 {
	x.L0 &^= y.L0
	x.L1 &^= y.L1
	x.L2 &^= y.L2
	x.L3 &^= y.L3
	return x
}

// Or returns x | y
func (x Uint256) Or(y Uint256) Uint256 {
	x.L0 |= y.L0
	x.L1 |= y.L1
	x.L2 |= y.L2
	x.L3 |= y.L3
	return x
}

// Not returns ^x
func (x Uint256) Not() Uint256 {
	x.L0 = ^x.L0
	x.L1 = ^x.L1
	x.L2 = ^x.L2
	x.L3 = ^x.L3
	return x
}

// Bit returns bit i of x, 0 past bit 255, it panics for i < 0
func (x Uint256) Bit(i int) uint {
	if i < 0 {
		panic("fat.Uint256.Bit: negative index")
	}
	u := uint64(i)
	// conditional moves keep x in registers
	lo, hi := x.L0, x.L2
	if u&64 != 0 {
		lo, hi = x.L1, x.L3
	}
	// u > 127 for u&128 saves an AND, past bit 255 the limb is 0
	if u > 127 {
		lo = hi
	}
	if u > 255 {
		lo = 0
	}
	return uint(lo >> (u & 63) & 1)
}

// WithBit returns x with bit i set to bit, a bit past 255 is dropped.
// It panics for i < 0 and for bit > 1
func (x Uint256) WithBit(i int, bit uint) Uint256 {
	if i < 0 {
		panic("fat.Uint256.WithBit: negative index")
	}
	if bit > 1 {
		panic("fat.Uint256.WithBit: bit not 0 or 1")
	}
	u := uint64(i)
	// a shift past 63 gives 0: only the limb holding the bit gets a mask,
	// past 255 none does
	b := -uint64(bit)
	x.L0 ^= (x.L0 ^ b) & (1 << u)
	x.L1 ^= (x.L1 ^ b) & (1 << (u - 64))
	x.L2 ^= (x.L2 ^ b) & (1 << (u - 128))
	x.L3 ^= (x.L3 ^ b) & (1 << (u - 192))
	return x
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

// --- mul ------------------------------------------------

// Mul returns x * y, it panics on overflow
func (x Uint256) Mul(y Uint256) Uint256 {
	// 14 multiplies in place of the 20 of Mulo, faster in our benchmarks
	// on Graviton, where the multiplier is the bound
	r, c := x.mulFit(y)
	if c {
		panic("fat.Uint256.Mul: overflow")
	}
	return r
}

// mulFit is Mulo for a caller that drops r on overflow.
// r is not the wrapped product when both high halves are nonzero.
// Two nonzero high halves overflow, so the operand with a zero one goes in
// as the 128-bit y of Mul128o by conditional moves: 14 multiplies
func (x Uint256) mulFit(y Uint256) (r Uint256, overflow bool) {
	xh, yh := x.L2|x.L3, y.L2|y.L3
	x0, x1, x2, x3, y0, y1 := x.L0, x.L1, x.L2, x.L3, y.L0, y.L1
	if yh != 0 {
		x0, x1, x2, x3, y0, y1 = y.L0, y.L1, y.L2, y.L3, x.L0, x.L1
	}
	// x * y0 = p0..p4
	h0, p0 := bits.Mul64(x0, y0)
	h1, p1 := bits.Mul64(x1, y0)
	h2, p2 := bits.Mul64(x2, y0)
	h3, p3 := bits.Mul64(x3, y0)
	var c uint64
	p1, c = bits.Add64(p1, h0, 0)
	p2, c = bits.Add64(p2, h1, c)
	p3, c = bits.Add64(p3, h2, c)
	p4, _ := bits.Add64(h3, 0, c)
	// x * y1 at limb 1 without x3 * y1, which overflows when nonzero
	g0, q1 := bits.Mul64(x0, y1)
	g1, q2 := bits.Mul64(x1, y1)
	g2, q3 := bits.Mul64(x2, y1)
	q2, c = bits.Add64(q2, g0, 0)
	q3, c = bits.Add64(q3, g1, c)
	q4, _ := bits.Add64(g2, 0, c)
	r1, c := bits.Add64(p1, q1, 0)
	r2, c := bits.Add64(p2, q2, c)
	r3, c := bits.Add64(p3, q3, c)
	// else the product is below 2^320 and limb 4 doesn't wrap
	r4, _ := bits.Add64(p4, q4, c)
	return Uint256{p0, r1, r2, r3}, r4|min(x3, y1)|min(xh, yh) != 0
}

// Mulo returns Mulu and whether x * y overflowed
func (x Uint256) Mulo(y Uint256) (r Uint256, overflow bool) {
	// rows cut at limb 4, products past it are caught by the min test below
	h0, p0 := bits.Mul64(x.L0, y.L0)
	h1, p1 := bits.Mul64(x.L1, y.L0)
	h2, p2 := bits.Mul64(x.L2, y.L0)
	h3, p3 := bits.Mul64(x.L3, y.L0)
	var c uint64
	p1, c = bits.Add64(p1, h0, 0)
	p2, c = bits.Add64(p2, h1, c)
	p3, c = bits.Add64(p3, h2, c)
	p4, _ := bits.Add64(h3, 0, c)
	g0, q1 := bits.Mul64(x.L0, y.L1)
	g1, q2 := bits.Mul64(x.L1, y.L1)
	g2, q3 := bits.Mul64(x.L2, y.L1)
	q2, c = bits.Add64(q2, g0, 0)
	q3, c = bits.Add64(q3, g1, c)
	q4, _ := bits.Add64(g2, 0, c)
	f0, s2 := bits.Mul64(x.L0, y.L2)
	f1, s3 := bits.Mul64(x.L1, y.L2)
	s3, c = bits.Add64(s3, f0, 0)
	s4, _ := bits.Add64(f1, 0, c)
	e0, t3 := bits.Mul64(x.L0, y.L3)
	r1, c := bits.Add64(p1, q1, 0)
	r2, c := bits.Add64(p2, q2, c)
	r3, c := bits.Add64(p3, q3, c)
	r4, _ := bits.Add64(p4, q4, c)
	r2, c = bits.Add64(r2, s2, 0)
	r3, c = bits.Add64(r3, s3, c)
	r4, _ = bits.Add64(r4, s4, c)
	r3, c = bits.Add64(r3, t3, 0)
	r4, _ = bits.Add64(r4, e0, c)
	// min != 0 is a nonzero product past limb 4, else r4 is exact limb 4
	y23 := y.L2 | y.L3
	return Uint256{p0, r1, r2, r3}, r4|min(x.L3, y.L1|y23)|min(x.L2, y23)|min(x.L1, y.L3) != 0
}

// Mulu returns x * y wrapped around as on a Go integer
func (x Uint256) Mulu(y Uint256) Uint256 {
	// row by row, carries ride ADCS chains
	h0, r0 := bits.Mul64(x.L0, y.L0)
	h1, p1 := bits.Mul64(x.L1, y.L0)
	h2, p2 := bits.Mul64(x.L2, y.L0)
	var c uint64
	p1, c = bits.Add64(p1, h0, 0)
	p2, c = bits.Add64(p2, h1, c)
	p3, _ := bits.Add64(x.L3*y.L0, h2, c)
	g0, q1 := bits.Mul64(x.L0, y.L1)
	g1, q2 := bits.Mul64(x.L1, y.L1)
	q2, c = bits.Add64(q2, g0, 0)
	q3, _ := bits.Add64(x.L2*y.L1, g1, c)
	f0, s2 := bits.Mul64(x.L0, y.L2)
	r1, c := bits.Add64(p1, q1, 0)
	r2, c := bits.Add64(p2, q2, c)
	r3, _ := bits.Add64(p3, q3, c)
	r2, c = bits.Add64(r2, s2, 0)
	r3, _ = bits.Add64(r3, x.L1*y.L2+f0+x.L0*y.L3, c)
	return Uint256{r0, r1, r2, r3}
}

// Mule returns the 512-bit product x * y as hi * 2^256 + lo
func (x Uint256) Mule(y Uint256) (hi, lo Uint256) {
	// row by row, each row a chain of the low halves into the running sum,
	// faster than one chain per half in our benchmarks
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
	return Uint256{p4, p5, p6, p7}, Uint256{r0, r1, r2, r3}
}

// Mul128 returns x * y, it panics on overflow
func (x Uint256) Mul128(y Uint128) Uint256 {
	r, c := x.Mul128o(y)
	if c {
		panic("fat.Uint256.Mul128: overflow")
	}
	return r
}

// Mul128o returns Mul128u and whether x * y overflowed
func (x Uint256) Mul128o(y Uint128) (r Uint256, overflow bool) {
	// x * y.L0 = p0..p4
	h0, p0 := bits.Mul64(x.L0, y.L0)
	h1, p1 := bits.Mul64(x.L1, y.L0)
	h2, p2 := bits.Mul64(x.L2, y.L0)
	h3, p3 := bits.Mul64(x.L3, y.L0)
	var c uint64
	p1, c = bits.Add64(p1, h0, 0)
	p2, c = bits.Add64(p2, h1, c)
	p3, c = bits.Add64(p3, h2, c)
	p4, _ := bits.Add64(h3, 0, c)
	// x * y.L1 at limb 1 without x.L3 * y.L1, which overflows when nonzero
	g0, q1 := bits.Mul64(x.L0, y.L1)
	g1, q2 := bits.Mul64(x.L1, y.L1)
	g2, q3 := bits.Mul64(x.L2, y.L1)
	q2, c = bits.Add64(q2, g0, 0)
	q3, c = bits.Add64(q3, g1, c)
	q4, _ := bits.Add64(g2, 0, c)
	r1, c := bits.Add64(p1, q1, 0)
	r2, c := bits.Add64(p2, q2, c)
	r3, c := bits.Add64(p3, q3, c)
	// else the product is below 2^320 and limb 4 doesn't wrap
	r4, _ := bits.Add64(p4, q4, c)
	return Uint256{p0, r1, r2, r3}, r4|min(x.L3, y.L1) != 0
}

// Mul128u returns x * y wrapped around as on a Go integer
func (x Uint256) Mul128u(y Uint128) Uint256 {
	// x * y.L0
	h0, r0 := bits.Mul64(x.L0, y.L0)
	h1, a1 := bits.Mul64(x.L1, y.L0)
	h2, a2 := bits.Mul64(x.L2, y.L0)
	var c uint64
	a1, c = bits.Add64(a1, h0, 0)
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
	return Uint256{r0, r1, r2, r3}
}

// Mul64 returns x * y, it panics on overflow
func (x Uint256) Mul64(y uint64) Uint256 {
	// Mul64o inlines and would bring the product and a stack frame
	// for the panic in here
	r, c := x.mul64o(y)
	if c {
		panic("fat.Uint256.Mul64: overflow")
	}
	return r
}

// Mul64o returns Mul64u and whether x * y overflowed
func (x Uint256) Mul64o(y uint64) (r Uint256, overflow bool) {
	// the product into r limb by limb stays within the inline budget,
	// a call spills the caller's values
	var h, c uint64
	h, r.L0 = bits.Mul64(x.L0, y)
	g, l := bits.Mul64(x.L1, y)
	r.L1, c = bits.Add64(l, h, 0)
	h, l = bits.Mul64(x.L2, y)
	r.L2, c = bits.Add64(l, g, c)
	g, l = bits.Mul64(x.L3, y)
	r.L3, c = bits.Add64(l, h, c)
	return r, g|c != 0
}

// mul64o is Mul64o in the form that does not inline, for Mul64
func (x Uint256) mul64o(y uint64) (r Uint256, overflow bool) {
	h0, r0 := bits.Mul64(x.L0, y)
	h1, r1 := bits.Mul64(x.L1, y)
	h2, r2 := bits.Mul64(x.L2, y)
	h3, r3 := bits.Mul64(x.L3, y)
	r1, c := bits.Add64(r1, h0, 0)
	r2, c = bits.Add64(r2, h1, c)
	r3, c = bits.Add64(r3, h2, c)
	return Uint256{r0, r1, r2, r3}, h3|c != 0
}

// Mul64u returns x * y wrapped around as on a Go integer
func (x Uint256) Mul64u(y uint64) Uint256 {
	v := y
	h0, r0 := bits.Mul64(x.L0, v)
	h1, r1 := bits.Mul64(x.L1, v)
	h2, r2 := bits.Mul64(x.L2, v)
	var c uint64
	r1, c = bits.Add64(r1, h0, 0)
	r2, c = bits.Add64(r2, h1, c)
	return Uint256{r0, r1, r2, x.L3*v + h2 + c}
}
