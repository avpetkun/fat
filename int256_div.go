package fat

import (
	"math"
	"math/bits"

	"github.com/avpetkun/fat/internal/arch"
)

// Div returns x / y truncated toward zero.
// It panics for y == 0 and on overflow at MinInt256 / -1
func (x Int256) Div(y Int256) Int256 {
	q, st := x.div(y, true)
	if st > 1 {
		panic("fat.Int256.Div: division by zero")
	}
	return q
}

// Divo returns Divu and whether x / y overflowed, it panics for y == 0
func (x Int256) Divo(y Int256) (q Int256, overflow bool) {
	q, st := x.div(y, false)
	if st > 1 {
		panic("fat.Int256.Divo: division by zero")
	}
	return q, st != 0
}

// Divu returns x / y wrapped around as on a Go integer, it panics for y == 0
func (x Int256) Divu(y Int256) Int256 {
	q, st := x.div(y, false)
	if st > 1 {
		panic("fat.Int256.Divu: division by zero")
	}
	return q
}

// Mod returns x % y with the sign of x, it panics for y == 0
func (x Int256) Mod(y Int256) Int256 {
	r, zero := x.mod(y)
	if zero {
		panic("fat.Int256.Mod: division by zero")
	}
	return r
}

// div is Divo with a status: 1 when x / y overflowed, 2 for y == 0.
// With fit it panics on overflow as Div does.
// It is divMod without the remainder, which the compiler drops
func (x Int256) div(y Int256, fit bool) (Int256, uint) {
	ux, uy := x.UnsignedAbs(), y.UnsignedAbs()
	// the quotient is negative when the signs differ
	mq := uint64(int64(x.L3^y.L3) >> 63)
	if uy.L3 != 0 {
		// one digit, x < y is not tested.
		// The digit comes out 0 for less than a mispredicted test costs
		q := ux.L3 / uy.L3
		if q > uy.L3 {
			// the top limb of y is too short for the estimate
			qa, _ := ux.divModKnuth(uy)
			return qa.withSign(mq), 0
		}
		// q is the digit or one above it, then x - q*y.
		// No fifth limb, as |x| <= 2^255 and q is at most its top limb
		k0, l0 := bits.Mul64(q, uy.L0)
		k1, l1 := bits.Mul64(q, uy.L1)
		k2, l2 := bits.Mul64(q, uy.L2)
		p1, c := bits.Add64(l1, k0, 0)
		p2, c := bits.Add64(l2, k1, c)
		p3, _ := bits.Add64(q*uy.L3, k2, c)
		_, b := bits.Sub64(ux.L0, l0, 0)
		_, b = bits.Sub64(ux.L1, p1, b)
		_, b = bits.Sub64(ux.L2, p2, b)
		_, b = bits.Sub64(ux.L3, p3, b)
		// one too big when the product is above |x|
		q -= b
		return Uint256{q, 0, 0, 0}.withSign(mq), 0
	}
	// half of mixed-length pairs have x < y, cheaper to test than to divide
	if ux.LessThan(uy) {
		return Int256{}, 0
	}
	if uy.L2 != 0 {
		// two digits, the first as for 4 limbs: <ux.L3, ux.L2, ux.L1> / y
		q1 := ux.L3 / uy.L2
		if q1 > uy.L2 {
			qa, _ := ux.divModKnuth(uy)
			return qa.withSign(mq), 0
		}
		// no fourth limb of q1*y: |x| <= 2^255 and q1 is at most its top limb
		k0, l0 := bits.Mul64(q1, uy.L0)
		k1, l1 := bits.Mul64(q1, uy.L1)
		p1, c := bits.Add64(l1, k0, 0)
		p2, _ := bits.Add64(q1*uy.L2, k1, c)
		w0, b := bits.Sub64(ux.L1, l0, 0)
		w1, b := bits.Sub64(ux.L2, p1, b)
		w2, b := bits.Sub64(ux.L3, p2, b)
		if b != 0 {
			q1--
			w0, c = bits.Add64(w0, uy.L0, 0)
			w1, c = bits.Add64(w1, uy.L1, c)
			w2, _ = bits.Add64(w2, uy.L2, c)
		}
		// the second by one step of algorithm D on the remainder over ux.L0
		s := uint(bits.LeadingZeros64(uy.L2)) & 63
		t := ^s & 63
		// >>1>>t is >>(64-s) that gives 0 at s = 0
		a2 := w2<<s | w1>>1>>t
		a1 := w1<<s | w0>>1>>t
		a0 := w0<<s | ux.L0>>1>>t
		u0 := ux.L0 << s
		e1 := uy.L2<<s | uy.L1>>1>>t
		e0 := uy.L1<<s | uy.L0>>1>>t
		d0 := uy.L0 << s
		// v = (2^192-1)/<e1, e0> - 2^64: reciprocal2by1, then Algorithm 6.
		// Algorithm 6 is from "Improved division by invariant integers".
		// Both adjustments go by carries, they follow no pattern
		var v uint64
		if arch.IsAmd64 {
			// one hardware 128-by-64 division there
			v, _ = bits.Div64(^e1, math.MaxUint64, e1)
		} else {
			v = (1<<47 - 1) / (e1>>40 + 1)
			lb := e1 & 1
			e := (v>>1)&-lb - v*(e1>>1+lb)
			hi, _ := bits.Mul64(e, e)
			e += hi >> 22
			hi, _ = bits.Mul64(v<<19, e)
			v = v<<41 + hi
			hi, lo := bits.Mul64(v, e1)
			_, c = bits.Add64(lo, e1, 0)
			v -= hi + e1 + c
		}
		p, c := bits.Add64(e1*v, e0, 0)
		_, b = bits.Sub64(p, e1, 0)
		k := c + c&^b
		v -= k
		p -= e1 * k
		t1, t0 := bits.Mul64(v, e0)
		p, c = bits.Add64(p, t1, 0)
		_, b = bits.Sub64(t0, e0, 0)
		_, b = bits.Sub64(p, e1, b)
		v -= c + c&^b
		if a2 == e1 && a1 == e0 {
			// rare: the digit is exactly 2^64-1 here
			return Uint256{math.MaxUint64, q1, 0, 0}.withSign(mq), 0
		}
		// the 3-by-2 division of Algorithm 5 by the top two limbs
		q0, f := bits.Mul64(v, a2)
		f, c = bits.Add64(f, a1, 0)
		q0, _ = bits.Add64(q0, a2, c)
		g1 := a1 - q0*e1
		t1, t0 = bits.Mul64(e0, q0)
		g0, b := bits.Sub64(a0, t0, 0)
		g1, _ = bits.Sub64(g1, t1, b)
		g0, b = bits.Sub64(g0, e0, 0)
		g1, _ = bits.Sub64(g1, e1, b)
		// the divisor goes back unless g1 < f, by a select: a branch is random
		m1, m0 := e1, e0
		if g1 < f {
			m1, m0 = 0, 0
			q0++
		}
		g0, c = bits.Add64(g0, m0, 0)
		g1, _ = bits.Add64(g1, m1, c)
		// rare
		if g1 > e1 || g1 == e1 && g0 >= e0 {
			q0++
			g0, b = bits.Sub64(g0, e0, 0)
			g1, _ = bits.Sub64(g1, e1, b)
		}
		t1, t0 = bits.Mul64(q0, d0)
		_, b = bits.Sub64(u0, t0, 0)
		_, b = bits.Sub64(g0, t1, b)
		_, b = bits.Sub64(g1, 0, b)
		// rare: the digit was one too big, add the divisor back
		if b != 0 {
			q0--
		}
		return Uint256{q0, q1, 0, 0}.withSign(mq), 0
	}
	if uy.L1 != 0 {
		// three digits, the first as for 4 limbs: <ux.L3, ux.L2> / y
		q2 := ux.L3 / uy.L1
		if q2 > uy.L1 {
			qa, _ := ux.divModKnuth(uy)
			return qa.withSign(mq), 0
		}
		// no third limb of q2*y: |x| <= 2^255 and q2 is at most its top limb
		k0, l0 := bits.Mul64(q2, uy.L0)
		p1 := q2*uy.L1 + k0
		w0, b := bits.Sub64(ux.L2, l0, 0)
		w1, b := bits.Sub64(ux.L3, p1, b)
		if b != 0 {
			q2--
			var c uint64
			w0, c = bits.Add64(w0, uy.L0, 0)
			w1, _ = bits.Add64(w1, uy.L1, c)
		}
		// the other two exactly by 3-by-2 divisions over ux.L1 and ux.L0
		s := uint(bits.LeadingZeros64(uy.L1)) & 63
		t := ^s & 63
		// >>1>>t is >>(64-s) that gives 0 at s = 0
		a1 := w1<<s | w0>>1>>t
		a0 := w0<<s | ux.L1>>1>>t
		u1 := ux.L1<<s | ux.L0>>1>>t
		u0 := ux.L0 << s
		e1 := uy.L1<<s | uy.L0>>1>>t
		e0 := uy.L0 << s
		// the reciprocal (2^192-1)/<e1, e0> - 2^64 written out as above
		var v uint64
		if arch.IsAmd64 {
			// one hardware 128-by-64 division there
			v, _ = bits.Div64(^e1, math.MaxUint64, e1)
		} else {
			v = (1<<47 - 1) / (e1>>40 + 1)
			lb := e1 & 1
			e := (v>>1)&-lb - v*(e1>>1+lb)
			hi, _ := bits.Mul64(e, e)
			e += hi >> 22
			hi, _ = bits.Mul64(v<<19, e)
			v = v<<41 + hi
			hi, lo := bits.Mul64(v, e1)
			_, c := bits.Add64(lo, e1, 0)
			v -= hi + e1 + c
		}
		p, c := bits.Add64(e1*v, e0, 0)
		_, b = bits.Sub64(p, e1, 0)
		k := c + c&^b
		v -= k
		p -= e1 * k
		t1, t0 := bits.Mul64(v, e0)
		p, c = bits.Add64(p, t1, 0)
		_, b = bits.Sub64(t0, e0, 0)
		_, b = bits.Sub64(p, e1, b)
		v -= c + c&^b
		q1, f := bits.Mul64(v, a1)
		f, c = bits.Add64(f, a0, 0)
		q1, _ = bits.Add64(q1, a1, c)
		g1 := a0 - q1*e1
		t1, t0 = bits.Mul64(e0, q1)
		g0, b := bits.Sub64(u1, t0, 0)
		g1, _ = bits.Sub64(g1, t1, b)
		g0, b = bits.Sub64(g0, e0, 0)
		g1, _ = bits.Sub64(g1, e1, b)
		// the divisor goes back unless g1 < f, by a select: a branch is random
		m1, m0 := e1, e0
		if g1 < f {
			m1, m0 = 0, 0
			q1++
		}
		g0, c = bits.Add64(g0, m0, 0)
		g1, _ = bits.Add64(g1, m1, c)
		// rare
		if g1 > e1 || g1 == e1 && g0 >= e0 {
			q1++
			g0, b = bits.Sub64(g0, e0, 0)
			g1, _ = bits.Sub64(g1, e1, b)
		}
		q0, f := bits.Mul64(v, g1)
		f, c = bits.Add64(f, g0, 0)
		q0, _ = bits.Add64(q0, g1, c)
		r1 := g0 - q0*e1
		t1, t0 = bits.Mul64(e0, q0)
		r0, b := bits.Sub64(u0, t0, 0)
		r1, _ = bits.Sub64(r1, t1, b)
		r0, b = bits.Sub64(r0, e0, 0)
		r1, _ = bits.Sub64(r1, e1, b)
		m1, m0 = e1, e0
		if r1 < f {
			m1, m0 = 0, 0
			q0++
		}
		r0, c = bits.Add64(r0, m0, 0)
		r1, _ = bits.Add64(r1, m1, c)
		// rare
		if r1 > e1 || r1 == e1 && r0 >= e0 {
			q0++
		}
		return Uint256{q0, q1, q2, 0}.withSign(mq), 0
	}
	// |y| < 2^64 here, a zero y is the only one to come this far
	if uy.L0 == 0 {
		return Int256{}, 2
	}
	qa, _ := ux.divMod64Flat(uy.L0)
	// |q| = 2^255 fits only as a negative quotient, it takes |y| = 1
	st := uint(qa.L3 &^ mq >> 63)
	if st != 0 && fit {
		panic("fat.Int256.Div: overflow")
	}
	return qa.withSign(mq), st
}

