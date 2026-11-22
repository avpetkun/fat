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

// ExpMod returns x^y % m, 0^0 is 1, it panics for m == 0
func (x Uint128) ExpMod(y, m Uint128) Uint128 {
	r, zero := x.expMod(y, m)
	if zero {
		panic("fat.Uint128.ExpMod: division by zero")
	}
	return r
}

// expMod is ExpMod and true for m == 0, with a zero result
func (x Uint128) expMod(y, m Uint128) (Uint128, bool) {
	// m == 1 gives 0, one test for both
	if m.L1 == 0 && m.L0 <= 1 {
		return Uint128{}, m.L0 == 0
	}
	if y.IsZero() {
		return Uint128{L0: 1}, false
	}
	b := x
	if !b.LessThan(m) {
		b = b.mod(m)
	}
	if m.L0&1 == 0 {
		return expModEven128(b, y, m), false
	}
	return expModOdd128(b, y.Uint256(), m), false
}

// GCD returns the greatest common divisor of x and y, x for y == 0
func (x Uint128) GCD(y Uint128) Uint128 {
	if x.L0|x.L1 == 0 {
		return y
	}
	if y.L0|y.L1 == 0 {
		return x
	}
	// one division takes the difference in length at once,
	// where a binary step removes a bit or two
	if x.LessThan(y) {
		x, y = y, x
	}
	x = x.mod(y)
	if x.L0|x.L1 == 0 {
		return y
	}
	// the common power of two, then both odd
	tx, ty := x.TrailingZeros(), y.TrailingZeros()
	return gcdOdd128(x.Rsh(tx), y.Rsh(ty)).Lsh(min(tx, ty))
}

// ModInverse returns y with x * y % m == 1 and true, or 0 and false where x
// and m share a factor and none exists; for m == 1 it is 0 and true.
// It panics for m == 0
func (x Uint128) ModInverse(m Uint128) (Uint128, bool) {
	if m.L0|m.L1 == 0 {
		panic("fat.Uint128.ModInverse: division by zero")
	}
	if !x.LessThan(m) {
		x = x.mod(m)
	}
	if m.L0&1 != 0 {
		return invOdd128(x, m)
	}
	// an even m takes an odd x, and 1 is its own inverse
	if x.L0&1 == 0 {
		return Uint128{}, false
	}
	if x.L1 == 0 && x.L0 == 1 {
		return x, true
	}
	// with m*t = 1 (mod x) from the odd modulus x, m*t = 1 + x*s
	// and x*(m - s) = 1 (mod m), where s = m*t / x < m
	t, ok := invOdd128(m.mod(x), x)
	if !ok {
		return Uint128{}, false
	}
	s, _, _ := m.mulDivMod(t, x)
	return m.Subu(s), true
}

// expModEven128 is expModEven on 128 bits
func expModEven128(b, y, m Uint128) Uint128 {
	k := m.TrailingZeros()
	mask := Uint128{}.Not().Rsh(128 - k)
	c := b.Expu(y).And(mask)
	q := m.Rsh(k)
	// q is odd, so q == 1 by one OR
	if q.L0^1|q.L1 == 0 {
		return c
	}
	if !b.LessThan(q) {
		b = b.mod(q)
	}
	a := expModOdd128(b, y.Uint256(), q)
	// 1/q % 2^128 by Newton steps
	v := q
	for range 6 {
		v = v.Mulu(Uint128{L0: 2}.Subu(q.Mulu(v)))
	}
	return a.Addu(q.Mulu(c.Subu(a).Mulu(v).And(mask)))
}

// expModOdd128 is expModOdd on 128 bits
func expModOdd128(b Uint128, y Uint256, m Uint128) Uint128 {
	if m.L1 == 0 {
		return Uint128{L0: expModMont64(b.L0, y, m.L0)}
	}
	return expModMont128(b, y, m)
}

