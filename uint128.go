package fat

import (
	"math"
	"math/big"
	"math/bits"
)

// Uint128 is an unsigned 128-bit integer, L0 is its low limb and L1 the high
type Uint128 struct{ L0, L1 uint64 }

// --- constructors ---------------------------------------

// NewUint128 returns v as a Uint128
func NewUint128(v uint64) Uint128 {
	return Uint128{v, 0}
}

// Uint128FromBig returns v, or zero and ErrNegative or ErrRange
func Uint128FromBig(v *big.Int) (Uint128, error) {
	// x is zero on an error
	x, e := uint128FromBig(v)
	return x, uint128FromBigErrs[e&3]
}

// MustUint128FromBig is Uint128FromBig that panics when v does not fit
func MustUint128FromBig(v *big.Int) Uint128 {
	x, err := uint128FromBig(v)
	if err != 0 {
		// a table lookup and not a call keeps it within the inline budget
		panic(mustUint128FromBigPanics[err&3])
	}
	return x
}

// uint128FromBigErrs are the errors of Uint128FromBig by error bits
var uint128FromBigErrs = [4]error{
	1: ErrNegative{"Uint128FromBig", ""},
	2: ErrRange("Uint128FromBig"),
	3: ErrNegative{"Uint128FromBig", ""},
}

// mustUint128FromBigPanics are the panics of MustUint128FromBig by error bits
var mustUint128FromBigPanics = [4]string{
	1: "fat.MustUint128FromBig: negative value",
	2: "fat.MustUint128FromBig: out of range",
	3: "fat.MustUint128FromBig: negative value",
}

// uint128FromBig is Uint128FromBig with error bits:
// 1 for a negative v, 2 for one past 128 bits
func uint128FromBig(v *big.Int) (Uint128, uint) {
	words := v.Bits()
	n := len(words)
	if bits.UintSize == 64 && uint(n-1) < 2 {
		// words[1], or words[0] masked off for one word: no length branch
		i := n - 1
		w0 := uint64(words[0])
		// all ones for a negative v
		neg := bigNeg(v, w0)
		return Uint128{w0 &^ neg, uint64(words[i]) & -uint64(i) &^ neg}, uint(neg & 1)
	}
	// zero, past 128 bits, or 32-bit words
	u, left := uint128FromWords(words)
	err := uint(v.Sign())>>(bits.UintSize-1) | uint(-left)>>(bits.UintSize-1)<<1
	// all ones when v fits
	ok := uint64(int64(uint64(err)-1) >> 63)
	return Uint128{u.L0 & ok, u.L1 & ok}, err
}

// --- getters --------------------------------------------

// IsZero reports whether x == 0
func (x Uint128) IsZero() bool {
	return x.L0|x.L1 == 0
}

// Sign returns 0 for x == 0 and 1 otherwise
func (x Uint128) Sign() int {
	// the top bit of n|-n is set when n != 0
	n := x.L0 | x.L1
	return int((n | -n) >> 63)
}

// IsUint64 reports whether x fits in a uint64
func (x Uint128) IsUint64() bool {
	return x.L1 == 0
}

// Uint64 returns x as a uint64, it panics when x does not fit
func (x Uint128) Uint64() uint64 {
	if !x.IsUint64() {
		panic("fat.Uint128.Uint64: out of range")
	}
	return x.L0
}

// Uint64o returns Uint64u and whether x does not fit
func (x Uint128) Uint64o() (r uint64, overflow bool) {
	return x.Uint64u(), !x.IsUint64()
}

// Uint64u returns the low 64 bits of x as a Go conversion does
func (x Uint128) Uint64u() uint64 {
	return x.L0
}

// Uint256 returns x widened to 256 bits
func (x Uint128) Uint256() Uint256 {
	return Uint256{x.L0, x.L1, 0, 0}
}

// IsInt64 reports whether x fits in an int64
func (x Uint128) IsInt64() bool {
	return x.L1|x.L0>>63 == 0
}

