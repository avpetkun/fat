package fat

import (
	"encoding/binary"
	"fmt"
	"math/bits"
)

// Uint256FromHex parses hex s with optional plus, 0x or 0X and leading zeros.
// It returns zero and ErrNegative for a negative s, ErrParse for other bad s
func Uint256FromHex[S string | []byte](s S) (Uint256, error) {
	x, ok := parseUint256([]byte(s), 16)
	if !ok {
		t := string(s)
		if isNegativeText(t) {
			return Uint256{}, ErrNegative{"Uint256FromHex", t}
		}
		return Uint256{}, ErrParse{"Uint256FromHex", t}
	}
	return x, nil
}

// MustUint256FromHex is Uint256FromHex that panics on a bad s
func MustUint256FromHex[S string | []byte](s S) Uint256 {
	x, ok := parseUint256([]byte(s), 16)
	if !ok {
		t := string(s)
		if isNegativeText(t) {
			panic(ErrNegative{"MustUint256FromHex", t})
		}
		panic(ErrParse{"MustUint256FromHex", t})
	}
	return x
}

// Uint256FromDec parses decimal s with an optional plus and leading zeros,
// or m e n as m * 10^n, such as 12e3 or 5E+18.
// It returns zero and ErrNegative for a negative s, ErrParse for other bad s
func Uint256FromDec[S string | []byte](s S) (Uint256, error) {
	// the bytes don't escape parseUint256, so the conversion copies nothing
	x, ok := parseUint256([]byte(s), 10)
	if !ok {
		return uint256FromDecSlow("Uint256FromDec", []byte(s))
	}
	return x, nil
}

// MustUint256FromDec is Uint256FromDec that panics on a bad s
func MustUint256FromDec[S string | []byte](s S) Uint256 {
	x, ok := parseUint256([]byte(s), 10)
	if !ok {
		var err error
		if x, err = uint256FromDecSlow("MustUint256FromDec", []byte(s)); err != nil {
			panic(err)
		}
	}
	return x
}

// uint256FromDecSlow is FromDec and Scan for s their parser turns down:
// m e n as m * 10^n, else the error of fn
func uint256FromDecSlow(fn string, s []byte) (Uint256, error) {
	m, n, ok := sciParts(s)
	if ok {
		if x, ok := parseUint256(m, 10); ok {
			if x.IsZero() {
				return x, nil
			}
			// a power below 2^64 takes the shorter product by one limb
			if n < len(pow10Table) {
				if r, over := x.Mul64o(pow10Table[n]); !over {
					return r, nil
				}
			} else if n < len(pow10x256) {
				if r, over := x.Mulo(pow10x256[n]); !over {
					return r, nil
				}
			}
		}
		// a minus before a zero mantissa is not negative, as for -0
		if isMinusZero(m) {
			return Uint256{}, ErrParse{fn, string(s)}
		}
	}
	if isNegativeText(s) {
		return Uint256{}, ErrNegative{fn, string(s)}
	}
	return Uint256{}, ErrParse{fn, string(s)}
}

var _ fmt.Stringer = (*Uint256)(nil)

// String is Dec
func (x Uint256) String() string {
	return x.Dec()
}

// Dec returns x in base 10
func (x Uint256) Dec() string {
	// fits 78 digits
	var buf [78]byte
	return string(x.appendDecimalSign(buf[:0], false))
}

// Hex returns x in lower-case base 16 with a 0x prefix
func (x Uint256) Hex() string {
	var buf [67]byte
	return string(buf[x.putHex(&buf):])
}

// AppendDec appends Dec to dst
func (x Uint256) AppendDec(dst []byte) []byte {
	return x.appendDecimal(dst)
}

// AppendHex appends Hex to dst
func (x Uint256) AppendHex(dst []byte) []byte {
	// Hex on the stack, then one append
	var buf [67]byte
	return append(dst, buf[x.putHex(&buf):]...)
}

// PrettyDec returns x in base 10 with sep between groups of 3 digits
func (x Uint256) PrettyDec(sep byte) string {
	var buf [78]byte
	return prettyDecimal(x.appendDecimal(buf[:0]), false, sep)
}