// montMul128 returns a * b / 2^128 % m by Montgomery's product, montMul
// on two limbs. m must be odd and a, b below it, mp = -1/m mod 2^64, else
// it is wrong
func montMul128(a, b, m Uint128, mp uint64) Uint128 {
	h0, p0 := bits.Mul64(a.L0, b.L0)
	h1, l1 := bits.Mul64(a.L1, b.L0)
	h2, l2 := bits.Mul64(a.L0, b.L1)
	h3, l3 := bits.Mul64(a.L1, b.L1)
	p1, c := bits.Add64(h0, l1, 0)
	p2, c := bits.Add64(h1, l3, c)
	p3 := h3 + c
	p1, c = bits.Add64(p1, l2, 0)
	p2, c = bits.Add64(p2, h2, c)
	p3 += c
	// k carries into the limb above each round; the low half of u*m0 is
	// -p_i by the choice of u, so only its high half is taken
	var u, k uint64
	u = p0 * mp
	c, _ = bits.Mul64(u, m.L0)
	c += uint64(nonzero(p0))
	c, p1 = madd(u, m.L1, p1, c)
	p2, k = bits.Add64(p2, c, 0)
	u = p1 * mp
	c, _ = bits.Mul64(u, m.L0)
	c += uint64(nonzero(p1))
	c, p2 = madd(u, m.L1, p2, c)
	p3, k = bits.Add64(p3, c, k)
	// p3:p2 + k*2^128 < 2m: less m unless below it, picked by a mask
	r0, br := bits.Sub64(p2, m.L0, 0)
	r1, br := bits.Sub64(p3, m.L1, br)
	keep := -(br &^ k)
	return Uint128{r0&^keep | p2&keep, r1&^keep | p3&keep}
}

// montSqr128 is montMul128(a, a, m, mp) by 3 limb products for 4
func montSqr128(a, m Uint128, mp uint64) Uint128 {
	h01, l01 := bits.Mul64(a.L0, a.L1)
	s0h, p0 := bits.Mul64(a.L0, a.L0)
	s1h, s1l := bits.Mul64(a.L1, a.L1)
	// the cross product doubled, the squares on the even limbs
	p1, c := bits.Add64(l01<<1, s0h, 0)
	p2, c := bits.Add64(h01<<1|l01>>63, s1l, c)
	p3 := h01>>63 + s1h + c
	// k carries into the limb above each round; the low half of u*m0 is
	// -p_i by the choice of u, so only its high half is taken
	var u, k uint64
	u = p0 * mp
	c, _ = bits.Mul64(u, m.L0)
	c += uint64(nonzero(p0))
	c, p1 = madd(u, m.L1, p1, c)
	p2, k = bits.Add64(p2, c, 0)
	u = p1 * mp
	c, _ = bits.Mul64(u, m.L0)
	c += uint64(nonzero(p1))
	c, p2 = madd(u, m.L1, p2, c)
	p3, k = bits.Add64(p3, c, k)
	// p3:p2 + k*2^128 < 2m: less m unless below it, picked by a mask
	r0, br := bits.Sub64(p2, m.L0, 0)
	r1, br := bits.Sub64(p3, m.L1, br)
	keep := -(br &^ k)
	return Uint128{r0&^keep | p2&keep, r1&^keep | p3&keep}
}

// expModMont128 is expModMont for an odd m in 2^64..2^128-1, y > 0 and
// b < m, by Montgomery products modulo R = 2^128
func expModMont128(b Uint128, y Uint256, m Uint128) Uint128 {
	mp := montInv(m.L0)
	// R % m from 2^128 - m, then R^2 % m to take b to R*b % m
	r := Uint128{}.Subu(m).mod(m)
	_, r2, _ := r.mulDivMod(r, m)
	var odd [8]Uint128
	odd[0] = montMul128(b, r2, m, mp)
	n, w := y.BitLen(), expWindow(y.BitLen())
	if w > 1 {
		b2 := montSqr128(odd[0], m, mp)
		for k := 1; k < 1<<(w-1); k++ {
			odd[k] = montMul128(odd[k-1], b2, m, mp)
		}
	}
	// the top bit of y is set, so the first window starts r
	started := false
	for i := n - 1; i >= 0; {
		if y.Bit(i) == 0 {
			r = montSqr128(r, m, mp)
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
				r = montSqr128(r, m, mp)
			}
		}
		if started {
			r = montMul128(r, odd[v>>1], m, mp)
		} else {
			r, started = odd[v>>1], true
		}
		i = j - 1
	}
	// out of the Montgomery form
	return montMul128(r, Uint128{L0: 1}, m, mp)
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

