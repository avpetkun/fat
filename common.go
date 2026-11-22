package fat

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"math/bits"
	"sync"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/avpetkun/fat/internal/arch"
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

// --- decimal digits -------------------------------------

// notDigits8 returns 0 when all 8 bytes of x are ASCII digits
func notDigits8(x uint64) uint64 {
	return (x&0xF0F0F0F0F0F0F0F0 | (x+0x0606060606060606)&0xF0F0F0F0F0F0F0F0>>4) ^ 0x3333333333333333
}

// conv8 converts 8 ASCII digits to a number, the first in the low byte.
// x must pass notDigits8, other bytes give garbage
func conv8(x uint64) uint64 {
	x = (x & 0x0F0F0F0F0F0F0F0F) * 2561 >> 8
	x = (x & 0x00FF00FF00FF00FF) * 6553601 >> 16
	return (x & 0x0000FFFF0000FFFF) * 42949672960001 >> 32
}

// pow10Table holds 10^i for every i that fits a uint64
var pow10Table = [20]uint64{
	1, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9,
	1e10, 1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19,
}

// count64Digits returns the decimal digit count of v.
// v must not be 0, which gives 19
func count64Digits(v uint64) int {
	t := &count64Table[bits.LeadingZeros64(v)&63]
	nd := t.n
	if v >= t.lim {
		nd++
	}
	return nd
}

// count64Table holds, by leading zero count, the digit count of the least
// value and the next power of ten
var count64Table = func() (t [64]struct {
	lim uint64
	n   int
}) {
	for lz := range 64 {
		v := uint64(1) << (63 - lz)
		n := 1
		for n < 20 && v >= pow10Table[n] {
			n++
		}
		t[lz].n = n
		t[lz].lim = math.MaxUint64
		if n < 20 {
			t[lz].lim = pow10Table[n]
		}
	}
	return t
}()

// digits8 converts x to 8 ASCII digits, the first in the low byte.
// x must be below 1e8, larger x gives garbage
func digits8(x uint64) uint64 {
	// 109951163 / 2^40 ~ 1/10000
	hi := x * 109951163 >> 40
	// two 4-digit halves side by side, hi + (x-hi*10000)<<32 as one msub
	merged := x<<32 - hi*(10000<<32-1)
	// 10486 / 2^20 ~ 1/100
	top := ((merged * 10486) >> 20) & (0x7F<<32 | 0x7F)
	// four 2-digit numbers in 16-bit lanes
	hundreds := merged<<16 - top*(100<<16-1)
	// 103 / 2^10 ~ 1/10
	tens := (hundreds * 103) >> 10
	tens &= 0xF<<48 | 0xF<<32 | 0xF<<16 | 0xF
	return hundreds<<8 + 0x3030303030303030 - tens*(10<<8-1)
}

// --- hex digits -----------------------------------------

// hexDigits8 packs 8 ASCII hex digits into 32 bits, the first on top.
// x must pass notHexDigits8, other bytes give garbage
func hexDigits8(x uint64) uint64 {
	// a letter has bit 6 set and is worth its low nibble plus 9
	t := x >> 6 & 0x0101010101010101
	x = x&0x0F0F0F0F0F0F0F0F + t + t<<3
	// nibbles close up pair by pair
	x = (x | x>>4) & 0x00FF00FF00FF00FF
	x = (x | x>>8) & 0x0000FFFF0000FFFF
	return (x | x>>16) & 0xFFFFFFFF
}

// notHexDigits8 returns 0 when all 8 bytes of x are ASCII hex digits
func notHexDigits8(x uint64) uint64 {
	// 7-bit lanes, so an add cannot carry into the next byte
	c := x & 0x7F7F7F7F7F7F7F7F
	// bit 7 is set for '0'..'9'
	d := (c + 0x5050505050505050) &^ (c + 0x4646464646464646)
	// and for 'a'..'f' with the case folded
	c |= 0x2020202020202020
	l := (c + 0x1F1F1F1F1F1F1F1F) &^ (c + 0x1919191919191919)
	return (x | ^(d | l)) & 0x8080808080808080
}

// hexChars8 spreads x < 2^32 into 8 lowercase hex digits, the first on top
func hexChars8(x uint64) uint64 {
	x = (x | x<<16) & 0x0000FFFF0000FFFF
	x = (x | x<<8) & 0x00FF00FF00FF00FF
	x = (x | x<<4) & 0x0F0F0F0F0F0F0F0F
	// 1 in each byte holding 10..15, those skip 'a'-'0'-10 = 39 chars
	m := (x + 0x0606060606060606) >> 4 & 0x0101010101010101
	return x + 0x3030303030303030 + m*39
}

// --- decimal output -------------------------------------

// 10^19 is the largest power of ten in a limb, its top bit is set
const decimalChunk = 1e19

// decimalChunkReciprocal is (2^128-1)/decimalChunk - 2^64, for div2by1
const decimalChunkReciprocal = 0xd83c94fb6d2ac34a

// prettyDecimal returns digits d with sep between groups of 3, a minus first
// when neg; d must hold 1 to 78 digits
func prettyDecimal(d []byte, neg bool, sep byte) string {
	// 78 digits and 25 separators, a signed type has a digit less
	var buf [103]byte
	sign := 0
	if neg {
		sign = 1
	}
	// the first digit overwrites the minus when there is no sign
	buf[0] = '-'
	h := (len(d)-1)%3 + 1
	j := sign + copy(buf[sign:], d[:h])
	for i := h; i+2 < len(d); i += 3 {
		buf[j], buf[j+1], buf[j+2], buf[j+3] = sep, d[i], d[i+1], d[i+2]
		j += 4
	}
	return string(buf[:j])
}

