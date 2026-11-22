package fat

import (
	"encoding"
	"encoding/binary"
	"encoding/gob"
	"math"
	"math/big"
	"math/bits"
)

// Int128 is a signed 128-bit integer in two's complement.
// L0 holds the low 64 bits and L1 the high ones
type Int128 struct{ L0, L1 uint64 }

// --- constructors ---------------------------------------

// NewInt128 returns v as an Int128
func NewInt128(v int64) Int128 {
	ext := uint64(v >> 63)
	return Int128{uint64(v), ext}
}

// Int128FromBig returns v, or zero and ErrRange when v does not fit
func Int128FromBig(v *big.Int) (Int128, error) {
	x, ok := int128FromBig(v)
	if !ok {
		return Int128{}, ErrRange("Int128FromBig")
	}
	return x, nil
}

// int128FromBig is Int128FromBig with ok in place of the error
func int128FromBig(v *big.Int) (Int128, bool) {
	words := v.Bits()
	n := len(words)
	if bits.UintSize == 64 && uint(n-1) < 2 {
		// a test of the cap would mispredict, a single word has cap 1
		// words[1], or words[0] masked off when there is one word
		i := n - 1
		l0 := uint64(words[0])
		l1 := uint64(words[i]) & -uint64(i)
		// all ones for a negative v, (x ^ m) - m then negates with no branch
		m := bigNeg(v, l0)
		r0, b := bits.Sub64(l0^m, m, 0)
		r1, _ := bits.Sub64(l1^m, m, b)
		// all ones when v fits: the sign flips out of range, MinInt128 keeps it
		ok := uint64(int64(^(r1 ^ m)) >> 63)
		return Int128{r0 & ok, r1 & ok}, ok != 0
	}
	// zero, past 128 bits, or 32-bit words
	u, left := uint128FromWords(words)
	m := uint64(v.Sign() >> 1)
	r0, b := bits.Sub64(u.L0^m, m, 0)
	r1, _ := bits.Sub64(u.L1^m, m, b)
	// all ones when v fits: no words left and the sign kept
	ok := uint64(int64(^(r1^m)&^uint64(-left)) >> 63)
	return Int128{r0 & ok, r1 & ok}, ok != 0
}

// MustInt128FromBig is Int128FromBig that panics when v does not fit
func MustInt128FromBig(v *big.Int) Int128 {
	x, ok := int128FromBig(v)
	if !ok {
		panic("fat.MustInt128FromBig: out of range")
	}
	return x
}

// Int128FromBytes reads big-endian b and whether it fits, zero if it does not.
// The last 16 bytes are the two's complement word, fewer are zero-extended.
// The bytes before them must be all 0, or all 0xFF when the word is negative
func Int128FromBytes(b []byte) (Int128, bool) {
	if n := len(b); n >= 16 {
		// the bytes in front are zeros, or 0xFF for a negative word.
		// Both in one loop: a call to Uint128FromBytes benchmarked slower
		s := byte(int8(b[n-16]) >> 7)
		var nz, ns byte
		for _, c := range b[:n-16] {
			nz |= c
			ns |= c ^ s
		}
		if nz != 0 && ns != 0 {
			return Int128{}, false
		}
		b = b[n-16:]
		return Int128{binary.BigEndian.Uint64(b[8:]), binary.BigEndian.Uint64(b)}, true
	}
	return Int128(uint128FromBytes(b)), true
}

// MustInt128FromBytes is Int128FromBytes that panics when b does not fit
func MustInt128FromBytes(b []byte) Int128 {
	x, ok := Int128FromBytes(b)
	if !ok {
		panic("fat.MustInt128FromBytes: out of range")
	}
	return x
}

// Int128FromFloat64 returns v truncated toward zero and whether it fits.
// It returns zero and false for a NaN, for v < -2^127 and for v >= 2^127
func Int128FromFloat64(v float64) (Int128, bool) {
	b := math.Float64bits(v)
	t := b<<1>>53 - 1023
	if t >= 127 {
		// |v| < 1 is zero, only -2^127 fits from that exponent on
		if b == 0xC7E<<52 {
			return MinInt128(), true
		}
		return Int128{}, b<<1 < 1023<<53
	}
	// Uint128FromFloat64 on |v|
	h := b<<11 | 1<<63
	lo, hi := h>>(^t&63), uint64(0)
	if under := h << 1 << (t & 63); t >= 64 {
		lo, hi = under, lo
	}
	// (x ^ m) - m negates for a negative v
	m := uint64(int64(b) >> 63)
	r0, c := bits.Sub64(lo^m, m, 0)
	r1, _ := bits.Sub64(hi^m, m, c)
	return Int128{r0, r1}, true
}

// MustInt128FromFloat64 is Int128FromFloat64 that panics when v does not fit
func MustInt128FromFloat64(v float64) Int128 {
	x, ok := Int128FromFloat64(v)
	if !ok {
		panic("fat.MustInt128FromFloat64: out of range")
	}
	return x
}

