package fat

import (
	"encoding/binary"
	"fmt"
	"math/bits"
)

// Uint128FromHex parses hex s with optional plus, 0x or 0X and leading zeros.
// It returns zero and ErrNegative for a negative s, ErrParse for other bad s
func Uint128FromHex[S string | []byte](s S) (Uint128, error) {
	x, ok := parseUint128([]byte(s), 16)
	if !ok {
		t := string(s)
		if isNegativeText(t) {
			return Uint128{}, ErrNegative{"Uint128FromHex", t}
		}
		return Uint128{}, ErrParse{"Uint128FromHex", t}
	}
	return x, nil
}

// MustUint128FromHex is Uint128FromHex that panics on a bad s
func MustUint128FromHex[S string | []byte](s S) Uint128 {
	x, ok := parseUint128([]byte(s), 16)
	if !ok {
		t := string(s)
		if isNegativeText(t) {
			panic(ErrNegative{"MustUint128FromHex", t})
		}
		panic(ErrParse{"MustUint128FromHex", t})
	}
	return x
}

// Uint128FromDec parses decimal s with an optional plus and leading zeros,
// or m e n as m * 10^n, such as 12e3 or 5E+18.
// It returns zero and ErrNegative for a negative s, ErrParse for other bad s
func Uint128FromDec[S string | []byte](s S) (Uint128, error) {
	// the bytes don't escape parseUint128, so the conversion copies nothing
	x, ok := parseUint128([]byte(s), 10)
	if !ok {
		return uint128FromDecSlow("Uint128FromDec", []byte(s))
	}
	return x, nil
}

// MustUint128FromDec is Uint128FromDec that panics on a bad s
func MustUint128FromDec[S string | []byte](s S) Uint128 {
	x, ok := parseUint128([]byte(s), 10)
	if !ok {
		var err error
		if x, err = uint128FromDecSlow("MustUint128FromDec", []byte(s)); err != nil {
			panic(err)
		}
	}
	return x
}

// uint128FromDecSlow is FromDec and Scan for s their parser turns down:
// m e n as m * 10^n, else the error of fn
func uint128FromDecSlow(fn string, s []byte) (Uint128, error) {
	m, n, ok := sciParts(s)
	if ok {
		if x, ok := parseUint128(m, 10); ok {
			if x.IsZero() {
				return x, nil
			}
			// a power below 2^64 takes the shorter product by one limb
			if n < len(pow10Table) {
				if r, over := x.Mul64o(pow10Table[n]); !over {
					return r, nil
				}
			} else if n < len(pow10x128) {
				if r, over := x.Mulo(pow10x128[n]); !over {
					return r, nil
				}
			}
		}
		// a minus before a zero mantissa is not negative, as for -0
		if isMinusZero(m) {
			return Uint128{}, ErrParse{fn, string(s)}
		}
	}
	if isNegativeText(s) {
		return Uint128{}, ErrNegative{fn, string(s)}
	}
	return Uint128{}, ErrParse{fn, string(s)}
}

var _ fmt.Stringer = (*Uint128)(nil)

// String is Dec
func (x Uint128) String() string {
	return x.Dec()
}

// Dec returns x in base 10
func (x Uint128) Dec() string {
	// 39 digits hold any x
	var buf [39]byte
	var d []byte
	// appendDecimal written out, a call less
	if x.L1 == 0 {
		d = appendDecimal64(buf[:0], x.L0)
	} else if x.L1 < decimalChunk {
		// the writer of appendDecimal on the buffer
		top, c0 := div1e19Mask(x.L1, x.L0)
		nt := count64Digits(top)
		b := buf[:nt+19]
		// the head as blocks of 4, 8 and 8 digits ending at nt.
		// A block starting before 0 is cut at 0,
		// the stores after it overwrite what it writes past
		hi := top / 1e8
		t0 := top - hi*1e8
		// 1e8 = 2^8 * 390625, 11529215047 / 2^52 ~ 1/390625
		t2 := (hi >> 8) * 11529215047 >> 52
		t1 := hi - t2*1e8
		ch := c0 / 1e16
		cq := c0 / 1e8
		// t2 < 1845 and ch < 1000 as 4 digits in 32-bit lanes,
		// 5243 / 2^19 ~ 1/100 and 103 / 2^10 ~ 1/10
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
		// put19Digits written out,
		// its 3-digit head from the lane above without the leading zero
		c := b[nt:]
		_ = c[18]
		// the 4th byte is overwritten by the next store
		binary.LittleEndian.PutUint32(c, uint32(g>>40))
		binary.LittleEndian.PutUint64(c[3:], digits8(cq-ch*1e8))
		binary.LittleEndian.PutUint64(c[11:], digits8(c0-cq*1e8))
		d = b
	} else {
		q, c0 := div2by1(x.L1-decimalChunk, x.L0, decimalChunk, decimalChunkReciprocal)
		top, c1 := div2by1(1, q, decimalChunk, decimalChunkReciprocal)
		d = appendDecimalParts(buf[:0], false, top, c1, c0, 0, 0, 2)
	}
	return string(d)
}