// appendDecimalParts appends the sign, head and the 19-digit chunks c1 to cn.
// It needs 0 < head < 1e19, chunks below 1e19 and n from 1 to 4,
// other values give garbage
func appendDecimalParts(dst []byte, neg bool, head, c1, c2, c3, c4 uint64, n int) []byte {
	sign := 0
	if neg {
		sign = 1
	}
	nd := count64Digits(head)
	size := sign + nd + 19*n
	start := len(dst)
	if cap(dst)-start < size {
		dst = append(dst, make([]byte, size)...)
	}
	dst = dst[:start+size]
	b := dst[start:]
	// the first digit overwrites the sign when there is none
	b[0] = '-'
	// the head as in appendDecimal64, the chunks overwrite what it writes past
	a := b[sign:]
	v := head
	switch {
	case v < 1e8:
		binary.LittleEndian.PutUint64(a, digits8(v)>>(8*uint(8-nd)&63))
	case v < 1e16:
		hi := v / 1e8
		lo := v - hi*1e8
		e := a[nd-8:]
		binary.LittleEndian.PutUint64(a, digits8(hi)>>(8*uint(16-nd)&63))
		binary.LittleEndian.PutUint64(e, digits8(lo))
	default:
		hi := v / 1e8
		lo := v - hi*1e8
		top := (hi >> 8) * 11529215047 >> 52
		mid := hi - top*1e8
		e := a[nd-16:]
		binary.LittleEndian.PutUint64(a, digits8(top)>>(8*uint(24-nd)&63))
		binary.LittleEndian.PutUint64(e, digits8(mid))
		binary.LittleEndian.PutUint64(e[8:], digits8(lo))
	}
	b = b[sign+nd:]
	switch n {
	case 1:
		put19Digits((*[19]byte)(b), c1)
	case 2:
		put38Digits((*[38]byte)(b), c1, c2)
	case 3:
		put57Digits((*[57]byte)(b), c1, c2, c3)
	default:
		put76Digits((*[76]byte)(b), c1, c2, c3, c4)
	}
	return dst
}

// put19Digits writes r < 1e19 as exactly 19 digits
func put19Digits(b *[19]byte, r uint64) {
	// both quotients straight from r, not one after the other
	hi := r / 1e16
	q := r / 1e8
	// the 4th byte is overwritten by the next store
	binary.LittleEndian.PutUint32(b[:], uint32(digits3(hi)))
	binary.LittleEndian.PutUint64(b[3:], digits8(q-hi*1e8))
	binary.LittleEndian.PutUint64(b[11:], digits8(r-q*1e8))
}

// put38Digits writes a and c < 1e19 as exactly 19 digits each
func put38Digits(b *[38]byte, a, c uint64) {
	ha, hc := a/1e16, c/1e16
	qa, qc := a/1e8, c/1e8
	// both 3-digit heads at once, in 32-bit lanes
	h := digits3(ha | hc<<32)
	// the 4th byte of a head is overwritten by the next store
	binary.LittleEndian.PutUint32(b[:], uint32(h))
	binary.LittleEndian.PutUint64(b[3:], digits8(qa-ha*1e8))
	binary.LittleEndian.PutUint64(b[11:], digits8(a-qa*1e8))
	binary.LittleEndian.PutUint32(b[19:], uint32(h>>32))
	binary.LittleEndian.PutUint64(b[22:], digits8(qc-hc*1e8))
	binary.LittleEndian.PutUint64(b[30:], digits8(c-qc*1e8))
}

// put57Digits writes a, c and e < 1e19 as exactly 19 digits each
func put57Digits(b *[57]byte, a, c, e uint64) {
	ha, hc, he := a/1e16, c/1e16, e/1e16
	qa, qc, qe := a/1e8, c/1e8, e/1e8
	// 3-digit heads two at a time, in 32-bit lanes
	h := digits3(ha | hc<<32)
	g := digits3(he)
	// the 4th byte of a head is overwritten by the next store
	binary.LittleEndian.PutUint32(b[:], uint32(h))
	binary.LittleEndian.PutUint64(b[3:], digits8(qa-ha*1e8))
	binary.LittleEndian.PutUint64(b[11:], digits8(a-qa*1e8))
	binary.LittleEndian.PutUint32(b[19:], uint32(h>>32))
	binary.LittleEndian.PutUint64(b[22:], digits8(qc-hc*1e8))
	binary.LittleEndian.PutUint64(b[30:], digits8(c-qc*1e8))
	binary.LittleEndian.PutUint32(b[38:], uint32(g))
	binary.LittleEndian.PutUint64(b[41:], digits8(qe-he*1e8))
	binary.LittleEndian.PutUint64(b[49:], digits8(e-qe*1e8))
}

