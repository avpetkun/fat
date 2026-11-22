package fat

import (
	"encoding/binary"
	"fmt"
	"math/bits"
)

// Int256FromHex parses hex s with an optional sign, 0x or 0X and leading zeros.
// It returns zero and ErrParse for a bad s or one out of range
func Int256FromHex[S string | []byte](s S) (x Int256, err error) {
	// named results keep it inlined with the text in the error
	x, ok := parseInt256([]byte(s), 16)
	if !ok {
		err = ErrParse{"Int256FromHex", string(s)}
	}
	return
}

// MustInt256FromHex is Int256FromHex that panics on a bad s
func MustInt256FromHex[S string | []byte](s S) (x Int256) {
	x, ok := parseInt256([]byte(s), 16)
	if !ok {
		panic(ErrParse{"MustInt256FromHex", string(s)})
	}
	return
}

// Int256FromDec parses decimal s with an optional sign and leading zeros,
// or m e n as m * 10^n, such as -12e3 or 5E+18.
// It returns zero and ErrParse for a bad s or one out of range
func Int256FromDec[S string | []byte](s S) (Int256, error) {
	// the bytes don't escape parseInt256, so the conversion copies nothing
	x, ok := parseInt256([]byte(s), 10)
	if !ok {
		return int256FromDecSlow("Int256FromDec", []byte(s))
	}
	return x, nil
}

// MustInt256FromDec is Int256FromDec that panics on a bad s
func MustInt256FromDec[S string | []byte](s S) (x Int256) {
	// a named result keeps it inlined
	x, ok := parseInt256([]byte(s), 10)
	if !ok {
		var err error
		if x, err = int256FromDecSlow("MustInt256FromDec", []byte(s)); err != nil {
			panic(err)
		}
	}
	return
}

// int256FromDecSlow is FromDec and Scan for s their parser turns down:
// m e n as m * 10^n, else the error of fn
func int256FromDecSlow(fn string, s []byte) (Int256, error) {
	if m, n, ok := sciParts(s); ok {
		if x, ok := parseInt256(m, 10); ok {
			if x.IsZero() {
				return x, nil
			}
			// a power below 2^63 takes the shorter product by one limb
			if n < len(pow10Table)-1 {
				if r, over := x.Mul64o(int64(pow10Table[n])); !over {
					return r, nil
				}
			} else if n < len(pow10x256)-1 {
				if r, over := x.Mulo(Int256(pow10x256[n])); !over {
					return r, nil
				}
			}
		}
	}
	return Int256{}, ErrParse{fn, string(s)}
}

var _ fmt.Stringer = (*Int256)(nil)

// String is x in base 10, as Dec
func (x Int256) String() string {
	return x.Dec()
}

// Dec is x in base 10
func (x Int256) Dec() string {
	var buf [78]byte
	return string(x.appendDecimal(buf[:0]))
}

// Hex is x in base 16 with a 0x prefix, -0x when negative
func (x Int256) Hex() string {
	var buf [67]byte
	return string(buf[x.putHex(&buf):])
}

// AppendDec appends x as Dec does
func (x Int256) AppendDec(dst []byte) []byte {
	return x.appendDecimal(dst)
}

// AppendHex appends x as Hex does
func (x Int256) AppendHex(dst []byte) []byte {
	// Hex on the stack, then one append
	var buf [67]byte
	return append(dst, buf[x.putHex(&buf):]...)
}

// PrettyDec is x in base 10 with sep between groups of 3 digits
func (x Int256) PrettyDec(sep byte) string {
	var buf [78]byte
	return prettyDecimal(x.UnsignedAbs().AppendDec(buf[:0]), x.IsNegative(), sep)
}

