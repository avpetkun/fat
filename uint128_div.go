package fat

import (
	"math"
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

// DivMod64 returns x / y and x % y, it panics for y == 0
func (x Uint128) DivMod64(y uint64) (Uint128, uint64) {
	if y == 0 {
		panic("fat.Uint128.DivMod64: division by zero")
	}
	return x.divMod64(y)
}

// divMod64 is DivMod64 for y != 0 only, y == 0 gives a wrong result or a panic
func (x Uint128) divMod64(y uint64) (Uint128, uint64) {
	if arch.IsAmd64 {
		// a hardware 128-by-64 division per quotient limb there.
		// The top limb takes one only when it is not below y,
		// so a quotient within 64 bits skips it by a predicted branch
		var q1 uint64
		r := x.L1
		if r >= y {
			q1 = r / y
			r -= q1 * y
		}
		q0, r := bits.Div64(r, x.L0, y)
		return Uint128{q0, q1}, r
	}
	// as div64, with the remainder from x = q*y + r in the low limb:
	// that of the shifted division would need a shift back
	s := uint(bits.LeadingZeros64(y))
	d := y << (s & 63)
	v := (1<<47 - 1) / (d>>40 + 1)
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
	return Uint128{q0, q1}, x.L0 - q0*y
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

// DivMod returns x / y and x % y, it panics for y == 0
func (x Uint128) DivMod(y Uint128) (q, r Uint128) {
	if y.L0|y.L1 == 0 {
		panic("fat.Uint128.DivMod: division by zero")
	}
	// returning the call itself costs 81 of the inlining budget of 80
	q, r = x.divMod(y)
	return
}

// divMod is DivMod with no calls and so no stack frame.
// For y != 0 only, y == 0 gives a wrong result or a panic
func (x Uint128) divMod(y Uint128) (Uint128, Uint128) {
	if arch.IsAmd64 {
		// a hardware 128-by-64 division per quotient limb there
		if y.L1 == 0 {
			var q1 uint64
			r := x.L1
			if r >= y.L0 {
				q1 = r / y.L0
				r -= q1 * y.L0
			}
			q0, r := bits.Div64(r, x.L0, y.L0)
			return Uint128{q0, q1}, Uint128{r, 0}
		}
		// y >= 2^64 leaves one quotient limb: x/2 by the top 64 bits of y,
		// then one correction, as divlu in Hacker's Delight 9-5
		n := uint(bits.LeadingZeros64(y.L1))
		d := y.L1<<(n&63) | y.L0>>1>>(^n&63)
		q, _ := bits.Div64(x.L1>>1, x.L1<<63|x.L0>>1, d)
		// one under the quotient or exact
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
		return Uint128{q + 1 - b, 0}, Uint128{r0, r1}
	}
	// x.L1 < y.L1, y >= 2^97 and both below 2^64 take one hardware division.
	// Conditional moves fold the three tests into one early branch
	xm, ym := x.L1, y.L1
	if xm > 1<<33-1 {
		xm = 1<<33 - 1
	}
	if ym == 0 {
		ym = 1
	}
	if xm < ym {
		// the halved top limbs with the divisor rounded up give q or q-1:
		// for y.L1 >= 2^33 under x / y by less than 2^63 / (2^32 * 2^32) + 1,
		// and for x.L1 < y.L1 both are 0
		n, d := x.L1>>1, y.L1>>1+1
		z := x.L1 | y.L1
		if z == 0 {
			n = x.L0
		}
		if z == 0 {
			d = y.L0
		}
		// d != 0 always: the test drops the call of panicdivide
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
		if b == 0 {
			q++
		}
		return Uint128{q, 0}, Uint128{r0, r1}
	}
	// y < 2^64 <= x: divMod64 written out, its call would cost a stack frame.
	// y.L0 != 0 always, the test tells the compiler so
	if y.L1 == 0 && y.L0 != 0 {
		y := y.L0
		q1 := x.L1 / y
		r := x.L1 - q1*y
		s := uint(bits.LeadingZeros64(y))
		d := y << (s & 63)
		// the reciprocal (2^128-1)/d - 2^64 as reciprocal2by1 does,
		// a call would spill the limbs
		v := (1<<47 - 1) / (d>>40 + 1)
		d0 := d & 1
		e := (v>>1)&-d0 - v*(d>>1+d0)
		hi, _ := bits.Mul64(e, e)
		e += hi >> 22
		hi, _ = bits.Mul64(v<<19, e)
		v = v<<41 + hi
		hi, lo := bits.Mul64(v, d)
		_, c := bits.Add64(lo, d, 0)
		// hi + d + c by one ADCS, a sum of three costs two instructions more
		hi, _ = bits.Add64(hi, d, c)
		v -= hi
		q0, r := div2by1(r<<(s&63)|x.L0>>1>>(^s&63), x.L0<<(s&63), d, v)
		return Uint128{q0, q1}, Uint128{r >> (s & 63), 0}
	}
	// 2^64 <= y < 2^97 leaves one quotient limb: x/2 by the top 64 bits of y,
	// then one correction, as divlu in Hacker's Delight 9-5.
	// x < y with equal top limbs gives 0 here too
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
	// one under the quotient or exact
	q >>= ^n & 63
	if q != 0 {
		q--
	}
	hi, lo = bits.Mul64(y.L0, q)
	r := x.Subu(Uint128{lo, hi + y.L1*q})
	if !r.LessThan(y) {
		q++
		r = r.Subu(y)
	}
	return Uint128{q, 0}, r
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

// uint128MulDivPanics, uint128MulDivModPanics and uint128MulDivCeilPanics are the panics by kernel status.
// One panic from a table keeps the wrappers within the inline budget
var (
	uint128MulDivPanics = [3]string{
		1: "fat.Uint128.MulDiv: overflow",
		2: "fat.Uint128.MulDiv: division by zero",
	}
	uint128MulDivModPanics = [3]string{
		1: "fat.Uint128.MulDivMod: overflow",
		2: "fat.Uint128.MulDivMod: division by zero",
	}
	uint128MulDivCeilPanics = [3]string{
		1: "fat.Uint128.MulDivCeil: overflow",
		2: "fat.Uint128.MulDivCeil: division by zero",
	}
)

// MulDiv returns x * y / d with the product in 256 bits.
// It panics for d == 0 and when the quotient overflows
func (x Uint128) MulDiv(y, d Uint128) (q Uint128) {
	q, _, st := x.mulDivMod(y, d)
	if st != 0 {
		panic(uint128MulDivPanics[st])
	}
	return
}

// MulDivo returns MulDivu and whether the quotient overflowed.
// It panics for d == 0
func (x Uint128) MulDivo(y, d Uint128) (q Uint128, overflow bool) {
	q, _, st := x.mulDivMod(y, d)
	if st > 1 {
		panic("fat.Uint128.MulDivo: division by zero")
	}
	return q, st != 0
}

// MulDivu returns x * y / d wrapped around, the product in 256 bits.
// It panics for d == 0
func (x Uint128) MulDivu(y, d Uint128) (q Uint128) {
	q, _, st := x.mulDivMod(y, d)
	if st > 1 {
		panic("fat.Uint128.MulDivu: division by zero")
	}
	return
}

// MulDivMod returns MulDiv and x * y % d.
// It panics for d == 0 and when the quotient overflows
func (x Uint128) MulDivMod(y, d Uint128) (q, r Uint128) {
	q, r, st := x.mulDivMod(y, d)
	if st != 0 {
		panic(uint128MulDivModPanics[st])
	}
	return
}

// MulDivCeil returns x * y / d rounded up, the product in 256 bits.
// It panics for d == 0 and when the quotient overflows
func (x Uint128) MulDivCeil(y, d Uint128) Uint128 {
	q, st := x.mulDivCeil(y, d)
	if st != 0 {
		panic(uint128MulDivCeilPanics[st])
	}
	return q
}

// AddMod returns (x + y) % m of the exact sum, it panics for m == 0
func (x Uint128) AddMod(y, m Uint128) Uint128 {
	if m.L0|m.L1 == 0 {
		panic("fat.Uint128.AddMod: division by zero")
	}
	s0, c := bits.Add64(x.L0, y.L0, 0)
	s1, c := bits.Add64(x.L1, y.L1, c)
	_, bx := bits.Sub64(x.L0, m.L0, 0)
	_, bx = bits.Sub64(x.L1, m.L1, bx)
	_, by := bits.Sub64(y.L0, m.L0, 0)
	_, by = bits.Sub64(y.L1, m.L1, by)
	// operands below m are the common case: x + y < 2m takes one subtraction,
	// picked by conditional moves as a carry or a sum past m is a coin flip
	if bx&by != 0 {
		t0, b := bits.Sub64(s0, m.L0, 0)
		t1, b := bits.Sub64(s1, m.L1, b)
		if c >= b {
			s0 = t0
		}
		if c >= b {
			s1 = t1
		}
		return Uint128{s0, s1}
	}
	if c != 0 {
		// 2^128 + s less m shifted to its top bit once or twice fits 128 bits,
		// with the same remainder
		h, l := m.L1, m.L0
		if h == 0 {
			h = m.L0
		}
		if m.L1 == 0 {
			l = 0
		}
		k := uint(bits.LeadingZeros64(h))
		h = h<<(k&63) | l>>1>>(^k&63)
		l <<= k & 63
		t0, b := bits.Sub64(s0, l, 0)
		t1, b := bits.Sub64(s1, h, b)
		u0, d := bits.Sub64(t0, l, 0)
		u1, _ := bits.Sub64(t1, h, d)
		if b == 0 {
			t0 = u0
		}
		if b == 0 {
			t1 = u1
		}
		s0, s1 = t0, t1
	}
	return Uint128{s0, s1}.mod(m)
}

// SubMod returns (x - y) mod m of the exact difference, in [0, m) for x < y too.
// It panics for m == 0
func (x Uint128) SubMod(y, m Uint128) Uint128 {
	if m.L0|m.L1 == 0 {
		panic("fat.Uint128.SubMod: division by zero")
	}
	d0, b := bits.Sub64(x.L0, y.L0, 0)
	d1, b := bits.Sub64(x.L1, y.L1, b)
	e0, c := bits.Add64(d0, m.L0, 0)
	e1, c := bits.Add64(d1, m.L1, c)
	_, t := bits.Sub64(d0, m.L0, 0)
	_, t = bits.Sub64(d1, m.L1, t)
	// operands below m are the common case: |x - y| < m takes x - y, or
	// x - y + m for x < y, picked by conditional moves as the sign is a coin flip.
	// The test is d < m without a borrow, y - x <= m by the carry of d + m with one
	if t&^b|c&b != 0 {
		if b != 0 {
			d0, d1 = e0, e1
		}
		return Uint128{d0, d1}
	}
	if b == 0 {
		return Uint128{d0, d1}.mod(m)
	}
	// m - (y - x) mod m, or 0 where m divides y - x
	d0, b = bits.Sub64(0, d0, 0)
	d1, _ = bits.Sub64(0, d1, b)
	r := Uint128{d0, d1}.mod(m)
	if r.L0|r.L1 == 0 {
		return r
	}
	r0, b := bits.Sub64(m.L0, r.L0, 0)
	r1, _ := bits.Sub64(m.L1, r.L1, b)
	return Uint128{r0, r1}
}

// MulMod returns x * y % m with the product in 256 bits, it panics for m == 0
func (x Uint128) MulMod(y, m Uint128) (r Uint128) {
	_, r, st := x.mulDivMod(y, m)
	if st > 1 {
		panic("fat.Uint128.MulMod: division by zero")
	}
	return
}

// mulDivMod is MulDivMod with the low 128 bits of q and a status:
// 1 when q overflowed, 2 for d == 0
func (x Uint128) mulDivMod(y, d Uint128) (q, r Uint128, st uint) {
	// the 256-bit product
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
	// a product within 128 bits
	if p2|p3 == 0 {
		if d.L0|d.L1 == 0 {
			return q, r, 2
		}
		q, r = Uint128{p0, p1}.divMod(d)
		return q, r, 0
	}
	// the quotient overflows when the high half is not below d
	_, b := bits.Sub64(p2, d.L0, 0)
	_, b = bits.Sub64(p3, d.L1, b)
	st = uint(1 - b)
	if d.L1 == 0 {
		// a zero d is the only one with no limb set
		if d.L0 == 0 {
			return q, r, 2
		}
		if st != 0 {
			qq, rr := Uint256{p0, p1, p2, p3}.divMod64(d.L0)
			return Uint128{qq.L0, qq.L1}, Uint128{rr, 0}, 1
		}
		if arch.IsAmd64 {
			// p3 = 0 and p2 < d leave two digits,
			// a hardware 128-by-64 division each there
			q1, rr := bits.Div64(p2, p1, d.L0)
			q0, rr := bits.Div64(rr, p0, d.L0)
			return Uint128{q0, q1}, Uint128{rr, 0}, 0
		}
		// p3 = 0 and p2 < d leave two digits,
		// by the reciprocal of d shifted to its top bit
		s := uint(bits.LeadingZeros64(d.L0)) & 63
		t := ^s & 63
		dn := d.L0 << s
		v := (1<<47 - 1) / (dn>>40 + 1)
		e := (v>>1)&-(dn&1) - v*(dn>>1+dn&1)
		hi, _ := bits.Mul64(e, e)
		e += hi >> 22
		hi, _ = bits.Mul64(v<<19, e)
		v = v<<41 + hi
		hi, lo := bits.Mul64(v, dn)
		_, c = bits.Add64(lo, dn, 0)
		v -= hi + dn + c
		q1, rr := div2by1(p2<<s|p1>>1>>t, p1<<s|p0>>1>>t, dn, v)
		q0, rr := div2by1(rr, p0<<s, dn, v)
		return Uint128{q0, q1}, Uint128{rr >> s, 0}, 0
	}
	// d >= 2^64 shifted to its top bit, >>1>>t is >>(64-s) giving 0 at s = 0
	s := uint(bits.LeadingZeros64(d.L1))
	t := ^s & 63
	d1 := d.L1<<s | d.L0>>1>>t
	d0 := d.L0 << s
	u4 := p3 >> 1 >> t
	u3 := p3<<s | p2>>1>>t
	u2 := p2<<s | p1>>1>>t
	u1 := p1<<s | p0>>1>>t
	u0 := p0 << s
	// reciprocal3by2m and two steps of udivrem3by2m written out:
	// faster than their calls on full-width operands in our benchmarks
	var v uint64
	if arch.IsAmd64 {
		// one hardware 128-by-64 division there
		v, _ = bits.Div64(^d1, math.MaxUint64, d1)
	} else {
		v = (1<<47 - 1) / (d1>>40 + 1)
		e := (v>>1)&-(d1&1) - v*(d1>>1+d1&1)
		hi, _ := bits.Mul64(e, e)
		e += hi >> 22
		hi, _ = bits.Mul64(v<<19, e)
		v = v<<41 + hi
		hi, lo := bits.Mul64(v, d1)
		_, c = bits.Add64(lo, d1, 0)
		v -= hi + d1 + c
	}
	p, c := bits.Add64(d1*v, d0, 0)
	_, b = bits.Sub64(p, d1, 0)
	m := c + c&^b
	v -= m
	p -= d1 * m
	t1, t0 := bits.Mul64(v, d0)
	p, c = bits.Add64(p, t1, 0)
	_, b = bits.Sub64(t0, d0, 0)
	_, b = bits.Sub64(p, d1, b)
	v -= c + c&^b
	r1, r0 := u3, u2
	if st != 0 {
		// the digit above 2^128 only reduces the high half
		_, r1, r0 = udivrem3by2m(u4, u3, u2, d1, d0, v)
	}
	q1, k := bits.Mul64(v, r1)
	k, c = bits.Add64(k, r0, 0)
	q1, _ = bits.Add64(q1, r1, c)
	r1 = r0 - q1*d1
	t1, t0 = bits.Mul64(d0, q1)
	r0, b = bits.Sub64(u1, t0, 0)
	r1, _ = bits.Sub64(r1, t1, b)
	s0, b := bits.Sub64(r0, d0, 0)
	s1, _ := bits.Sub64(r1, d1, b)
	if s1 < k {
		q1, r0, r1 = q1+1, s0, s1
	}
	if r1 >= d1 {
		if r1 > d1 || r0 >= d0 {
			q1++
			r0, b = bits.Sub64(r0, d0, 0)
			r1, _ = bits.Sub64(r1, d1, b)
		}
	}
	q0, k := bits.Mul64(v, r1)
	k, c = bits.Add64(k, r0, 0)
	q0, _ = bits.Add64(q0, r1, c)
	r1 = r0 - q0*d1
	t1, t0 = bits.Mul64(d0, q0)
	r0, b = bits.Sub64(u0, t0, 0)
	r1, _ = bits.Sub64(r1, t1, b)
	s0, b = bits.Sub64(r0, d0, 0)
	s1, _ = bits.Sub64(r1, d1, b)
	if s1 < k {
		q0, r0, r1 = q0+1, s0, s1
	}
	if r1 >= d1 {
		if r1 > d1 || r0 >= d0 {
			q0++
			r0, b = bits.Sub64(r0, d0, 0)
			r1, _ = bits.Sub64(r1, d1, b)
		}
	}
	return Uint128{q0, q1}, Uint128{r0>>s | r1<<1<<t, r1 >> s}, st
}

// mulDivCeil is MulDivCeil with the low 128 bits of q and a status as of mulDivMod
func (x Uint128) mulDivCeil(y, d Uint128) (Uint128, uint) {
	q, r, st := x.mulDivMod(y, d)
	// a remainder adds one, which carries out only of an overflowed q
	// or of the maximum. A zero d leaves r = 0
	nz := r.L0 | r.L1
	q0, c := bits.Add64(q.L0, (nz|-nz)>>63, 0)
	q1, c := bits.Add64(q.L1, 0, c)
	return Uint128{q0, q1}, st | uint(c)
}