// Int64 returns x as an int64, it panics when x does not fit
func (x Uint128) Int64() int64 {
	if !x.IsInt64() {
		panic("fat.Uint128.Int64: out of range")
	}
	return int64(x.L0)
}

// Int64o returns Int64u and whether x does not fit
func (x Uint128) Int64o() (r int64, overflow bool) {
	return x.Int64u(), !x.IsInt64()
}

// Int64u returns the low 64 bits of x as a Go conversion does
func (x Uint128) Int64u() int64 {
	return int64(x.L0)
}

// IsInt128 reports whether x fits in an Int128
func (x Uint128) IsInt128() bool {
	return x.L1>>63 == 0
}

// Int128 returns x as an Int128, it panics when x does not fit
func (x Uint128) Int128() Int128 {
	if !x.IsInt128() {
		panic("fat.Uint128.Int128: out of range")
	}
	return Int128(x)
}

// Int128o returns Int128u and whether x does not fit
func (x Uint128) Int128o() (r Int128, overflow bool) {
	return x.Int128u(), !x.IsInt128()
}

// Int128u returns the bits of x as a Go conversion does
func (x Uint128) Int128u() Int128 {
	return Int128(x)
}

// Int256 returns x widened to 256 bits
func (x Uint128) Int256() Int256 {
	return Int256{x.L0, x.L1, 0, 0}
}

// Big returns x as a new big.Int
func (x Uint128) Big() *big.Int {
	// the Int and its words in one allocation
	p := new(struct {
		b big.Int
		w [2 * limbWords]big.Word
	})
	x.putWords(p.w[:])
	return p.b.SetBits(p.w[:x.words()*limbWords])
}

// PutBig sets b to x and returns it, reusing the words of b if they fit
func (x Uint128) PutBig(b *big.Int) *big.Int {
	words := b.Bits()
	if cap(words) < 2*limbWords {
		words = make([]big.Word, 2*limbWords)
	}
	w := words[:2*limbWords]
	x.putWords(w)
	// the exact length, so SetBits has at most one zero 32-bit word to trim
	return b.SetBits(w[:x.words()*limbWords])
}

// words returns the limb count of x without zero top limbs
func (x Uint128) words() int {
	// a sum of two flags: a select of 2 compiles to a branch
	var lo, hi int
	if x.L0|x.L1 != 0 {
		lo = 1
	}
	if x.L1 != 0 {
		hi = 1
	}
	return lo + hi
}

// putWords writes x into w for any word size, w must hold 2*limbWords words
func (x Uint128) putWords(w []big.Word) {
	// high halves of 32-bit words, for 64-bit ones the stores below overwrite
	w[2*limbWords-1] = big.Word(x.L1 >> (bits.UintSize % 64))
	w[limbWords-1] = big.Word(x.L0 >> (bits.UintSize % 64))
	// low words next to each other, they pair up
	w[0] = big.Word(x.L0)
	w[limbWords] = big.Word(x.L1)
}

// uint128FromWords reads the low 2 limbs of |v| for any word size.
// left is the count of words past them
func uint128FromWords(words []big.Word) (x Uint128, left int) {
	n := min(len(words), 2*limbWords)
	// the top word first, in registers: an array would cost a stack frame
	var l0, l1 uint64
	for i := n - 1; i >= 0; i-- {
		// a word shifts in, for 64-bit words two half shifts move a limb
		l1 = l1<<(bits.UintSize/2)<<(bits.UintSize/2) | l0>>(64-bits.UintSize)
		l0 = l0<<(bits.UintSize/2)<<(bits.UintSize/2) | uint64(words[i])
	}
	return Uint128{l0, l1}, len(words) - n
}

// --- comparison -----------------------------------------

// Cmp returns -1 for x < y, 0 for x = y and 1 for x > y
func (x Uint128) Cmp(y Uint128) int {
	// two independent borrow chains, x < y and y < x
	_, lt := bits.Sub64(x.L0, y.L0, 0)
	_, lt = bits.Sub64(x.L1, y.L1, lt)
	_, gt := bits.Sub64(y.L0, x.L0, 0)
	_, gt = bits.Sub64(y.L1, x.L1, gt)
	return int(gt) - int(lt)
}