// mod is Mod and true for y == 0, with a zero remainder.
// It is divMod without the quotient, which the compiler drops
func (x Int256) mod(y Int256) (Int256, bool) {
	ux, uy := x.UnsignedAbs(), y.UnsignedAbs()
	// the remainder takes the sign of x
	mr := uint64(int64(x.L3) >> 63)
	if uy.L3 != 0 {
		// one digit, x < y is not tested.
		// The digit comes out 0 for less than a mispredicted test costs
		q := ux.L3 / uy.L3
		if q > uy.L3 {
			// the top limb of y is too short for the estimate
			_, ra := ux.divModKnuth(uy)
			return ra.withSign(mr), false
		}
		// q is the digit or one above it, then x - q*y.
		// No fifth limb, as |x| <= 2^255 and q is at most its top limb
		k0, l0 := bits.Mul64(q, uy.L0)
		k1, l1 := bits.Mul64(q, uy.L1)
		k2, l2 := bits.Mul64(q, uy.L2)
		p1, c := bits.Add64(l1, k0, 0)
		p2, c := bits.Add64(l2, k1, c)
		p3, _ := bits.Add64(q*uy.L3, k2, c)
		r0, b := bits.Sub64(ux.L0, l0, 0)
		r1, b := bits.Sub64(ux.L1, p1, b)
		r2, b := bits.Sub64(ux.L2, p2, b)
		r3, b := bits.Sub64(ux.L3, p3, b)
		// rare for a long top limb: one too big, add the divisor back
		if b != 0 {
			r0, c = bits.Add64(r0, uy.L0, 0)
			r1, c = bits.Add64(r1, uy.L1, c)
			r2, c = bits.Add64(r2, uy.L2, c)
			r3, _ = bits.Add64(r3, uy.L3, c)
		}
		return Uint256{r0, r1, r2, r3}.withSign(mr), false
	}
	// half of mixed-length pairs have x < y, cheaper to test than to divide
	if ux.LessThan(uy) {
		return x, false
	}
	if uy.L2 != 0 {
		// two digits, the first as for 4 limbs: <ux.L3, ux.L2, ux.L1> / y
		q1 := ux.L3 / uy.L2
		if q1 > uy.L2 {
			_, ra := ux.divModKnuth(uy)
			return ra.withSign(mr), false
		}
		// no fourth limb of q1*y: |x| <= 2^255 and q1 is at most its top limb
		k0, l0 := bits.Mul64(q1, uy.L0)
		k1, l1 := bits.Mul64(q1, uy.L1)
		p1, c := bits.Add64(l1, k0, 0)
		p2, _ := bits.Add64(q1*uy.L2, k1, c)
		w0, b := bits.Sub64(ux.L1, l0, 0)
		w1, b := bits.Sub64(ux.L2, p1, b)
		w2, b := bits.Sub64(ux.L3, p2, b)
		if b != 0 {
			w0, c = bits.Add64(w0, uy.L0, 0)
			w1, c = bits.Add64(w1, uy.L1, c)
			w2, _ = bits.Add64(w2, uy.L2, c)
		}
		// the second by one step of algorithm D on the remainder over ux.L0
		s := uint(bits.LeadingZeros64(uy.L2)) & 63
		t := ^s & 63
		// >>1>>t is >>(64-s) that gives 0 at s = 0
		a2 := w2<<s | w1>>1>>t
		a1 := w1<<s | w0>>1>>t
		a0 := w0<<s | ux.L0>>1>>t
		u0 := ux.L0 << s
		e1 := uy.L2<<s | uy.L1>>1>>t
		e0 := uy.L1<<s | uy.L0>>1>>t
		d0 := uy.L0 << s
		// v = (2^192-1)/<e1, e0> - 2^64: reciprocal2by1, then Algorithm 6.
		// Algorithm 6 is from "Improved division by invariant integers".
		// Both adjustments go by carries, they follow no pattern
		var v uint64
		if arch.IsAmd64 {
			// one hardware 128-by-64 division there
			v, _ = bits.Div64(^e1, math.MaxUint64, e1)
		} else {
			v = (1<<47 - 1) / (e1>>40 + 1)
			lb := e1 & 1
			e := (v>>1)&-lb - v*(e1>>1+lb)
			hi, _ := bits.Mul64(e, e)
			e += hi >> 22
			hi, _ = bits.Mul64(v<<19, e)
			v = v<<41 + hi
			hi, lo := bits.Mul64(v, e1)
			_, c = bits.Add64(lo, e1, 0)
			v -= hi + e1 + c
		}
		p, c := bits.Add64(e1*v, e0, 0)
		_, b = bits.Sub64(p, e1, 0)
		k := c + c&^b
		v -= k
		p -= e1 * k
		t1, t0 := bits.Mul64(v, e0)
		p, c = bits.Add64(p, t1, 0)
		_, b = bits.Sub64(t0, e0, 0)
		_, b = bits.Sub64(p, e1, b)
		v -= c + c&^b
		if a2 == e1 && a1 == e0 {
			// rare: the digit is 2^64-1, so a - q*d is a + d - d<<64
			r0, c := bits.Add64(u0, d0, 0)
			r1, c := bits.Add64(a0, e0, c)
			r2, _ := bits.Add64(a1, e1, c)
			r1, b = bits.Sub64(r1, d0, 0)
			r2, _ = bits.Sub64(r2, e0, b)
			return Uint256{r0>>s | r1<<1<<t, r1>>s | r2<<1<<t, r2 >> s, 0}.withSign(mr), false
		}
		// the 3-by-2 division of Algorithm 5 by the top two limbs
		q0, f := bits.Mul64(v, a2)
		f, c = bits.Add64(f, a1, 0)
		q0, _ = bits.Add64(q0, a2, c)
		g1 := a1 - q0*e1
		t1, t0 = bits.Mul64(e0, q0)
		g0, b := bits.Sub64(a0, t0, 0)
		g1, _ = bits.Sub64(g1, t1, b)
		g0, b = bits.Sub64(g0, e0, 0)
		g1, _ = bits.Sub64(g1, e1, b)
		// the divisor goes back unless g1 < f, by a select: a branch is random
		m1, m0 := e1, e0
		if g1 < f {
			m1, m0 = 0, 0
			q0++
		}
		g0, c = bits.Add64(g0, m0, 0)
		g1, _ = bits.Add64(g1, m1, c)
		// rare
		if g1 > e1 || g1 == e1 && g0 >= e0 {
			q0++
			g0, b = bits.Sub64(g0, e0, 0)
			g1, _ = bits.Sub64(g1, e1, b)
		}
		t1, t0 = bits.Mul64(q0, d0)
		r0, b := bits.Sub64(u0, t0, 0)
		r1, b := bits.Sub64(g0, t1, b)
		r2, b := bits.Sub64(g1, 0, b)
		// rare: the digit was one too big, add the divisor back
		if b != 0 {
			r0, c = bits.Add64(r0, d0, 0)
			r1, c = bits.Add64(r1, e0, c)
			r2, _ = bits.Add64(r2, e1, c)
		}
		return Uint256{r0>>s | r1<<1<<t, r1>>s | r2<<1<<t, r2 >> s, 0}.withSign(mr), false
	}
	if uy.L1 != 0 {
		// three digits, the first as for 4 limbs: <ux.L3, ux.L2> / y
		q2 := ux.L3 / uy.L1
		if q2 > uy.L1 {
			_, ra := ux.divModKnuth(uy)
			return ra.withSign(mr), false
		}
		// no third limb of q2*y: |x| <= 2^255 and q2 is at most its top limb
		k0, l0 := bits.Mul64(q2, uy.L0)
		p1 := q2*uy.L1 + k0
		w0, b := bits.Sub64(ux.L2, l0, 0)
		w1, b := bits.Sub64(ux.L3, p1, b)
		if b != 0 {
			var c uint64
			w0, c = bits.Add64(w0, uy.L0, 0)
			w1, _ = bits.Add64(w1, uy.L1, c)
		}
		// the other two exactly by 3-by-2 divisions over ux.L1 and ux.L0
		s := uint(bits.LeadingZeros64(uy.L1)) & 63
		t := ^s & 63
		// >>1>>t is >>(64-s) that gives 0 at s = 0
		a1 := w1<<s | w0>>1>>t
		a0 := w0<<s | ux.L1>>1>>t
		u1 := ux.L1<<s | ux.L0>>1>>t
		u0 := ux.L0 << s
		e1 := uy.L1<<s | uy.L0>>1>>t
		e0 := uy.L0 << s
		// the reciprocal (2^192-1)/<e1, e0> - 2^64 written out as above
		var v uint64
		if arch.IsAmd64 {
			// one hardware 128-by-64 division there
			v, _ = bits.Div64(^e1, math.MaxUint64, e1)
		} else {
			v = (1<<47 - 1) / (e1>>40 + 1)
			lb := e1 & 1
			e := (v>>1)&-lb - v*(e1>>1+lb)
			hi, _ := bits.Mul64(e, e)
			e += hi >> 22
			hi, _ = bits.Mul64(v<<19, e)
			v = v<<41 + hi
			hi, lo := bits.Mul64(v, e1)
			_, c := bits.Add64(lo, e1, 0)
			v -= hi + e1 + c
		}
		p, c := bits.Add64(e1*v, e0, 0)
		_, b = bits.Sub64(p, e1, 0)
		k := c + c&^b
		v -= k
		p -= e1 * k
		t1, t0 := bits.Mul64(v, e0)
		p, c = bits.Add64(p, t1, 0)
		_, b = bits.Sub64(t0, e0, 0)
		_, b = bits.Sub64(p, e1, b)
		v -= c + c&^b
		q1, f := bits.Mul64(v, a1)
		f, c = bits.Add64(f, a0, 0)
		q1, _ = bits.Add64(q1, a1, c)
		g1 := a0 - q1*e1
		t1, t0 = bits.Mul64(e0, q1)
		g0, b := bits.Sub64(u1, t0, 0)
		g1, _ = bits.Sub64(g1, t1, b)
		g0, b = bits.Sub64(g0, e0, 0)
		g1, _ = bits.Sub64(g1, e1, b)
		// the divisor goes back unless g1 < f, by a select: a branch is random
		m1, m0 := e1, e0
		if g1 < f {
			m1, m0 = 0, 0
		}
		g0, c = bits.Add64(g0, m0, 0)
		g1, _ = bits.Add64(g1, m1, c)
		// rare
		if g1 > e1 || g1 == e1 && g0 >= e0 {
			g0, b = bits.Sub64(g0, e0, 0)
			g1, _ = bits.Sub64(g1, e1, b)
		}
		q0, f := bits.Mul64(v, g1)
		f, c = bits.Add64(f, g0, 0)
		q0, _ = bits.Add64(q0, g1, c)
		r1 := g0 - q0*e1
		t1, t0 = bits.Mul64(e0, q0)
		r0, b := bits.Sub64(u0, t0, 0)
		r1, _ = bits.Sub64(r1, t1, b)
		r0, b = bits.Sub64(r0, e0, 0)
		r1, _ = bits.Sub64(r1, e1, b)
		m1, m0 = e1, e0
		if r1 < f {
			m1, m0 = 0, 0
		}
		r0, c = bits.Add64(r0, m0, 0)
		r1, _ = bits.Add64(r1, m1, c)
		// rare
		if r1 > e1 || r1 == e1 && r0 >= e0 {
			r0, b = bits.Sub64(r0, e0, 0)
			r1, _ = bits.Sub64(r1, e1, b)
		}
		return Uint256{r0>>s | r1<<1<<t, r1 >> s, 0, 0}.withSign(mr), false
	}
	// |y| < 2^64 here, a zero y is the only one to come this far
	if uy.L0 == 0 {
		return Int256{}, true
	}
	_, r := ux.divMod64Flat(uy.L0)
	return Uint256{r, 0, 0, 0}.withSign(mr), false
}

