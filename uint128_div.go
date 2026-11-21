package fat

import (
	"math/bits"

	"github.com/avpetkun/fat/internal/arch"
)

// Div64 returns x / y, it panics for y == 0
func (x Uint128) Div64(y uint64) Uint128 {
	if y == 0 {
		panic("fat.Uint128.Div64: division by zero")
	}
	return x.div64(y)
}

// Mod64 returns x % y, it panics for y == 0
func (x Uint128) Mod64(y uint64) uint64 {
	if y == 0 {
		panic("fat.Uint128.Mod64: division by zero")
	}
	return x.mod64(y)
}

// div64 is divMod64 without the remainder and its multiplication.
// For y != 0 only, y == 0 gives a wrong result or a panic
func (x Uint128) div64(y uint64) Uint128 {
	if arch.IsAmd64 {
		// a hardware 128-by-64 division per quotient limb there
		var q1 uint64
		r := x.L1
		if r >= y {
			q1 = r / y
			r -= q1 * y
		}
		q0, _ := bits.Div64(r, x.L0, y)
		return Uint128{q0, q1}
	}
	// the low limb by the reciprocal of y shifted to its top bit.
	// reciprocal2by1 is written out, a call would spill the limbs.
	// Graviton overlaps about three of these and divides one at a time,
	// so the length of the chain counts on independent values too
	s := uint(bits.LeadingZeros64(y))
	d := y << (s & 63)
	v := (1<<47 - 1) / (d>>40 + 1)
	// the top limb by a hardware division beside the reciprocal,
	// a branch on the length would mispredict.
	// y != 0 always: the test drops the call of panicdivide and the frame
	q1 := x.L1
	if y != 0 {
		q1 = x.L1 / y
	}
	r := x.L1 - q1*y
	d0 := d & 1
	e := (v>>1)&-d0 - v*(d>>1+d0)
	hi, _ := bits.Mul64(e, e)
	e += hi >> 22
	hi, _ = bits.Mul64(v<<19, e)
	v = v<<41 + hi
	hi, lo := bits.Mul64(v, d)
	_, c := bits.Add64(lo, d, 0)
	t, _ := bits.Add64(hi, d, c)
	// the last step of the reciprocal runs beside the product, not ahead:
	// the exact reciprocal is v - t with t = 0 or -1,
	// so the product takes uh&t more
	uh, ul := r<<(s&63)|x.L0>>1>>(^s&63), x.L0<<(s&63)
	qh, ql := bits.Mul64(v, uh)
	ql, c = bits.Add64(ql, ul, 0)
	qh, _ = bits.Add64(qh, uh, c)
	ql, c = bits.Add64(ql, uh&t, 0)
	qh, _ = bits.Add64(qh, 0, c)
	r = ul - d - qh*d
	q0 := qh + 1
	if r > ql {
		q0 = qh
		r += d
	}
	if r >= d {
		q0++
	}
	return Uint128{q0, q1}
}

// mod64 is divMod64 without the quotient.
// For y != 0 only, y == 0 gives a wrong result or a panic
func (x Uint128) mod64(y uint64) uint64 {
	if arch.IsAmd64 {
		// a hardware 128-by-64 division per quotient limb there
		r := x.L1
		if r >= y {
			r %= y
		}
		_, r = bits.Div64(r, x.L0, y)
		return r
	}
	// as div64, with the remainder of the shifted division shifted back
	r := x.L1
	if y != 0 {
		r = x.L1 % y
	}
	s := uint(bits.LeadingZeros64(y))
	d := y << (s & 63)
	v := (1<<47 - 1) / (d>>40 + 1)
	d0 := d & 1
	e := (v>>1)&-d0 - v*(d>>1+d0)
	hi, _ := bits.Mul64(e, e)
	e += hi >> 22
	hi, _ = bits.Mul64(v<<19, e)
	v = v<<41 + hi
	hi, lo := bits.Mul64(v, d)
	_, c := bits.Add64(lo, d, 0)
	t, _ := bits.Add64(hi, d, c)
	uh, ul := r<<(s&63)|x.L0>>1>>(^s&63), x.L0<<(s&63)
	qh, ql := bits.Mul64(v, uh)
	ql, c = bits.Add64(ql, ul, 0)
	qh, _ = bits.Add64(qh, uh, c)
	ql, c = bits.Add64(ql, uh&t, 0)
	qh, _ = bits.Add64(qh, 0, c)
	r = ul - d - qh*d
	if r > ql {
		r += d
	}
	if r >= d {
		r -= d
	}
	return r >> (s & 63)
}