// --- getters --------------------------------------------

// IsZero reports whether x == 0
func (x Int128) IsZero() bool {
	return x.L0|x.L1 == 0
}

// Sign returns -1 for x < 0, 0 for x == 0 and 1 for x > 0
func (x Int128) Sign() int {
	// -1 or 0 from the sign, | 1 if nonzero, no branches
	return int(uint64(int64(x.L1)>>63) | min(x.L0|x.L1, 1))
}

// IsNegative reports whether x < 0
func (x Int128) IsNegative() bool {
	return x.L1>>63 == 1
}

// IsPositive reports whether x > 0
func (x Int128) IsPositive() bool {
	// -x is negative and x is not, && would branch
	_, b := bits.Sub64(0, x.L0, 0)
	return int64((-x.L1-b)&^x.L1) < 0
}

// IsInt64 reports whether x fits in an int64
func (x Int128) IsInt64() bool {
	return x.L1 == uint64(int64(x.L0)>>63)
}

// Int64 returns x as an int64, it panics when x does not fit
func (x Int128) Int64() int64 {
	if !x.IsInt64() {
		panic("fat.Int128.Int64: out of range")
	}
	return int64(x.L0)
}

// Int64o returns Int64u and whether x does not fit
func (x Int128) Int64o() (r int64, overflow bool) {
	return x.Int64u(), !x.IsInt64()
}

// Int64u returns the low 64 bits of x as a Go conversion does
func (x Int128) Int64u() int64 {
	return int64(x.L0)
}

// Int256 returns x widened to 256 bits
func (x Int128) Int256() Int256 {
	m := uint64(int64(x.L1) >> 63)
	return Int256{x.L0, x.L1, m, m}
}

// IsUint64 reports whether x fits in a uint64
func (x Int128) IsUint64() bool {
	return x.L1 == 0
}

// Uint64 returns x as a uint64, it panics when x does not fit
func (x Int128) Uint64() uint64 {
	if !x.IsUint64() {
		if x.IsNegative() {
			panic("fat.Int128.Uint64: negative value")
		}
		panic("fat.Int128.Uint64: out of range")
	}
	return x.L0
}

// Uint64o returns Uint64u and whether x does not fit
func (x Int128) Uint64o() (r uint64, overflow bool) {
	return x.Uint64u(), !x.IsUint64()
}

// Uint64u returns the low 64 bits of x as a Go conversion does
func (x Int128) Uint64u() uint64 {
	return x.L0
}

// IsUint128 reports whether x fits in a Uint128
func (x Int128) IsUint128() bool {
	return int64(x.L1) >= 0
}

// Uint128 returns x as a Uint128, it panics when x does not fit
func (x Int128) Uint128() Uint128 {
	if !x.IsUint128() {
		panic("fat.Int128.Uint128: negative value")
	}
	return Uint128(x)
}

// Uint128o returns Uint128u and whether x does not fit
func (x Int128) Uint128o() (r Uint128, overflow bool) {
	return x.Uint128u(), !x.IsUint128()
}

// Uint128u returns the bits of x as a Go conversion does
func (x Int128) Uint128u() Uint128 {
	return Uint128(x)
}

// IsUint256 reports whether x fits in a Uint256
func (x Int128) IsUint256() bool {
	return int64(x.L1) >= 0
}

// Uint256 returns x as a Uint256, it panics when x does not fit
func (x Int128) Uint256() Uint256 {
	if !x.IsUint256() {
		panic("fat.Int128.Uint256: negative value")
	}
	return Uint256{x.L0, x.L1, 0, 0}
}

// Uint256o returns Uint256u and whether x does not fit
func (x Int128) Uint256o() (r Uint256, overflow bool) {
	return x.Uint256u(), !x.IsUint256()
}

// Uint256u returns x sign-extended to 256 bits as a Go conversion does
func (x Int128) Uint256u() Uint256 {
	m := uint64(int64(x.L1) >> 63)
	return Uint256{x.L0, x.L1, m, m}
}

// Float64 returns the float64 nearest to x, a tie goes to the even one
func (x Int128) Float64() float64 {
	// Uint128.Float64 with a signed high field and an unsigned low one:
	// bits 0 to 103 for -2^104 <= x < 2^104, else 24 to 127 over sticky bits
	lb, hb := x.L0, int64(x.L0>>52|x.L1<<12)
	e := uint64(1023 << 52)
	// lb2 in the if statement is cheaper to inline and still a conditional move
	if lb2 := x.L0>>24 | x.L1<<40 | x.L0&0xFFFFFF; x.L1+1<<40 >= 1<<41 {
		lb, hb, e = lb2, int64(x.L1)>>12, (1023+24)<<52
	}
	return (float64(hb)*0x1p52 + float64(int64(lb&(1<<52-1)))) * math.Float64frombits(e)
}