// Div64 returns x / y truncated toward zero.
// It panics for y == 0 and on overflow at MinInt256 / -1
func (x Int256) Div64(y int64) Int256 {
	if y == 0 {
		panic("fat.Int256.Div64: division by zero")
	}
	return x.div64(y, "fat.Int256.Div64: overflow")
}

// Div64o returns Div64u and whether x / y overflowed, it panics for y == 0
func (x Int256) Div64o(y int64) (q Int256, overflow bool) {
	if y == 0 {
		panic("fat.Int256.Div64o: division by zero")
	}
	return x.div64o(y)
}

// Div64u returns x / y wrapped around as on a Go integer, it panics for y == 0
func (x Int256) Div64u(y int64) Int256 {
	if y == 0 {
		panic("fat.Int256.Div64u: division by zero")
	}
	q, _ := x.div64o(y)
	return q
}

// Mod64 returns x % y with the sign of x, it panics for y == 0
func (x Int256) Mod64(y int64) int64 {
	if y == 0 {
		panic("fat.Int256.Mod64: division by zero")
	}
	return x.mod64(y)
}

// div64 is divMod64 without the remainder, whose product ends the chain.
// For y != 0 only, y == 0 gives a runtime panic
func (x Int256) div64(y int64, overflow string) (q Int256) {
	if arch.IsAmd64 {
		// a hardware 128-by-64 division per quotient limb there
		my := uint64(y >> 63)
		mq := uint64(int64(x.L3)>>63) ^ my
		u := x.UnsignedAbs()
		w := uint64(y) ^ my - my
		var q3 uint64
		ra := u.L3
		if ra >= w {
			q3 = ra / w
			ra -= q3 * w
		}
		wm := w & mq
		l0, c := bits.Add64(u.L0^mq, wm, 0)
		l1, c := bits.Add64(u.L1^mq, 0, c)
		l2, c := bits.Add64(u.L2^mq, 0, c)
		top, _ := bits.Add64(ra^mq, wm, c)
		q.L3 = q3 ^ mq
		if top == w {
			top = 0
			q.L3++
		}
		q.L2, ra = bits.Div64(top, l2, w)
		q.L1, ra = bits.Div64(ra, l1, w)
		q.L0, _ = bits.Div64(ra, l0, w)
		if int64(q3&^mq) < 0 {
			panic(overflow)
		}
		return q
	}
	// as in divMod64
	my := uint64(y >> 63)
	mq := uint64(int64(x.L3)>>63) ^ my
	u := x.UnsignedAbs()
	w := uint64(y) ^ my - my
	wm := w & mq
	s := uint(bits.LeadingZeros64(w)) & 63
	d := w << s
	// reciprocal2by1 written out, a call would spill the limbs
	v := (1<<47 - 1) / (d>>40 + 1)
	d0 := d & 1
	e := (v>>1)&-d0 - v*(d>>1+d0)
	hi, _ := bits.Mul64(e, e)
	e += hi >> 22
	hi, _ = bits.Mul64(v<<19, e)
	v = v<<41 + hi
	hi, lo := bits.Mul64(v, d)
	_, b := bits.Sub64(^d, lo, 0)
	v, _ = bits.Sub64(v, hi+d, b)
	t := ^s & 63
	l0, c := bits.Add64(u.L0^mq, wm, 0)
	if u.L3|u.L2 == 0 {
		q1 := u.L1 / w
		ra := u.L1 - q1*w
		top, _ := bits.Add64(ra^mq, wm, c)
		var k uint64
		if top == w {
			top, k = 0, 1
		}
		q.L1, c = bits.Add64(q1^mq, k, 0)
		q.L2, c = bits.Add64(mq, 0, c)
		q.L3 = mq + c
		q.L0 = quo2by1(top<<s|l0>>1>>t, l0<<s, d, v)
		return q
	}
	q3 := u.L3 / w
	ra := u.L3 - q3*w
	l1, c := bits.Add64(u.L1^mq, 0, c)
	l2, c := bits.Add64(u.L2^mq, 0, c)
	top, _ := bits.Add64(ra^mq, wm, c)
	q.L3 = q3 ^ mq
	if top == w {
		top = 0
		q.L3++
	}
	q.L2, ra = div2by1r(top<<s|l2>>1>>t, l2<<s|l1>>1>>t, d, v)
	q.L1, ra = div2by1r(ra, l1<<s|l0>>1>>t, d, v)
	q.L0 = quo2by1(ra, l0<<s, d, v)
	if int64(q3&^mq) < 0 {
		panic(overflow)
	}
	return q
}

