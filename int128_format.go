package fat

import (
	"encoding/binary"
	"fmt"
	"math/bits"
)

// Int128FromHex parses hex s with an optional sign, 0x or 0X and leading zeros.
// It returns zero and ErrParse for a bad s or one out of range
func Int128FromHex[S string | []byte](s S) (x Int128, err error) {
	// named results keep it inlined with the text in the error
	x, ok := parseInt128([]byte(s), 16)
	if !ok {
		err = ErrParse{"Int128FromHex", string(s)}
	}
	return
}

// MustInt128FromHex is Int128FromHex that panics on a bad s
func MustInt128FromHex[S string | []byte](s S) (x Int128) {
	x, ok := parseInt128([]byte(s), 16)
	if !ok {
		panic(ErrParse{"MustInt128FromHex", string(s)})
	}
	return
}

// Int128FromDec parses decimal s with an optional sign and leading zeros,
// or m e n as m * 10^n, such as -12e3 or 5E+18.
// It returns zero and ErrParse for a bad s or one out of range
func Int128FromDec[S string | []byte](s S) (Int128, error) {
	// the bytes don't escape parseInt128, so the conversion copies nothing
	x, ok := parseInt128([]byte(s), 10)
	if !ok {
		return int128FromDecSlow("Int128FromDec", []byte(s))
	}
	return x, nil
}

// MustInt128FromDec is Int128FromDec that panics on a bad s
func MustInt128FromDec[S string | []byte](s S) (x Int128) {
	// a named result keeps it inlined
	x, ok := parseInt128([]byte(s), 10)
	if !ok {
		var err error
		if x, err = int128FromDecSlow("MustInt128FromDec", []byte(s)); err != nil {
			panic(err)
		}
	}
	return
}

// int128FromDecSlow is FromDec and Scan for s their parser turns down:
// m e n as m * 10^n, else the error of fn
func int128FromDecSlow(fn string, s []byte) (Int128, error) {
	if m, n, ok := sciParts(s); ok {
		if x, ok := parseInt128(m, 10); ok {
			if x.IsZero() {
				return x, nil
			}
			// a power below 2^63 takes the shorter product by one limb
			if n < len(pow10Table)-1 {
				if r, over := x.Mul64o(int64(pow10Table[n])); !over {
					return r, nil
				}
			} else if n < len(pow10x128) {
				if r, over := x.Mulo(Int128(pow10x128[n])); !over {
					return r, nil
				}
			}
		}
	}
	return Int128{}, ErrParse{fn, string(s)}
}

var _ fmt.Stringer = (*Int128)(nil)

// String is x in base 10, as Dec
func (x Int128) String() string {
	return x.Dec()
}

// Dec is x in base 10
func (x Int128) Dec() string {
	// fits a sign and 39 digits
	var buf [40]byte
	var d []byte
	a, neg := x.UnsignedAbs(), int64(x.L1) < 0
	// appendDecimal written out, a call less
	if a.L1 == 0 {
		d = appendDecimalSign64(buf[:0], neg, a.L0)
	} else {
		// a.L1 <= 2^63 < 1e19 leaves one division, top > 0 as |x| >= 2^64
		top, c0 := div2by1(a.L1, a.L0, decimalChunk, decimalChunkReciprocal)
		d = appendDecimalParts(buf[:0], neg, top, c0, 0, 0, 0, 1)
	}
	return string(d)
}

// Hex is x in base 16 with a 0x prefix, -0x when negative
func (x Int128) Hex() string {
	var buf [35]byte
	return string(buf[x.putHex(&buf):])
}

// AppendDec appends x as Dec does
func (x Int128) AppendDec(dst []byte) []byte {
	return x.appendDecimal(dst)
}

// AppendHex appends x as Hex does
func (x Int128) AppendHex(dst []byte) []byte {
	// Hex on the stack, then one append
	var buf [35]byte
	return append(dst, buf[x.putHex(&buf):]...)
}