// Big returns x as a new big.Int
func (x Int128) Big() *big.Int {
	// the Int and its words in one allocation
	p := new(struct {
		b big.Int
		w [2 * limbWords]big.Word
	})
	// PutBig written out, the words need no capacity check
	a := x.UnsignedAbs()
	a.putWords(p.w[:])
	p.b.SetBits(p.w[:a.words()*limbWords])
	if x.IsNegative() {
		p.b.Neg(&p.b)
	}
	return &p.b
}

// PutBig sets b to x and returns it, reusing the words of b if they fit
func (x Int128) PutBig(b *big.Int) *big.Int {
	// Uint128.PutBig on |x| written out, UnsignedAbs is right for MinInt128 too
	a := x.UnsignedAbs()
	words := b.Bits()
	if cap(words) < 2*limbWords {
		words = make([]big.Word, 2*limbWords)
	}
	w := words[:2*limbWords]
	a.putWords(w)
	b.SetBits(w[:a.words()*limbWords])
	if x.IsNegative() {
		b.Neg(b)
	}
	return b
}

// --- interfaces -----------------------------------------

var (
	_ encoding.BinaryMarshaler   = (*Int128)(nil)
	_ encoding.BinaryAppender    = (*Int128)(nil)
	_ encoding.BinaryUnmarshaler = (*Int128)(nil)
)

// MarshalBinary writes 16 bytes big-endian in two's complement
func (x Int128) MarshalBinary() ([]byte, error) {
	b := make([]byte, 16)
	binary.BigEndian.PutUint64(b, x.L1)
	binary.BigEndian.PutUint64(b[8:], x.L0)
	return b, nil
}

// AppendBinary appends the 16 bytes of MarshalBinary to b
func (x Int128) AppendBinary(b []byte) ([]byte, error) {
	b = binary.BigEndian.AppendUint64(b, x.L1)
	return binary.BigEndian.AppendUint64(b, x.L0), nil
}

// UnmarshalBinary reads 16 bytes big-endian in two's complement.
// It returns ErrBinary for any other length
func (x *Int128) UnmarshalBinary(b []byte) error {
	if len(b) != 16 {
		return ErrBinary{"Int128.UnmarshalBinary", len(b), 16}
	}
	*x = Int128{binary.BigEndian.Uint64(b[8:]), binary.BigEndian.Uint64(b)}
	return nil
}

var (
	_ gob.GobEncoder = (*Int128)(nil)
	_ gob.GobDecoder = (*Int128)(nil)
)

// GobEncode is MarshalBinary
func (x Int128) GobEncode() ([]byte, error) {
	return x.MarshalBinary()
}

// GobDecode is UnmarshalBinary
func (x *Int128) GobDecode(b []byte) error {
	if len(b) != 16 {
		return ErrBinary{"Int128.GobDecode", len(b), 16}
	}
	*x = Int128{binary.BigEndian.Uint64(b[8:]), binary.BigEndian.Uint64(b)}
	return nil
}

// --- bytes ----------------------------------------------

// ByteLen returns the length of Bytes, 16 for x < 0
func (x Int128) ByteLen() int {
	return Uint128(x).ByteLen()
}

// Byte returns byte i of the 16 big-endian bytes of x in two's complement.
// It returns 0 for i >= 16 and panics for i < 0
func (x Int128) Byte(i int) byte {
	if uint(i) >= 16 {
		if i < 0 {
			panic("fat.Int128.Byte: negative index")
		}
		return 0
	}
	// the bit offset of the byte, the limb by a conditional move
	s := uint(15-i) * 8
	v := x.L0
	if s >= 64 {
		v = x.L1
	}
	return byte(v >> (s & 63))
}

// ExtendSign returns x sign-extended from byte b, counted from the low end.
// It returns x for b >= 15 and panics for b < 0
func (x Int128) ExtendSign(b int) Int128 {
	if uint(b) >= 15 {
		if b < 0 {
			panic("fat.Int128.ExtendSign: negative index")
		}
		return x
	}
	// as Uint128.ExtendSign, a call to it would panic under its name
	k := uint(^b&7) * 8
	lo := uint64(int64(x.L0<<k) >> k)
	hi := uint64(int64(lo) >> 63)
	if b >= 8 {
		lo, hi = x.L0, uint64(int64(x.L1<<k)>>k)
	}
	return Int128{lo, hi}
}

// Bytes32 returns x as 32 bytes big-endian in two's complement
func (x Int128) Bytes32() (b [32]byte) {
	m := uint64(int64(x.L1) >> 63)
	binary.BigEndian.PutUint64(b[:], m)
	binary.BigEndian.PutUint64(b[8:], m)
	binary.BigEndian.PutUint64(b[16:], x.L1)
	binary.BigEndian.PutUint64(b[24:], x.L0)
	return
}