// put76Digits writes a, c, e and f < 1e19 as exactly 19 digits each
func put76Digits(b *[76]byte, a, c, e, f uint64) {
	ha, hc, he, hf := a/1e16, c/1e16, e/1e16, f/1e16
	qa, qc, qe, qf := a/1e8, c/1e8, e/1e8, f/1e8
	// 3-digit heads two at a time, in 32-bit lanes
	h := digits3(ha | hc<<32)
	g := digits3(he | hf<<32)
	// the 4th byte of a head is overwritten by the next store
	binary.LittleEndian.PutUint32(b[:], uint32(h))
	binary.LittleEndian.PutUint64(b[3:], digits8(qa-ha*1e8))
	binary.LittleEndian.PutUint64(b[11:], digits8(a-qa*1e8))
	binary.LittleEndian.PutUint32(b[19:], uint32(h>>32))
	binary.LittleEndian.PutUint64(b[22:], digits8(qc-hc*1e8))
	binary.LittleEndian.PutUint64(b[30:], digits8(c-qc*1e8))
	binary.LittleEndian.PutUint32(b[38:], uint32(g))
	binary.LittleEndian.PutUint64(b[41:], digits8(qe-he*1e8))
	binary.LittleEndian.PutUint64(b[49:], digits8(e-qe*1e8))
	binary.LittleEndian.PutUint32(b[57:], uint32(g>>32))
	binary.LittleEndian.PutUint64(b[60:], digits8(qf-hf*1e8))
	binary.LittleEndian.PutUint64(b[68:], digits8(f-qf*1e8))
}