// parseUint256 parses s with an optional plus in base 10 or 16.
// Base 16 takes an optional 0x or 0X, base 0 picks either by that prefix.
// A minus or a second sign fails, any other base parses as 16
func parseUint256(s []byte, base int) (Uint256, bool) {
	// one plus, a second one is a bad symbol
	if len(s) > 0 && s[0] == '+' {
		s = s[1:]
	}
	hex := len(s) > 2 && s[0] == '0' && s[1]|0x20 == 'x'
	if base == 10 || base == 0 && !hex {
		// a 0x prefix fails on the x
		// decimal written out here, parseDecimalUint64 in it up to 16 digits:
		// two calls less, faster in our benchmarks
		// leading zeros don't count towards the length limit
		for len(s) > 1 && s[0] == '0' {
			s = s[1:]
		}
		n := len(s)
		if n <= 16 {
			// parseDecimalUint64 up to 16 digits written out, a call less
			if n <= 8 {
				if n == 0 {
					return Uint256{}, false
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
					return Uint256{}, false
				}
				return Uint256{conv8(x), 0, 0, 0}, true
			}
			// two overlapping blocks
			a := binary.LittleEndian.Uint64(s)
			b := binary.LittleEndian.Uint64(s[n-8:])
			if notDigits8(a)|notDigits8(b) != 0 {
				return Uint256{}, false
			}
			// drop digits shared with a
			b &= ^uint64(0) << (8 * uint(16-n))
			return Uint256{conv8(a)*pow10Table[n-8] + conv8(b), 0, 0, 0}, true
		}
		if n <= 19 {
			v, ok := parseDecimalUint64(s)
			if !ok {
				return Uint256{}, false
			}
			return Uint256{v, 0, 0, 0}, true
		}
		if n > 78 {
			return Uint256{}, false
		}
		// k whole 16-digit windows from the start, then a tail of t digits
		k := (n - 1) >> 4
		t := n - 16*k
		p := pow10Table[t]
		// the window ending at len(s):
		// its first 16-t bytes are digits of the last window
		ta := binary.LittleEndian.Uint64(s[n-16:])
		tb := binary.LittleEndian.Uint64(s[n-8:])
		bad := notDigits8(ta) | notDigits8(tb)
		m := 16 - t
		dm := m - 8
		dm &^= dm >> (bits.UintSize - 1)
		// zero bytes count as zero digits, a shift past 63 clears the word
		ta &= ^uint64(0) << (8 * uint(m))
		tb &= ^uint64(0) << (8 * uint(dm))
		tail := conv8(ta)*1e8 + conv8(tb)
		var l0, l1, l2, l3, h, lo, c, w uint64
		switch k {
		case 1:
			_ = s[15]
			a, b := binary.LittleEndian.Uint64(s), binary.LittleEndian.Uint64(s[8:])
			bad |= notDigits8(a) | notDigits8(b)
			w0 := conv8(a)*1e8 + conv8(b)
			// x*10^t + tail
			h, lo = bits.Mul64(w0, p)
			l0, c = bits.Add64(lo, tail, 0)
			l1 = h + c
		case 2:
			_ = s[31]
			a, b := binary.LittleEndian.Uint64(s), binary.LittleEndian.Uint64(s[8:])
			e, f := binary.LittleEndian.Uint64(s[16:]), binary.LittleEndian.Uint64(s[24:])
			bad |= notDigits8(a) | notDigits8(b) | notDigits8(e) | notDigits8(f)
			w0 := conv8(a)*1e8 + conv8(b)
			w1 := conv8(e)*1e8 + conv8(f)
			h, lo = bits.Mul64(w0, 1e16)
			l0, c = bits.Add64(lo, w1, 0)
			l1 = h + c
			// a high half never reaches 2^64-1 so +carry can't wrap
			h, lo = bits.Mul64(l0, p)
			l0, c = bits.Add64(lo, tail, 0)
			w = h + c
			h, lo = bits.Mul64(l1, p)
			l1, c = bits.Add64(lo, w, 0)
			l2 = h + c
		case 3:
			_ = s[47]
			a, b := binary.LittleEndian.Uint64(s), binary.LittleEndian.Uint64(s[8:])
			e, f := binary.LittleEndian.Uint64(s[16:]), binary.LittleEndian.Uint64(s[24:])
			g, i := binary.LittleEndian.Uint64(s[32:]), binary.LittleEndian.Uint64(s[40:])
			bad |= notDigits8(a) | notDigits8(b) | notDigits8(e) | notDigits8(f) | notDigits8(g) | notDigits8(i)
			w0 := conv8(a)*1e8 + conv8(b)
			w1 := conv8(e)*1e8 + conv8(f)
			w2 := conv8(g)*1e8 + conv8(i)
			h, lo = bits.Mul64(w0, 1e16)
			l0, c = bits.Add64(lo, w1, 0)
			l1 = h + c
			h, lo = bits.Mul64(l0, 1e16)
			l0, c = bits.Add64(lo, w2, 0)
			w = h + c
			h, lo = bits.Mul64(l1, 1e16)
			l1, c = bits.Add64(lo, w, 0)
			l2 = h + c
			h, lo = bits.Mul64(l0, p)
			l0, c = bits.Add64(lo, tail, 0)
			w = h + c
			h, lo = bits.Mul64(l1, p)
			l1, c = bits.Add64(lo, w, 0)
			w = h + c
			h, lo = bits.Mul64(l2, p)
			l2, c = bits.Add64(lo, w, 0)
			l3 = h + c
		default:
			_ = s[63]
			a, b := binary.LittleEndian.Uint64(s), binary.LittleEndian.Uint64(s[8:])
			e, f := binary.LittleEndian.Uint64(s[16:]), binary.LittleEndian.Uint64(s[24:])
			g, i := binary.LittleEndian.Uint64(s[32:]), binary.LittleEndian.Uint64(s[40:])
			j, q := binary.LittleEndian.Uint64(s[48:]), binary.LittleEndian.Uint64(s[56:])
			bad |= notDigits8(a) | notDigits8(b) | notDigits8(e) | notDigits8(f) | notDigits8(g) | notDigits8(i) | notDigits8(j) | notDigits8(q)
			w0 := conv8(a)*1e8 + conv8(b)
			w1 := conv8(e)*1e8 + conv8(f)
			w2 := conv8(g)*1e8 + conv8(i)
			w3 := conv8(j)*1e8 + conv8(q)
			h, lo = bits.Mul64(w0, 1e16)
			l0, c = bits.Add64(lo, w1, 0)
			l1 = h + c
			h, lo = bits.Mul64(l0, 1e16)
			l0, c = bits.Add64(lo, w2, 0)
			w = h + c
			h, lo = bits.Mul64(l1, 1e16)
			l1, c = bits.Add64(lo, w, 0)
			l2 = h + c
			h, lo = bits.Mul64(l0, 1e16)
			l0, c = bits.Add64(lo, w3, 0)
			w = h + c
			h, lo = bits.Mul64(l1, 1e16)
			l1, c = bits.Add64(lo, w, 0)
			w = h + c
			h, lo = bits.Mul64(l2, 1e16)
			l2, c = bits.Add64(lo, w, 0)
			l3 = h + c
			h, lo = bits.Mul64(l0, p)
			l0, c = bits.Add64(lo, tail, 0)
			w = h + c
			h, lo = bits.Mul64(l1, p)
			l1, c = bits.Add64(lo, w, 0)
			w = h + c
			h, lo = bits.Mul64(l2, p)
			l2, c = bits.Add64(lo, w, 0)
			w = h + c
			h, lo = bits.Mul64(l3, p)
			l3, c = bits.Add64(lo, w, 0)
			if h+c != 0 {
				return Uint256{}, false
			}
		}
		if bad != 0 {
			return Uint256{}, false
		}
		return Uint256{l0, l1, l2, l3}, true
	}
	// base 16 takes hex with no prefix too
	if hex {
		s = s[2:]
	} else if len(s) == 0 {
		return Uint256{}, false
	}
	return uint256FromHex(s)
}