// Bytes20 returns the low 20 bytes of Bytes32
func (x Int128) Bytes20() (b [20]byte) {
	// Uint128.Bytes20 with the sign on top.
	// On Graviton the stores at 0, 8 and 16 forward to the copy of the result
	binary.BigEndian.PutUint64(b[:], uint64(int64(x.L1)>>32))
	binary.BigEndian.PutUint64(b[8:], x.L1<<32|x.L0>>32)
	binary.BigEndian.PutUint32(b[16:], uint32(x.L0))
	return
}

// Bytes16 returns x as 16 big-endian bytes, two's complement
func (x Int128) Bytes16() (b [16]byte) {
	binary.BigEndian.PutUint64(b[:], x.L1)
	binary.BigEndian.PutUint64(b[8:], x.L0)
	return
}

// PutBytes32 writes Bytes32 to dst[:32], it panics when len(dst) < 32
func (x Int128) PutBytes32(dst []byte) {
	if len(dst) < 32 {
		panic("fat.Int128.PutBytes32: short buffer")
	}
	m := uint64(int64(x.L1) >> 63)
	binary.BigEndian.PutUint64(dst, m)
	binary.BigEndian.PutUint64(dst[8:], m)
	binary.BigEndian.PutUint64(dst[16:], x.L1)
	binary.BigEndian.PutUint64(dst[24:], x.L0)
}

// PutBytes20 writes Bytes20 to dst[:20], it panics when len(dst) < 20
func (x Int128) PutBytes20(dst []byte) {
	if len(dst) < 20 {
		panic("fat.Int128.PutBytes20: short buffer")
	}
	binary.BigEndian.PutUint32(dst, uint32(int64(x.L1)>>63))
	binary.BigEndian.PutUint64(dst[4:], x.L1)
	binary.BigEndian.PutUint64(dst[12:], x.L0)
}

// PutBytes16 writes Bytes16 to dst[:16], it panics when len(dst) < 16
func (x Int128) PutBytes16(dst []byte) {
	if len(dst) < 16 {
		panic("fat.Int128.PutBytes16: short buffer")
	}
	binary.BigEndian.PutUint64(dst, x.L1)
	binary.BigEndian.PutUint64(dst[8:], x.L0)
}

// Bytes returns x in two's complement big-endian without leading zero bytes.
// It is 16 bytes long for x < 0 and empty for x == 0
func (x Int128) Bytes() []byte {
	return Uint128(x).Bytes()
}

// AppendBytes appends Bytes to dst
func (x Int128) AppendBytes(dst []byte) []byte {
	return Uint128(x).AppendBytes(dst)
}

// --- sign -----------------------------------------------

// Neg returns -x, it panics for MinInt128
func (x Int128) Neg() Int128 {
	r0, c := bits.Sub64(0, x.L0, 0)
	r1, _ := bits.Sub64(0, x.L1, c)
	// only MinInt128 stays negative
	if int64(x.L1&r1) < 0 {
		panic("fat.Int128.Neg: overflow")
	}
	return Int128{r0, r1}
}

// Abs returns |x|, it panics for MinInt128
func (x Int128) Abs() Int128 {
	r := x.UnsignedAbs()
	if int64(r.L1) < 0 {
		panic("fat.Int128.Abs: overflow")
	}
	return Int128(r)
}

// UnsignedAbs returns |x| as a Uint128, which holds 2^127 for MinInt128
func (x Int128) UnsignedAbs() Uint128 {
	// -x picked by conditional moves
	r0, b := bits.Sub64(0, x.L0, 0)
	r1, _ := bits.Sub64(0, x.L1, b)
	if int64(x.L1) >= 0 {
		r0, r1 = x.L0, x.L1
	}
	return Uint128{r0, r1}
}

// --- comparison -----------------------------------------

// flipping the sign bits maps signed order onto unsigned order

// Cmp returns -1 for x < y, 0 for x = y and 1 for x > y
func (x Int128) Cmp(y Int128) int {
	// two borrow chains, x < y and y < x
	x1, y1 := x.L1^(1<<63), y.L1^(1<<63)
	_, lt := bits.Sub64(x.L0, y.L0, 0)
	_, lt = bits.Sub64(x1, y1, lt)
	_, gt := bits.Sub64(y.L0, x.L0, 0)
	_, gt = bits.Sub64(y1, x1, gt)
	return int(gt) - int(lt)
}

// LessThan reports whether x < y
func (x Int128) LessThan(y Int128) bool {
	_, carry := bits.Sub64(x.L0, y.L0, 0)
	_, carry = bits.Sub64(x.L1^(1<<63), y.L1^(1<<63), carry)
	return carry != 0
}

// LessOrEqual reports whether x <= y
func (x Int128) LessOrEqual(y Int128) bool {
	_, carry := bits.Sub64(y.L0, x.L0, 0)
	_, carry = bits.Sub64(y.L1^(1<<63), x.L1^(1<<63), carry)
	return carry == 0
}

// GreaterThan reports whether x > y
func (x Int128) GreaterThan(y Int128) bool {
	_, carry := bits.Sub64(y.L0, x.L0, 0)
	_, carry = bits.Sub64(y.L1^(1<<63), x.L1^(1<<63), carry)
	return carry != 0
}