// Div returns x / y, it panics for y == 0
func (x Uint128) Div(y Uint128) Uint128 {
	if y.L0|y.L1 == 0 {
		panic("fat.Uint128.Div: division by zero")
	}
	return x.div(y)
}

// Mod returns x % y, it panics for y == 0
func (x Uint128) Mod(y Uint128) Uint128 {
	if y.L0|y.L1 == 0 {
		panic("fat.Uint128.Mod: division by zero")
	}
	return x.mod(y)
}

// div is divMod without the remainder.
// For y != 0 only, y == 0 gives a wrong result or a panic
func (x Uint128) div(y Uint128) Uint128 {
	if arch.IsAmd64 {
		// a hardware 128-by-64 division per quotient limb there
		if y.L1 == 0 {
			var q1 uint64
			r := x.L1
			if r >= y.L0 {
				q1 = r / y.L0
				r -= q1 * y.L0
			}
			q0, _ := bits.Div64(r, x.L0, y.L0)
			return Uint128{q0, q1}
		}
		n := uint(bits.LeadingZeros64(y.L1))
		d := y.L1<<(n&63) | y.L0>>1>>(^n&63)
		q, _ := bits.Div64(x.L1>>1, x.L1<<63|x.L0>>1, d)
		q >>= ^n & 63
		if q != 0 {
			q--
		}
		hi, lo := bits.Mul64(y.L0, q)
		r0, b := bits.Sub64(x.L0, lo, 0)
		r1, _ := bits.Sub64(x.L1, hi+y.L1*q, b)
		_, b = bits.Sub64(r0, y.L0, 0)
		_, b = bits.Sub64(r1, y.L1, b)
		return Uint128{q + 1 - b, 0}
	}
	xm, ym := x.L1, y.L1
	if xm > 1<<33-1 {
		xm = 1<<33 - 1
	}
	if ym == 0 {
		ym = 1
	}
	if xm < ym {
		n, d := x.L1>>1, y.L1>>1+1
		z := x.L1 | y.L1
		if z == 0 {
			n = x.L0
		}
		if z == 0 {
			d = y.L0
		}
		q := n
		if d != 0 {
			q = n / d
		}
		hi, lo := bits.Mul64(q, y.L0)
		hi += q * y.L1
		r0, b := bits.Sub64(x.L0, lo, 0)
		r1, _ := bits.Sub64(x.L1, hi, b)
		_, b = bits.Sub64(r0, y.L0, 0)
		_, b = bits.Sub64(r1, y.L1, b)
		return Uint128{q + 1 - b, 0}
	}
	if y.L1 == 0 && y.L0 != 0 {
		y := y.L0
		q1 := x.L1 / y
		r := x.L1 - q1*y
		s := uint(bits.LeadingZeros64(y))
		d := y << (s & 63)
		v := (1<<47 - 1) / (d>>40 + 1)
		d0 := d & 1
		e := (v>>1)&-d0 - v*(d>>1+d0)
		hi, _ := bits.Mul64(e, e)
		e += hi >> 22
		hi, _ = bits.Mul64(v<<19, e)
		v = v<<41 + hi
		hi, lo := bits.Mul64(v, d)
		_, c := bits.Add64(lo, d, 0)
		hi, _ = bits.Add64(hi, d, c)
		v -= hi
		q0, _ := div2by1(r<<(s&63)|x.L0>>1>>(^s&63), x.L0<<(s&63), d, v)
		return Uint128{q0, q1}
	}
	n := uint(bits.LeadingZeros64(y.L1))
	d := y.L1<<(n&63) | y.L0>>1>>(^n&63)
	v := (1<<47 - 1) / (d>>40 + 1)
	d0 := d & 1
	e := (v>>1)&-d0 - v*(d>>1+d0)
	hi, _ := bits.Mul64(e, e)
	e += hi >> 22
	hi, _ = bits.Mul64(v<<19, e)
	v = v<<41 + hi
	hi, lo := bits.Mul64(v, d)
	_, c := bits.Add64(lo, d, 0)
	hi, _ = bits.Add64(hi, d, c)
	v -= hi
	q, _ := div2by1(x.L1>>1, x.L1<<63|x.L0>>1, d, v)
	q >>= ^n & 63
	if q != 0 {
		q--
	}
	hi, lo = bits.Mul64(y.L0, q)
	r := x.Subu(Uint128{lo, hi + y.L1*q})
	if !r.LessThan(y) {
		q++
	}
	return Uint128{q, 0}
}