// Hex returns x in lower-case base 16 with a 0x prefix
func (x Uint128) Hex() string {
	var buf [35]byte
	return string(buf[x.putHex(&buf):])
}

// AppendDec appends Dec to dst
func (x Uint128) AppendDec(dst []byte) []byte {
	return x.appendDecimal(dst)
}

// AppendHex appends Hex to dst
func (x Uint128) AppendHex(dst []byte) []byte {
	// Hex on the stack, then one append
	var buf [35]byte
	return append(dst, buf[x.putHex(&buf):]...)
}

// PrettyDec returns x in base 10 with sep between groups of 3 digits
func (x Uint128) PrettyDec(sep byte) string {
	var buf [39]byte
	return prettyDecimal(x.appendDecimal(buf[:0]), false, sep)
}

// parseUint128 parses s with an optional plus in base 10 or 16.
// Base 16 takes an optional 0x or 0X, base 0 picks either by that prefix.
// A minus or a second sign fails, any other base parses as 16
func parseUint128(s []byte, base int) (Uint128, bool) {
	// one plus, a second one is a bad symbol
	if len(s) > 0 && s[0] == '+' {
		s = s[1:]
	}
	hex := len(s) > 2 && s[0] == '0' && s[1]|0x20 == 'x'
	if base == 10 || base == 0 && !hex {
		// a 0x prefix fails on the x
		// decimal written out here, a call less
		// leading zeros don't count towards the length limit
		for len(s) > 1 && s[0] == '0' {
			s = s[1:]
		}
		n := len(s)
		if n <= 16 {
			// parseDecimalUint64 up to 16 digits written out, a call less
			if n <= 8 {
				if n == 0 {
					return Uint128{}, false
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
					return Uint128{}, false
				}
				return Uint128{conv8(x), 0}, true
			}
			// two overlapping blocks
			a := binary.LittleEndian.Uint64(s)
			b := binary.LittleEndian.Uint64(s[n-8:])
			if notDigits8(a)|notDigits8(b) != 0 {
				return Uint128{}, false
			}
			// drop digits shared with a
			b &= ^uint64(0) << (8 * uint(16-n))
			return Uint128{conv8(a)*pow10Table[n-8] + conv8(b), 0}, true
		}
		if n > 39 {
			return Uint128{}, false
		}
		// k whole 16-digit windows from the start, then a tail of t digits
		k := (n - 1) >> 4
		t := n - 16*k
		p := pow10Table[t]
		// the window ending at len(s), its first 16-t bytes are of the last one
		ta := binary.LittleEndian.Uint64(s[n-16:])
		tb := binary.LittleEndian.Uint64(s[n-8:])
		bad := notDigits8(ta) | notDigits8(tb)
		m := 16 - t
		dm := m - 8
		dm &^= dm >> (bits.UintSize - 1)
		// zero bytes count as zero digits, a shift by 64 or more clears it
		ta &= ^uint64(0) << (8 * uint(m))
		tb &= ^uint64(0) << (8 * uint(dm))
		tail := conv8(ta)*1e8 + conv8(tb)
		a, b := binary.LittleEndian.Uint64(s), binary.LittleEndian.Uint64(s[8:])
		bad |= notDigits8(a) | notDigits8(b)
		w0 := conv8(a)*1e8 + conv8(b)
		if k == 1 {
			if bad != 0 {
				return Uint128{}, false
			}
			// w0*10^t + tail stays below 1e32
			h, l := bits.Mul64(w0, p)
			l, c := bits.Add64(l, tail, 0)
			return Uint128{l, h + c}, true
		}
		_ = s[31]
		e, f := binary.LittleEndian.Uint64(s[16:]), binary.LittleEndian.Uint64(s[24:])
		bad |= notDigits8(e) | notDigits8(f)
		w1 := conv8(e)*1e8 + conv8(f)
		h, l := bits.Mul64(w0, 1e16)
		l, c := bits.Add64(l, w1, 0)
		h += c
		// h:l * 10^t + tail, the top word must stay zero
		hh, hl := bits.Mul64(h, p)
		lh, ll := bits.Mul64(l, p)
		ll, c = bits.Add64(ll, tail, 0)
		lh, c = bits.Add64(lh, hl, c)
		if bad|hh|c != 0 {
			return Uint128{}, false
		}
		return Uint128{ll, lh}, true
	}
	// hex written out, a call less, a sign is a bad symbol.
	// Base 16 takes it with no prefix too
	if hex {
		s = s[2:]
	} else if len(s) == 0 {
		return Uint128{}, false
	}
	// only zeros past 32 digits are dropped, the rest is parsed as is
	for len(s) > 32 && s[0] == '0' {
		s = s[1:]
	}
	n := len(s)
	if n > 32 {
		return Uint128{}, false
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
			return Uint128{}, false
		}
		return Uint128{hexDigits8(v), 0}, true
	}
	// the top limb holds the first m chars, the low one the last 16
	m := n - 16*((n-1)>>4)
	// its low block starts at o = max(m-8, 0), the high block ends there
	d := m - 8
	neg := d >> (bits.UintSize - 1)
	o := d &^ neg
	sl := 8 * uint(-d&neg) & 63
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
	if bad != 0 {
		return Uint128{}, false
	}
	return Uint128{l0, l1}, true
}