// GreaterOrEqual reports whether x >= y
func (x Int128) GreaterOrEqual(y Int128) bool {
	_, carry := bits.Sub64(x.L0, y.L0, 0)
	_, carry = bits.Sub64(x.L1^(1<<63), y.L1^(1<<63), carry)
	return carry == 0
}

// Max returns the larger of x and y, as the builtin max
func (x Int128) Max(y Int128) Int128 {
	// the borrow chain of LessThan written out picks by CSEL
	_, carry := bits.Sub64(x.L0, y.L0, 0)
	_, carry = bits.Sub64(x.L1^(1<<63), y.L1^(1<<63), carry)
	if carry != 0 {
		x = y
	}
	return x
}

// Min returns the smaller of x and y, as the builtin min
func (x Int128) Min(y Int128) Int128 {
	_, carry := bits.Sub64(x.L0, y.L0, 0)
	_, carry = bits.Sub64(x.L1^(1<<63), y.L1^(1<<63), carry)
	if carry == 0 {
		x = y
	}
	return x
}

// Cmp64 returns -1 for x < y, 0 for x = y and 1 for x > y
func (x Int128) Cmp64(y int64) int {
	ext := uint64(y >> 63)
	// two borrow chains, x < y and y < x
	_, lt := bits.Sub64(x.L0, uint64(y), 0)
	_, lt = bits.Sub64(x.L1^(1<<63), ext^(1<<63), lt)
	_, gt := bits.Sub64(uint64(y), x.L0, 0)
	_, gt = bits.Sub64(ext^(1<<63), x.L1^(1<<63), gt)
	return int(gt) - int(lt)
}

// CmpBig returns -1 for x < y, 0 for x = y and 1 for x > y
func (x Int128) CmpBig(y *big.Int) int {
	words := y.Bits()
	n := len(words)
	if bits.UintSize == 64 && uint(n-1) < 2 {
		// as in Uint128.CmpBig, words[0] stands in for a missing words[1]
		i := n - 1
		l0 := uint64(words[0])
		l1 := uint64(words[i]) & -uint64(i)
		// y as 192-bit two's complement needs no range checks
		// all ones for a negative y, Sign would branch
		m := bigNeg(y, l0)
		y0, b := bits.Sub64(l0^m, m, 0)
		y1, b := bits.Sub64(l1^m, m, b)
		y2, _ := bits.Sub64(m, m, b)
		d0, c := bits.Sub64(x.L0, y0, 0)
		d1, c := bits.Sub64(x.L1, y1, c)
		d2, c := bits.Sub64(uint64(int64(x.L1)>>63)^(1<<63), y2^(1<<63), c)
		return int(-c | min(d0|d1|d2, 1))
	}
	// zero, past 128 bits, or 32-bit words
	y128, ok := int128FromBig(y)
	if !ok {
		// out of range, x is below a positive y and above a negative one
		return -y.Sign()
	}
	return x.Cmp(y128)
}

// --- bits -----------------------------------------------

// BitLen returns the length of |x| in bits, 0 for zero
func (x Int128) BitLen() int {
	// |x| by conditional moves, right even for MinInt128
	return x.UnsignedAbs().BitLen()
}

// Lsh returns x << n, it panics for n < 0
func (x Int128) Lsh(n int) Int128 {
	if n < 0 {
		panic("fat.Int128.Lsh: negative shift")
	}
	// a copy of Uint128.Lsh, a conversion wrapper exceeds the inline budget
	s, t := n&63, ^n&63
	r1 := x.L1<<s | x.L0>>1>>t
	r0 := x.L0 << s
	if n&64 != 0 {
		r1, r0 = r0, 0
	}
	// nothing is left past 127 bits
	if n > 127 {
		r1, r0 = 0, 0
	}
	return Int128{r0, r1}
}

// Rsh returns x >> n rounded toward negative infinity, it panics for n < 0
func (x Int128) Rsh(n int) Int128 {
	if n < 0 {
		panic("fat.Int128.Rsh: negative shift")
	}
	// as Uint128.Rsh with the sign as fill, past 127 bits only the sign is left
	n = min(n, 127)
	s, t := n&63, ^n&63
	r0 := x.L0>>s | x.L1<<1<<t
	r1 := uint64(int64(x.L1) >> s)
	if n&64 != 0 {
		r0, r1 = r1, uint64(int64(x.L1)>>63)
	}
	return Int128{r0, r1}
}

// Xor returns x ^ y
func (x Int128) Xor(y Int128) Int128 {
	return Int128(Uint128(x).Xor(Uint128(y)))
}

// And returns x & y
func (x Int128) And(y Int128) Int128 {
	return Int128(Uint128(x).And(Uint128(y)))
}

// AndNot returns x &^ y
func (x Int128) AndNot(y Int128) Int128 {
	return Int128(Uint128(x).AndNot(Uint128(y)))
}