// PrettyDec is x in base 10 with sep between groups of 3 digits
func (x Int128) PrettyDec(sep byte) string {
	var buf [39]byte
	return prettyDecimal(x.UnsignedAbs().appendDecimal(buf[:0]), x.IsNegative(), sep)
}

// parseInt128 parses s with an optional sign in base 10 or 16.
// Base 16 takes s with or without 0x or 0X, base 0 picks either by that prefix.
// Any other base is taken as 10
func parseInt128(s []byte, base int) (Int128, bool) {
	// decimal written out here: faster than a call in our benchmarks
	var c byte
	if len(s) > 0 {
		c = s[0]
	}
	// '+' and '-' differ only in bit 1, m is all ones for a minus
	k := 0
	if (c-'+')&^2 == 0 {
		k = 1
	}
	var m uint64
	if c == '-' {
		m = ^uint64(0)
	}
	s = s[k:]
	hex := len(s) > 2 && s[0] == '0' && s[1]|0x20 == 'x'
	if base == 16 || base == 0 && hex {
		// hex written out, a call less, and a second sign is a bad symbol.
		// Base 16 takes s with no prefix too
		if hex {
			s = s[2:]
		} else if len(s) == 0 {
			return Int128{}, false
		}
		// only zeros past 32 digits are dropped, the rest is parsed as is
		for len(s) > 32 && s[0] == '0' {
			s = s[1:]
		}
		n := len(s)
		if n > 32 {
			return Int128{}, false
		}
		if n <= 8 {
			// chars go to the low n bytes, first char on top
			var v uint64
			if n >= 4 {
				// two overlapping 4-byte loads
				v = uint64(binary.BigEndian.Uint32(s))<<(8*uint(n-4)) | uint64(binary.BigEndian.Uint32(s[n-4:]))
			} else {
				// first, middle and last byte cover 1..3 chars
				v = uint64(s[0])<<(8*uint(n-1)) | uint64(s[(n-1)>>1])<<(8*uint(n-1-(n-1)>>1)) | uint64(s[n-1])
			}
			// left-pad with '0' to a full block
			v |= 0x3030303030303030 &^ (^uint64(0) >> (8 * uint(8-n) & 63))
			if notHexDigits8(v) != 0 {
				return Int128{}, false
			}
			// below 2^32, (v ^ m) - m with the borrow as the high limb
			r0, b := bits.Sub64(hexDigits8(v)^m, m, 0)
			return Int128{r0, -b}, true
		}
		// the top limb holds the first t chars, the low one the last 16
		t := n - 16*((n-1)>>4)
		// its low block starts at o = max(t-8, 0), its high block is s[:o]
		d := t - 8
		dn := d >> (bits.UintSize - 1)
		o := d &^ dn
		sl := 8 * uint(-d&dn) & 63
		sh := 8 * uint(8-o)
		lo := binary.BigEndian.Uint64(s[o:])
		hi := binary.BigEndian.Uint64(s)
		// chars of the lower blocks are shifted out, '0' pads the top
		lo = lo>>sl | 0x3030303030303030&^(^uint64(0)>>sl)
		hi = hi>>sh | 0x3030303030303030&^(^uint64(0)>>sh)
		bad := notHexDigits8(lo) | notHexDigits8(hi)
		top := hexDigits8(hi)<<32 | hexDigits8(lo)
		var l0, l1 uint64
		if n <= 16 {
			l0 = top
		} else {
			a, b := binary.BigEndian.Uint64(s[n-16:]), binary.BigEndian.Uint64(s[n-8:])
			bad |= notHexDigits8(a) | notHexDigits8(b)
			l0, l1 = hexDigits8(a)<<32|hexDigits8(b), top
		}
		// it fits 127 bits, or is 2^127 with a minus.
		// So 2^127-1+minus - u must not borrow
		_, over := bits.Sub64(^m, l0, 0)
		_, over = bits.Sub64(1<<63-1-m, l1, over)
		if bad|over != 0 {
			return Int128{}, false
		}
		r0, bw := bits.Sub64(l0^m, m, 0)
		r1, _ := bits.Sub64(l1^m, m, bw)
		return Int128{r0, r1}, true
	}
	// a 0x prefix fails on the x, leading zeros do not count toward the limit
	for len(s) > 1 && s[0] == '0' {
		s = s[1:]
	}
	n := len(s)
	if n <= 19 {
		var l0 uint64
		if n <= 8 {
			if n == 0 {
				return Int128{}, false
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
				return Int128{}, false
			}
			l0 = conv8(x)
		} else {
			a := binary.LittleEndian.Uint64(s)
			b := binary.LittleEndian.Uint64(s[n-8:])
			if n <= 16 {
				// two overlapping blocks
				if notDigits8(a)|notDigits8(b) != 0 {
					return Int128{}, false
				}
				// drop digits shared with a
				b &= ^uint64(0) << (8 * uint(16-n))
				l0 = conv8(a)*pow10Table[n-8] + conv8(b)
			} else {
				e := binary.LittleEndian.Uint64(s[8:])
				if notDigits8(a)|notDigits8(e)|notDigits8(b) != 0 {
					return Int128{}, false
				}
				// drop digits shared with e
				b &= ^uint64(0) << (8 * uint(24-n))
				l0 = (conv8(a)*1e8+conv8(e))*pow10Table[n-16] + conv8(b)
			}
		}
		// (l0 ^ m) - m, the high limb is the borrow
		r0, b := bits.Sub64(l0^m, m, 0)
		return Int128{r0, -b}, true
	}
	if n > 39 {
		return Int128{}, false
	}
	// one or two 16-digit windows from the start, then a tail of t digits
	w := (n - 1) >> 4
	t := n - 16*w
	p := pow10Table[t]
	// the window ending at len(s), its first 16-t bytes belong to the last one
	ta := binary.LittleEndian.Uint64(s[n-16:])
	tb := binary.LittleEndian.Uint64(s[n-8:])
	_ = s[15]
	a, b := binary.LittleEndian.Uint64(s), binary.LittleEndian.Uint64(s[8:])
	bad := notDigits8(ta) | notDigits8(tb) | notDigits8(a) | notDigits8(b)
	sm := 16 - t
	dm := sm - 8
	dm &^= dm >> (bits.UintSize - 1)
	// zero bytes count as zero digits, a shift by 64 or more clears the word
	ta &= ^uint64(0) << (8 * uint(sm))
	tb &= ^uint64(0) << (8 * uint(dm))
	tail := conv8(ta)*1e8 + conv8(tb)
	w0 := conv8(a)*1e8 + conv8(b)
	if w == 1 {
		if bad != 0 {
			return Int128{}, false
		}
		// w0*10^t + tail, below 1e32
		h, lo := bits.Mul64(w0, p)
		lo, c := bits.Add64(lo, tail, 0)
		r0, bw := bits.Sub64(lo^m, m, 0)
		r1, _ := bits.Sub64((h+c)^m, m, bw)
		return Int128{r0, r1}, true
	}
	_ = s[31]
	e, f := binary.LittleEndian.Uint64(s[16:]), binary.LittleEndian.Uint64(s[24:])
	bad |= notDigits8(e) | notDigits8(f)
	w1 := conv8(e)*1e8 + conv8(f)
	h, lo := bits.Mul64(w0, 1e16)
	l0, cc := bits.Add64(lo, w1, 0)
	l1 := h + cc
	// (l1:l0)*10^t + tail, the top word must stay zero
	h, lo = bits.Mul64(l0, p)
	l0, cc = bits.Add64(lo, tail, 0)
	u := h + cc
	h, lo = bits.Mul64(l1, p)
	l1, cc = bits.Add64(lo, u, 0)
	// overflows if 2^127-1+minus - u borrows, |MinInt128| = 2^127
	_, over := bits.Sub64(^m, l0, 0)
	_, over = bits.Sub64(1<<63-1-m, l1, over)
	if bad|h|cc|over != 0 {
		return Int128{}, false
	}
	r0, bw := bits.Sub64(l0^m, m, 0)
	r1, _ := bits.Sub64(l1^m, m, bw)
	return Int128{r0, r1}, true
}