// appendDecimal appends decimal x
func (x Uint256) appendDecimal(dst []byte) []byte {
	return x.appendDecimalSign(dst, false)
}

// putHex writes hex x at the end of buf after 0x and returns where 0x starts.
// It writes the limbs up to the top nonzero one, 0x goes over the zeros
func (x Uint256) putHex(buf *[67]byte) int {
	nd := max((x.BitLen()+3)>>2, 1)
	switch (nd - 1) >> 4 {
	case 3:
		binary.BigEndian.PutUint64(buf[3:], hexChars8(x.L3>>32))
		binary.BigEndian.PutUint64(buf[11:], hexChars8(x.L3&0xFFFFFFFF))
		fallthrough
	case 2:
		binary.BigEndian.PutUint64(buf[19:], hexChars8(x.L2>>32))
		binary.BigEndian.PutUint64(buf[27:], hexChars8(x.L2&0xFFFFFFFF))
		fallthrough
	case 1:
		binary.BigEndian.PutUint64(buf[35:], hexChars8(x.L1>>32))
		binary.BigEndian.PutUint64(buf[43:], hexChars8(x.L1&0xFFFFFFFF))
		fallthrough
	default:
		binary.BigEndian.PutUint64(buf[51:], hexChars8(x.L0>>32))
		binary.BigEndian.PutUint64(buf[59:], hexChars8(x.L0&0xFFFFFFFF))
	}
	// after the digits, the stores above write leading zeros over it
	i := 65 - nd
	buf[i], buf[i+1] = '0', 'x'
	return i
}