// Or returns x | y
func (x Int128) Or(y Int128) Int128 {
	return Int128(Uint128(x).Or(Uint128(y)))
}

// Not returns ^x
func (x Int128) Not() Int128 {
	return Int128(Uint128(x).Not())
}

// Bit returns bit i of x in two's complement, the sign bit past 127.
// It panics for i < 0
func (x Int128) Bit(i int) uint {
	if i < 0 {
		panic("fat.Int128.Bit: negative index")
	}
	// past bit 127 every bit is the sign
	u := min(uint64(i), 127)
	// the limb by a conditional move, x stays in registers
	w := x.L0
	if u&64 != 0 {
		w = x.L1
	}
	return uint(w >> (u & 63) & 1)
}

// WithBit returns x with bit i set to bit, a bit past 127 is dropped.
// It panics for i < 0 and for bit > 1
func (x Int128) WithBit(i int, bit uint) Int128 {
	if i < 0 {
		panic("fat.Int128.WithBit: negative index")
	}
	if bit > 1 {
		panic("fat.Int128.WithBit: bit not 0 or 1")
	}
	u := uint64(i)
	// a shift past 63 is 0: only the limb of bit i gets a mask, none past 127
	b := -uint64(bit)
	x.L0 ^= (x.L0 ^ b) & (1 << u)
	x.L1 ^= (x.L1 ^ b) & (1 << (u - 64))
	return x
}

// --- add ------------------------------------------------

// signed overflow of the o forms: the operands share a sign the result lacks

// Add returns x + y, it panics on overflow
func (x Int128) Add(y Int128) Int128 {
	// written out: a call of Addo leaves the NOP of its inline mark in a loop
	x1 := x.L1
	var carry uint64
	x.L0, carry = bits.Add64(x.L0, y.L0, 0)
	x.L1, _ = bits.Add64(x.L1, y.L1, carry)
	if int64((x1^x.L1)&(y.L1^x.L1)) < 0 {
		panic("fat.Int128.Add: overflow")
	}
	return x
}

// Addo returns Addu and whether x + y overflowed
func (x Int128) Addo(y Int128) (r Int128, overflow bool) {
	x1 := x.L1
	var carry uint64
	x.L0, carry = bits.Add64(x.L0, y.L0, 0)
	x.L1, _ = bits.Add64(x.L1, y.L1, carry)
	return x, int64((x1^x.L1)&(y.L1^x.L1)) < 0
}

// Addu returns x + y wrapped around as on a Go integer
func (x Int128) Addu(y Int128) Int128 {
	var carry uint64
	x.L0, carry = bits.Add64(x.L0, y.L0, 0)
	x.L1, _ = bits.Add64(x.L1, y.L1, carry)
	return x
}

// Add64 returns x + y, it panics on overflow
func (x Int128) Add64(y int64) Int128 {
	// written out: a call of Add64o leaves the NOP of its inline mark in a loop
	x1 := x.L1
	ext := uint64(y >> 63)
	var carry uint64
	x.L0, carry = bits.Add64(x.L0, uint64(y), 0)
	x.L1, _ = bits.Add64(x.L1, ext, carry)
	if int64((x1^x.L1)&(ext^x.L1)) < 0 {
		panic("fat.Int128.Add64: overflow")
	}
	return x
}

// Add64o returns Add64u and whether x + y overflowed
func (x Int128) Add64o(y int64) (r Int128, overflow bool) {
	x1 := x.L1
	ext := uint64(y >> 63)
	var carry uint64
	x.L0, carry = bits.Add64(x.L0, uint64(y), 0)
	x.L1, _ = bits.Add64(x.L1, ext, carry)
	return x, int64((x1^x.L1)&(ext^x.L1)) < 0
}

// Add64u returns x + y wrapped around as on a Go integer
func (x Int128) Add64u(y int64) Int128 {
	ext := uint64(y >> 63)
	var carry uint64
	x.L0, carry = bits.Add64(x.L0, uint64(y), 0)
	x.L1, _ = bits.Add64(x.L1, ext, carry)
	return x
}

// --- sub ------------------------------------------------

// signed overflow of the o forms: the signs differ, the result lost that of x

// Sub returns x - y, it panics on overflow
func (x Int128) Sub(y Int128) Int128 {
	// written out: a call of Subo leaves the NOP of its inline mark in a loop
	x1 := x.L1
	var carry uint64
	x.L0, carry = bits.Sub64(x.L0, y.L0, 0)
	x.L1, _ = bits.Sub64(x.L1, y.L1, carry)
	if int64((x1^y.L1)&(x1^x.L1)) < 0 {
		panic("fat.Int128.Sub: overflow")
	}
	return x
}

// Subo returns Subu and whether x - y overflowed
func (x Int128) Subo(y Int128) (r Int128, overflow bool) {
	x1 := x.L1
	var carry uint64
	x.L0, carry = bits.Sub64(x.L0, y.L0, 0)
	x.L1, _ = bits.Sub64(x.L1, y.L1, carry)
	return x, int64((x1^y.L1)&(x1^x.L1)) < 0
}