// digits3 converts x < 1000 in each 32-bit lane to 3 ASCII digits and a 0
func digits3(x uint64) uint64 {
	// 41 / 2^12 ~ 1/100, 103 / 2^10 ~ 1/10, no lane overflows
	h := (x * 41 >> 12) & (0xF<<32 | 0xF)
	r := x - h*100
	t := (r * 103 >> 10) & (0xF<<32 | 0xF)
	return h | t<<8 | (r-t*10)<<16 | 0x00303030_00303030
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

// appendDecimal64 appends decimal v
func appendDecimal64(dst []byte, v uint64) []byte {
	nd := count64Digits(v | 1)
	n := len(dst)
	if cap(dst)-n < nd {
		// append of make allocates twice without optimizations; nd <= 20
		var zeros [24]byte
		dst = append(dst, zeros[:nd]...)[:n]
	}
	dst = dst[:n+nd]
	a := dst[n:]
	// branch on the value, nd is ready much later
	switch {
	case v < 1000:
		// 41 / 2^12 ~ 1/100, 103 / 2^10 ~ 1/10
		h := v * 41 >> 12
		r := v - h*100
		t := r * 103 >> 10
		// one bounds check for all three stores
		_ = a[nd-1]
		// a leading zero is overwritten by the next digit
		a[0] = byte('0' + h)
		a[(nd-1)>>1] = byte('0' + t)
		a[nd-1] = byte('0' + r - t*10)
	case v < 1e8:
		// drop leading zeros
		x := digits8(v) >> (8 * uint(8-nd) & 63)
		// two overlapping 4-byte stores
		b := a[nd-4:]
		binary.LittleEndian.PutUint32(a, uint32(x))
		binary.LittleEndian.PutUint32(b, uint32(x>>(8*uint(nd-4)&63)))
	case v < 1e16:
		hi := v / 1e8
		lo := v - hi*1e8
		// tail of the first block is overwritten by the second
		b := a[nd-8:]
		binary.LittleEndian.PutUint64(a, digits8(hi)>>(8*uint(16-nd)&63))
		binary.LittleEndian.PutUint64(b, digits8(lo))
	default:
		hi := v / 1e8
		lo := v - hi*1e8
		// 1e8 = 2^8 * 390625, 11529215047 / 2^52 ~ 1/390625
		top := (hi >> 8) * 11529215047 >> 52
		mid := hi - top*1e8
		b := a[nd-16:]
		binary.LittleEndian.PutUint64(a, digits8(top)>>(8*uint(24-nd)&63))
		binary.LittleEndian.PutUint64(b, digits8(mid))
		binary.LittleEndian.PutUint64(b[8:], digits8(lo))
	}
	return dst
}

// formatInt64 returns decimal v, for error texts
func formatInt64(v int64) string {
	// a minus and 19 digits, string copies them out
	var buf [20]byte
	neg := v < 0
	u := uint64(v)
	if neg {
		u = -u
	}
	return string(appendDecimalSign64(buf[:0], neg, u))
}

// appendDecimalSign64 appends a minus when neg, then v
func appendDecimalSign64(dst []byte, neg bool, v uint64) []byte {
	sign := 0
	if neg {
		sign = 1
	}
	// v|1 has the digit count of v, and 1 digit for zero
	nd := count64Digits(v | 1)
	size := sign + nd
	start := len(dst)
	if cap(dst)-start < size {
		dst = append(dst, make([]byte, size)...)
	}
	dst = dst[:start+size]
	b := dst[start:]
	// the first digit overwrites the sign when there is none
	b[0] = '-'
	// the switch of appendDecimal64 written out, a call less
	a := b[sign:]
	switch {
	case v < 1000:
		h := v * 41 >> 12
		r := v - h*100
		t := r * 103 >> 10
		_ = a[nd-1]
		a[0] = byte('0' + h)
		a[(nd-1)>>1] = byte('0' + t)
		a[nd-1] = byte('0' + r - t*10)
	case v < 1e8:
		x := digits8(v) >> (8 * uint(8-nd) & 63)
		e := a[nd-4:]
		binary.LittleEndian.PutUint32(a, uint32(x))
		binary.LittleEndian.PutUint32(e, uint32(x>>(8*uint(nd-4)&63)))
	case v < 1e16:
		hi := v / 1e8
		lo := v - hi*1e8
		e := a[nd-8:]
		binary.LittleEndian.PutUint64(a, digits8(hi)>>(8*uint(16-nd)&63))
		binary.LittleEndian.PutUint64(e, digits8(lo))
	default:
		hi := v / 1e8
		lo := v - hi*1e8
		top := (hi >> 8) * 11529215047 >> 52
		mid := hi - top*1e8
		e := a[nd-16:]
		binary.LittleEndian.PutUint64(a, digits8(top)>>(8*uint(24-nd)&63))
		binary.LittleEndian.PutUint64(e, digits8(mid))
		binary.LittleEndian.PutUint64(e[8:], digits8(lo))
	}
	return dst
}

// --- decimal input --------------------------------------

// sciParts splits s of the form m e n at its last e or E for FromDec and
// Scan: n is decimal digits after an optional plus, not empty, and m all
// before the e, left to the parser to check. It goes from the end, as n is
// short and m can be long. An n past 999 comes back as 999,
// beyond every power of ten that fits
func sciParts(s []byte) (m []byte, n int, ok bool) {
	k := len(s)
	for k > 0 && s[k-1]-'0' <= 9 {
		k--
	}
	j := k
	if j > 0 && s[j-1] == '+' {
		j--
	}
	if k == len(s) || j == 0 || s[j-1]|0x20 != 'e' {
		return nil, 0, false
	}
	for _, c := range s[k:] {
		n = min(n*10+int(c-'0'), 999)
	}
	return s[:j-1], n, true
}

// isMinusZero reports whether m is a minus and zeros only, as -0 or -000
func isMinusZero(m []byte) bool {
	if len(m) < 2 || m[0] != '-' {
		return false
	}
	for _, c := range m[1:] {
		if c != '0' {
			return false
		}
	}
	return true
}

// parseDecimalUint64 parses decimal s with no sign, false when it cannot.
// No error value: building one would cost a call and a frame on every parse
func parseDecimalUint64(s []byte) (uint64, bool) {
	n := len(s)
	if n <= 8 {
		if n == 0 {
			return 0, false
		}
		// chars go to the top n bytes
		sh := 8 * uint(8-n) & 63
		var x uint64
		if n >= 4 {
			// two overlapping 4-byte loads
			x = uint64(binary.LittleEndian.Uint32(s))<<sh | uint64(binary.LittleEndian.Uint32(s[n-4:]))<<32
		} else {
			// first, middle and last byte cover 1..3 chars
			x = (uint64(s[0]) | uint64(s[(n-1)>>1])<<(8*uint((n-1)>>1)) | uint64(s[n-1])<<(8*uint(n-1))) << sh
		}
		// left-pad with '0' to a full block
		x |= 0x3030303030303030 &^ (^uint64(0) << sh)
		if notDigits8(x) != 0 {
			return 0, false
		}
		return conv8(x), true
	}
	if n <= 16 {
		// two overlapping blocks
		a := binary.LittleEndian.Uint64(s)
		b := binary.LittleEndian.Uint64(s[n-8:])
		if notDigits8(a)|notDigits8(b) != 0 {
			return 0, false
		}
		// drop digits shared with a
		b &= ^uint64(0) << (8 * uint(16-n))
		return conv8(a)*pow10Table[n-8] + conv8(b), true
	}
	a := binary.LittleEndian.Uint64(s)
	b := binary.LittleEndian.Uint64(s[8:])
	if n <= 19 {
		c := binary.LittleEndian.Uint64(s[n-8:])
		if notDigits8(a)|notDigits8(b)|notDigits8(c) != 0 {
			return 0, false
		}
		// drop digits shared with b
		c &= ^uint64(0) << (8 * uint(24-n))
		return (conv8(a)*1e8+conv8(b))*pow10Table[n-16] + conv8(c), true
	}
	// first 20 chars, the last block adds 4 digits
	c := binary.LittleEndian.Uint64(s[12:])
	if notDigits8(a)|notDigits8(b)|notDigits8(c) != 0 {
		return 0, false
	}
	hi, lo := bits.Mul64(conv8(a)*1e8+conv8(b), 1e4)
	v, carry := bits.Add64(lo, conv8(c&^0xFFFFFFFF), 0)
	rest := s[20:]
	if hi|carry != 0 {
		return 0, false
	}
	// more digits fit only after leading zeros
	var d byte
	var m uint64
	for _, ch := range rest {
		d = ch - '0'
		if d > 9 {
			return 0, false
		}
		// v*10 must not wrap
		if v > math.MaxUint64/10 {
			return 0, false
		}
		m = v*10 + uint64(d)
		if m < v {
			return 0, false
		}
		v = m
	}
	return v, true
}

// --- fmt ------------------------------------------------

// formatBase returns the base of a Format verb, 0 for an unknown verb
func formatBase(verb rune) int {
	switch verb {
	case 'b':
		return 2
	case 'o', 'O':
		return 8
	case 'd', 's', 'v':
		return 10
	case 'x', 'X':
		return 16
	}
	return 0
}

// formatPlain reports whether verb has no flags, width or precision,
// so the digits are all Format writes
func formatPlain(f fmt.State, verb rune) bool {
	_, width := f.Width()
	_, precision := f.Precision()
	return !width && !precision && !f.Flag('+') && !f.Flag(' ') && !f.Flag('#') && verb != 'O'
}

// upperHex turns the letters of digits to uppercase for %X
func upperHex(digits []byte, verb rune) []byte {
	if verb == 'X' {
		for i, d := range digits {
			// letters have bit 6 set and digits do not; no branch, they mix
			digits[i] = d &^ (d >> 1 & 0x20)
		}
	}
	return digits
}

// fmtBufs holds the output buffers of Format.
// A slice given to f.Write goes to the heap anyway, and a Writer must not
// keep it, so pooling saves an allocation per call in our benchmarks
var fmtBufs = sync.Pool{New: func() any { return new([]byte) }}

// getFmtBuf takes an empty buffer from fmtBufs
func getFmtBuf() *[]byte { return fmtBufs.Get().(*[]byte) }

// putFmtBuf returns b to fmtBufs in p, a buffer over 1 KB is dropped
func putFmtBuf(p *[]byte, b []byte) {
	if cap(b) <= 1024 {
		*p = b[:0]
		fmtBufs.Put(p)
	}
}

// formatNumber writes the magnitude digits with what the flags of f add,
// as big.Int.Format does.
// A sign: '-', or '+' or ' ' by the flags.
// A prefix: 0b, 0, 0x or 0X by '#', and 0o for %O.
// At least precision digits, and at least width bytes: spaces on the left,
// on the right after '-', or zeros after the prefix by '0' and no precision.
// A zero with precision 0 writes nothing, width included
func formatNumber(f fmt.State, verb rune, neg bool, digits []byte) {
	prec, hasPrec := f.Precision()
	if hasPrec && prec == 0 && len(digits) == 1 && digits[0] == '0' {
		return
	}
	// sign and prefix, 3 bytes at most
	var head [3]byte
	n := 0
	if neg || f.Flag('+') || f.Flag(' ') {
		head[0] = '-'
		if !neg {
			// '+' wins over ' '
			head[0] = ' '
			if f.Flag('+') {
				head[0] = '+'
			}
		}
		n = 1
	}
	if verb == 'O' || f.Flag('#') && verb != 'd' && verb != 's' && verb != 'v' {
		head[n] = '0'
		n++
		switch verb {
		case 'b', 'x', 'X':
			head[n] = byte(verb)
			n++
		case 'O':
			head[n] = 'o'
			n++
		}
	}
	zeros := 0
	if hasPrec && prec > len(digits) {
		zeros = prec - len(digits)
	}
	left, right := 0, 0
	if width, ok := f.Width(); ok {
		if pad := width - n - zeros - len(digits); pad > 0 {
			switch {
			case f.Flag('-'):
				right = pad
			case f.Flag('0') && !hasPrec:
				zeros += pad
			default:
				left = pad
			}
		}
	}
	// the whole output in one pooled slice of its size for one write
	p := getFmtBuf()
	size := left + n + zeros + len(digits) + right
	out := *p
	if cap(out) < size {
		out = make([]byte, size)
	}
	out = out[:size]
	i := 0
	for ; i < left; i++ {
		out[i] = ' '
	}
	i += copy(out[i:], head[:n])
	for end := i + zeros; i < end; i++ {
		out[i] = '0'
	}
	if verb == 'X' {
		// uppercase on the way, letters have bit 6 set and digits do not
		for _, d := range digits {
			out[i] = d &^ (d >> 1 & 0x20)
			i++
		}
	} else {
		i += copy(out[i:], digits)
	}
	for ; i < len(out); i++ {
		out[i] = ' '
	}
	f.Write(out)
	putFmtBuf(p, out)
}

// --- roots ----------------------------------------------

// sqrtSeed holds ceil(sqrt((t+1) * 2^55)) / 2^17 rounded up by the top 9 bits t
// of a limb >= 2^62: a root from above within 2^-7.9.
// 512 entries keep the index in range without a check
var sqrtSeed = func() (s [512]uint16) {
	for t := 128; t < 512; t++ {
		// ceil(sqrt((t+1) * 2^21)) by integer Newton steps from above
		v := uint64(t+1) << 21
		z := v
		for {
			n := (z + v/z) >> 1
			if n >= z {
				break
			}
			z = n
		}
		if z*z < v {
			z++
		}
		s[t] = uint16(z)
	}
	return
}()

// sqrtInvSeed holds 2^15 / sqrt((t+0.5) / 512) rounded by the top 9 bits t
// of a limb >= 2^62: the inverse root within 2^-9, for roots with no division.
// The entries below 128 stay zero and give the root 0 for a limb 0
var sqrtInvSeed = func() (s [512]uint16) {
	for t := 128; t < 512; t++ {
		// floor(sqrt(2^42 / (2t+1))) is twice the seed, by Newton steps
		v := uint64(1<<42) / uint64(2*t+1)
		z := v
		for {
			n := (z + v/z) >> 1
			if n >= z {
				break
			}
			z = n
		}
		s[t] = uint16((z + 1) >> 1)
	}
	return
}()

// sqrt64n returns the square root of a rounded down.
// a must be at least 2^62, a smaller a gives garbage
func sqrt64n(a uint64) uint64 {
	if !arch.IsAmd64 {
		// no division, amd64 keeps the division-based root below: on Graviton
		// UDIV is slow and not pipelined, so this was faster in our benchmarks.
		// y = 2^63/sqrt(a) by one Newton step on the inverse root from a seed
		// within 2^-9: y0*(3 - a*y0^2)/2 is within 2^-17.4.
		// 2^-29 is taken off to stay below.
		// Then s = a*y and one step on the root, s + (a - s^2)*y/2,
		// gives the root or one less
		y0 := uint64(sqrtInvSeed[a>>55])
		a32 := a >> 32
		y, _ := bits.Mul64(y0<<17, 3<<62-1<<34-a32*(y0*y0))
		s := a32 * y >> 31
		d, _ := bits.Mul64(a-s*s, y)
		s += d
		// one more when the remainder is past 2s, by a conditional increment
		if a-s*s > 2*s {
			s++
		}
		return s
	}
	z := uint64(sqrtSeed[a>>55]) << 17
	// Newton steps from above: 2^-7.9 off the root, then 2^-16.8,
	// then less than 1/8 past it
	z = (z + a/z) >> 1
	z = (z + a/z) >> 1
	// z is the root or one more, z*z-1 wraps to 2^64-1 for z = 2^32
	_, b := bits.Sub64(a, z*z-1, 1)
	return z - b
}

// sqrtrem64n is the amd64 sqrt64n with the remainder a - s^2 <= 2s.
// a must be at least 2^62
func sqrtrem64n(a uint64) (s, r uint64) {
	z := uint64(sqrtSeed[a>>55]) << 17
	z = (z + a/z) >> 1
	z = (z + a/z) >> 1
	// a - z^2 is exact as int64 for z = 2^32 too, negative for a root too big
	r = a - z*z
	m := uint64(int64(r) >> 63)
	return z + m, r + (2*z-1)&m
}

// sqrt64 returns the square root of x rounded down
func sqrt64(x uint64) uint64 {
	if x == 0 {
		return 0
	}
	k := uint(bits.LeadingZeros64(x)) &^ 1
	return sqrt64n(x<<k) >> (k >> 1)
}

// sqrt128n returns the square root of <a1, a0> rounded down.
// a1 must be at least 2^62, a smaller a1 gives garbage.
// The root s1 of a1 with its remainder r1, then 32 more bits by a division
func sqrt128n(a1, a0 uint64) uint64 {
	s1, r1 := sqrtrem64n(a1)
	// q = (r1*2^32 + a0>>32) / (2*s1) as (r1*2^31 + a0>>33) / s1,
	// 2^32 at most and then one less
	n := r1<<31 | a0>>33
	q := n / s1
	q -= q >> 32
	u := n - q*s1
	// the remainder (2u + bit 32 of a0)*2^32 + low 32 bits of a0 - q^2
	// has u>>31 above 2^64, and is negative for a root one too big
	_, b := bits.Sub64(u<<33|a0&(1<<33-1), q*q, 0)
	return (s1<<32 | q) - b&((u>>31-1)>>63)
}

// sqrtrem128n is sqrt128n with the remainder <rh, rl> <= 2s.
// a1 must be at least 2^62
func sqrtrem128n(a1, a0 uint64) (s, rh, rl uint64) {
	if !arch.IsAmd64 {
		// the root of a1 as in sqrt64n, written out as its remainder is needed.
		// amd64 keeps the division-based root below.
		// The division stays: a quotient from y*(2 - s1*y) is as long a chain.
		// s1 has bit 31 set, so the division needs no zero check;
		// nothing here calls, so there is no frame
		y0 := uint64(sqrtInvSeed[a1>>55])
		a32 := a1 >> 32
		y, _ := bits.Mul64(y0<<17, 3<<62-1<<34-a32*(y0*y0))
		s1 := a32 * y >> 31
		d, _ := bits.Mul64(a1-s1*s1, y)
		s1 += d
		r1 := a1 - s1*s1
		if s2, r2 := s1+1, r1-2*s1-1; int64(r2) >= 0 {
			s1, r1 = s2, r2
		}
		n := r1<<31 | a0>>33
		q := n / (s1 | 1<<31)
		q -= q >> 32
		u := n - q*s1
		s = s1<<32 | q
		var b, c uint64
		rl, b = bits.Sub64(u<<33|a0&(1<<33-1), q*q, 0)
		rh = u>>31 - b
		m := uint64(int64(rh) >> 63)
		rl, c = bits.Add64(rl, s&m, 0)
		rh += c
		s += m
		rl, c = bits.Add64(rl, s&m, 0)
		rh += c
		return
	}
	s1, r1 := sqrtrem64n(a1)
	n := r1<<31 | a0>>33
	q := n / s1
	q -= q >> 32
	u := n - q*s1
	s = s1<<32 | q
	rl, b := bits.Sub64(u<<33|a0&(1<<33-1), q*q, 0)
	rh = u>>31 - b
	// one less root for a negative remainder, r + 2s - 1 as r + s + (s-1).
	// A mask, not a branch: a quarter of the roots are one too big at random
	m := uint64(int64(rh) >> 63)
	rl, c := bits.Add64(rl, s&m, 0)
	rh += c
	s += m
	rl, c = bits.Add64(rl, s&m, 0)
	rh += c
	return
}

// --- log10 ----------------------------------------------

// pow10x128 holds 10^i for every i that fits a Uint128
var pow10x128 = func() (p [39]Uint128) {
	p[0] = Uint128{1, 0}
	for i := 1; i < len(p); i++ {
		p[i] = p[i-1].Mul64u(10)
	}
	return
}()

// pow10x256 holds 10^i for every i that fits a Uint256
var pow10x256 = func() (p [78]Uint256) {
	p[0] = Uint256{1, 0, 0, 0}
	for i := 1; i < len(p); i++ {
		p[i] = p[i-1].Mul64u(10)
	}
	return
}()

// --- JSON text ------------------------------------------

// isNegativeText reports whether s is a negative number for ErrNegative.
// -0 is not negative
func isNegativeText[S string | []byte](s S) bool {
	if len(s) < 2 || s[0] != '-' {
		return false
	}
	s = s[1:]
	if len(s) > 2 && s[0] == '0' && s[1]|0x20 == 'x' {
		s = s[2:]
	}
	for i := 0; i < len(s); i++ {
		if s[i] != '0' {
			return true
		}
	}
	return false
}

// jsonUnescape appends the text of s, the inside of a JSON string, to dst
// when s has escapes. It is false for s with none and for s that is not
// valid JSON, as encoding/json would fail it; bad UTF-8 turns to U+FFFD there
func jsonUnescape[S string | []byte](dst []byte, s S) ([]byte, bool) {
	i := 0
	for i < len(s) && s[i] != '\\' {
		i++
	}
	if i == len(s) {
		return nil, false
	}
	// written out, not the decoder of encoding/json: no reflection, no string
	// and no copy of s, the text goes to a buffer of the caller on its stack
	b := dst
	for j := 0; j < len(s); {
		c := s[j]
		if c < utf8.RuneSelf && c != '\\' {
			if c < 0x20 || c == '"' {
				return nil, false
			}
			b = append(b, c)
			j++
			continue
		}
		if c >= utf8.RuneSelf {
			r, n := decodeRune(s[j:])
			b = utf8.AppendRune(b, r)
			j += n
			continue
		}
		if j+1 == len(s) {
			return nil, false
		}
		c, j = s[j+1], j+2
		switch c {
		case '"', '\\', '/':
			b = append(b, c)
		case 'b':
			b = append(b, '\b')
		case 'f':
			b = append(b, '\f')
		case 'n':
			b = append(b, '\n')
		case 'r':
			b = append(b, '\r')
		case 't':
			b = append(b, '\t')
		case 'u':
			r := jsonHex4(s[j:])
			if r < 0 {
				return nil, false
			}
			j += 4
			if utf16.IsSurrogate(r) {
				// a pair as one rune, a lone half as U+FFFD, as encoding/json
				r2 := rune(-1)
				if len(s)-j >= 6 && s[j] == '\\' && s[j+1] == 'u' {
					r2 = jsonHex4(s[j+2:])
				}
				if r = utf16.DecodeRune(r, r2); r != utf8.RuneError {
					j += 6
				}
			}
			b = utf8.AppendRune(b, r)
		default:
			return nil, false
		}
	}
	return b, true
}

// jsonHex4 returns the rune of the 4 hex digits that start s, else -1
func jsonHex4[S string | []byte](s S) rune {
	if len(s) < 4 {
		return -1
	}
	r := rune(0)
	for i := range 4 {
		c := s[i]
		switch {
		case c-'0' <= 9:
			c -= '0'
		case c|0x20-'a' <= 5:
			c = c | 0x20 - 'a' + 10
		default:
			return -1
		}
		r = r<<4 | rune(c)
	}
	return r
}

// decodeRune is utf8.DecodeRuneInString for bytes or a string.
// The conversion takes at most 4 bytes, so it copies them to the stack
func decodeRune[S string | []byte](s S) (rune, int) {
	return utf8.DecodeRuneInString(string(s[:min(len(s), utf8.UTFMax)]))
}

// --- Montgomery -----------------------------------------

// nonzero returns 1 for v != 0, else 0, without a branch
func nonzero(v uint64) uint {
	return uint((v | -v) >> 63)
}

// madd returns a*b + c + d as hi, lo, which never overflows 128 bits
func madd(a, b, c, d uint64) (hi, lo uint64) {
	hi, lo = bits.Mul64(a, b)
	var k uint64
	lo, k = bits.Add64(lo, c, 0)
	hi += k
	lo, k = bits.Add64(lo, d, 0)
	return hi + k, lo
}

// montInv returns -1/m0 mod 2^64 for an odd m0: each Newton step doubles
// the low bits that are right, 3 of them in m0 itself
func montInv(m0 uint64) uint64 {
	v := m0
	v *= 2 - m0*v
	v *= 2 - m0*v
	v *= 2 - m0*v
	v *= 2 - m0*v
	v *= 2 - m0*v
	return -v
}

// montMul64 returns a * b / 2^64 % m by Montgomery's product, montMul on
// one limb. m must be odd and a, b below it, mp = -1/m mod 2^64, else it
// is wrong
func montMul64(a, b, m, mp uint64) uint64 {
	h, l := bits.Mul64(a, b)
	hu, _ := bits.Mul64(l*mp, m)
	// the low half of l + u*m is 0, with a carry for l != 0
	t, c := bits.Add64(h, hu, uint64(nonzero(l)))
	// t + c*2^64 < 2m
	r, br := bits.Sub64(t, m, 0)
	keep := -(br &^ c)
	return r&^keep | t&keep
}

// expModMont64 is expModMont for an odd m in 2..2^64-1, y > 0 and b < m,
// by Montgomery products modulo R = 2^64
func expModMont64(b uint64, y Uint256, m uint64) uint64 {
	mp := montInv(m)
	// R % m, then R^2 % m to take b to R*b % m
	r := -m % m
	h, l := bits.Mul64(r, r)
	r2 := bits.Rem64(h, l, m)
	var odd [8]uint64
	odd[0] = montMul64(b, r2, m, mp)
	n, w := y.BitLen(), expWindow(y.BitLen())
	if w > 1 {
		b2 := montMul64(odd[0], odd[0], m, mp)
		for k := 1; k < 1<<(w-1); k++ {
			odd[k] = montMul64(odd[k-1], b2, m, mp)
		}
	}
	// the top bit of y is set, so the first window starts r
	started := false
	for i := n - 1; i >= 0; {
		if y.Bit(i) == 0 {
			r = montMul64(r, r, m, mp)
			i--
			continue
		}
		j := max(i-w+1, 0)
		for y.Bit(j) == 0 {
			j++
		}
		v := 0
		for k := i; k >= j; k-- {
			v = v<<1 | int(y.Bit(k))
			if started {
				r = montMul64(r, r, m, mp)
			}
		}
		if started {
			r = montMul64(r, odd[v>>1], m, mp)
		} else {
			r, started = odd[v>>1], true
		}
		i = j - 1
	}
	// out of the Montgomery form
	return montMul64(r, 1, m, mp)
}

// gcdSteps31 runs 31 halvings of the binary GCD on a and b, the approximations
// of Pornin's optimized binary GCD (eprint 2020/972): the low 31 bits of a
// value exact, its top 33 bits at the length of the longer one. It returns
// f0, g0, f1, g1 with a*f0 + b*g0 and a*f1 + b*g1 multiples of 2^31 for the
// exact values, |f| + |g| <= 2^31 in each pair. For an odd b only
func gcdSteps31(a, b uint64) (f0, g0, f1, g1 int64) {
	f0, g1 = 1, 1
	// the factors of 2 of a at once: only the low 31 - i bits of a are exact
	i := bits.TrailingZeros64(a)
	if i >= 31 {
		return f0, g0, f1 << 31, g1 << 31
	}
	a >>= uint(i)
	f1 <<= uint(i)
	g1 <<= uint(i)
	for {
		// an odd a: |a - b| and the smaller one by masks, as which one is
		// larger follows no pattern, the factors along. The zeros of a - b
		// are those of |a - b|, counted beside the negation
		d, bw := bits.Sub64(a, b, 0)
		k := bits.TrailingZeros64(d)
		m := -bw
		b += d & m
		a = (d ^ m) - m
		sm := int64(m)
		df := f0 - f1
		f1 += df & sm
		f0 = (df ^ sm) - sm
		dg := g0 - g1
		g1 += dg & sm
		g0 = (dg ^ sm) - sm
		if i+k >= 31 {
			k = 31 - i
			return f0, g0, f1 << uint(k), g1 << uint(k)
		}
		i += k
		a >>= uint(k) & 63
		f1 <<= uint(k) & 63
		g1 <<= uint(k) & 63
	}
}

// foldSigns negates the factors of a round where its sum came out negative,
// na and nb being 0 or ^0, so the sums of the next round need no sign
func foldSigns(f0, g0, f1, g1 int64, na, nb uint64) (int64, int64, int64, int64) {
	sa, sb := int64(na), int64(nb)
	return (f0 ^ sa) - sa, (g0 ^ sa) - sa, (f1 ^ sb) - sb, (g1 ^ sb) - sb
}

// matMul returns the factors of the round f after the round p. Two rounds of
// gcdSteps31 give |f| + |g| <= 2^62 in each pair, within an int64
func matMul(f0, g0, f1, g1, p0, q0, p1, q1 int64) (int64, int64, int64, int64) {
	return f0*p0 + g0*p1, f0*q0 + g0*q1, f1*p0 + g1*p1, f1*q0 + g1*q1
}

// gcdOdd64 is the binary GCD of a and b.
// For odd a and b only, else it gives a wrong result or never returns
func gcdOdd64(a, b uint64) uint64 {
	for a != b {
		// the smaller one and |a - b| by masks, which is larger follows no
		// pattern; the zeros of a - b are those of |a - b|, counted beside it
		d, bw := bits.Sub64(a, b, 0)
		k := uint(bits.TrailingZeros64(d)) & 63
		m := -bw
		b += d & m
		a = ((d ^ m) - m) >> k
	}
	return a
}

// invOdd64 returns 1/x mod m and true, or false where x and m share a factor,
// as invOdd128 on one limb. For an odd m and x < m only
func invOdd64(x, m uint64) (uint64, bool) {
	if x == 0 {
		return 0, m == 1
	}
	u, v, r, s := m, x, uint64(0), uint64(1)
	k := bits.TrailingZeros64(v)
	v >>= uint(k)
	for u != v {
		d, bw := bits.Sub64(u, v, 0)
		sw := -bw
		pl, ps := s^(r^s)&sw, r^(r^s)&sw
		d = (d ^ sw) - sw
		t := bits.TrailingZeros64(d)
		d >>= uint(t)
		sum, pl := ps+pl, pl<<uint(t)
		u, v = d^(u^d)&sw, v^(v^d)&sw
		s, r = pl^(sum^pl)&sw, sum^(pl^sum)&sw
		k += t
	}
	if u != 1 {
		return 0, false
	}
	// the last step of the almost inverse, then m - r = 2^k/x mod m
	r2, c := bits.Add64(r, r, 0)
	r3, b := bits.Sub64(r2, m, 0)
	if c >= b {
		r2 = r3
	}
	return halve64(m-r2, k+1, m, montInv(m)), true
}

// halve64 returns a / 2^k mod m as halve128 does on one limb.
// For a < m, an odd m and mp = -1/m mod 2^64 only
func halve64(a uint64, k int, m, mp uint64) uint64 {
	for k > 0 {
		s := uint(min(k, 63))
		j := a * mp & (1<<s - 1)
		h, l := bits.Mul64(j, m)
		l, c := bits.Add64(l, a, 0)
		a = l>>s | (h+c)<<(64-s)
		k -= int(s)
	}
	return a
}