// uint256FromHex parses hex digits of either case with no prefix or sign.
// For a nonempty s only, an empty s panics on bounds
func uint256FromHex(s []byte) (Uint256, bool) {
	// only zeros past 64 digits are dropped, the rest is parsed as is
	for len(s) > 64 && s[0] == '0' {
		s = s[1:]
	}
	n := len(s)
	if n > 64 {
		return Uint256{}, false
	}
	if n <= 8 {
		// chars go to the low n bytes, first char on top
		var x uint64
		if n >= 4 {
			// two overlapping 4-byte loads
			x = uint64(binary.BigEndian.Uint32(s))<<(8*uint(n-4)) | uint64(binary.BigEndian.Uint32(s[n-4:]))
		} else {
			// first, middle and last byte cover 1..3 chars
			x = uint64(s[0])<<(8*uint(n-1)) | uint64(s[(n-1)>>1])<<(8*uint(n-1-(n-1)>>1)) | uint64(s[n-1])
		}
		// left-pad with '0' to a full block
		x |= 0x3030303030303030 &^ (^uint64(0) >> (8 * uint(8-n) & 63))
		if notHexDigits8(x) != 0 {
			return Uint256{}, false
		}
		return Uint256{hexDigits8(x), 0, 0, 0}, true
	}
	// limbs of 16 chars from the end, the top one holds the first m chars
	m := (n-1)&15 + 1
	// its low block starts at max(m-8, 0),
	// the high block is the o chars before it
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
	var l0, l1, l2, l3 uint64
	// by n, the loads from the end need no bounds checks
	switch {
	case n <= 16:
		l0 = top
	case n <= 32:
		a, b := binary.BigEndian.Uint64(s[n-16:]), binary.BigEndian.Uint64(s[n-8:])
		bad |= notHexDigits8(a) | notHexDigits8(b)
		l0, l1 = hexDigits8(a)<<32|hexDigits8(b), top
	case n <= 48:
		a, b := binary.BigEndian.Uint64(s[n-16:]), binary.BigEndian.Uint64(s[n-8:])
		c, e := binary.BigEndian.Uint64(s[n-32:]), binary.BigEndian.Uint64(s[n-24:])
		bad |= notHexDigits8(a) | notHexDigits8(b) | notHexDigits8(c) | notHexDigits8(e)
		l0, l1, l2 = hexDigits8(a)<<32|hexDigits8(b), hexDigits8(c)<<32|hexDigits8(e), top
	default:
		a, b := binary.BigEndian.Uint64(s[n-16:]), binary.BigEndian.Uint64(s[n-8:])
		c, e := binary.BigEndian.Uint64(s[n-32:]), binary.BigEndian.Uint64(s[n-24:])
		f, g := binary.BigEndian.Uint64(s[n-48:]), binary.BigEndian.Uint64(s[n-40:])
		bad |= notHexDigits8(a) | notHexDigits8(b) | notHexDigits8(c) | notHexDigits8(e) | notHexDigits8(f) | notHexDigits8(g)
		l0, l1, l2, l3 = hexDigits8(a)<<32|hexDigits8(b), hexDigits8(c)<<32|hexDigits8(e), hexDigits8(f)<<32|hexDigits8(g), top
	}
	if bad != 0 {
		return Uint256{}, false
	}
	return Uint256{l0, l1, l2, l3}, true
}