// DivToFloat64 returns the float64 nearest to x / y, ties go to the even one.
// It panics for y == 0
func (x Uint128) DivToFloat64(y Uint128) float64 {
	// the top limb of each and the one below, nx and ny count the limbs above
	xh, xm, nx := x.L1, x.L0, uint64(0)
	if x.L1 == 0 {
		xh, xm, nx = x.L0, 0, 1
	}
	yh, ym, ny := y.L1, y.L0, uint64(0)
	if y.L1 == 0 {
		yh, ym, ny = y.L0, 0, 1
	}
	// u and d are the top 64 bits of x and y, u halved when not below d.
	// The digit q = u * 2^64 / d then has 64 bits,
	// and the quotient of the whole x and y is within 2 of q.
	// So q rounds as the quotient does unless a middle between two floats,
	// an odd multiple of 2^10, lies as near:
	// those and y = 0, which has no top bit, are left to divFloatExact
	if arch.IsAmd64 {
		// the shifts by the bit lengths, BSR gives those there
		lx, ly := uint(bits.Len64(xh)), uint(bits.Len64(yh))
		u := xh<<(-lx&63) | xm>>1>>((lx-1)&63)
		d := yh<<(-ly&63) | ym>>1>>((ly-1)&63)
		var h uint64
		if u >= d {
			h = 1
		}
		u >>= h & 1
		// the exponent of half the digit
		e := uint64(lx) - uint64(ly) + (ny-nx)<<6 + h + (1023 - 64 + 1)
		if int64(d) < 0 {
			// one hardware division gives the digit itself,
			// and the quotient lies above q-2 and below q+3.
			// Away from a middle, bit 0 of q changes nothing,
			// and q>>1 converts as a signed word with no branch
			q, _ := bits.Div64(u, 0, d)
			if (q-1<<10+2)&(1<<11-1) > 3 {
				return float64(int64(q>>1)) * math.Float64frombits(e<<52)
			}
		}
	} else {
		sx, sy := uint(bits.LeadingZeros64(xh)), uint(bits.LeadingZeros64(yh))
		tx := ^sx
		u := xh<<(sx&63) | xm>>1>>(tx&63)
		d := yh<<(sy&63) | ym>>1>>(^sy&63)
		var h uint64
		if u >= d {
			h = 1
		}
		u >>= h & 1
		// the exponent of the digit, ^sx is -sx - 1
		e := uint64(tx) + uint64(sy) + (ny-nx)<<6 + h + (1023 - 64 + 1)
		if int64(d) < 0 {
			// reciprocal2by1 without its last step,
			// which gives (2^128-1)/d - 2^64 or one less.
			// For t = d/2^64 the digit by it is less than 2t + 1 short
			// and the bits under u add less than 1/t,
			// so the quotient lies above q-2 and below q+4,
			// as 2t + 1/t is at most 3 for t from 1/2 to 1
			v := (1<<47 - 1) / (d>>40 + 1)
			d0 := d & 1
			t := (v>>1)&-d0 - v*(d>>1+d0)
			hi, _ := bits.Mul64(t, t)
			t += hi >> 22
			hi, _ = bits.Mul64(v<<19, t)
			v = v<<41 + hi
			hi, _ = bits.Mul64(v, u)
			q := u + hi
			if (q-1<<10+3)&(1<<11-1) > 4 {
				return float64(q) * math.Float64frombits(e<<52)
			}
		}
	}
	return divFloatExact(Uint256{x.L0, x.L1, 0, 0}, Uint256{y.L0, y.L1, 0, 0}, 0, "fat.Uint128.DivToFloat64: division by zero")
}