// mod is divMod without the quotient.
// For y != 0 only, y == 0 gives a wrong result or a panic
func (x Uint128) mod(y Uint128) Uint128 {
	if arch.IsAmd64 {
		// a hardware 128-by-64 division per quotient limb there
		if y.L1 == 0 {
			r := x.L1
			if r >= y.L0 {
				r %= y.L0
			}
			_, r = bits.Div64(r, x.L0, y.L0)
			return Uint128{r, 0}
		}
		n := uint(bits.LeadingZeros64(y.L1))
		d := y.L1<<(n&63) | y.L0>>1>>(^n&63)
		q, _ := bits.Div64(x.L1>>1, x.L1<<63|x.L0>>1, d)
		q >>= ^n & 63
		if q != 0 {
			q--
		}
		hi, lo := bits.Mul64(y.L0, q)
		r0, b := bits.Sub64(x.L0, lo, 0)
		r1, _ := bits.Sub64(x.L1, hi+y.L1*q, b)
		t0, b := bits.Sub64(r0, y.L0, 0)
		t1, b := bits.Sub64(r1, y.L1, b)
		if b == 0 {
			r0 = t0
		}
		if b == 0 {
			r1 = t1
		}
		return Uint128{r0, r1}
	}
	xm, ym := x.L1, y.L1
	if xm > 1<<33-1 {
		xm = 1<<33 - 1
	}
	if ym == 0 {
		ym = 1
	}
	if xm < ym {
		n, d := x.L1>>1, y.L1>>1+1
		z := x.L1 | y.L1
		if z == 0 {
			n = x.L0
		}
		if z == 0 {
			d = y.L0
		}
		q := n
		if d != 0 {
			q = n / d
		}
		hi, lo := bits.Mul64(q, y.L0)
		hi += q * y.L1
		r0, b := bits.Sub64(x.L0, lo, 0)
		r1, _ := bits.Sub64(x.L1, hi, b)
		t0, b := bits.Sub64(r0, y.L0, 0)
		t1, b := bits.Sub64(r1, y.L1, b)
		if b == 0 {
			r0 = t0
		}
		if b == 0 {
			r1 = t1
		}
		return Uint128{r0, r1}
	}
	if y.L1 == 0 && y.L0 != 0 {
		y := y.L0
		q1 := x.L1 / y
		r := x.L1 - q1*y
		s := uint(bits.LeadingZeros64(y))
		d := y << (s & 63)
		v := (1<<47 - 1) / (d>>40 + 1)
		d0 := d & 1
		e := (v>>1)&-d0 - v*(d>>1+d0)
		hi, _ := bits.Mul64(e, e)
		e += hi >> 22
		hi, _ = bits.Mul64(v<<19, e)
		v = v<<41 + hi
		hi, lo := bits.Mul64(v, d)
		_, c := bits.Add64(lo, d, 0)
		hi, _ = bits.Add64(hi, d, c)
		v -= hi
		_, r = div2by1(r<<(s&63)|x.L0>>1>>(^s&63), x.L0<<(s&63), d, v)
		return Uint128{r >> (s & 63), 0}
	}
	n := uint(bits.LeadingZeros64(y.L1))
	d := y.L1<<(n&63) | y.L0>>1>>(^n&63)
	v := (1<<47 - 1) / (d>>40 + 1)
	d0 := d & 1
	e := (v>>1)&-d0 - v*(d>>1+d0)
	hi, _ := bits.Mul64(e, e)
	e += hi >> 22
	hi, _ = bits.Mul64(v<<19, e)
	v = v<<41 + hi
	hi, lo := bits.Mul64(v, d)
	_, c := bits.Add64(lo, d, 0)
	hi, _ = bits.Add64(hi, d, c)
	v -= hi
	q, _ := div2by1(x.L1>>1, x.L1<<63|x.L0>>1, d, v)
	q >>= ^n & 63
	if q != 0 {
		q--
	}
	hi, lo = bits.Mul64(y.L0, q)
	r := x.Subu(Uint128{lo, hi + y.L1*q})
	if !r.LessThan(y) {
		r = r.Subu(y)
	}
	return r
}