// div64o is div64 with MinInt256 / -1 wrapped and reported.
// For y != 0 only, y == 0 gives a wrong result or a panic
func (x Int256) div64o(y int64) (q Int256, over bool) {
	if arch.IsAmd64 {
		// a hardware 128-by-64 division per quotient limb there
		my := uint64(y >> 63)
		mq := uint64(int64(x.L3)>>63) ^ my
		u := x.UnsignedAbs()
		w := uint64(y) ^ my - my
		var q3 uint64
		ra := u.L3
		if ra >= w {
			q3 = ra / w
			ra -= q3 * w
		}
		wm := w & mq
		l0, c := bits.Add64(u.L0^mq, wm, 0)
		l1, c := bits.Add64(u.L1^mq, 0, c)
		l2, c := bits.Add64(u.L2^mq, 0, c)
		top, _ := bits.Add64(ra^mq, wm, c)
		q.L3 = q3 ^ mq
		if top == w {
			top = 0
			q.L3++
		}
		q.L2, ra = bits.Div64(top, l2, w)
		q.L1, ra = bits.Div64(ra, l1, w)
		q.L0, _ = bits.Div64(ra, l0, w)
		return q, int64(q3&^mq) < 0
	}
	// as in divMod64
	my := uint64(y >> 63)
	mq := uint64(int64(x.L3)>>63) ^ my
	u := x.UnsignedAbs()
	w := uint64(y) ^ my - my
	wm := w & mq
	s := uint(bits.LeadingZeros64(w)) & 63
	d := w << s
	// reciprocal2by1 written out, a call would spill the limbs
	v := (1<<47 - 1) / (d>>40 + 1)
	d0 := d & 1
	e := (v>>1)&-d0 - v*(d>>1+d0)
	hi, _ := bits.Mul64(e, e)
	e += hi >> 22
	hi, _ = bits.Mul64(v<<19, e)
	v = v<<41 + hi
	hi, lo := bits.Mul64(v, d)
	_, b := bits.Sub64(^d, lo, 0)
	v, _ = bits.Sub64(v, hi+d, b)
	t := ^s & 63
	l0, c := bits.Add64(u.L0^mq, wm, 0)
	if u.L3|u.L2 == 0 {
		q1 := u.L1
		if w != 0 {
			q1 = u.L1 / w
		}
		ra := u.L1 - q1*w
		top, _ := bits.Add64(ra^mq, wm, c)
		var k uint64
		if top == w {
			top, k = 0, 1
		}
		q.L1, c = bits.Add64(q1^mq, k, 0)
		q.L2, c = bits.Add64(mq, 0, c)
		q.L3 = mq + c
		q.L0 = quo2by1(top<<s|l0>>1>>t, l0<<s, d, v)
		return q, false
	}
	q3 := u.L3
	if w != 0 {
		q3 = u.L3 / w
	}
	ra := u.L3 - q3*w
	l1, c := bits.Add64(u.L1^mq, 0, c)
	l2, c := bits.Add64(u.L2^mq, 0, c)
	top, _ := bits.Add64(ra^mq, wm, c)
	q.L3 = q3 ^ mq
	if top == w {
		top = 0
		q.L3++
	}
	q.L2, ra = div2by1r(top<<s|l2>>1>>t, l2<<s|l1>>1>>t, d, v)
	q.L1, ra = div2by1r(ra, l1<<s|l0>>1>>t, d, v)
	q.L0 = quo2by1(ra, l0<<s, d, v)
	return q, int64(q3&^mq) < 0
}