// Subu returns x - y wrapped around as on a Go integer
func (x Int128) Subu(y Int128) Int128 {
	var carry uint64
	x.L0, carry = bits.Sub64(x.L0, y.L0, 0)
	x.L1, _ = bits.Sub64(x.L1, y.L1, carry)
	return x
}

// Sub64 returns x - y, it panics on overflow
func (x Int128) Sub64(y int64) Int128 {
	// written out: a call of Sub64o leaves the NOP of its inline mark in a loop
	x1 := x.L1
	ext := uint64(y >> 63)
	var carry uint64
	x.L0, carry = bits.Sub64(x.L0, uint64(y), 0)
	x.L1, _ = bits.Sub64(x.L1, ext, carry)
	if int64((x1^ext)&(x1^x.L1)) < 0 {
		panic("fat.Int128.Sub64: overflow")
	}
	return x
}

// Sub64o returns Sub64u and whether x - y overflowed
func (x Int128) Sub64o(y int64) (r Int128, overflow bool) {
	x1 := x.L1
	ext := uint64(y >> 63)
	var carry uint64
	x.L0, carry = bits.Sub64(x.L0, uint64(y), 0)
	x.L1, _ = bits.Sub64(x.L1, ext, carry)
	return x, int64((x1^ext)&(x1^x.L1)) < 0
}

// Sub64u returns x - y wrapped around as on a Go integer
func (x Int128) Sub64u(y int64) Int128 {
	ext := uint64(y >> 63)
	var carry uint64
	x.L0, carry = bits.Sub64(x.L0, uint64(y), 0)
	x.L1, _ = bits.Sub64(x.L1, ext, carry)
	return x
}

// --- mul ------------------------------------------------

// Mul returns x * y, it panics on overflow
func (x Int128) Mul(y Int128) Int128 {
	r, c := x.Mulo(y)
	if c {
		panic("fat.Int128.Mul: overflow")
	}
	return r
}

// Mulo returns Mulu and whether x * y overflowed
func (x Int128) Mulo(y Int128) (r Int128, overflow bool) {
	// the product with no abs: x * y as unsigned is p0..p3.
	// A 4th product costs less than |x|, |y| and the negation
	h0, p0 := bits.Mul64(x.L0, y.L0)
	h1, p1 := bits.Mul64(x.L1, y.L0)
	g0, q1 := bits.Mul64(x.L0, y.L1)
	g1, q2 := bits.Mul64(x.L1, y.L1)
	p1, c := bits.Add64(p1, h0, 0)
	p2, _ := bits.Add64(h1, 0, c)
	q2, c = bits.Add64(q2, g0, 0)
	p3, _ := bits.Add64(g1, 0, c)
	r1, c := bits.Add64(p1, q1, 0)
	p2, c = bits.Add64(p2, q2, c)
	p3, _ = bits.Add64(p3, 0, c)
	// signed = unsigned - 2^128*x if y < 0 - 2^128*y if x < 0
	my := uint64(int64(y.L1) >> 63)
	mx := uint64(int64(x.L1) >> 63)
	p2, b := bits.Sub64(p2, x.L0&my, 0)
	p3, _ = bits.Sub64(p3, x.L1&my, b)
	p2, b = bits.Sub64(p2, y.L0&mx, 0)
	p3, _ = bits.Sub64(p3, y.L1&mx, b)
	// fits if limbs 2 and 3 only extend the sign of r1
	s := uint64(int64(r1) >> 63)
	return Int128{p0, r1}, (p2^s)|(p3^s) != 0
}

// Mulu returns x * y wrapped around as on a Go integer
func (x Int128) Mulu(y Int128) Int128 {
	return Int128(Uint128(x).Mulu(Uint128(y)))
}

// Mule returns the 256-bit product x * y
func (x Int128) Mule(y Int128) Int256 {
	// Uint128.Mule written out, faster than its call in our benchmarks
	h0, p0 := bits.Mul64(x.L0, y.L0)
	h1, l1 := bits.Mul64(x.L1, y.L0)
	h2, l2 := bits.Mul64(x.L0, y.L1)
	h3, l3 := bits.Mul64(x.L1, y.L1)
	p1, c := bits.Add64(h0, l1, 0)
	p2, c := bits.Add64(h1, l3, c)
	p3 := h3 + c
	p1, c = bits.Add64(p1, l2, 0)
	p2, c = bits.Add64(p2, h2, c)
	p3 += c
	// signed = unsigned - 2^128*y if x < 0 - 2^128*x if y < 0,
	// the corrections summed off the path of the product
	mx := uint64(int64(x.L1) >> 63)
	my := uint64(int64(y.L1) >> 63)
	k0, c := bits.Add64(y.L0&mx, x.L0&my, 0)
	k1, _ := bits.Add64(y.L1&mx, x.L1&my, c)
	p2, c = bits.Sub64(p2, k0, 0)
	p3, _ = bits.Sub64(p3, k1, c)
	return Int256{p0, p1, p2, p3}
}