// appendDecimalSign appends decimal x, after a minus sign when neg.
// For x < 2^64 neg is dropped: Int256 writes that case itself
func (x Uint256) appendDecimalSign(dst []byte, neg bool) []byte {
	l0, l1, l2, l3 := x.L0, x.L1, x.L2, x.L3
	// 19-digit chunks c0, c1, ... least significant first, then the top.
	// The divisions inline: a call keeps the quotients of the passes in memory
	// between the steps, slower on Graviton in our benchmarks
	var q, r, c0, c1, c2, c3, top uint64
	switch {
	case l3 != 0:
		// 58 to 78 digits
		q, r = div1e19Top(l3)
		l2, r = div2by1(r, l2, decimalChunk, decimalChunkReciprocal)
		l1, r = div2by1(r, l1, decimalChunk, decimalChunkReciprocal)
		l0, c0 = div2by1(r, l0, decimalChunk, decimalChunkReciprocal)
		l2, r = div2by1(q, l2, decimalChunk, decimalChunkReciprocal)
		l1, r = div2by1(r, l1, decimalChunk, decimalChunkReciprocal)
		l0, c1 = div2by1(r, l0, decimalChunk, decimalChunkReciprocal)
		l1, r = div2by1(l2, l1, decimalChunk, decimalChunkReciprocal)
		l0, c2 = div2by1(r, l0, decimalChunk, decimalChunkReciprocal)
		top, c3 = div2by1(l1, l0, decimalChunk, decimalChunkReciprocal)
		if top != 0 {
			return appendDecimalParts(dst, neg, top, c3, c2, c1, c0, 4)
		}
		return appendDecimalParts(dst, neg, c3, c2, c1, c0, 0, 3)
	case l2 != 0:
		// 39 to 58 digits
		q, r = div1e19Top(l2)
		l1, r = div2by1(r, l1, decimalChunk, decimalChunkReciprocal)
		l0, c0 = div2by1(r, l0, decimalChunk, decimalChunkReciprocal)
		l1, r = div2by1(q, l1, decimalChunk, decimalChunkReciprocal)
		l0, c1 = div2by1(r, l0, decimalChunk, decimalChunkReciprocal)
		top, c2 = div2by1(l1, l0, decimalChunk, decimalChunkReciprocal)
		if top != 0 {
			return appendDecimalParts(dst, neg, top, c2, c1, c0, 0, 3)
		}
		return appendDecimalParts(dst, neg, c2, c1, c0, 0, 0, 2)
	case l1 != 0:
		// 20 to 39 digits
		if l1 < decimalChunk {
			// the writer of Uint128.appendDecimal after the sign,
			// one division and no call: faster in our benchmarks
			top, c0 := div1e19Mask(l1, l0)
			nt := count64Digits(top)
			sign := 0
			if neg {
				sign = 1
			}
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
		// l1 >= 1e19: the quotient has a second limb of 1, so top > 0
		l0, c0 = div2by1(l1-decimalChunk, l0, decimalChunk, decimalChunkReciprocal)
		top, c1 = div2by1(1, l0, decimalChunk, decimalChunkReciprocal)
		return appendDecimalParts(dst, neg, top, c1, c0, 0, 0, 2)
	}
	// Int256 sends one limb to appendDecimalSign64 itself, neg is false here
	return appendDecimal64(dst, l0)
}

// div1e19Top returns v / 1e19 and the remainder, the quotient is 0 or 1
func div1e19Top(v uint64) (q, r uint64) {
	d, borrow := bits.Sub64(v, decimalChunk, 0)
	return borrow ^ 1, d + decimalChunk&-borrow
}