// DivToFloat64x returns DivToFloat64 and whether it is exactly x / y.
// It panics for y == 0
func (x Uint128) DivToFloat64x(y Uint128) (float64, bool) {
	// as DivToFloat64, digits near a multiple of 2^10 go to divFloatNear128,
	// as a float equal to the quotient ends at a multiple of 2^11, x = 0 too.
	// Only y = 0 has no top bit in d
	xh, xm, nx := x.L1, x.L0, uint64(0)
	if x.L1 == 0 {
		xh, xm, nx = x.L0, 0, 1
	}
	yh, ym, ny := y.L1, y.L0, uint64(0)
	if y.L1 == 0 {
		yh, ym, ny = y.L0, 0, 1
	}
	if arch.IsAmd64 {
		lx, ly := uint(bits.Len64(xh)), uint(bits.Len64(yh))
		u := xh<<(-lx&63) | xm>>1>>((lx-1)&63)
		d := yh<<(-ly&63) | ym>>1>>((ly-1)&63)
		var h uint64
		if u >= d {
			h = 1
		}
		u1 := u
		u >>= h & 1
		e := uint64(lx) - uint64(ly) + (ny-nx)<<6 + h + (1023 - 64 + 1)
		if int64(d) < 0 {
			q, _ := bits.Div64(u, 0, d)
			if (q+2)&(1<<10-1) > 3 {
				return float64(int64(q>>1)) * math.Float64frombits(e<<52), false
			}
			return divFloatNear128(u1, xm<<(-lx&63), d, ym<<(-ly&63), q, (q+2)&^(1<<10-1), h, e-1, 0)
		}
	} else {
		sx, sy := uint(bits.LeadingZeros64(xh)), uint(bits.LeadingZeros64(yh))
		tx := ^sx
		u := xh<<(sx&63) | xm>>1>>(tx&63)
		d := yh<<(sy&63) | ym>>1>>(^sy&63)
		var h uint64
		if u >= d {
			h = 1
		}
		u1 := u
		u >>= h & 1
		e := uint64(tx) + uint64(sy) + (ny-nx)<<6 + h + (1023 - 64 + 1)
		if int64(d) < 0 {
			v := (1<<47 - 1) / (d>>40 + 1)
			d0 := d & 1
			t := (v>>1)&-d0 - v*(d>>1+d0)
			hi, _ := bits.Mul64(t, t)
			t += hi >> 22
			hi, _ = bits.Mul64(v<<19, t)
			v = v<<41 + hi
			hi, _ = bits.Mul64(v, u)
			q := u + hi
			if (q+3)&(1<<10-1) > 4 {
				return float64(q) * math.Float64frombits(e<<52), false
			}
			return divFloatNear128(u1, xm<<(sx&63), d, ym<<(sy&63), q, (q+3)&^(1<<10-1), h, e, 0)
		}
	}
	panic("fat.Uint128.DivToFloat64x: division by zero")
}

// divFloatNear128 is divFloatNear on two limbs, for DivToFloat64x only.
// <u1, u0> and <d1, d0> are x and y shifted to their top bits,
// q the digit, c the multiple of 2^10 next to it, h 1 when u1 was halved,
// e the exponent of the digit and neg the sign bit.
// Other inputs give a wrong result
func divFloatNear128(u1, u0, d1, d0, q, c, h, e, neg uint64) (float64, bool) {
	if c == 0 {
		// a zero numerator, else c is 2^64 with q all ones under it
		if q == 0 {
			return 0, true
		}
		return float64(q) * math.Float64frombits(e<<52|neg), false
	}
	n2, n1, n0 := u1, u0, uint64(0)
	if h != 0 {
		n2, n1, n0 = u1>>1, u1<<63|u0>>1, u0<<63
	}
	h0, p0 := bits.Mul64(c, d0)
	h1, l1 := bits.Mul64(c, d1)
	p1, cy := bits.Add64(h0, l1, 0)
	p2 := h1 + cy
	_, b := bits.Sub64(n0, p0, 0)
	_, b = bits.Sub64(n1, p1, b)
	_, b = bits.Sub64(n2, p2, b)
	if (n0^p0)|(n1^p1)|(n2^p2) == 0 {
		return float64(c) * math.Float64frombits(e<<52|neg), bits.Len64(c)-bits.TrailingZeros64(c) <= 53
	}
	return float64(c+1-2*b) * math.Float64frombits(e<<52|neg), false
}