// putHex writes x as Hex at the end of buf and returns where it starts.
// It is Uint128.putHex on |x| written out, a call less
func (x Int128) putHex(buf *[35]byte) int {
	u := x.UnsignedAbs()
	// z leading zero chars of the 32, one digit stays for zero
	z := u.LeadingZeros() >> 2
	z -= z >> 5
	if u.L1 != 0 {
		binary.BigEndian.PutUint64(buf[3:], hexChars8(u.L1>>32))
		binary.BigEndian.PutUint64(buf[11:], hexChars8(u.L1&0xFFFFFFFF))
	}
	binary.BigEndian.PutUint64(buf[19:], hexChars8(u.L0>>32))
	binary.BigEndian.PutUint64(buf[27:], hexChars8(u.L0&0xFFFFFFFF))
	// after the digit stores, which would write leading zeros over it
	buf[z], buf[z+1], buf[z+2] = '-', '0', 'x'
	return 1 + z - int(x.L1>>63)
}

// appendDecimal appends decimal x
func (x Int128) appendDecimal(dst []byte) []byte {
	// a sign version of Uint128.appendDecimal written out, a call less
	a, neg := x.UnsignedAbs(), int64(x.L1) < 0
	if a.L1 == 0 {
		return appendDecimalSign64(dst, neg, a.L0)
	}
	// a.L1 <= 2^63 < 1e19 leaves one division, top > 0 as |x| >= 2^64.
	// Then the writer of Uint128.appendDecimal after the sign, written out:
	// faster than calls of appendDecimalParts and put19Digits in our benchmarks
	top, c0 := div1e19Mask(a.L1, a.L0)
	nt := count64Digits(top)
	sign := int(x.L1 >> 63)
	nd := sign + nt + 19
	n := len(dst)
	if cap(dst)-n < nd {
		dst = append(dst, make([]byte, nd)...)
	}
	dst = dst[:n+nd]
	// the first digit overwrites the minus when there is no sign
	dst[n] = '-'
	b := dst[n+sign:]
	hi := top / 1e8
	t0 := top - hi*1e8
	t2 := (hi >> 8) * 11529215047 >> 52
	t1 := hi - t2*1e8
	ch := c0 / 1e16
	cq := c0 / 1e8
	g := t2 | ch<<32
	gh := (g * 5243 >> 19) & (0x7F<<32 | 0x7F)
	g = g<<16 - gh*(100<<16-1)
	gt := (g * 103 >> 10) & (0xF<<48 | 0xF<<32 | 0xF<<16 | 0xF)
	g = g<<8 + 0x3030303030303030 - gt*(10<<8-1)
	p1, p0 := nt-16, nt-8
	q1 := p1 &^ (p1 >> (bits.UintSize - 1))
	q0 := p0 &^ (p0 >> (bits.UintSize - 1))
	binary.LittleEndian.PutUint32(b, uint32(g>>(8*uint(20-nt)&63)))
	binary.LittleEndian.PutUint64(b[q1:], digits8(t1)>>(8*uint(q1-p1)&63))
	binary.LittleEndian.PutUint64(b[q0:], digits8(t0)>>(8*uint(q0-p0)&63))
	c := b[nt:]
	_ = c[18]
	binary.LittleEndian.PutUint32(c, uint32(g>>40))
	binary.LittleEndian.PutUint64(c[3:], digits8(cq-ch*1e8))
	binary.LittleEndian.PutUint64(c[11:], digits8(c0-cq*1e8))
	return dst
}