// LessThan reports whether x < y
func (x Uint128) LessThan(y Uint128) bool {
	_, carry := bits.Sub64(x.L0, y.L0, 0)
	_, carry = bits.Sub64(x.L1, y.L1, carry)
	return carry != 0
}

// LessOrEqual reports whether x <= y
func (x Uint128) LessOrEqual(y Uint128) bool {
	_, carry := bits.Sub64(y.L0, x.L0, 0)
	_, carry = bits.Sub64(y.L1, x.L1, carry)
	return carry == 0
}

// GreaterThan reports whether x > y
func (x Uint128) GreaterThan(y Uint128) bool {
	_, carry := bits.Sub64(y.L0, x.L0, 0)
	_, carry = bits.Sub64(y.L1, x.L1, carry)
	return carry != 0
}

// GreaterOrEqual reports whether x >= y
func (x Uint128) GreaterOrEqual(y Uint128) bool {
	_, carry := bits.Sub64(x.L0, y.L0, 0)
	_, carry = bits.Sub64(x.L1, y.L1, carry)
	return carry == 0
}

// Max returns the larger of x and y, as the builtin max
func (x Uint128) Max(y Uint128) Uint128 {
	// the borrow chain of LessThan written out picks by CSEL
	_, carry := bits.Sub64(x.L0, y.L0, 0)
	_, carry = bits.Sub64(x.L1, y.L1, carry)
	if carry != 0 {
		x = y
	}
	return x
}

// Min returns the smaller of x and y, as the builtin min
func (x Uint128) Min(y Uint128) Uint128 {
	_, carry := bits.Sub64(x.L0, y.L0, 0)
	_, carry = bits.Sub64(x.L1, y.L1, carry)
	if carry == 0 {
		x = y
	}
	return x
}

// Cmp64 returns -1 for x < y, 0 for x = y and 1 for x > y
func (x Uint128) Cmp64(y uint64) int {
	// two independent borrow chains, x < y and y < x
	_, lt := bits.Sub64(x.L0, y, 0)
	_, lt = bits.Sub64(x.L1, 0, lt)
	_, gt := bits.Sub64(y, x.L0, 0)
	_, gt = bits.Sub64(0, x.L1, gt)
	return int(gt) - int(lt)
}

// CmpBig returns -1 for x < y, 0 for x = y and 1 for x > y
func (x Uint128) CmpBig(y *big.Int) int {
	words := y.Bits()
	n := len(words)
	if bits.UintSize == 64 && uint(n-1) < 2 {
		// as in uint128FromBig, words[0] stands in for a missing words[1]
		i := n - 1
		l0 := uint64(words[0])
		c := x.Cmp(Uint128{l0, uint64(words[i]) & -uint64(i)})
		// every x is above a negative y
		if bigIsNeg(y, l0) {
			c = 1
		}
		return c
	}
	// zero, past 128 bits, or 32-bit words
	y128, e := uint128FromBig(y)
	if e != 0 {
		// x is above a negative y and below one past 128 bits
		return -y.Sign()
	}
	return x.Cmp(y128)
}

// --- add ------------------------------------------------

// Add returns x + y, it panics on overflow
func (x Uint128) Add(y Uint128) Uint128 {
	// written out: a call of Addo leaves the NOP of its inline mark in a loop
	var carry uint64
	x.L0, carry = bits.Add64(x.L0, y.L0, 0)
	x.L1, carry = bits.Add64(x.L1, y.L1, carry)
	if carry != 0 {
		panic("fat.Uint128.Add: overflow")
	}
	return x
}

// Addo returns Addu and whether x + y overflowed
func (x Uint128) Addo(y Uint128) (r Uint128, overflow bool) {
	var carry uint64
	x.L0, carry = bits.Add64(x.L0, y.L0, 0)
	x.L1, carry = bits.Add64(x.L1, y.L1, carry)
	return x, carry != 0
}