// mod64 is divMod64 without the quotient.
// For y != 0 only, y == 0 gives a wrong result or a panic
func (x Int256) mod64(y int64) int64 {
	if arch.IsAmd64 {
		// a hardware 128-by-64 division per quotient limb there
		my := uint64(y >> 63)
		mx := uint64(int64(x.L3) >> 63)
		u := x.UnsignedAbs()
		w := uint64(y) ^ my - my
		ra := u.L3
		if ra >= w {
			ra %= w
		}
		_, ra = bits.Div64(ra, u.L2, w)
		_, ra = bits.Div64(ra, u.L1, w)
		_, ra = bits.Div64(ra, u.L0, w)
		// the remainder takes the sign of x
		return int64(ra ^ mx - mx)
	}
	// as in divMod64
	my := uint64(y >> 63)
	mx := uint64(int64(x.L3) >> 63)
	u := x.UnsignedAbs()
	w := uint64(y) ^ my - my
	s := uint(bits.LeadingZeros64(w)) & 63
	d := w << s
	// reciprocal2by1 written out, a call would spill the limbs
	v := (1<<47 - 1) / (d>>40 + 1)
	d0 := d & 1
	e := (v>>1)&-d0 - v*(d>>1+d0)
	hi, _ := bits.Mul64(e, e)
	e += hi >> 22
	hi, _ = bits.Mul64(v<<19, e)
	v = v<<41 + hi
	hi, lo := bits.Mul64(v, d)
	_, b := bits.Sub64(^d, lo, 0)
	v, _ = bits.Sub64(v, hi+d, b)
	t := ^s & 63
	if u.L3|u.L2 == 0 {
		ra := u.L1
		if w != 0 {
			ra = u.L1 % w
		}
		q0 := quo2by1(ra<<s|u.L0>>1>>t, u.L0<<s, d, v)
		return int64(x.L0 - q0*(w^mx-mx))
	}
	ra := u.L3
	if w != 0 {
		ra = u.L3 % w
	}
	// the steps correct by selects, a branch is slower here in our benchmarks
	_, ra = div2by1(ra<<s|u.L2>>1>>t, u.L2<<s|u.L1>>1>>t, d, v)
	_, ra = div2by1(ra, u.L1<<s|u.L0>>1>>t, d, v)
	q0 := quo2by1(ra, u.L0<<s, d, v)
	// x = q*y + r in the low limb, q*y has the sign of x
	return int64(x.L0 - q0*(w^mx-mx))
}