// putHex writes Hex to the end of buf and returns the index it starts at.
// It is Uint256.putHex on |x| written out, a call less
func (x Int256) putHex(buf *[67]byte) int {
	u := x.UnsignedAbs()
	nd := max((u.BitLen()+3)>>2, 1)
	switch (nd - 1) >> 4 {
	case 3:
		binary.BigEndian.PutUint64(buf[3:], hexChars8(u.L3>>32))
		binary.BigEndian.PutUint64(buf[11:], hexChars8(u.L3&0xFFFFFFFF))
		fallthrough
	case 2:
		binary.BigEndian.PutUint64(buf[19:], hexChars8(u.L2>>32))
		binary.BigEndian.PutUint64(buf[27:], hexChars8(u.L2&0xFFFFFFFF))
		fallthrough
	case 1:
		binary.BigEndian.PutUint64(buf[35:], hexChars8(u.L1>>32))
		binary.BigEndian.PutUint64(buf[43:], hexChars8(u.L1&0xFFFFFFFF))
		fallthrough
	default:
		binary.BigEndian.PutUint64(buf[51:], hexChars8(u.L0>>32))
		binary.BigEndian.PutUint64(buf[59:], hexChars8(u.L0&0xFFFFFFFF))
	}
	// after the digits, the stores above write leading zeros over it
	i := 64 - nd
	buf[i], buf[i+1], buf[i+2] = '-', '0', 'x'
	return i + 1 - int(x.L3>>63)
}

// appendDecimal appends decimal x
func (x Int256) appendDecimal(dst []byte) []byte {
	u, neg := x.UnsignedAbs(), int64(x.L3) < 0
	if u.L1|u.L2|u.L3 == 0 {
		// appendDecimalSign would branch on the sign here
		return appendDecimalSign64(dst, neg, u.L0)
	}
	return u.appendDecimalSign(dst, neg)
}