// appendDecimal appends decimal x, one limb goes to the 64-bit writer
func (x Uint128) appendDecimal(dst []byte) []byte {
	if x.L1 == 0 {
		return appendDecimal64(dst, x.L0)
	}
	if x.L1 >= decimalChunk {
		q, c0 := div2by1(x.L1-decimalChunk, x.L0, decimalChunk, decimalChunkReciprocal)
		top, c1 := div2by1(1, q, decimalChunk, decimalChunkReciprocal)
		return appendDecimalParts(dst, false, top, c1, c0, 0, 0, 2)
	}
	// a head of 1 to 20 digits and 19 more
	top, c0 := div1e19Mask(x.L1, x.L0)
	nt := count64Digits(top)
	nd := nt + 19
	n := len(dst)
	if cap(dst)-n < nd {
		dst = append(dst, make([]byte, nd)...)
	}
	dst = dst[:n+nd]
	b := dst[n:]
	// the head as blocks of 4, 8 and 8 digits ending at nt.
	// A block starting before 0 is cut at 0,
	// the stores after it overwrite what it writes past
	hi := top / 1e8
	t0 := top - hi*1e8
	// 1e8 = 2^8 * 390625, 11529215047 / 2^52 ~ 1/390625
	t2 := (hi >> 8) * 11529215047 >> 52
	t1 := hi - t2*1e8
	ch := c0 / 1e16
	cq := c0 / 1e8
	// t2 < 1845 and ch < 1000 as 4 digits in 32-bit lanes,
	// 5243 / 2^19 ~ 1/100 and 103 / 2^10 ~ 1/10
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
	// put19Digits written out,
	// its 3-digit head from the lane above without the leading zero
	c := b[nt:]
	_ = c[18]
	// the 4th byte is overwritten by the next store
	binary.LittleEndian.PutUint32(c, uint32(g>>40))
	binary.LittleEndian.PutUint64(c[3:], digits8(cq-ch*1e8))
	binary.LittleEndian.PutUint64(c[11:], digits8(c0-cq*1e8))
	return dst
}

// putHex writes hex x at the end of buf after 0x and returns where 0x starts
func (x Uint128) putHex(buf *[35]byte) int {
	// z leading zero chars of the 32, one digit stays for zero
	z := x.LeadingZeros() >> 2
	z -= z >> 5
	if x.L1 != 0 {
		binary.BigEndian.PutUint64(buf[3:], hexChars8(x.L1>>32))
		binary.BigEndian.PutUint64(buf[11:], hexChars8(x.L1&0xFFFFFFFF))
	}
	binary.BigEndian.PutUint64(buf[19:], hexChars8(x.L0>>32))
	binary.BigEndian.PutUint64(buf[27:], hexChars8(x.L0&0xFFFFFFFF))
	// after the digits, the stores above write leading zeros over it
	buf[z+1], buf[z+2] = '0', 'x'
	return 1 + z
}

// div1e19Mask is div2by1 by decimalChunk, the likely correction by a mask.
// A branch on it would go at random.
// For uh < decimalChunk only, a larger uh gives a wrong result
func div1e19Mask(uh, ul uint64) (q, r uint64) {
	qh, ql := bits.Mul64(decimalChunkReciprocal, uh)
	ql, c := bits.Add64(ql, ul, 0)
	qh, _ = bits.Add64(qh, uh, c)
	qh++
	r = ul - qh*decimalChunk
	// r > ql takes one off the quotient
	_, b := bits.Sub64(ql, r, 0)
	qh -= b
	r += decimalChunk & -b
	// rare, a branch predicts well
	if r >= decimalChunk {
		qh++
		r -= decimalChunk
	}
	return qh, r
}