// Addu returns x + y wrapped around as on a Go integer
func (x Uint128) Addu(y Uint128) Uint128 {
	var carry uint64
	x.L0, carry = bits.Add64(x.L0, y.L0, 0)
	x.L1, _ = bits.Add64(x.L1, y.L1, carry)
	return x
}

// Add64 returns x + y, it panics on overflow
func (x Uint128) Add64(y uint64) Uint128 {
	// written out: a call of Add64o leaves the NOP of its inline mark in a loop
	// only a full top limb carries, known before the low add
	var full uint64
	if x.L1 == math.MaxUint64 {
		full = 1
	}
	var carry uint64
	x.L0, carry = bits.Add64(x.L0, y, 0)
	x.L1 += carry
	if carry&full != 0 {
		panic("fat.Uint128.Add64: overflow")
	}
	return x
}

// Add64o returns Add64u and whether x + y overflowed
func (x Uint128) Add64o(y uint64) (r Uint128, overflow bool) {
	// only a full top limb carries, known before the low add
	var full uint64
	if x.L1 == math.MaxUint64 {
		full = 1
	}
	var carry uint64
	x.L0, carry = bits.Add64(x.L0, y, 0)
	x.L1 += carry
	return x, carry&full != 0
}

// Add64u returns x + y wrapped around as on a Go integer
func (x Uint128) Add64u(y uint64) Uint128 {
	var carry uint64
	x.L0, carry = bits.Add64(x.L0, y, 0)
	x.L1, _ = bits.Add64(x.L1, 0, carry)
	return x
}

// --- sub ------------------------------------------------

// Sub returns x - y, it panics when y > x
func (x Uint128) Sub(y Uint128) Uint128 {
	// written out: a call of Subo leaves the NOP of its inline mark in a loop
	var carry uint64
	x.L0, carry = bits.Sub64(x.L0, y.L0, 0)
	x.L1, carry = bits.Sub64(x.L1, y.L1, carry)
	if carry != 0 {
		panic("fat.Uint128.Sub: negative value")
	}
	return x
}

// Subo returns Subu and whether y > x
func (x Uint128) Subo(y Uint128) (r Uint128, overflow bool) {
	var carry uint64
	x.L0, carry = bits.Sub64(x.L0, y.L0, 0)
	x.L1, carry = bits.Sub64(x.L1, y.L1, carry)
	return x, carry != 0
}

// Subu returns x - y wrapped around as on a Go integer
func (x Uint128) Subu(y Uint128) Uint128 {
	var carry uint64
	x.L0, carry = bits.Sub64(x.L0, y.L0, 0)
	x.L1, _ = bits.Sub64(x.L1, y.L1, carry)
	return x
}

// Sub64 returns x - y, it panics when y > x
func (x Uint128) Sub64(y uint64) Uint128 {
	// written out: a call of Sub64o leaves the NOP of its inline mark in a loop
	var carry uint64
	if x.L0 < y {
		carry = 1
	}
	// only a zero top limb borrows
	over := x.L1 < carry
	x.L0 -= y
	x.L1 -= carry
	if over {
		panic("fat.Uint128.Sub64: negative value")
	}
	return x
}

// Sub64o returns Sub64u and whether y > x
func (x Uint128) Sub64o(y uint64) (r Uint128, overflow bool) {
	var carry uint64
	if x.L0 < y {
		carry = 1
	}
	// only a zero top limb borrows
	over := x.L1 < carry
	x.L0 -= y
	x.L1 -= carry
	return x, over
}

// Sub64u returns x - y wrapped around as on a Go integer
func (x Uint128) Sub64u(y uint64) Uint128 {
	var carry uint64
	x.L0, carry = bits.Sub64(x.L0, y, 0)
	x.L1, _ = bits.Sub64(x.L1, 0, carry)
	return x
}