// gcdOdd128 is the GCD of a and b, by Pornin's optimized binary GCD while one
// is past 64 bits, then by gcdOdd64. For an odd b and a nonzero a only
func gcdOdd128(a, b Uint128) Uint128 {
	for a.L1|b.L1 != 0 {
		// approxOne128 inlines here, approx128 is a call: 1.5% on full widths
		s := 95 - a.Or(b).LeadingZeros()
		f0, g0, f1, g1 := gcdSteps31(approxOne128(a, s), approxOne128(b, s))
		a, b = linAbs128(a, b, f0, g0), linAbs128(a, b, f1, g1)
		if a.L0|a.L1 == 0 {
			return b
		}
	}
	// a stays nonzero: the loop returns on a zero one
	return Uint128{gcdOdd64(a.L0>>uint(bits.TrailingZeros64(a.L0)), b.L0), 0}
}

// invOdd128 returns 1/x mod m and true, or false where x and m share a factor.
// For an odd m and x < m only, else it gives a wrong result.
// Pornin's optimized binary GCD of x and m keeps a = u*x and b = v*x (mod m),
// u and v divided by 2^31 mod m per round as a and b are, so b = 1 leaves
// v = 1/x with nothing to divide out. u and v take two rounds at once by the
// product of their factors, and a and b go on in one limb once they fit
func invOdd128(x, m Uint128) (Uint128, bool) {
	if m.L1 == 0 {
		r, ok := invOdd64(x.L0, m.L0)
		return Uint128{r, 0}, ok
	}
	mp := montInv(m.L0)
	a, b, u, v := x, m, Uint128{1, 0}, Uint128{}
	// the factors of a round that waits for the next one
	var p0, q0, p1, q1 int64
	odd := false
	for a.L1|b.L1 != 0 {
		// a zero a leaves b, the GCD, past 64 bits
		if a.L0|a.L1 == 0 {
			return Uint128{}, false
		}
		f0, g0, f1, g1 := gcdSteps31(approx128(a, b))
		na, sa := linOne128(a, b, f0, g0)
		nb, sb := linOne128(a, b, f1, g1)
		a, b = na, nb
		f0, g0, f1, g1 = foldSigns(f0, g0, f1, g1, sa, sb)
		if odd {
			f0, g0, f1, g1 = matMul(f0, g0, f1, g1, p0, q0, p1, q1)
			u, v = modLin128(u, v, f0, g0, 62, m, mp), modLin128(u, v, f1, g1, 62, m, mp)
		}
		p0, q0, p1, q1 = f0, g0, f1, g1
		odd = !odd
	}
	// within 64 bits the approximations are the values themselves
	c, d := a.L0, b.L0
	for c != 0 {
		f0, g0, f1, g1 := gcdSteps31(c, d)
		nc, sc := linOne64(c, d, f0, g0)
		nd, sd := linOne64(c, d, f1, g1)
		c, d = nc, nd
		f0, g0, f1, g1 = foldSigns(f0, g0, f1, g1, sc, sd)
		if odd {
			f0, g0, f1, g1 = matMul(f0, g0, f1, g1, p0, q0, p1, q1)
			u, v = modLin128(u, v, f0, g0, 62, m, mp), modLin128(u, v, f1, g1, 62, m, mp)
		}
		p0, q0, p1, q1 = f0, g0, f1, g1
		odd = !odd
	}
	if d != 1 {
		return Uint128{}, false
	}
	if odd {
		v = modLin128(u, v, p1, q1, 31, m, mp)
	}
	return v, true
}

// linOne64 is linOne128 on one limb
func linOne64(a, b uint64, f, g int64) (uint64, uint64) {
	// a negative f is uint64(f) - 2^64
	h, l := bits.Mul64(a, uint64(f))
	h -= a & uint64(f>>63)
	k, m := bits.Mul64(b, uint64(g))
	k -= b & uint64(g>>63)
	l, c := bits.Add64(l, m, 0)
	h += k + c
	// below 2^95 in size: the shifted value is below 2^64 in size
	n := uint64(int64(h) >> 63)
	return (l>>31 | h<<33 ^ n) - n, n
}

// approxOne128 returns the low 31 bits of a under its bits s..s+32, one value
// of approx128. For a < 2^(s+33) and s in 31..95 only
func approxOne128(a Uint128, s int) uint64 {
	return a.Rsh(s).L0<<31 | a.L0&(1<<31-1)
}