// Mul64 returns x * y, it panics on overflow
func (x Int128) Mul64(y int64) Int128 {
	// x0 * y and x1 * y as unsigned, then signed high limbs by masks
	h0, r0 := bits.Mul64(x.L0, uint64(y))
	h1, r1 := bits.Mul64(x.L1, uint64(y))
	// signed high limb of x0 * y, it fits 64 bits
	h0 -= x.L0 & uint64(y>>63)
	// a signed h0 adds its sign to limb 2
	r1, c := bits.Add64(r1, h0, 0)
	// fits if limb 2, h1 + c - sign(h0) - corrections, extends the sign of r1.
	// One expression, so it inlines
	if h1+c+r1>>63 != h0>>63+x.L1&uint64(y>>63)+uint64(y)&-(x.L1>>63) {
		panic("fat.Int128.Mul64: overflow")
	}
	return Int128{r0, r1}
}

// Mul64o returns Mul64u and whether x * y overflowed
func (x Int128) Mul64o(y int64) (r Int128, overflow bool) {
	// x0 * y and x1 * y as unsigned, then signed high limbs by masks
	h0, r0 := bits.Mul64(x.L0, uint64(y))
	h1, r1 := bits.Mul64(x.L1, uint64(y))
	// signed high limb of x0 * y, it fits 64 bits
	h0 -= x.L0 & uint64(y>>63)
	// a signed h0 adds its sign to limb 2
	r1, c := bits.Add64(r1, h0, 0)
	// fits if limb 2, h1 + c - sign(h0) - corrections, extends the sign of r1.
	// One expression, so it inlines
	return Int128{r0, r1}, h1+c+r1>>63 != h0>>63+x.L1&uint64(y>>63)+uint64(y)&-(x.L1>>63)
}

// Mul64u returns x * y wrapped around as on a Go integer
func (x Int128) Mul64u(y int64) Int128 {
	r := Uint128(x).Mul64u(uint64(y))
	// a negative y adds -2^64 * x
	r.L1 -= x.L0 & uint64(y>>63)
	return Int128(r)
}

// --- pow ------------------------------------------------

// Exp returns x^y, 0^0 is 1, it panics for y < 0 and on overflow
func (x Int128) Exp(y Int128) Int128 {
	if int64(y.L1) < 0 {
		panic("fat.Int128.Exp: negative exponent")
	}
	r, over := x.expo(y)
	if over {
		panic("fat.Int128.Exp: overflow")
	}
	return r
}

// Expo returns Expu and whether x^y overflowed, it panics for y < 0
func (x Int128) Expo(y Int128) (r Int128, overflow bool) {
	if int64(y.L1) < 0 {
		panic("fat.Int128.Expo: negative exponent")
	}
	return x.expo(y)
}

// expo is Expo for y >= 0 only, a negative y gives a wrong result.
// It returns |x|^y with the sign of x for an odd y
func (x Int128) expo(y Int128) (r Int128, over bool) {
	a := x.UnsignedAbs()
	if y.L1 != 0 || y.L0 >= 128 {
		a, over = a.Expu(Uint128(y)), a.L1 != 0 || a.L0 > 1
	} else {
		a, over = a.exp(y.L0)
	}
	// n is 1 for a negative power: it fits up to 2^127, a positive one below
	n := x.L1 >> 63 & y.L0 & 1
	m := -n
	_, b := bits.Sub64(^m, a.L0, 0)
	_, b = bits.Sub64(1<<63-1+n, a.L1, b)
	r0, c := bits.Sub64(a.L0^m, m, 0)
	r1, _ := bits.Sub64(a.L1^m, m, c)
	return Int128{r0, r1}, over || b != 0
}

// Expu returns x^y wrapped around as on a Go integer, 0^0 is 1.
// It panics for y < 0
func (x Int128) Expu(y Int128) Int128 {
	if int64(y.L1) < 0 {
		panic("fat.Int128.Expu: negative exponent")
	}
	return Int128(Uint128(x).Expu(Uint128(y)))
}

// --- roots ----------------------------------------------

// Sqrt returns the square root of x rounded down, it panics for x < 0
func (x Int128) Sqrt() Int128 {
	if int64(x.L1) < 0 {
		panic("fat.Int128.Sqrt: negative value")
	}
	return Int128(Uint128(x).Sqrt())
}

// Log10 returns log10(x) rounded down, it panics for x <= 0
func (x Int128) Log10() int {
	// a negative x stays within the table as a large unsigned one,
	// then one sign test takes it and the -1 of 0
	n := Uint128(x).log10()
	if int64(x.L1)|int64(n) < 0 {
		panic("fat.Int128.Log10: non-positive value")
	}
	return n
}