// Div128 returns x / y truncated toward zero.
// It panics for y == 0 and for MinInt256 / -1, which overflows
func (x Int256) Div128(y Int128) Int256 {
	q, st := x.div128(y, true)
	if st > 1 {
		panic("fat.Int256.Div128: division by zero")
	}
	return q
}

// Div128o returns Div128u and whether x / y overflowed.
// It panics for y == 0
func (x Int256) Div128o(y Int128) (q Int256, overflow bool) {
	q, st := x.div128(y, false)
	if st > 1 {
		panic("fat.Int256.Div128o: division by zero")
	}
	return q, st != 0
}

// Div128u returns x / y wrapped around as on a Go integer.
// It panics for y == 0
func (x Int256) Div128u(y Int128) Int256 {
	q, st := x.div128(y, false)
	if st > 1 {
		panic("fat.Int256.Div128u: division by zero")
	}
	return q
}

// Mod128 returns x % y with the sign of x, it panics for y == 0
func (x Int256) Mod128(y Int128) Int128 {
	r, zero := x.mod128(y)
	if zero {
		panic("fat.Int256.Mod128: division by zero")
	}
	return r
}

// div128 is Div128o with a status: 1 when x / y overflowed, 2 for y == 0.
// With fit it panics on overflow as Div128 does
func (x Int256) div128(y Int128, fit bool) (Int256, uint) {
	qa, _, zero := x.uabs().divMod128(y.UnsignedAbs())
	// the quotient is negative when the signs differ
	mq := uint64(int64(x.L3)>>63) ^ uint64(int64(y.L1)>>63)
	// |q| = 2^255 fits only as a negative quotient, a zero y leaves qa = 0
	st := uint(qa.L3 &^ mq >> 63)
	if st != 0 && fit {
		panic("fat.Int256.Div128: overflow")
	}
	if zero {
		st = 2
	}
	return qa.withSign(mq), st
}

// mod128 is Mod128 and true for y == 0, with a zero remainder
func (x Int256) mod128(y Int128) (Int128, bool) {
	_, ra, zero := x.uabs().divMod128(y.UnsignedAbs())
	return ra.withSign(uint64(int64(x.L3) >> 63)), zero
}