// approx128 returns the approximations of a and b for gcdSteps31: the low 31
// bits under the 33 from the top bit of the longer one, or the values
// themselves within 64 bits. For an a or b past 2^31 only
func approx128(a, b Uint128) (uint64, uint64) {
	// the window starts at bit s in 31..95
	s := uint(max(95-a.Or(b).LeadingZeros(), 31))
	r, hi := s&63, s&64 != 0
	// <<1<<(^r&63) is <<(64-r), 0 at r = 0
	wa := a.L0>>r | a.L1<<1<<(^r&63)
	wb := b.L0>>r | b.L1<<1<<(^r&63)
	if hi {
		wa = a.L1 >> r
	}
	if hi {
		wb = b.L1 >> r
	}
	return wa<<31 | a.L0&(1<<31-1), wb<<31 | b.L0&(1<<31-1)
}

// mulS128 returns a * f as three limbs of two's complement
func mulS128(a Uint128, f int64) (p0, p1, p2 uint64) {
	h0, p0 := bits.Mul64(a.L0, uint64(f))
	h1, l1 := bits.Mul64(a.L1, uint64(f))
	p1, c := bits.Add64(h0, l1, 0)
	p2 = h1 + c
	// a negative f is uint64(f) - 2^64
	mf := uint64(f >> 63)
	p1, c = bits.Sub64(p1, a.L0&mf, 0)
	p2 -= a.L1&mf + c
	return
}

// linAbs128 is linOne128 without the sign, for the GCD
func linAbs128(a, b Uint128, f, g int64) Uint128 {
	r, _ := linOne128(a, b, f, g)
	return r
}

// linOne128 returns |a*f + b*g| / 2^31 and ^0 for a negative sum. For a sum
// that is a multiple of 2^31 and |f| + |g| <= 2^31 only
func linOne128(a, b Uint128, f, g int64) (Uint128, uint64) {
	p0, p1, p2 := mulS128(a, f)
	q0, q1, q2 := mulS128(b, g)
	p0, c := bits.Add64(p0, q0, 0)
	p1, c = bits.Add64(p1, q1, c)
	p2 += q2 + c
	// below 2^159 in size: the shifted value and its sign fit 129 bits
	n := uint64(int64(p2) >> 63)
	r0, r1 := p0>>31|p1<<33, p1>>31|p2<<33
	r0, c = bits.Sub64(r0^n, n, 0)
	r1, _ = bits.Sub64(r1^n, n, c)
	return Uint128{r0, r1}, n
}

// modLin128 returns (u*f + v*g) / 2^s mod m, s 31 or 62. For u, v < m, an odd
// m, mp = -1/m mod 2^64 and |f| + |g| <= 2^s only
func modLin128(u, v Uint128, f, g int64, s uint, m Uint128, mp uint64) Uint128 {
	p0, p1, p2 := mulS128(u, f)
	q0, q1, q2 := mulS128(v, g)
	p0, c := bits.Add64(p0, q0, 0)
	p1, c = bits.Add64(p1, q1, c)
	p2 += q2 + c
	// plus j*m, a multiple of 2^s, below (2^(s+1) - 1) * m: over 2^s it lies in -m..2m
	j := p0 * mp & (1<<s - 1)
	h0, l0 := bits.Mul64(j, m.L0)
	h1, l1 := bits.Mul64(j, m.L1)
	l1, c = bits.Add64(l1, h0, 0)
	h1 += c
	p0, c = bits.Add64(p0, l0, 0)
	p1, c = bits.Add64(p1, l1, c)
	p2 += h1 + c
	r0, r1, r2 := p0>>s|p1<<(64-s), p1>>s|p2<<(64-s), uint64(int64(p2)>>s)
	// m back for a negative r, then m off unless that borrows
	mn := uint64(int64(r2) >> 63)
	r0, c = bits.Add64(r0, m.L0&mn, 0)
	r1, c = bits.Add64(r1, m.L1&mn, c)
	r2 += c
	t0, b := bits.Sub64(r0, m.L0, 0)
	t1, b := bits.Sub64(r1, m.L1, b)
	_, b = bits.Sub64(r2, 0, b)
	keep := -b
	return Uint128{t0 ^ (r0^t0)&keep, t1 ^ (r1^t1)&keep}
}
