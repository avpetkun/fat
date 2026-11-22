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

// DivMod returns Div and Mod.
// It panics for y == 0 and on overflow at MinInt256 / -1
func (x Int256) DivMod(y Int256) (q, r Int256) {
	q, r, zero := x.divMod(y)
	if zero {
		panic("fat.Int256.DivMod: division by zero")
	}
	return
}

// divMod is DivMod and true for y == 0, with a zero quotient and remainder.
// The zero test is on one limb, where the branches on the length leave it.
// It is Uint256.divMod on |x| and |y| written out, faster than its call
func (x Int256) divMod(y Int256) (Int256, Int256, bool) {
	ux, uy := x.UnsignedAbs(), y.UnsignedAbs()
	// q is negative when the signs differ, the remainder takes the sign of x
	mr := uint64(int64(x.L3) >> 63)
	mq := mr ^ uint64(int64(y.L3)>>63)
	if uy.L3 != 0 {
		// one digit, x < y is not tested.
		// The digit comes out 0 for less than a mispredicted test costs
		q := ux.L3 / uy.L3
		if q > uy.L3 {
			// the top limb of y is too short for the estimate
			qa, ra := ux.divModKnuth(uy)
			return qa.withSign(mq), ra.withSign(mr), false
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
			q--
			r0, c = bits.Add64(r0, uy.L0, 0)
			r1, c = bits.Add64(r1, uy.L1, c)
			r2, c = bits.Add64(r2, uy.L2, c)
			r3, _ = bits.Add64(r3, uy.L3, c)
		}
		return Uint256{q, 0, 0, 0}.withSign(mq), Uint256{r0, r1, r2, r3}.withSign(mr), false
	}
	// half of mixed-length pairs have x < y, cheaper to test than to divide
	if ux.LessThan(uy) {
		return Int256{}, x, false
	}
	if uy.L2 != 0 {
		// two digits, the first as for 4 limbs: <ux.L3, ux.L2, ux.L1> / y
		q1 := ux.L3 / uy.L2
		if q1 > uy.L2 {
			qa, ra := ux.divModKnuth(uy)
			return qa.withSign(mq), ra.withSign(mr), false
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
			// rare: the digit is 2^64-1, so a - q*d is a + d - d<<64
			r0, c := bits.Add64(u0, d0, 0)
			r1, c := bits.Add64(a0, e0, c)
			r2, _ := bits.Add64(a1, e1, c)
			r1, b = bits.Sub64(r1, d0, 0)
			r2, _ = bits.Sub64(r2, e0, b)
			return Uint256{math.MaxUint64, q1, 0, 0}.withSign(mq), Uint256{r0>>s | r1<<1<<t, r1>>s | r2<<1<<t, r2 >> s, 0}.withSign(mr), false
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
			q0--
			r0, c = bits.Add64(r0, d0, 0)
			r1, c = bits.Add64(r1, e0, c)
			r2, _ = bits.Add64(r2, e1, c)
		}
		return Uint256{q0, q1, 0, 0}.withSign(mq), Uint256{r0>>s | r1<<1<<t, r1>>s | r2<<1<<t, r2 >> s, 0}.withSign(mr), false
	}
	if uy.L1 != 0 {
		// three digits, the first as for 4 limbs: <ux.L3, ux.L2> / y
		q2 := ux.L3 / uy.L1
		if q2 > uy.L1 {
			qa, ra := ux.divModKnuth(uy)
			return qa.withSign(mq), ra.withSign(mr), false
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
			r0, b = bits.Sub64(r0, e0, 0)
			r1, _ = bits.Sub64(r1, e1, b)
		}
		return Uint256{q0, q1, q2, 0}.withSign(mq), Uint256{r0>>s | r1<<1<<t, r1 >> s, 0, 0}.withSign(mr), false
	}
	// |y| < 2^64 here, a zero y is the only one to come this far
	if uy.L0 == 0 {
		return Int256{}, Int256{}, true
	}
	qa, r := ux.divMod64Flat(uy.L0)
	// |q| = 2^255 fits only as a negative quotient, it takes |y| = 1
	if qa.L3&^mq>>63 != 0 {
		panic("fat.Int256.DivMod: overflow")
	}
	return qa.withSign(mq), Uint256{r, 0, 0, 0}.withSign(mr), false
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

// DivMod64 returns Div64 and Mod64.
// It panics for y == 0 and on overflow at MinInt256 / -1
func (x Int256) DivMod64(y int64) (Int256, int64) {
	if y == 0 {
		panic("fat.Int256.DivMod64: division by zero")
	}
	return x.divMod64(y, "fat.Int256.DivMod64: overflow")
}

// divMod64 is DivMod64 for y != 0 only, y == 0 gives a runtime panic.
// For MinInt256 / -1 it panics with overflow as the message
func (x Int256) divMod64(y int64, overflow string) (q Int256, r int64) {
	if arch.IsAmd64 {
		// a hardware 128-by-64 division per quotient limb there
		// Uint256.divMod64 on |x| and |y| written out, faster than its call
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
		// a negative quotient comes out of the division itself.
		// Negating it after would wait for the low limb.
		// It uses -(n/w) = (w*2^192 + w-1 - n) / w mod 2^192.
		// Here n = <ra, u.L2, u.L1, u.L0>, the limb above takes ^q3
		wm := w & mq
		l0, c := bits.Add64(u.L0^mq, wm, 0)
		l1, c := bits.Add64(u.L1^mq, 0, c)
		l2, c := bits.Add64(u.L2^mq, 0, c)
		top, _ := bits.Add64(ra^mq, wm, c)
		// n < w leaves top = w, a low quotient of 0 whose negation carries up
		q.L3 = q3 ^ mq
		if top == w {
			top = 0
			q.L3++
		}
		q.L2, ra = bits.Div64(top, l2, w)
		q.L1, ra = bits.Div64(ra, l1, w)
		q.L0, _ = bits.Div64(ra, l0, w)
		// |q| = 2^255 fits only as a negative quotient
		if int64(q3&^mq) < 0 {
			panic(overflow)
		}
		// x = q*y + r in the low limb
		return q, int64(x.L0 - q.L0*uint64(y))
	}
	// Uint256.divMod64 on |x| and |y| written out, faster than its call.
	// After the reciprocal |x| < 2^128 skips two steps by a branch, faster too
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
		// the negative quotient as below, on n = <ra, u.L0>.
		// It uses -(n/w) = (w*2^64 + w-1 - n) / w mod 2^64.
		// Above it ^q1 and the sign take the carry of top = w when n < w
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
		return q, int64(x.L0 - q.L0*uint64(y))
	}
	q3 := u.L3 / w
	ra := u.L3 - q3*w
	// a negative quotient comes out of the division itself.
	// Negating it after would wait for the low limb.
	// It uses -(n/w) = (w*2^192 + w-1 - n) / w mod 2^192.
	// Here n = <ra, u.L2, u.L1, u.L0>, the limb above takes ^q3
	l1, c := bits.Add64(u.L1^mq, 0, c)
	l2, c := bits.Add64(u.L2^mq, 0, c)
	top, _ := bits.Add64(ra^mq, wm, c)
	// n < w leaves top = w, a low quotient of 0 whose negation carries up
	q.L3 = q3 ^ mq
	if top == w {
		top = 0
		q.L3++
	}
	q.L2, ra = div2by1r(top<<s|l2>>1>>t, l2<<s|l1>>1>>t, d, v)
	q.L1, ra = div2by1r(ra, l1<<s|l0>>1>>t, d, v)
	q.L0 = quo2by1(ra, l0<<s, d, v)
	// |q| = 2^255 fits only as a negative quotient
	if int64(q3&^mq) < 0 {
		panic(overflow)
	}
	// x = q*y + r in the low limb
	return q, int64(x.L0 - q.L0*uint64(y))
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

// DivMod128 returns Div128 and Mod128.
// It panics for y == 0 and for MinInt256 / -1
func (x Int256) DivMod128(y Int128) (q Int256, r Int128) {
	q, r, zero := x.divMod128(y)
	if zero {
		panic("fat.Int256.DivMod128: division by zero")
	}
	return
}

// divMod128 is DivMod128 and true for y == 0, with a zero quotient and remainder.
// It is the unsigned division of |x| by |y|
func (x Int256) divMod128(y Int128) (Int256, Int128, bool) {
	qa, ra, zero := x.uabs().divMod128(y.UnsignedAbs())
	// q is negative when the signs differ, the remainder takes the sign of x
	mr := uint64(int64(x.L3) >> 63)
	mq := mr ^ uint64(int64(y.L1)>>63)
	// |q| = 2^255 fits only as a negative quotient, a zero y leaves qa = 0
	if qa.L3&^mq>>63 != 0 {
		panic("fat.Int256.DivMod128: overflow")
	}
	return qa.withSign(mq), ra.withSign(mr), zero
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

// int256MulDivPanics, int256MulDivModPanics and int256MulDivCeilPanics are the panics by kernel status.
// One panic from a table keeps the wrappers within the inline budget
var (
	int256MulDivPanics = [3]string{
		1: "fat.Int256.MulDiv: overflow",
		2: "fat.Int256.MulDiv: division by zero",
	}
	int256MulDivModPanics = [3]string{
		1: "fat.Int256.MulDivMod: overflow",
		2: "fat.Int256.MulDivMod: division by zero",
	}
	int256MulDivCeilPanics = [3]string{
		1: "fat.Int256.MulDivCeil: overflow",
		2: "fat.Int256.MulDivCeil: division by zero",
	}
)

// MulDiv returns x * y / d truncated toward zero, the product in 512 bits.
// It panics for d == 0 and when the quotient overflows
func (x Int256) MulDiv(y, d Int256) (q Int256) {
	q, _, st := x.mulDivMod(y, d)
	if st != 0 {
		panic(int256MulDivPanics[st])
	}
	return
}

// MulDivo returns MulDivu and whether the quotient overflowed.
// It panics for d == 0
func (x Int256) MulDivo(y, d Int256) (q Int256, overflow bool) {
	q, _, st := x.mulDivMod(y, d)
	if st > 1 {
		panic("fat.Int256.MulDivo: division by zero")
	}
	return q, st != 0
}

// MulDivu returns x * y / d wrapped around, the product in 512 bits.
// It panics for d == 0
func (x Int256) MulDivu(y, d Int256) (q Int256) {
	q, _, st := x.mulDivMod(y, d)
	if st > 1 {
		panic("fat.Int256.MulDivu: division by zero")
	}
	return
}

// MulDivMod returns MulDiv and x * y % d with the sign of x * y.
// It panics for d == 0 and when the quotient overflows
func (x Int256) MulDivMod(y, d Int256) (q, r Int256) {
	q, r, st := x.mulDivMod(y, d)
	if st != 0 {
		panic(int256MulDivModPanics[st])
	}
	return
}

// MulDivCeil returns x * y / d rounded toward +Inf, the product in 512 bits.
// It panics for d == 0 and when the quotient overflows
func (x Int256) MulDivCeil(y, d Int256) Int256 {
	q, st := x.mulDivCeil(y, d)
	if st != 0 {
		panic(int256MulDivCeilPanics[st])
	}
	return q
}

// mulDivMod is MulDivMod with the quotient wrapped and a status:
// 1 when it overflowed, 2 for d == 0
func (x Int256) mulDivMod(y, d Int256) (q, r Int256, st uint) {
	qa, ra, st := x.UnsignedAbs().mulDivMod(y.UnsignedAbs(), d.UnsignedAbs())
	// q takes the sign of x * y / d, r that of x * y, both by (v ^ m) - m
	mr := uint64(int64(x.L3^y.L3) >> 63)
	mq := mr ^ uint64(int64(d.L3)>>63)
	var b uint64
	q.L0, b = bits.Sub64(qa.L0^mq, mq, 0)
	q.L1, b = bits.Sub64(qa.L1^mq, mq, b)
	q.L2, b = bits.Sub64(qa.L2^mq, mq, b)
	q.L3, _ = bits.Sub64(qa.L3^mq, mq, b)
	r.L0, b = bits.Sub64(ra.L0^mr, mr, 0)
	r.L1, b = bits.Sub64(ra.L1^mr, mr, b)
	r.L2, b = bits.Sub64(ra.L2^mr, mr, b)
	r.L3, _ = bits.Sub64(ra.L3^mr, mr, b)
	// a nonzero q that fits keeps its sign, 2^255 only as a negative one.
	// The unsigned status carries over: a zero d leaves q = 0 and f = 0
	_, f := bits.Sub64(0, qa.L0|qa.L1|qa.L2|qa.L3, 0)
	f &= (q.L3 ^ mq) >> 63
	return q, r, st | uint(f)
}

// mulDivCeil is MulDivCeil with the quotient wrapped and a status as of mulDivMod
func (x Int256) mulDivCeil(y, d Int256) (q Int256, st uint) {
	qa, ra, st := x.UnsignedAbs().mulDivMod(y.UnsignedAbs(), d.UnsignedAbs())
	// truncation has rounded a negative quotient up already,
	// a positive one with a remainder adds one
	mq := uint64(int64(x.L3^y.L3^d.L3) >> 63)
	nz := ra.L0 | ra.L1 | ra.L2 | ra.L3
	var c uint64
	qa.L0, c = bits.Add64(qa.L0, (nz|-nz)>>63&^mq, 0)
	qa.L1, c = bits.Add64(qa.L1, 0, c)
	qa.L2, c = bits.Add64(qa.L2, 0, c)
	qa.L3, _ = bits.Add64(qa.L3, 0, c)
	// the sign by (v ^ m) - m and the overflow test as in mulDivMod.
	// A carry out of qa comes only from an overflowed q
	var b uint64
	q.L0, b = bits.Sub64(qa.L0^mq, mq, 0)
	q.L1, b = bits.Sub64(qa.L1^mq, mq, b)
	q.L2, b = bits.Sub64(qa.L2^mq, mq, b)
	q.L3, _ = bits.Sub64(qa.L3^mq, mq, b)
	_, f := bits.Sub64(0, qa.L0|qa.L1|qa.L2|qa.L3, 0)
	f &= (q.L3 ^ mq) >> 63
	return q, st | uint(f)
}

// AddMod returns (x + y) % m of the exact sum, with the sign of the sum.
// It panics for m == 0
func (x Int256) AddMod(y, m Int256) Int256 {
	if m.IsZero() {
		panic("fat.Int256.AddMod: division by zero")
	}
	s, over := x.Addo(y)
	if over {
		// only a sum of one sign overflows: |x| + |y| mod |m|, the sign of x
		return x.UnsignedAbs().AddMod(y.UnsignedAbs(), m.UnsignedAbs()).withSign(uint64(int64(x.L3) >> 63))
	}
	// mod of the sum written out, its |s| < |m| test becomes |s| < 2|m|.
	// Sums of operands within m land on either side of m at random, no division
	us, um := s.UnsignedAbs(), m.UnsignedAbs()
	var ra Uint256
	if um.L3 != 0 {
		// one digit as in divMod, with no fifth limb either.
		// An overflowed sum went above, so |s| <= 2^255
		k := us.L3 / um.L3
		if k > um.L3 {
			_, ra = us.divModKnuth(um)
		} else {
			k0, l0 := bits.Mul64(k, um.L0)
			k1, l1 := bits.Mul64(k, um.L1)
			k2, l2 := bits.Mul64(k, um.L2)
			p1, c := bits.Add64(l1, k0, 0)
			p2, c := bits.Add64(l2, k1, c)
			p3, _ := bits.Add64(k*um.L3, k2, c)
			r0, b := bits.Sub64(us.L0, l0, 0)
			r1, b := bits.Sub64(us.L1, p1, b)
			r2, b := bits.Sub64(us.L2, p2, b)
			r3, b := bits.Sub64(us.L3, p3, b)
			// rare for a long top limb: one too big, add the divisor back
			if b != 0 {
				r0, c = bits.Add64(r0, um.L0, 0)
				r1, c = bits.Add64(r1, um.L1, c)
				r2, c = bits.Add64(r2, um.L2, c)
				r3, _ = bits.Add64(r3, um.L3, c)
			}
			ra = Uint256{r0, r1, r2, r3}
		}
	} else {
		t0, b := bits.Sub64(us.L0, um.L0, 0)
		t1, b := bits.Sub64(us.L1, um.L1, b)
		t2, b := bits.Sub64(us.L2, um.L2, b)
		t3, b := bits.Sub64(us.L3, 0, b)
		_, d := bits.Sub64(t0, um.L0, 0)
		_, d = bits.Sub64(t1, um.L1, d)
		_, d = bits.Sub64(t2, um.L2, d)
		_, d = bits.Sub64(t3, 0, d)
		if b|d != 0 {
			// |s| or |s| - |m| picked without a branch
			if b != 0 {
				t0, t1, t2, t3 = us.L0, us.L1, us.L2, us.L3
			}
			ra = Uint256{t0, t1, t2, t3}
		} else {
			_, ra, _ = us.divMod(um)
		}
	}
	// the sign of s
	return ra.withSign(uint64(int64(s.L3) >> 63))
}

// SubMod returns (x - y) % m of the exact difference, with the sign of the
// difference. It panics for m == 0
func (x Int256) SubMod(y, m Int256) Int256 {
	if m.IsZero() {
		panic("fat.Int256.SubMod: division by zero")
	}
	s, over := x.Subo(y)
	if over {
		// only a difference of opposite signs overflows: |x| + |y| mod |m|, the sign of x
		return x.UnsignedAbs().AddMod(y.UnsignedAbs(), m.UnsignedAbs()).withSign(uint64(int64(x.L3) >> 63))
	}
	// mod of the difference written out, its |s| < |m| test becomes |s| < 2|m|.
	// Differences of operands within m land on either side of m at random, no division
	us, um := s.UnsignedAbs(), m.UnsignedAbs()
	var ra Uint256
	if um.L3 != 0 {
		// one digit as in divMod, with no fifth limb either.
		// An overflowed difference went above, so |s| <= 2^255
		k := us.L3 / um.L3
		if k > um.L3 {
			_, ra = us.divModKnuth(um)
		} else {
			k0, l0 := bits.Mul64(k, um.L0)
			k1, l1 := bits.Mul64(k, um.L1)
			k2, l2 := bits.Mul64(k, um.L2)
			p1, c := bits.Add64(l1, k0, 0)
			p2, c := bits.Add64(l2, k1, c)
			p3, _ := bits.Add64(k*um.L3, k2, c)
			r0, b := bits.Sub64(us.L0, l0, 0)
			r1, b := bits.Sub64(us.L1, p1, b)
			r2, b := bits.Sub64(us.L2, p2, b)
			r3, b := bits.Sub64(us.L3, p3, b)
			// rare for a long top limb: one too big, add the divisor back
			if b != 0 {
				r0, c = bits.Add64(r0, um.L0, 0)
				r1, c = bits.Add64(r1, um.L1, c)
				r2, c = bits.Add64(r2, um.L2, c)
				r3, _ = bits.Add64(r3, um.L3, c)
			}
			ra = Uint256{r0, r1, r2, r3}
		}
	} else {
		t0, b := bits.Sub64(us.L0, um.L0, 0)
		t1, b := bits.Sub64(us.L1, um.L1, b)
		t2, b := bits.Sub64(us.L2, um.L2, b)
		t3, b := bits.Sub64(us.L3, 0, b)
		_, d := bits.Sub64(t0, um.L0, 0)
		_, d = bits.Sub64(t1, um.L1, d)
		_, d = bits.Sub64(t2, um.L2, d)
		_, d = bits.Sub64(t3, 0, d)
		if b|d != 0 {
			// |s| or |s| - |m| picked without a branch
			if b != 0 {
				t0, t1, t2, t3 = us.L0, us.L1, us.L2, us.L3
			}
			ra = Uint256{t0, t1, t2, t3}
		} else {
			_, ra, _ = us.divMod(um)
		}
	}
	// the sign of s
	return ra.withSign(uint64(int64(s.L3) >> 63))
}

// MulMod returns x * y % m with the sign of x * y, the product in 512 bits.
// It panics for m == 0
func (x Int256) MulMod(y, m Int256) Int256 {
	_, ra, st := x.UnsignedAbs().mulDivMod(y.UnsignedAbs(), m.UnsignedAbs())
	if st > 1 {
		panic("fat.Int256.MulMod: division by zero")
	}
	// the sign of x * y by (v ^ m) - m
	mr := uint64(int64(x.L3^y.L3) >> 63)
	r0, b := bits.Sub64(ra.L0^mr, mr, 0)
	r1, b := bits.Sub64(ra.L1^mr, mr, b)
	r2, b := bits.Sub64(ra.L2^mr, mr, b)
	r3, _ := bits.Sub64(ra.L3^mr, mr, b)
	return Int256{r0, r1, r2, r3}
}

// ExpMod returns x^y % m with the sign of x^y, 0^0 is 1.
// It panics for y < 0 and m == 0
func (x Int256) ExpMod(y, m Int256) Int256 {
	if int64(y.L3) < 0 {
		panic("fat.Int256.ExpMod: negative exponent")
	}
	ra, zero := x.UnsignedAbs().expMod(Uint256(y), m.UnsignedAbs())
	if zero {
		panic("fat.Int256.ExpMod: division by zero")
	}
	r := Int256(ra)
	// x^y < 0 for x < 0 and y odd; r < |m| <= 2^255, so -r fits
	if int64(x.L3) < 0 && y.L0&1 != 0 {
		return r.Neg()
	}
	return r
}

// GCD returns the greatest common divisor of |x| and |y|, |x| for y == 0, as a
// Uint256: GCD(MinInt256, 0) is 2^255
func (x Int256) GCD(y Int256) Uint256 {
	return x.UnsignedAbs().GCD(y.UnsignedAbs())
}

// DivToFloat64 returns the float64 nearest to x / y, ties go to the even one.
// It panics for y == 0
func (x Int256) DivToFloat64(y Int256) float64 {
	// Uint256.DivToFloat64 on |x| and |y|, those picked by conditional moves
	x0, b := bits.Sub64(0, x.L0, 0)
	x1, b := bits.Sub64(0, x.L1, b)
	x2, b := bits.Sub64(0, x.L2, b)
	x3, _ := bits.Sub64(0, x.L3, b)
	if int64(x.L3) >= 0 {
		x0, x1, x2, x3 = x.L0, x.L1, x.L2, x.L3
	}
	y0, b := bits.Sub64(0, y.L0, 0)
	y1, b := bits.Sub64(0, y.L1, b)
	y2, b := bits.Sub64(0, y.L2, b)
	y3, _ := bits.Sub64(0, y.L3, b)
	if int64(y.L3) >= 0 {
		y0, y1, y2, y3 = y.L0, y.L1, y.L2, y.L3
	}
	neg := (x.L3 ^ y.L3) & (1 << 63)
	xh, xm, xl, nx := x3, x2, x1, uint64(0)
	if x3|x2 == 0 {
		xh, xm, xl, nx = x1, x0, 0, 2
	}
	if xh == 0 {
		xh, xm, nx = xm, xl, nx+1
	}
	yh, ym, yl, ny := y3, y2, y1, uint64(0)
	if y3|y2 == 0 {
		yh, ym, yl, ny = y1, y0, 0, 2
	}
	if yh == 0 {
		yh, ym, ny = ym, yl, ny+1
	}
	if arch.IsAmd64 {
		lx, ly := uint(bits.Len64(xh)), uint(bits.Len64(yh))
		u := xh<<(-lx&63) | xm>>1>>((lx-1)&63)
		d := yh<<(-ly&63) | ym>>1>>((ly-1)&63)
		var h uint64
		if u >= d {
			h = 1
		}
		u >>= h & 1
		e := uint64(lx) - uint64(ly) + (ny-nx)<<6 + h + (1023 - 64 + 1)
		if int64(d) < 0 {
			q, _ := bits.Div64(u, 0, d)
			if (q-1<<10+2)&(1<<11-1) > 3 {
				return float64(int64(q>>1))*math.Float64frombits(e<<52|neg) + 0
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
			if (q-1<<10+3)&(1<<11-1) > 4 {
				return float64(q)*math.Float64frombits(e<<52|neg) + 0
			}
		}
	}
	return divFloatExact(Uint256{x0, x1, x2, x3}, Uint256{y0, y1, y2, y3}, neg, "fat.Int256.DivToFloat64: division by zero")
}

// DivToFloat64x returns DivToFloat64 and whether it is exactly x / y.
// It panics for y == 0
func (x Int256) DivToFloat64x(y Int256) (float64, bool) {
	// DivToFloat64 with digits next to a multiple of 2^10 left to divFloatNear.
	// An exact quotient ends at a multiple of 2^11, zero numerators included.
	// Only y = 0 has no top bit in d
	x0, b := bits.Sub64(0, x.L0, 0)
	x1, b := bits.Sub64(0, x.L1, b)
	x2, b := bits.Sub64(0, x.L2, b)
	x3, _ := bits.Sub64(0, x.L3, b)
	if int64(x.L3) >= 0 {
		x0, x1, x2, x3 = x.L0, x.L1, x.L2, x.L3
	}
	y0, b := bits.Sub64(0, y.L0, 0)
	y1, b := bits.Sub64(0, y.L1, b)
	y2, b := bits.Sub64(0, y.L2, b)
	y3, _ := bits.Sub64(0, y.L3, b)
	if int64(y.L3) >= 0 {
		y0, y1, y2, y3 = y.L0, y.L1, y.L2, y.L3
	}
	neg := (x.L3 ^ y.L3) & (1 << 63)
	xh, xm, xl, nx := x3, x2, x1, uint64(0)
	if x3|x2 == 0 {
		xh, xm, xl, nx = x1, x0, 0, 2
	}
	if xh == 0 {
		xh, xm, nx = xm, xl, nx+1
	}
	yh, ym, yl, ny := y3, y2, y1, uint64(0)
	if y3|y2 == 0 {
		yh, ym, yl, ny = y1, y0, 0, 2
	}
	if yh == 0 {
		yh, ym, ny = ym, yl, ny+1
	}
	if arch.IsAmd64 {
		lx, ly := uint(bits.Len64(xh)), uint(bits.Len64(yh))
		u := xh<<(-lx&63) | xm>>1>>((lx-1)&63)
		d := yh<<(-ly&63) | ym>>1>>((ly-1)&63)
		var h uint64
		if u >= d {
			h = 1
		}
		u >>= h & 1
		e := uint64(lx) - uint64(ly) + (ny-nx)<<6 + h + (1023 - 64 + 1)
		if int64(d) < 0 {
			q, _ := bits.Div64(u, 0, d)
			if (q+2)&(1<<10-1) > 3 {
				return float64(int64(q>>1))*math.Float64frombits(e<<52|neg) + 0, false
			}
			return divFloatNear(Uint256{x0, x1, x2, x3}, Uint256{y0, y1, y2, y3}, q, (q+2)&^(1<<10-1), (nx-ny)<<6+uint64(ly)-uint64(lx)+64-h, e-1, neg)
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
				return float64(q)*math.Float64frombits(e<<52|neg) + 0, false
			}
			return divFloatNear(Uint256{x0, x1, x2, x3}, Uint256{y0, y1, y2, y3}, q, (q+3)&^(1<<10-1), (nx-ny)<<6+uint64(sx)-uint64(sy)+64-h, e, neg)
		}
	}
	panic("fat.Int256.DivToFloat64x: division by zero")
}