// parseInt256 parses s after an optional sign in base 10 or 16.
// Base 16 takes an optional 0x or 0X, base 0 is 16 after one and 10 otherwise
func parseInt256(s []byte, base int) (Int256, bool) {
	if len(s) == 0 {
		return Int256{}, false
	}
	// '+' and '-' differ only in bit 1, the sign is skipped without a branch
	c := s[0]
	k := 0
	if (c-'+')&^2 == 0 {
		k = 1
	}
	var neg uint64
	if c == '-' {
		neg = 1
	}
	// a second sign fails in the digit parsers
	t := s[k:]
	var u Uint256
	var ok bool
	hex := len(t) > 2 && t[0] == '0' && t[1]|0x20 == 'x'
	if base == 16 || base == 0 && hex {
		// a sign after the prefix is a bad symbol, base 16 needs no prefix
		if hex {
			t = t[2:]
		} else if len(t) == 0 {
			return Int256{}, false
		}
		u, ok = uint256FromHex(t)
	} else {
		// decimal written out here, a call less; a 0x prefix fails on the x
		// leading zeros don't count towards the length limit
		for len(t) > 1 && t[0] == '0' {
			t = t[1:]
		}
		n := len(t)
		if n <= 19 {
			var l0 uint64
			if n <= 8 {
				if n == 0 {
					return Int256{}, false
				}
				// chars go to the top n bytes
				sh := 8 * uint(8-n) & 63
				var x uint64
				if n >= 4 {
					// two overlapping 4-byte loads
					x = uint64(binary.LittleEndian.Uint32(t))<<sh | uint64(binary.LittleEndian.Uint32(t[n-4:]))<<32
				} else {
					// first, middle and last byte cover 1..3 chars
					x = (uint64(t[0]) | uint64(t[(n-1)>>1])<<(8*uint((n-1)>>1)) | uint64(t[n-1])<<(8*uint(n-1))) << sh
				}
				// left-pad with '0' to a full block
				x |= 0x3030303030303030 &^ (^uint64(0) << sh)
				if notDigits8(x) != 0 {
					return Int256{}, false
				}
				l0 = conv8(x)
			} else {
				a := binary.LittleEndian.Uint64(t)
				b := binary.LittleEndian.Uint64(t[n-8:])
				if n <= 16 {
					// two overlapping blocks
					if notDigits8(a)|notDigits8(b) != 0 {
						return Int256{}, false
					}
					// drop digits shared with a
					b &= ^uint64(0) << (8 * uint(16-n))
					l0 = conv8(a)*pow10Table[n-8] + conv8(b)
				} else {
					e := binary.LittleEndian.Uint64(t[8:])
					if notDigits8(a)|notDigits8(e)|notDigits8(b) != 0 {
						return Int256{}, false
					}
					// drop digits shared with e
					b &= ^uint64(0) << (8 * uint(24-n))
					l0 = (conv8(a)*1e8+conv8(e))*pow10Table[n-16] + conv8(b)
				}
			}
			// (l0 ^ m) - m, the higher limbs are the borrow
			m := -neg
			r0, b := bits.Sub64(l0^m, m, 0)
			return Int256{r0, -b, -b, -b}, true
		}
		if n > 78 {
			return Int256{}, false
		}
		// k 16-digit windows from the start, the nt digits left are the tail
		k := (n - 1) >> 4
		nt := n - 16*k
		p := pow10Table[nt]
		// the window ending at len(t), whose first 16-nt bytes overlap the last
		ta := binary.LittleEndian.Uint64(t[n-16:])
		tb := binary.LittleEndian.Uint64(t[n-8:])
		bad := notDigits8(ta) | notDigits8(tb)
		sm := 16 - nt
		dm := sm - 8
		dm &^= dm >> (bits.UintSize - 1)
		// zeroed bytes count as zero digits, a shift of 64 or more clears it
		ta &= ^uint64(0) << (8 * uint(sm))
		tb &= ^uint64(0) << (8 * uint(dm))
		tail := conv8(ta)*1e8 + conv8(tb)
		var l0, l1, l2, l3, h, lo, c, w uint64
		switch k {
		case 1:
			_ = t[15]
			a, b := binary.LittleEndian.Uint64(t), binary.LittleEndian.Uint64(t[8:])
			bad |= notDigits8(a) | notDigits8(b)
			w0 := conv8(a)*1e8 + conv8(b)
			// x*10^nt + tail
			h, lo = bits.Mul64(w0, p)
			l0, c = bits.Add64(lo, tail, 0)
			l1 = h + c
		case 2:
			_ = t[31]
			a, b := binary.LittleEndian.Uint64(t), binary.LittleEndian.Uint64(t[8:])
			e, f := binary.LittleEndian.Uint64(t[16:]), binary.LittleEndian.Uint64(t[24:])
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
			_ = t[47]
			a, b := binary.LittleEndian.Uint64(t), binary.LittleEndian.Uint64(t[8:])
			e, f := binary.LittleEndian.Uint64(t[16:]), binary.LittleEndian.Uint64(t[24:])
			g, i := binary.LittleEndian.Uint64(t[32:]), binary.LittleEndian.Uint64(t[40:])
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
			_ = t[63]
			a, b := binary.LittleEndian.Uint64(t), binary.LittleEndian.Uint64(t[8:])
			e, f := binary.LittleEndian.Uint64(t[16:]), binary.LittleEndian.Uint64(t[24:])
			g, i := binary.LittleEndian.Uint64(t[32:]), binary.LittleEndian.Uint64(t[40:])
			j, q := binary.LittleEndian.Uint64(t[48:]), binary.LittleEndian.Uint64(t[56:])
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
				return Int256{}, false
			}
		}
		u, ok = Uint256{l0, l1, l2, l3}, bad == 0
	}
	// u must fit 255 bits, or be exactly 2^255 when negative
	if !ok || u.L3>>63 != 0 && !(neg == 1 && u.L0|u.L1|u.L2 == 0 && u.L3 == 1<<63) {
		return Int256{}, false
	}
	// (u ^ m) - m negates when m is all ones
	m := -neg
	r0, b := bits.Sub64(u.L0^m, m, 0)
	r1, b := bits.Sub64(u.L1^m, m, b)
	r2, b := bits.Sub64(u.L2^m, m, b)
	r3, _ := bits.Sub64(u.L3^m, m, b)
	return Int256{r0, r1, r2, r3}, true
}
