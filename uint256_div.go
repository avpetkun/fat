package fat

import (
	"math"
	"math/bits"

	"github.com/avpetkun/fat/internal/arch"
)

// Div64 returns x / y, it panics for y == 0
func (x Uint256) Div64(y uint64) Uint256 {
	if y == 0 {
		panic("fat.Uint256.Div64: division by zero")
	}
	return x.div64(y)
}

// Mod64 returns x % y, it panics for y == 0
func (x Uint256) Mod64(y uint64) uint64 {
	if y == 0 {
		panic("fat.Uint256.Mod64: division by zero")
	}
	return x.mod64(y)
}

// DivMod64 returns x / y and x % y, it panics for y == 0
func (x Uint256) DivMod64(y uint64) (Uint256, uint64) {
	if y == 0 {
		panic("fat.Uint256.DivMod64: division by zero")
	}
	return x.divMod64(y)
}

// divMod64 is DivMod64 for y != 0 only, y == 0 can panic or loop forever
func (x Uint256) divMod64(y uint64) (Uint256, uint64) {
	if arch.IsAmd64 {
		// a hardware 128-by-64 division per quotient limb.
		// The top two limbs take one only when the quotient has a limb there:
		// a short x skips them by branches, faster than four divisions.
		// A third branch, on the next limb, mispredicts for more than it saves
		var q3, q2 uint64
		r := x.L3
		if r >= y {
			q3 = r / y
			r -= q3 * y
		}
		if r|x.L2 != 0 {
			q2, r = bits.Div64(r, x.L2, y)
		}
		q1, r := bits.Div64(r, x.L1, y)
		q0, r := bits.Div64(r, x.L0, y)
		return Uint256{q0, q1, q2, q3}, r
	}
	// the top limb by a hardware division beside the reciprocal of y shifted
	// to its top bit, the rest by that reciprocal.
	// x below 2^128 skips two steps by a branch,
	// faster on Graviton even with half of the branches mispredicted.
	// The branch follows the reciprocal, which survives a misprediction;
	// placed ahead of it, the branch made mixed lengths slower
	s := uint(bits.LeadingZeros64(y)) & 63
	d := y << s
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
	if x.L3|x.L2 == 0 {
		// y != 0, the test keeps panicdivide and its frame out of the kernel
		q1 := x.L1
		if y != 0 {
			q1 = x.L1 / y
		}
		r := x.L1 - q1*y
		q0 := quo2by1(r<<s|x.L0>>1>>t, x.L0<<s, d, v)
		return Uint256{q0, q1, 0, 0}, x.L0 - q0*y
	}
	q3 := x.L3
	if y != 0 {
		q3 = x.L3 / y
	}
	r := x.L3 - q3*y
	q2, r := div2by1r(r<<s|x.L2>>1>>t, x.L2<<s|x.L1>>1>>t, d, v)
	q1, r := div2by1r(r, x.L1<<s|x.L0>>1>>t, d, v)
	q0 := quo2by1(r, x.L0<<s, d, v)
	// x = q*y + r in the low limb:
	// the remainder of the shifted division would need a shift back
	return Uint256{q0, q1, q2, q3}, x.L0 - q0*y
}

// divMod64Flat is divMod64 for the one-limb divisor of divMod, y != 0 only.
// Among divisors of any length the branches of divMod64 mispredict on amd64,
// so there it is four divisions and no branch
func (x Uint256) divMod64Flat(y uint64) (Uint256, uint64) {
	if arch.IsAmd64 {
		q3, r := bits.Div64(0, x.L3, y)
		q2, r := bits.Div64(r, x.L2, y)
		q1, r := bits.Div64(r, x.L1, y)
		q0, r := bits.Div64(r, x.L0, y)
		return Uint256{q0, q1, q2, q3}, r
	}
	// the branch of divMod64 stays here, faster for DivMod on Graviton
	return x.divMod64(y)
}

// div64 is divMod64 without the remainder, whose multiply ends the chain.
// y must not be 0, as for divMod64
func (x Uint256) div64(y uint64) Uint256 {
	if arch.IsAmd64 {
		// a hardware 128-by-64 division per quotient limb there
		var q3, q2 uint64
		r := x.L3
		if r >= y {
			q3 = r / y
			r -= q3 * y
		}
		if r|x.L2 != 0 {
			q2, r = bits.Div64(r, x.L2, y)
		}
		q1, r := bits.Div64(r, x.L1, y)
		q0, _ := bits.Div64(r, x.L0, y)
		return Uint256{q0, q1, q2, q3}
	}
	// as in divMod64
	s := uint(bits.LeadingZeros64(y)) & 63
	d := y << s
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
	if x.L3|x.L2 == 0 {
		q1 := x.L1
		if y != 0 {
			q1 = x.L1 / y
		}
		r := x.L1 - q1*y
		return Uint256{quo2by1(r<<s|x.L0>>1>>t, x.L0<<s, d, v), q1, 0, 0}
	}
	q3 := x.L3
	if y != 0 {
		q3 = x.L3 / y
	}
	r := x.L3 - q3*y
	q2, r := div2by1r(r<<s|x.L2>>1>>t, x.L2<<s|x.L1>>1>>t, d, v)
	q1, r := div2by1r(r, x.L1<<s|x.L0>>1>>t, d, v)
	return Uint256{quo2by1(r, x.L0<<s, d, v), q1, q2, q3}
}

// mod64 is divMod64 without the quotient, y must not be 0 as for divMod64
func (x Uint256) mod64(y uint64) uint64 {
	if arch.IsAmd64 {
		// a hardware 128-by-64 division per quotient limb there
		r := x.L3
		if r >= y {
			r %= y
		}
		if r|x.L2 != 0 {
			_, r = bits.Div64(r, x.L2, y)
		}
		_, r = bits.Div64(r, x.L1, y)
		_, r = bits.Div64(r, x.L0, y)
		return r
	}
	// as in divMod64
	s := uint(bits.LeadingZeros64(y)) & 63
	d := y << s
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
	if x.L3|x.L2 == 0 {
		// the one step corrects by selects, a branch after it is slower
		r := x.L1
		if y != 0 {
			r = x.L1 % y
		}
		_, r = div2by1(r<<s|x.L0>>1>>t, x.L0<<s, d, v)
		return r >> s
	}
	r := x.L3
	if y != 0 {
		r = x.L3 % y
	}
	_, r = div2by1r(r<<s|x.L2>>1>>t, x.L2<<s|x.L1>>1>>t, d, v)
	_, r = div2by1r(r, x.L1<<s|x.L0>>1>>t, d, v)
	_, r = div2by1r(r, x.L0<<s, d, v)
	return r >> s
}

// div2by1r is div2by1 with its second correction as a branch.
// It needs u1 < d and the top bit of d set, as div2by1 does:
// other inputs give a wrong result, and d == 0 never returns.
// One step in a thousand takes the branch,
// and the two instructions it saves wait for the remainder
func div2by1r(u1, u0, d, v uint64) (uint64, uint64) {
	q1, q0 := bits.Mul64(v, u1)
	q0, c := bits.Add64(q0, u0, 0)
	q1, _ = bits.Add64(q1, u1, c)
	r := u0 - d - q1*d
	q := q1 + 1
	if r > q0 {
		q = q1
		r += d
	}
	// one pass at most, a loop stays a branch where an if turns into selects
	for r >= d {
		q++
		r -= d
	}
	return q, r
}

// Div returns x / y, it panics for y == 0
func (x Uint256) Div(y Uint256) Uint256 {
	q, _, zero := x.divMod(y)
	if zero {
		panic("fat.Uint256.Div: division by zero")
	}
	return q
}

// Mod returns x % y, it panics for y == 0
func (x Uint256) Mod(y Uint256) Uint256 {
	_, r, zero := x.divMod(y)
	if zero {
		panic("fat.Uint256.Mod: division by zero")
	}
	return r
}

// DivMod returns x / y and x % y, it panics for y == 0
func (x Uint256) DivMod(y Uint256) (q, r Uint256) {
	q, r, zero := x.divMod(y)
	if zero {
		panic("fat.Uint256.DivMod: division by zero")
	}
	return
}

// Div128 returns x / y, it panics for y == 0
func (x Uint256) Div128(y Uint128) Uint256 {
	q, _, zero := x.divMod128(y)
	if zero {
		panic("fat.Uint256.Div128: division by zero")
	}
	return q
}

// Mod128 returns x % y, it panics for y == 0
func (x Uint256) Mod128(y Uint128) Uint128 {
	_, r, zero := x.divMod128(y)
	if zero {
		panic("fat.Uint256.Mod128: division by zero")
	}
	return r
}

// DivMod128 returns x / y and x % y, it panics for y == 0
func (x Uint256) DivMod128(y Uint128) (q Uint256, r Uint128) {
	q, r, zero := x.divMod128(y)
	if zero {
		panic("fat.Uint256.DivMod128: division by zero")
	}
	return
}

// divMod is DivMod and true for y == 0, with a zero quotient and remainder.
// The zero test is on one limb, where the branches on the length leave it.
// The top quotient digit comes from the top limbs by a hardware division,
// the rest by Knuth's algorithm D on limbs normalized to the top bit of y
func (x Uint256) divMod(y Uint256) (Uint256, Uint256, bool) {
	if y.L3 != 0 {
		// one digit; x < y is not tested, the digit comes out 0
		// for less than a mispredicted test costs
		q := x.L3 / y.L3
		if q > y.L3 {
			// the top limb of y is too short for the estimate
			q, r := x.divModKnuth(y)
			return q, r, false
		}
		// q is the digit or one above it: x - q*y,
		// the product has no fifth limb unless it is above x
		k0, l0 := bits.Mul64(q, y.L0)
		k1, l1 := bits.Mul64(q, y.L1)
		k2, l2 := bits.Mul64(q, y.L2)
		p1, c := bits.Add64(l1, k0, 0)
		p2, c := bits.Add64(l2, k1, c)
		p3, c := bits.Add64(q*y.L3, k2, c)
		r0, b := bits.Sub64(x.L0, l0, 0)
		r1, b := bits.Sub64(x.L1, p1, b)
		r2, b := bits.Sub64(x.L2, p2, b)
		r3, b := bits.Sub64(x.L3, p3, b)
		// rare for a long top limb: one too big, add the divisor back
		if b|c != 0 {
			q--
			r0, c = bits.Add64(r0, y.L0, 0)
			r1, c = bits.Add64(r1, y.L1, c)
			r2, c = bits.Add64(r2, y.L2, c)
			r3, _ = bits.Add64(r3, y.L3, c)
		}
		return Uint256{q, 0, 0, 0}, Uint256{r0, r1, r2, r3}, false
	}
	// half of mixed-length pairs have x < y,
	// the test costs less than the division it skips
	if x.LessThan(y) {
		return Uint256{}, x, false
	}
	if y.L2 != 0 {
		// a power of two is a shift and a mask, as on 2 limbs.
		// Only the divisors of 2 and 3 limbs take the test: there the
		// division is twice as long, and a test for 1 or 4 limbs made those
		// 4..6% slower in our benchmarks, they divide a power fast anyway.
		// >>1>>t is >>(64-s) and <<1<<t is <<(64-s), 0 at s = 0
		if y.L0|y.L1 == 0 && y.L2&(y.L2-1) == 0 {
			s := uint(bits.TrailingZeros64(y.L2))
			t := ^s & 63
			return Uint256{x.L2>>s | x.L3<<1<<t, x.L3 >> s, 0, 0}, Uint256{x.L0, x.L1, x.L2 & (y.L2 - 1), 0}, false
		}
		// two digits, the first as for 4 limbs: <x.L3, x.L2, x.L1> / y
		q1 := x.L3 / y.L2
		if q1 > y.L2 {
			q, r := x.divModKnuth(y)
			return q, r, false
		}
		k0, l0 := bits.Mul64(q1, y.L0)
		k1, l1 := bits.Mul64(q1, y.L1)
		p1, c := bits.Add64(l1, k0, 0)
		p2, c := bits.Add64(q1*y.L2, k1, c)
		w0, b := bits.Sub64(x.L1, l0, 0)
		w1, b := bits.Sub64(x.L2, p1, b)
		w2, b := bits.Sub64(x.L3, p2, b)
		if b|c != 0 {
			q1--
			w0, c = bits.Add64(w0, y.L0, 0)
			w1, c = bits.Add64(w1, y.L1, c)
			w2, _ = bits.Add64(w2, y.L2, c)
		}
		// the second by one step of algorithm D on the remainder over x.L0
		s := uint(bits.LeadingZeros64(y.L2)) & 63
		t := ^s & 63
		// >>1>>t is >>(64-s) that gives 0 at s = 0
		a2 := w2<<s | w1>>1>>t
		a1 := w1<<s | w0>>1>>t
		a0 := w0<<s | x.L0>>1>>t
		u0 := x.L0 << s
		e1 := y.L2<<s | y.L1>>1>>t
		e0 := y.L1<<s | y.L0>>1>>t
		d0 := y.L0 << s
		// the reciprocal (2^192-1)/<e1, e0> - 2^64 written out:
		// reciprocal2by1 of e1, then Algorithm 6 of
		// "Improved division by invariant integers",
		// with both adjustments by carries as they follow no pattern
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
			return Uint256{math.MaxUint64, q1, 0, 0}, Uint256{r0>>s | r1<<1<<t, r1>>s | r2<<1<<t, r2 >> s, 0}, false
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
		// the divisor goes back by a select unless g1 < f:
		// a branch here follows no pattern
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
		return Uint256{q0, q1, 0, 0}, Uint256{r0>>s | r1<<1<<t, r1>>s | r2<<1<<t, r2 >> s, 0}, false
	}
	if y.L1 != 0 {
		// a power of two is a shift and a mask, as on 3 limbs
		if y.L0 == 0 && y.L1&(y.L1-1) == 0 {
			s := uint(bits.TrailingZeros64(y.L1))
			t := ^s & 63
			return Uint256{x.L1>>s | x.L2<<1<<t, x.L2>>s | x.L3<<1<<t, x.L3 >> s, 0}, Uint256{x.L0, x.L1 & (y.L1 - 1), 0, 0}, false
		}
		// three digits, the first as for 4 limbs: <x.L3, x.L2> / y
		q2 := x.L3 / y.L1
		if q2 > y.L1 {
			q, r := x.divModKnuth(y)
			return q, r, false
		}
		k0, l0 := bits.Mul64(q2, y.L0)
		p1, c := bits.Add64(q2*y.L1, k0, 0)
		w0, b := bits.Sub64(x.L2, l0, 0)
		w1, b := bits.Sub64(x.L3, p1, b)
		if b|c != 0 {
			q2--
			w0, c = bits.Add64(w0, y.L0, 0)
			w1, _ = bits.Add64(w1, y.L1, c)
		}
		// the other two exact by the 3-by-2 division
		// on the remainder over x.L1 and x.L0
		s := uint(bits.LeadingZeros64(y.L1)) & 63
		t := ^s & 63
		// >>1>>t is >>(64-s) that gives 0 at s = 0
		a1 := w1<<s | w0>>1>>t
		a0 := w0<<s | x.L1>>1>>t
		u1 := x.L1<<s | x.L0>>1>>t
		u0 := x.L0 << s
		e1 := y.L1<<s | y.L0>>1>>t
		e0 := y.L0 << s
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
		q1, f := bits.Mul64(v, a1)
		f, c = bits.Add64(f, a0, 0)
		q1, _ = bits.Add64(q1, a1, c)
		g1 := a0 - q1*e1
		t1, t0 = bits.Mul64(e0, q1)
		g0, b := bits.Sub64(u1, t0, 0)
		g1, _ = bits.Sub64(g1, t1, b)
		g0, b = bits.Sub64(g0, e0, 0)
		g1, _ = bits.Sub64(g1, e1, b)
		// the divisor goes back by a select unless g1 < f:
		// a branch here follows no pattern
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
		return Uint256{q0, q1, q2, 0}, Uint256{r0>>s | r1<<1<<t, r1 >> s, 0, 0}, false
	}
	// y < 2^64 here, a zero y is the only one to come this far
	if y.L0 == 0 {
		return Uint256{}, Uint256{}, true
	}
	q, r := x.divMod64Flat(y.L0)
	return q, Uint256{r, 0, 0, 0}, false
}

// divMod128 is DivMod128 and true for y == 0, with a zero quotient and remainder.
// It is the 2-limb divisor of divMod written out
func (x Uint256) divMod128(y Uint128) (Uint256, Uint128, bool) {
	if y.L1 == 0 {
		if y.L0 == 0 {
			return Uint256{}, Uint128{}, true
		}
		q, r := x.divMod64Flat(y.L0)
		return q, Uint128{r, 0}, false
	}
	// a quarter of mixed-length pairs have x < y
	_, b := bits.Sub64(x.L0, y.L0, 0)
	_, b = bits.Sub64(x.L1, y.L1, b)
	_, b = bits.Sub64(x.L2, 0, b)
	_, b = bits.Sub64(x.L3, 0, b)
	if b != 0 {
		return Uint256{}, Uint128{x.L0, x.L1}, false
	}
	// three digits, the first from the top limbs by a hardware division:
	// <x.L3, x.L2> / y
	q2 := x.L3 / y.L1
	if q2 > y.L1 {
		q, r := x.divModKnuth(Uint256{y.L0, y.L1, 0, 0})
		return q, Uint128{r.L0, r.L1}, false
	}
	k0, l0 := bits.Mul64(q2, y.L0)
	p1, c := bits.Add64(q2*y.L1, k0, 0)
	w0, b := bits.Sub64(x.L2, l0, 0)
	w1, b := bits.Sub64(x.L3, p1, b)
	if b|c != 0 {
		q2--
		w0, c = bits.Add64(w0, y.L0, 0)
		w1, _ = bits.Add64(w1, y.L1, c)
	}
	// the other two exact by the 3-by-2 division
	// on the remainder over x.L1 and x.L0
	s := uint(bits.LeadingZeros64(y.L1)) & 63
	t := ^s & 63
	// >>1>>t is >>(64-s) that gives 0 at s = 0
	a1 := w1<<s | w0>>1>>t
	a0 := w0<<s | x.L1>>1>>t
	u1 := x.L1<<s | x.L0>>1>>t
	u0 := x.L0 << s
	e1 := y.L1<<s | y.L0>>1>>t
	e0 := y.L0 << s
	// the reciprocal (2^192-1)/<e1, e0> - 2^64 written out as in divMod
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
	q1, f := bits.Mul64(v, a1)
	f, c = bits.Add64(f, a0, 0)
	q1, _ = bits.Add64(q1, a1, c)
	g1 := a0 - q1*e1
	t1, t0 = bits.Mul64(e0, q1)
	g0, b := bits.Sub64(u1, t0, 0)
	g1, _ = bits.Sub64(g1, t1, b)
	g0, b = bits.Sub64(g0, e0, 0)
	g1, _ = bits.Sub64(g1, e1, b)
	// the divisor goes back by a select unless g1 < f:
	// a branch here follows no pattern
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
	return Uint256{q0, q1, q2, 0}, Uint128{r0>>s | r1<<1<<t, r1 >> s}, false
}

// divModKnuth is divMod for y >= 2^64 only, a smaller y gives a wrong result.
// divMod calls it when the top limb of y is too short to estimate a digit.
// It is Knuth's algorithm D on limbs shifted to set the top bit of y,
// with the digits from the 3-by-2 division by its top two limbs
func (x Uint256) divModKnuth(y Uint256) (Uint256, Uint256) {
	top := y.L1
	if y.L2 != 0 {
		top = y.L2
	}
	if y.L3 != 0 {
		top = y.L3
	}
	s := uint(bits.LeadingZeros64(top)) & 63
	t := ^s & 63
	// >>1>>t is >>(64-s) that gives 0 at s = 0
	u4 := x.L3 >> 1 >> t
	u3 := x.L3<<s | x.L2>>1>>t
	u2 := x.L2<<s | x.L1>>1>>t
	u1 := x.L1<<s | x.L0>>1>>t
	u0 := x.L0 << s
	d3 := y.L3<<s | y.L2>>1>>t
	d2 := y.L2<<s | y.L1>>1>>t
	d1 := y.L1<<s | y.L0>>1>>t
	d0 := y.L0 << s
	// the top two limbs of the divisor
	e1, e0 := d1, d0
	if y.L2 != 0 {
		e1, e0 = d2, d1
	}
	if y.L3 != 0 {
		e1, e0 = d3, d2
	}
	// the reciprocal (2^192-1)/<e1, e0> - 2^64 written out:
	// reciprocal2by1 of e1, then Algorithm 6 of
	// "Improved division by invariant integers",
	// with both adjustments by carries as they follow no pattern
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
	_, b := bits.Sub64(p, e1, 0)
	k := c + c&^b
	v -= k
	p -= e1 * k
	t1, t0 := bits.Mul64(v, e0)
	p, c = bits.Add64(p, t1, 0)
	_, b = bits.Sub64(t0, e0, 0)
	_, b = bits.Sub64(p, e1, b)
	v -= c + c&^b
	// the first digit by the 3-by-2 division of Algorithm 5:
	// u4 < 2^s is below e1, so it does not overflow
	qf, f := bits.Mul64(v, u4)
	f, c = bits.Add64(f, u3, 0)
	qf, _ = bits.Add64(qf, u4, c)
	h1 := u3 - qf*e1
	t1, t0 = bits.Mul64(e0, qf)
	h0, b := bits.Sub64(u2, t0, 0)
	h1, _ = bits.Sub64(h1, t1, b)
	h0, b = bits.Sub64(h0, e0, 0)
	h1, _ = bits.Sub64(h1, e1, b)
	// the divisor goes back by a select unless h1 < f:
	// a branch here follows no pattern
	m1, m0 := e1, e0
	if h1 < f {
		m1, m0 = 0, 0
		qf++
	}
	h0, c = bits.Add64(h0, m0, 0)
	h1, _ = bits.Add64(h1, m1, c)
	// rare
	if h1 > e1 || h1 == e1 && h0 >= e0 {
		qf++
		h0, b = bits.Sub64(h0, e0, 0)
		h1, _ = bits.Sub64(h1, e1, b)
	}
	if y.L3 != 0 {
		// one digit
		p1, p0 := bits.Mul64(qf, d0)
		p2, pm := bits.Mul64(qf, d1)
		p1, c = bits.Add64(p1, pm, 0)
		p2 += c
		r0, b := bits.Sub64(u0, p0, 0)
		r1, b := bits.Sub64(u1, p1, b)
		r2, b := bits.Sub64(h0, p2, b)
		r3, b := bits.Sub64(h1, 0, b)
		// rare: the digit was one too big, add the divisor back
		if b != 0 {
			qf--
			r0, c = bits.Add64(r0, d0, 0)
			r1, c = bits.Add64(r1, d1, c)
			r2, c = bits.Add64(r2, d2, c)
			r3, _ = bits.Add64(r3, d3, c)
		}
		return Uint256{qf, 0, 0, 0}, Uint256{r0>>s | r1<<1<<t, r1>>s | r2<<1<<t, r2>>s | r3<<1<<t, r3 >> s}
	}
	if y.L2 != 0 {
		// two digits
		t1, t0 = bits.Mul64(qf, d0)
		a0, b := bits.Sub64(u1, t0, 0)
		a1, b := bits.Sub64(h0, t1, b)
		a2, b := bits.Sub64(h1, 0, b)
		// rare: the digit was one too big, add the divisor back
		if b != 0 {
			qf--
			a0, c = bits.Add64(a0, d0, 0)
			a1, c = bits.Add64(a1, d1, c)
			a2, _ = bits.Add64(a2, d2, c)
		}
		if a2 == d2 && a1 == d1 {
			// rare: the digit is 2^64-1, so a - q*d is a + d - d<<64
			r0, c := bits.Add64(u0, d0, 0)
			r1, c := bits.Add64(a0, d1, c)
			r2, _ := bits.Add64(a1, d2, c)
			r1, b = bits.Sub64(r1, d0, 0)
			r2, _ = bits.Sub64(r2, d1, b)
			return Uint256{math.MaxUint64, qf, 0, 0}, Uint256{r0>>s | r1<<1<<t, r1>>s | r2<<1<<t, r2 >> s, 0}
		}
		q0, f := bits.Mul64(v, a2)
		f, c = bits.Add64(f, a1, 0)
		q0, _ = bits.Add64(q0, a2, c)
		g1 := a1 - q0*e1
		t1, t0 = bits.Mul64(e0, q0)
		g0, b := bits.Sub64(a0, t0, 0)
		g1, _ = bits.Sub64(g1, t1, b)
		g0, b = bits.Sub64(g0, e0, 0)
		g1, _ = bits.Sub64(g1, e1, b)
		m1, m0 = e1, e0
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
			r1, c = bits.Add64(r1, d1, c)
			r2, _ = bits.Add64(r2, d2, c)
		}
		return Uint256{q0, qf, 0, 0}, Uint256{r0>>s | r1<<1<<t, r1>>s | r2<<1<<t, r2 >> s, 0}
	}
	// three digits, each exact
	q1, f := bits.Mul64(v, h1)
	f, c = bits.Add64(f, h0, 0)
	q1, _ = bits.Add64(q1, h1, c)
	g1 := h0 - q1*e1
	t1, t0 = bits.Mul64(e0, q1)
	g0, b := bits.Sub64(u1, t0, 0)
	g1, _ = bits.Sub64(g1, t1, b)
	g0, b = bits.Sub64(g0, e0, 0)
	g1, _ = bits.Sub64(g1, e1, b)
	m1, m0 = e1, e0
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
	return Uint256{q0, q1, qf, 0}, Uint256{r0>>s | r1<<1<<t, r1 >> s, 0, 0}
}

// uint256MulDivPanics, uint256MulDivModPanics and uint256MulDivCeilPanics are the panics by kernel status.
// One panic from a table keeps the wrappers within the inline budget
var (
	uint256MulDivPanics = [3]string{
		1: "fat.Uint256.MulDiv: overflow",
		2: "fat.Uint256.MulDiv: division by zero",
	}
	uint256MulDivModPanics = [3]string{
		1: "fat.Uint256.MulDivMod: overflow",
		2: "fat.Uint256.MulDivMod: division by zero",
	}
	uint256MulDivCeilPanics = [3]string{
		1: "fat.Uint256.MulDivCeil: overflow",
		2: "fat.Uint256.MulDivCeil: division by zero",
	}
)

// MulDiv returns x * y / d with the product in 512 bits.
// It panics for d == 0 and when the quotient overflows
func (x Uint256) MulDiv(y, d Uint256) (q Uint256) {
	q, _, st := x.mulDivMod(y, d)
	if st != 0 {
		panic(uint256MulDivPanics[st])
	}
	return
}

// MulDivo returns MulDivu and whether the quotient overflowed.
// It panics for d == 0
func (x Uint256) MulDivo(y, d Uint256) (q Uint256, overflow bool) {
	q, _, st := x.mulDivMod(y, d)
	if st > 1 {
		panic("fat.Uint256.MulDivo: division by zero")
	}
	return q, st != 0
}

// MulDivu returns x * y / d wrapped around, the product in 512 bits.
// It panics for d == 0
func (x Uint256) MulDivu(y, d Uint256) (q Uint256) {
	q, _, st := x.mulDivMod(y, d)
	if st > 1 {
		panic("fat.Uint256.MulDivu: division by zero")
	}
	return
}

// MulDivMod returns MulDiv and x * y % d.
// It panics for d == 0 and when the quotient overflows
func (x Uint256) MulDivMod(y, d Uint256) (q, r Uint256) {
	q, r, st := x.mulDivMod(y, d)
	if st != 0 {
		panic(uint256MulDivModPanics[st])
	}
	return
}

// MulDivCeil returns x * y / d rounded up, the product in 512 bits.
// It panics for d == 0 and when the quotient overflows
func (x Uint256) MulDivCeil(y, d Uint256) Uint256 {
	q, st := x.mulDivCeil(y, d)
	if st != 0 {
		panic(uint256MulDivCeilPanics[st])
	}
	return q
}

// AddMod returns (x + y) % m of the exact sum, it panics for m == 0
func (x Uint256) AddMod(y, m Uint256) Uint256 {
	if m.IsZero() {
		panic("fat.Uint256.AddMod: division by zero")
	}
	s0, c := bits.Add64(x.L0, y.L0, 0)
	s1, c := bits.Add64(x.L1, y.L1, c)
	s2, c := bits.Add64(x.L2, y.L2, c)
	s3, c := bits.Add64(x.L3, y.L3, c)
	// the 257-bit sum less m and less 2m:
	// a sum below 2m is the common case, as of operands below m
	t0, b := bits.Sub64(s0, m.L0, 0)
	t1, b := bits.Sub64(s1, m.L1, b)
	t2, b := bits.Sub64(s2, m.L2, b)
	t3, b := bits.Sub64(s3, m.L3, b)
	_, d := bits.Sub64(t0, m.L0, 0)
	_, d = bits.Sub64(t1, m.L1, d)
	_, d = bits.Sub64(t2, m.L2, d)
	_, d = bits.Sub64(t3, m.L3, d)
	// s < m with no carry, or s - m in 256 bits and below m
	if b&^c|d&^(b^c) != 0 {
		// picked without a branch, sums fall on both sides of m at random
		if b > c {
			t0, t1, t2, t3 = s0, s1, s2, s3
		}
		return Uint256{t0, t1, t2, t3}
	}
	if c == 0 {
		// one division of the sum
		_, r, _ := Uint256{s0, s1, s2, s3}.divMod(m)
		return r
	}
	// rare unless the operands are full width:
	// a carried sum takes both reduced, then it is below 2m
	_, x, _ = x.divMod(m)
	_, y, _ = y.divMod(m)
	s0, c = bits.Add64(x.L0, y.L0, 0)
	s1, c = bits.Add64(x.L1, y.L1, c)
	s2, c = bits.Add64(x.L2, y.L2, c)
	s3, c = bits.Add64(x.L3, y.L3, c)
	t0, b = bits.Sub64(s0, m.L0, 0)
	t1, b = bits.Sub64(s1, m.L1, b)
	t2, b = bits.Sub64(s2, m.L2, b)
	t3, b = bits.Sub64(s3, m.L3, b)
	if b > c {
		t0, t1, t2, t3 = s0, s1, s2, s3
	}
	return Uint256{t0, t1, t2, t3}
}

// SubMod returns (x - y) mod m of the exact difference, in [0, m) for x < y too.
// It panics for m == 0
func (x Uint256) SubMod(y, m Uint256) Uint256 {
	if m.IsZero() {
		panic("fat.Uint256.SubMod: division by zero")
	}
	d0, b := bits.Sub64(x.L0, y.L0, 0)
	d1, b := bits.Sub64(x.L1, y.L1, b)
	d2, b := bits.Sub64(x.L2, y.L2, b)
	d3, b := bits.Sub64(x.L3, y.L3, b)
	e0, c := bits.Add64(d0, m.L0, 0)
	e1, c := bits.Add64(d1, m.L1, c)
	e2, c := bits.Add64(d2, m.L2, c)
	e3, c := bits.Add64(d3, m.L3, c)
	_, t := bits.Sub64(d0, m.L0, 0)
	_, t = bits.Sub64(d1, m.L1, t)
	_, t = bits.Sub64(d2, m.L2, t)
	_, t = bits.Sub64(d3, m.L3, t)
	// operands below m are the common case: |x - y| < m takes x - y, or
	// x - y + m for x < y, picked by conditional moves as the sign is a coin flip.
	// The test is d < m without a borrow, y - x <= m by the carry of d + m with one
	if t&^b|c&b != 0 {
		if b != 0 {
			d0, d1, d2, d3 = e0, e1, e2, e3
		}
		return Uint256{d0, d1, d2, d3}
	}
	if b == 0 {
		_, r, _ := Uint256{d0, d1, d2, d3}.divMod(m)
		return r
	}
	// m - (y - x) mod m, or 0 where m divides y - x
	d0, b = bits.Sub64(0, d0, 0)
	d1, b = bits.Sub64(0, d1, b)
	d2, b = bits.Sub64(0, d2, b)
	d3, _ = bits.Sub64(0, d3, b)
	_, r, _ := Uint256{d0, d1, d2, d3}.divMod(m)
	if r.IsZero() {
		return r
	}
	return m.Subu(r)
}

// MulMod returns x * y % m with the product in 512 bits, it panics for m == 0
func (x Uint256) MulMod(y, m Uint256) (r Uint256) {
	_, r, st := x.mulDivMod(y, m)
	if st > 1 {
		panic("fat.Uint256.MulMod: division by zero")
	}
	return
}

// ExpMod returns x^y % m, 0^0 is 1, it panics for m == 0
func (x Uint256) ExpMod(y, m Uint256) Uint256 {
	r, zero := x.expMod(y, m)
	if zero {
		panic("fat.Uint256.ExpMod: division by zero")
	}
	return r
}

// expMod is ExpMod and true for m == 0, with a zero result
func (x Uint256) expMod(y, m Uint256) (Uint256, bool) {
	// m == 1 gives 0, one test for both
	if m.L1|m.L2|m.L3 == 0 && m.L0 <= 1 {
		return Uint256{}, m.L0 == 0
	}
	if y.IsZero() {
		return Uint256{L0: 1}, false
	}
	b := x
	if !b.LessThan(m) {
		_, b, _ = b.divMod(m)
	}
	if m.L0&1 == 0 {
		return expModEven(b, y, m), false
	}
	return expModOdd(b, y, m), false
}

// GCD returns the greatest common divisor of x and y, x for y == 0
func (x Uint256) GCD(y Uint256) Uint256 {
	if x.IsZero() {
		return y
	}
	if y.IsZero() {
		return x
	}
	// one division takes the difference in length at once,
	// where a binary step removes a bit or two
	if x.LessThan(y) {
		x, y = y, x
	}
	_, x, _ = x.divMod(y)
	if x.IsZero() {
		return y
	}
	// the common power of two, then both odd
	tx, ty := x.TrailingZeros(), y.TrailingZeros()
	return gcdOdd256(x.Rsh(tx), y.Rsh(ty)).Lsh(min(tx, ty))
}

// ModInverse returns y with x * y % m == 1 and true, or 0 and false where x
// and m share a factor and none exists; for m == 1 it is 0 and true.
// It panics for m == 0
func (x Uint256) ModInverse(m Uint256) (Uint256, bool) {
	if m.IsZero() {
		panic("fat.Uint256.ModInverse: division by zero")
	}
	if !x.LessThan(m) {
		_, x, _ = x.divMod(m)
	}
	if m.L2|m.L3 == 0 {
		r, ok := Uint128{x.L0, x.L1}.ModInverse(Uint128{m.L0, m.L1})
		return Uint256{r.L0, r.L1, 0, 0}, ok
	}
	if m.L0&1 != 0 {
		return invOdd256(x, m)
	}
	// an even m takes an odd x, and 1 is its own inverse
	if x.L0&1 == 0 {
		return Uint256{}, false
	}
	if x == (Uint256{L0: 1}) {
		return x, true
	}
	// with m*t = 1 (mod x) from the odd modulus x, m*t = 1 + x*s
	// and x*(m - s) = 1 (mod m), where s = m*t / x < m
	_, mx, _ := m.divMod(x)
	t, ok := invOdd256(mx, x)
	if !ok {
		return Uint256{}, false
	}
	s, _, _ := m.mulDivMod(t, x)
	return m.Subu(s), true
}

// expModEven is expMod for an even m, y > 0 and b < m by the Chinese
// remainder theorem on m = 2^k * q with q odd: b^y % q by expModOdd and
// b^y % 2^k as the low bits of Expu, joined by Garner's formula
// a + q * ((c - a) / q % 2^k). A window of products reduced by division
// was twice as slow
func expModEven(b, y, m Uint256) Uint256 {
	k := m.TrailingZeros()
	mask := Uint256{}.Not().Rsh(256 - k)
	c := b.Expu(y).And(mask)
	q := m.Rsh(k)
	// q is odd, so q == 1 by one OR
	if q.L0^1|q.L1|q.L2|q.L3 == 0 {
		return c
	}
	if !b.LessThan(q) {
		_, b, _ = b.divMod(q)
	}
	a := expModOdd(b, y, q)
	// 1/q % 2^256 by Newton steps from the 3 right bits of q itself,
	// each doubles them
	v := q
	for range 7 {
		v = v.Mulu(Uint256{L0: 2}.Subu(q.Mulu(v)))
	}
	// a + q*t < q + q*(2^k-1) = m
	return a.Addu(q.Mulu(c.Subu(a).Mulu(v).And(mask)))
}

// expModOdd is expMod for an odd m > 1, y > 0 and b < m: Montgomery
// products on as many limbs as m has, but 3, which take those of 4
func expModOdd(b, y, m Uint256) Uint256 {
	if m.L1|m.L2|m.L3 == 0 {
		return Uint256{L0: expModMont64(b.L0, y, m.L0)}
	}
	if m.L2|m.L3 == 0 {
		return expModMont128(Uint128{b.L0, b.L1}, y, Uint128{m.L0, m.L1}).Uint256()
	}
	return expModMont(b, y, m)
}

// expWindow returns the width of the sliding window for a y of n bits:
// the 2^(w-1) odd powers of the table pay off from these lengths on
func expWindow(n int) int {
	switch {
	case n > 80:
		return 4
	case n > 17:
		return 3
	}
	return 1
}

// expModMont is expMod for an odd m > 1, y > 0 and b < m by Montgomery
// products modulo R = 2^256 with a sliding window over y: a square per bit
// and a product per window, by an odd power of b from a table.
// It is 2.6 times as fast as the products reduced by mulDivMod at 256 bits
func expModMont(b, y, m Uint256) Uint256 {
	mp := montInv(m.L0)
	// R % m from 2^256 - m, then R^2 % m to take b to R*b % m
	_, r, _ := Uint256{}.Subu(m).divMod(m)
	_, r2, _ := r.mulDivMod(r, m)
	var odd [8]Uint256
	odd[0] = montMul(b, r2, m, mp)
	n, w := y.BitLen(), expWindow(y.BitLen())
	if w > 1 {
		b2 := montSqr(odd[0], m, mp)
		for k := 1; k < 1<<(w-1); k++ {
			odd[k] = montMul(odd[k-1], b2, m, mp)
		}
	}
	// from the top bit of y, which is set, so the first window starts r.
	// From the low bit the squares and the products overlap as two chains,
	// but that is 10% slower: the products, not their latency, are the bound
	started := false
	for i := n - 1; i >= 0; {
		if y.Bit(i) == 0 {
			r = montSqr(r, m, mp)
			i--
			continue
		}
		// the window of up to w bits from i down to a set bit
		j := max(i-w+1, 0)
		for y.Bit(j) == 0 {
			j++
		}
		v := 0
		for k := i; k >= j; k-- {
			v = v<<1 | int(y.Bit(k))
			if started {
				r = montSqr(r, m, mp)
			}
		}
		if started {
			r = montMul(r, odd[v>>1], m, mp)
		} else {
			r, started = odd[v>>1], true
		}
		i = j - 1
	}
	// out of the Montgomery form
	return montMul(r, Uint256{L0: 1}, m, mp)
}

// montMul returns a * b / 2^256 % m by Montgomery's product: the 512-bit
// product, then four rounds that each add a multiple of m to clear a limb.
// The 16 limb products of the first part run in parallel: 11% faster in
// ExpMod of 256 bits than a reduction that follows each row of b (CIOS).
// m must be odd and a, b below it, mp = -1/m mod 2^64, else it is wrong
func montMul(a, b, m Uint256, mp uint64) Uint256 {
	// the 512-bit product p7..p0 row by row,
	// each row a chain of the low and one of the high halves
	h0, p0 := bits.Mul64(a.L0, b.L0)
	h1, l1 := bits.Mul64(a.L0, b.L1)
	h2, l2 := bits.Mul64(a.L0, b.L2)
	h3, l3 := bits.Mul64(a.L0, b.L3)
	p1, c := bits.Add64(h0, l1, 0)
	p2, c := bits.Add64(h1, l2, c)
	p3, c := bits.Add64(h2, l3, c)
	p4 := h3 + c
	h0, l0 := bits.Mul64(a.L1, b.L0)
	h1, l1 = bits.Mul64(a.L1, b.L1)
	h2, l2 = bits.Mul64(a.L1, b.L2)
	h3, l3 = bits.Mul64(a.L1, b.L3)
	p1, c = bits.Add64(p1, l0, 0)
	p2, c = bits.Add64(p2, l1, c)
	p3, c = bits.Add64(p3, l2, c)
	p4, c = bits.Add64(p4, l3, c)
	p5 := c
	p2, c = bits.Add64(p2, h0, 0)
	p3, c = bits.Add64(p3, h1, c)
	p4, c = bits.Add64(p4, h2, c)
	p5 += h3 + c
	h0, l0 = bits.Mul64(a.L2, b.L0)
	h1, l1 = bits.Mul64(a.L2, b.L1)
	h2, l2 = bits.Mul64(a.L2, b.L2)
	h3, l3 = bits.Mul64(a.L2, b.L3)
	p2, c = bits.Add64(p2, l0, 0)
	p3, c = bits.Add64(p3, l1, c)
	p4, c = bits.Add64(p4, l2, c)
	p5, c = bits.Add64(p5, l3, c)
	p6 := c
	p3, c = bits.Add64(p3, h0, 0)
	p4, c = bits.Add64(p4, h1, c)
	p5, c = bits.Add64(p5, h2, c)
	p6 += h3 + c
	h0, l0 = bits.Mul64(a.L3, b.L0)
	h1, l1 = bits.Mul64(a.L3, b.L1)
	h2, l2 = bits.Mul64(a.L3, b.L2)
	h3, l3 = bits.Mul64(a.L3, b.L3)
	p3, c = bits.Add64(p3, l0, 0)
	p4, c = bits.Add64(p4, l1, c)
	p5, c = bits.Add64(p5, l2, c)
	p6, c = bits.Add64(p6, l3, c)
	p7 := c
	p4, c = bits.Add64(p4, h0, 0)
	p5, c = bits.Add64(p5, h1, c)
	p6, c = bits.Add64(p6, h2, c)
	p7 += h3 + c
	// k carries into the limb above each round. The low half of u*m0 is
	// -p_i by the choice of u, so only its high half is taken, and the
	// carry out of the low limb is 1 for p_i != 0
	var u, k uint64
	u = p0 * mp
	c, _ = bits.Mul64(u, m.L0)
	c += uint64(nonzero(p0))
	c, p1 = madd(u, m.L1, p1, c)
	c, p2 = madd(u, m.L2, p2, c)
	c, p3 = madd(u, m.L3, p3, c)
	p4, k = bits.Add64(p4, c, 0)
	u = p1 * mp
	c, _ = bits.Mul64(u, m.L0)
	c += uint64(nonzero(p1))
	c, p2 = madd(u, m.L1, p2, c)
	c, p3 = madd(u, m.L2, p3, c)
	c, p4 = madd(u, m.L3, p4, c)
	p5, k = bits.Add64(p5, c, k)
	u = p2 * mp
	c, _ = bits.Mul64(u, m.L0)
	c += uint64(nonzero(p2))
	c, p3 = madd(u, m.L1, p3, c)
	c, p4 = madd(u, m.L2, p4, c)
	c, p5 = madd(u, m.L3, p5, c)
	p6, k = bits.Add64(p6, c, k)
	u = p3 * mp
	c, _ = bits.Mul64(u, m.L0)
	c += uint64(nonzero(p3))
	c, p4 = madd(u, m.L1, p4, c)
	c, p5 = madd(u, m.L2, p5, c)
	c, p6 = madd(u, m.L3, p6, c)
	p7, k = bits.Add64(p7, c, k)
	// p7..p4 + k*2^256 < 2m: less m unless below it, picked by a mask,
	// as the subtraction follows no pattern
	r0, br := bits.Sub64(p4, m.L0, 0)
	r1, br := bits.Sub64(p5, m.L1, br)
	r2, br := bits.Sub64(p6, m.L2, br)
	r3, br := bits.Sub64(p7, m.L3, br)
	keep := -(br &^ k)
	return Uint256{r0&^keep | p4&keep, r1&^keep | p5&keep, r2&^keep | p6&keep, r3&^keep | p7&keep}
}

// montSqr is montMul(a, a, m, mp) with the product of a square:
// the 6 cross products doubled and the 4 squares of the limbs, 10 limb
// products for 16; the squares are most of the steps of ExpMod
func montSqr(a, m Uint256, mp uint64) Uint256 {
	h01, l01 := bits.Mul64(a.L0, a.L1)
	h02, l02 := bits.Mul64(a.L0, a.L2)
	h03, l03 := bits.Mul64(a.L0, a.L3)
	h12, l12 := bits.Mul64(a.L1, a.L2)
	h13, l13 := bits.Mul64(a.L1, a.L3)
	h23, l23 := bits.Mul64(a.L2, a.L3)
	// the cross sum c6..c1: a0 * a3..a1 at limb 1,
	// a1 * a3..a2 at limb 3, a2 * a3 at limb 5
	c2, c := bits.Add64(h01, l02, 0)
	c3, c := bits.Add64(h02, l03, c)
	c4 := h03 + c
	u4, c := bits.Add64(h12, l13, 0)
	u5 := h13 + c
	c3, c = bits.Add64(c3, l12, 0)
	c4, c = bits.Add64(c4, u4, c)
	c5 := u5 + c
	c5, c = bits.Add64(c5, l23, 0)
	c6 := h23 + c
	// doubled into c7..c1
	c7 := c6 >> 63
	c6 = c6<<1 | c5>>63
	c5 = c5<<1 | c4>>63
	c4 = c4<<1 | c3>>63
	c3 = c3<<1 | c2>>63
	c2 = c2<<1 | l01>>63
	c1 := l01 << 1
	// and the squares on the even limbs
	s0h, p0 := bits.Mul64(a.L0, a.L0)
	s1h, s1l := bits.Mul64(a.L1, a.L1)
	s2h, s2l := bits.Mul64(a.L2, a.L2)
	s3h, s3l := bits.Mul64(a.L3, a.L3)
	p1, c := bits.Add64(c1, s0h, 0)
	p2, c := bits.Add64(c2, s1l, c)
	p3, c := bits.Add64(c3, s1h, c)
	p4, c := bits.Add64(c4, s2l, c)
	p5, c := bits.Add64(c5, s2h, c)
	p6, c := bits.Add64(c6, s3l, c)
	p7, _ := bits.Add64(c7, s3h, c)
	// k carries into the limb above each round. The low half of u*m0 is
	// -p_i by the choice of u, so only its high half is taken, and the
	// carry out of the low limb is 1 for p_i != 0
	var u, k uint64
	u = p0 * mp
	c, _ = bits.Mul64(u, m.L0)
	c += uint64(nonzero(p0))
	c, p1 = madd(u, m.L1, p1, c)
	c, p2 = madd(u, m.L2, p2, c)
	c, p3 = madd(u, m.L3, p3, c)
	p4, k = bits.Add64(p4, c, 0)
	u = p1 * mp
	c, _ = bits.Mul64(u, m.L0)
	c += uint64(nonzero(p1))
	c, p2 = madd(u, m.L1, p2, c)
	c, p3 = madd(u, m.L2, p3, c)
	c, p4 = madd(u, m.L3, p4, c)
	p5, k = bits.Add64(p5, c, k)
	u = p2 * mp
	c, _ = bits.Mul64(u, m.L0)
	c += uint64(nonzero(p2))
	c, p3 = madd(u, m.L1, p3, c)
	c, p4 = madd(u, m.L2, p4, c)
	c, p5 = madd(u, m.L3, p5, c)
	p6, k = bits.Add64(p6, c, k)
	u = p3 * mp
	c, _ = bits.Mul64(u, m.L0)
	c += uint64(nonzero(p3))
	c, p4 = madd(u, m.L1, p4, c)
	c, p5 = madd(u, m.L2, p5, c)
	c, p6 = madd(u, m.L3, p6, c)
	p7, k = bits.Add64(p7, c, k)
	// p7..p4 + k*2^256 < 2m: less m unless below it, picked by a mask,
	// as the subtraction follows no pattern
	r0, br := bits.Sub64(p4, m.L0, 0)
	r1, br := bits.Sub64(p5, m.L1, br)
	r2, br := bits.Sub64(p6, m.L2, br)
	r3, br := bits.Sub64(p7, m.L3, br)
	keep := -(br &^ k)
	return Uint256{r0&^keep | p4&keep, r1&^keep | p5&keep, r2&^keep | p6&keep, r3&^keep | p7&keep}
}

// mulDivMod is MulDivMod with the low bits of the quotient and a status:
// 1 when the quotient overflowed, 2 for d == 0
func (x Uint256) mulDivMod(y, d Uint256) (q, r Uint256, st uint) {
	// the 512-bit product p7..p0 row by row,
	// each row a chain of the low and one of the high halves
	h0, p0 := bits.Mul64(x.L0, y.L0)
	h1, l1 := bits.Mul64(x.L0, y.L1)
	h2, l2 := bits.Mul64(x.L0, y.L2)
	h3, l3 := bits.Mul64(x.L0, y.L3)
	p1, c := bits.Add64(h0, l1, 0)
	p2, c := bits.Add64(h1, l2, c)
	p3, c := bits.Add64(h2, l3, c)
	p4 := h3 + c
	h0, l0 := bits.Mul64(x.L1, y.L0)
	h1, l1 = bits.Mul64(x.L1, y.L1)
	h2, l2 = bits.Mul64(x.L1, y.L2)
	h3, l3 = bits.Mul64(x.L1, y.L3)
	p1, c = bits.Add64(p1, l0, 0)
	p2, c = bits.Add64(p2, l1, c)
	p3, c = bits.Add64(p3, l2, c)
	p4, c = bits.Add64(p4, l3, c)
	p5 := c
	p2, c = bits.Add64(p2, h0, 0)
	p3, c = bits.Add64(p3, h1, c)
	p4, c = bits.Add64(p4, h2, c)
	p5 += h3 + c
	h0, l0 = bits.Mul64(x.L2, y.L0)
	h1, l1 = bits.Mul64(x.L2, y.L1)
	h2, l2 = bits.Mul64(x.L2, y.L2)
	h3, l3 = bits.Mul64(x.L2, y.L3)
	p2, c = bits.Add64(p2, l0, 0)
	p3, c = bits.Add64(p3, l1, c)
	p4, c = bits.Add64(p4, l2, c)
	p5, c = bits.Add64(p5, l3, c)
	p6 := c
	p3, c = bits.Add64(p3, h0, 0)
	p4, c = bits.Add64(p4, h1, c)
	p5, c = bits.Add64(p5, h2, c)
	p6 += h3 + c
	h0, l0 = bits.Mul64(x.L3, y.L0)
	h1, l1 = bits.Mul64(x.L3, y.L1)
	h2, l2 = bits.Mul64(x.L3, y.L2)
	h3, l3 = bits.Mul64(x.L3, y.L3)
	p3, c = bits.Add64(p3, l0, 0)
	p4, c = bits.Add64(p4, l1, c)
	p5, c = bits.Add64(p5, l2, c)
	p6, c = bits.Add64(p6, l3, c)
	p7 := c
	p4, c = bits.Add64(p4, h0, 0)
	p5, c = bits.Add64(p5, h1, c)
	p6, c = bits.Add64(p6, h2, c)
	p7 += h3 + c
	// a product within 256 bits
	if p4|p5|p6|p7 == 0 {
		q, r, zero := Uint256{p0, p1, p2, p3}.divMod(d)
		if zero {
			st = 2
		}
		return q, r, st
	}
	return div512o(Uint256{p4, p5, p6, p7}, Uint256{p0, p1, p2, p3}, d)
}

// mulDivCeil is MulDivCeil with the low 256 bits of q and a status as of mulDivMod
func (x Uint256) mulDivCeil(y, d Uint256) (Uint256, uint) {
	q, r, st := x.mulDivMod(y, d)
	// a remainder adds one, which carries out only of an overflowed q
	// or of the maximum. A zero d leaves r = 0
	nz := r.L0 | r.L1 | r.L2 | r.L3
	q0, c := bits.Add64(q.L0, (nz|-nz)>>63, 0)
	q1, c := bits.Add64(q.L1, 0, c)
	q2, c := bits.Add64(q.L2, 0, c)
	q3, c := bits.Add64(q.L3, 0, c)
	return Uint256{q0, q1, q2, q3}, st | uint(c)
}

// div512o divides hi:lo by d and returns the low 256 bits of the quotient,
// the remainder and a status: 1 when the quotient overflowed, 2 for d == 0.
// It is long division by limbs shifted so d has its top bit set,
// the remainder carried in as many limbs as d has.
// An overflow first takes the steps of the quotient limbs above 2^256,
// which only reduce hi
func div512o(hi, lo, d Uint256) (q, r Uint256, st uint) {
	// the quotient overflows when hi is not below d
	_, b := bits.Sub64(hi.L0, d.L0, 0)
	_, b = bits.Sub64(hi.L1, d.L1, b)
	_, b = bits.Sub64(hi.L2, d.L2, b)
	_, b = bits.Sub64(hi.L3, d.L3, b)
	st = uint(1 - b)
	// in every case >>1>>t is >>(64-s) that gives 0 at s = 0,
	// and hi.L3>>1>>t < 2^s is below the top limb of the shifted divisor
	switch {
	case d.L3 != 0:
		s := uint(bits.LeadingZeros64(d.L3))
		t := ^s & 63
		d3 := d.L3<<s | d.L2>>1>>t
		d2 := d.L2<<s | d.L1>>1>>t
		d1 := d.L1<<s | d.L0>>1>>t
		d0 := d.L0 << s
		r3 := hi.L3<<s | hi.L2>>1>>t
		r2 := hi.L2<<s | hi.L1>>1>>t
		r1 := hi.L1<<s | hi.L0>>1>>t
		r0 := hi.L0<<s | lo.L3>>1>>t
		u3 := lo.L3<<s | lo.L2>>1>>t
		u2 := lo.L2<<s | lo.L1>>1>>t
		u1 := lo.L1<<s | lo.L0>>1>>t
		u0 := lo.L0 << s
		v := reciprocal3by2m(d3, d2)
		if st != 0 {
			_, r3, r2, r1, r0 = knuthStep4m(hi.L3>>1>>t, r3, r2, r1, r0, d3, d2, d1, d0, v)
		}
		q.L3, r3, r2, r1, r0 = knuthStep4m(r3, r2, r1, r0, u3, d3, d2, d1, d0, v)
		q.L2, r3, r2, r1, r0 = knuthStep4m(r3, r2, r1, r0, u2, d3, d2, d1, d0, v)
		q.L1, r3, r2, r1, r0 = knuthStep4m(r3, r2, r1, r0, u1, d3, d2, d1, d0, v)
		q.L0, r3, r2, r1, r0 = knuthStep4m(r3, r2, r1, r0, u0, d3, d2, d1, d0, v)
		return q, Uint256{r0>>s | r1<<1<<t, r1>>s | r2<<1<<t, r2>>s | r3<<1<<t, r3 >> s}, st
	case d.L2 != 0:
		s := uint(bits.LeadingZeros64(d.L2))
		t := ^s & 63
		d2 := d.L2<<s | d.L1>>1>>t
		d1 := d.L1<<s | d.L0>>1>>t
		d0 := d.L0 << s
		r2 := hi.L2<<s | hi.L1>>1>>t
		r1 := hi.L1<<s | hi.L0>>1>>t
		r0 := hi.L0<<s | lo.L3>>1>>t
		u3 := lo.L3<<s | lo.L2>>1>>t
		u2 := lo.L2<<s | lo.L1>>1>>t
		u1 := lo.L1<<s | lo.L0>>1>>t
		u0 := lo.L0 << s
		v := reciprocal3by2m(d2, d1)
		if st != 0 {
			_, r2, r1, r0 = knuthStep3m(hi.L3>>1>>t, hi.L3<<s|hi.L2>>1>>t, r2, r1, d2, d1, d0, v)
			_, r2, r1, r0 = knuthStep3m(r2, r1, r0, hi.L0<<s|lo.L3>>1>>t, d2, d1, d0, v)
		}
		q.L3, r2, r1, r0 = knuthStep3m(r2, r1, r0, u3, d2, d1, d0, v)
		q.L2, r2, r1, r0 = knuthStep3m(r2, r1, r0, u2, d2, d1, d0, v)
		q.L1, r2, r1, r0 = knuthStep3m(r2, r1, r0, u1, d2, d1, d0, v)
		q.L0, r2, r1, r0 = knuthStep3m(r2, r1, r0, u0, d2, d1, d0, v)
		return q, Uint256{r0>>s | r1<<1<<t, r1>>s | r2<<1<<t, r2 >> s, 0}, st
	case d.L1 != 0:
		s := uint(bits.LeadingZeros64(d.L1))
		t := ^s & 63
		d1 := d.L1<<s | d.L0>>1>>t
		d0 := d.L0 << s
		r1 := hi.L1<<s | hi.L0>>1>>t
		r0 := hi.L0<<s | lo.L3>>1>>t
		u3 := lo.L3<<s | lo.L2>>1>>t
		u2 := lo.L2<<s | lo.L1>>1>>t
		u1 := lo.L1<<s | lo.L0>>1>>t
		u0 := lo.L0 << s
		v := reciprocal3by2m(d1, d0)
		if st != 0 {
			_, r1, r0 = udivrem3by2m(hi.L3>>1>>t, hi.L3<<s|hi.L2>>1>>t, hi.L2<<s|hi.L1>>1>>t, d1, d0, v)
			_, r1, r0 = udivrem3by2m(r1, r0, hi.L1<<s|hi.L0>>1>>t, d1, d0, v)
			_, r1, r0 = udivrem3by2m(r1, r0, hi.L0<<s|lo.L3>>1>>t, d1, d0, v)
		}
		q.L3, r1, r0 = udivrem3by2m(r1, r0, u3, d1, d0, v)
		q.L2, r1, r0 = udivrem3by2m(r1, r0, u2, d1, d0, v)
		q.L1, r1, r0 = udivrem3by2m(r1, r0, u1, d1, d0, v)
		q.L0, r1, r0 = udivrem3by2m(r1, r0, u0, d1, d0, v)
		return q, Uint256{r0>>s | r1<<1<<t, r1 >> s, 0, 0}, st
	}
	// d < 2^64 here, a zero d is the only one to come this far
	if d.L0 == 0 {
		return q, r, 2
	}
	if arch.IsAmd64 {
		// one limb: a hardware division per quotient limb,
		// hi < d leaves hi.L0 below d.L0
		r0 := hi.L0
		if st != 0 {
			_, r0 = bits.Div64(0, hi.L3, d.L0)
			_, r0 = bits.Div64(r0, hi.L2, d.L0)
			_, r0 = bits.Div64(r0, hi.L1, d.L0)
			_, r0 = bits.Div64(r0, hi.L0, d.L0)
		}
		q.L3, r0 = bits.Div64(r0, lo.L3, d.L0)
		q.L2, r0 = bits.Div64(r0, lo.L2, d.L0)
		q.L1, r0 = bits.Div64(r0, lo.L1, d.L0)
		q.L0, r0 = bits.Div64(r0, lo.L0, d.L0)
		return q, Uint256{r0, 0, 0, 0}, st
	}
	s := uint(bits.LeadingZeros64(d.L0)) & 63
	t := ^s & 63
	d0 := d.L0 << s
	r0 := hi.L0<<s | lo.L3>>1>>t
	v := reciprocal2by1(d0)
	// the steps with their second correction as a branch,
	// faster on Graviton in our benchmarks
	if st != 0 {
		_, r0 = div2by1r(hi.L3>>1>>t, hi.L3<<s|hi.L2>>1>>t, d0, v)
		_, r0 = div2by1r(r0, hi.L2<<s|hi.L1>>1>>t, d0, v)
		_, r0 = div2by1r(r0, hi.L1<<s|hi.L0>>1>>t, d0, v)
		_, r0 = div2by1r(r0, hi.L0<<s|lo.L3>>1>>t, d0, v)
	}
	q.L3, r0 = div2by1r(r0, lo.L3<<s|lo.L2>>1>>t, d0, v)
	q.L2, r0 = div2by1r(r0, lo.L2<<s|lo.L1>>1>>t, d0, v)
	q.L1, r0 = div2by1r(r0, lo.L1<<s|lo.L0>>1>>t, d0, v)
	q.L0, r0 = div2by1r(r0, lo.L0<<s, d0, v)
	return q, Uint256{r0 >> s, 0, 0, 0}, st
}

// reciprocal3by2m returns the reciprocal (2^192-1)/<d1, d0> - 2^64.
// d1 must have its top bit set, else the result is wrong or it panics.
// The corrections go by flags: as branches they follow the data and mispredict
func reciprocal3by2m(d1, d0 uint64) uint64 {
	var v uint64
	if arch.IsAmd64 {
		// one hardware 128-by-64 division there
		v, _ = bits.Div64(^d1, math.MaxUint64, d1)
	} else {
		v = reciprocal2by1(d1)
	}
	p, c := bits.Add64(d1*v, d0, 0)
	_, b := bits.Sub64(p, d1, 0)
	// a carry takes one off v, two when p stays at d1 or above
	m := c + c&^b
	v -= m
	p -= d1 * m
	t1, t0 := bits.Mul64(v, d0)
	p, c = bits.Add64(p, t1, 0)
	_, b = bits.Sub64(t0, d0, 0)
	_, b = bits.Sub64(p, d1, b)
	return v - c - c&^b
}

// udivrem3by2m returns <u2, u1, u0> / <d1, d0> and the remainder <r1, r0>,
// v from reciprocal3by2m.
// It needs <u2, u1> < <d1, d0> and the top bit of d1 set,
// other inputs give a wrong result.
// The first correction goes by conditional moves, that branch is a coin toss
func udivrem3by2m(u2, u1, u0, d1, d0, v uint64) (q, r1, r0 uint64) {
	q, k := bits.Mul64(v, u2)
	k, c := bits.Add64(k, u1, 0)
	q, _ = bits.Add64(q, u2, c)
	r1 = u1 - q*d1
	t1, t0 := bits.Mul64(d0, q)
	r0, b := bits.Sub64(u0, t0, 0)
	r1, _ = bits.Sub64(r1, t1, b)
	// r - d with its top limb below k keeps q + 1, else q and r:
	// only moves under the if keep it branchless
	s0, b := bits.Sub64(r0, d0, 0)
	s1, _ := bits.Sub64(r1, d1, b)
	if s1 < k {
		q, r0, r1 = q+1, s0, s1
	}
	// rare
	if r1 >= d1 {
		if r1 > d1 || r0 >= d0 {
			q++
			r0, b = bits.Sub64(r0, d0, 0)
			r1, _ = bits.Sub64(r1, d1, b)
		}
	}
	return q, r1, r0
}

// knuthStep3m is a step of Knuth's algorithm D: <a3, a2, a1, a0> / d
// and the remainder for d = <d2, d1, d0>, the digit as in udivrem3by2m.
// It needs <a3, a2, a1> < d, the top bit of d2 set,
// and v = reciprocal3by2m(d2, d1); other inputs give a wrong result
func knuthStep3m(a3, a2, a1, a0, d2, d1, d0, v uint64) (q, r2, r1, r0 uint64) {
	var b, c uint64
	if a3 == d2 && a2 == d1 {
		// the digit is exactly 2^64-1 here, so a - q*d is a + d - d<<64
		r0, c = bits.Add64(a0, d0, 0)
		r1, c = bits.Add64(a1, d1, c)
		r2, _ = bits.Add64(a2, d2, c)
		r1, b = bits.Sub64(r1, d0, 0)
		r2, _ = bits.Sub64(r2, d1, b)
		return math.MaxUint64, r2, r1, r0
	}
	q, k := bits.Mul64(v, a3)
	k, c = bits.Add64(k, a2, 0)
	q, _ = bits.Add64(q, a3, c)
	r2 = a2 - q*d2
	t1, t0 := bits.Mul64(d1, q)
	r1, b = bits.Sub64(a1, t0, 0)
	r2, _ = bits.Sub64(r2, t1, b)
	s0, b := bits.Sub64(r1, d1, 0)
	s1, _ := bits.Sub64(r2, d2, b)
	if s1 < k {
		q, r1, r2 = q+1, s0, s1
	}
	// rare
	if r2 >= d2 {
		if r2 > d2 || r1 >= d1 {
			q++
			r1, b = bits.Sub64(r1, d1, 0)
			r2, _ = bits.Sub64(r2, d2, b)
		}
	}
	// q*d0 off the low limb
	t1, t0 = bits.Mul64(q, d0)
	r0, b = bits.Sub64(a0, t0, 0)
	r1, b = bits.Sub64(r1, t1, b)
	r2, b = bits.Sub64(r2, 0, b)
	// rare: the digit was one too big, add the divisor back
	if b != 0 {
		q--
		r0, c = bits.Add64(r0, d0, 0)
		r1, c = bits.Add64(r1, d1, c)
		r2, _ = bits.Add64(r2, d2, c)
	}
	return q, r2, r1, r0
}

// knuthStep4m is a step of Knuth's algorithm D: <a4, a3, a2, a1, a0> / d
// and the remainder for d = <d3, d2, d1, d0>, the digit as in udivrem3by2m.
// It needs <a4, a3, a2, a1> < d, the top bit of d3 set,
// and v = reciprocal3by2m(d3, d2); other inputs give a wrong result
func knuthStep4m(a4, a3, a2, a1, a0, d3, d2, d1, d0, v uint64) (q, r3, r2, r1, r0 uint64) {
	var b, c uint64
	if a4 == d3 && a3 == d2 {
		// the digit is exactly 2^64-1 here, so a - q*d is a + d - d<<64
		r0, c = bits.Add64(a0, d0, 0)
		r1, c = bits.Add64(a1, d1, c)
		r2, c = bits.Add64(a2, d2, c)
		r3, _ = bits.Add64(a3, d3, c)
		r1, b = bits.Sub64(r1, d0, 0)
		r2, b = bits.Sub64(r2, d1, b)
		r3, _ = bits.Sub64(r3, d2, b)
		return math.MaxUint64, r3, r2, r1, r0
	}
	q, k := bits.Mul64(v, a4)
	k, c = bits.Add64(k, a3, 0)
	q, _ = bits.Add64(q, a4, c)
	r3 = a3 - q*d3
	t1, t0 := bits.Mul64(d2, q)
	r2, b = bits.Sub64(a2, t0, 0)
	r3, _ = bits.Sub64(r3, t1, b)
	s0, b := bits.Sub64(r2, d2, 0)
	s1, _ := bits.Sub64(r3, d3, b)
	if s1 < k {
		q, r2, r3 = q+1, s0, s1
	}
	// rare
	if r3 >= d3 {
		if r3 > d3 || r2 >= d2 {
			q++
			r2, b = bits.Sub64(r2, d2, 0)
			r3, _ = bits.Sub64(r3, d3, b)
		}
	}
	// q*<d1, d0> off the low limbs
	t1, t0 = bits.Mul64(q, d0)
	t2, t := bits.Mul64(q, d1)
	t1, c = bits.Add64(t1, t, 0)
	t2 += c
	r0, b = bits.Sub64(a0, t0, 0)
	r1, b = bits.Sub64(a1, t1, b)
	r2, b = bits.Sub64(r2, t2, b)
	r3, b = bits.Sub64(r3, 0, b)
	// rare: the digit was one too big, add the divisor back
	if b != 0 {
		q--
		r0, c = bits.Add64(r0, d0, 0)
		r1, c = bits.Add64(r1, d1, c)
		r2, c = bits.Add64(r2, d2, c)
		r3, _ = bits.Add64(r3, d3, c)
	}
	return q, r3, r2, r1, r0
}

// DivToFloat64 returns the float64 nearest to x / y, ties go to the even one.
// It panics for y == 0
func (x Uint256) DivToFloat64(y Uint256) float64 {
	// the top limb of each and the one under it,
	// first the half with it and then the limb in the half.
	// nx and ny count the limbs above them
	xh, xm, xl, nx := x.L3, x.L2, x.L1, uint64(0)
	if x.L3|x.L2 == 0 {
		xh, xm, xl, nx = x.L1, x.L0, 0, 2
	}
	if xh == 0 {
		xh, xm, nx = xm, xl, nx+1
	}
	yh, ym, yl, ny := y.L3, y.L2, y.L1, uint64(0)
	if y.L3|y.L2 == 0 {
		yh, ym, yl, ny = y.L1, y.L0, 0, 2
	}
	if yh == 0 {
		yh, ym, ny = ym, yl, ny+1
	}
	// u and d are the top 64 bits of x and y from their top bits on,
	// u halved when it is not below d.
	// The digit q = u * 2^64 / d has 64 bits,
	// and the rest of x and y moves the quotient by less than 2 either way.
	// So q rounds to the float of the quotient unless a middle between floats,
	// an odd multiple of 2^10, lies as near.
	// Those and y = 0, which has no top bit, are left to divFloatExact
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
			// so the quotient lies above q-2 and below q+3.
			// Away from a middle bit 0 of q changes nothing,
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
			// reciprocal2by1 without its last step is exact or one less.
			// For t = d/2^64 the digit by it is less than 2t + 1 short
			// and the bits under u add less than 1/t,
			// so the quotient lies above q-2 and below q+4:
			// 2t + 1/t is at most 3 for t from 1/2 to 1
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
	return divFloatExact(x, y, 0, "fat.Uint256.DivToFloat64: division by zero")
}

// DivToFloat64x returns DivToFloat64 and whether it is exactly x / y.
// It panics for y == 0
func (x Uint256) DivToFloat64x(y Uint256) (float64, bool) {
	// DivToFloat64 with the digits next to any multiple of 2^10
	// left to divFloatNear: an exact quotient ends at a multiple of 2^11,
	// a zero numerator among them.
	// Only y = 0 has no top bit in d
	xh, xm, xl, nx := x.L3, x.L2, x.L1, uint64(0)
	if x.L3|x.L2 == 0 {
		xh, xm, xl, nx = x.L1, x.L0, 0, 2
	}
	if xh == 0 {
		xh, xm, nx = xm, xl, nx+1
	}
	yh, ym, yl, ny := y.L3, y.L2, y.L1, uint64(0)
	if y.L3|y.L2 == 0 {
		yh, ym, yl, ny = y.L1, y.L0, 0, 2
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
				return float64(int64(q>>1)) * math.Float64frombits(e<<52), false
			}
			return divFloatNear(x, y, q, (q+2)&^(1<<10-1), (nx-ny)<<6+uint64(ly)-uint64(lx)+64-h, e-1, 0)
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
				return float64(q) * math.Float64frombits(e<<52), false
			}
			return divFloatNear(x, y, q, (q+3)&^(1<<10-1), (nx-ny)<<6+uint64(sx)-uint64(sy)+64-h, e, 0)
		}
	}
	panic("fat.Uint256.DivToFloat64x: division by zero")
}

// divFloatExact is DivToFloat64 by the whole of x and y, neg is the sign bit.
// It panics with zero for y == 0.
// The 64-bit digit with its remainder as one more bit rounds the quotients
// that the top bits alone do not
func divFloatExact(x, y Uint256, neg uint64, zero string) float64 {
	if y.IsZero() {
		panic(zero)
	}
	// both shifted to their top bits,
	// x halved when its top limb is not below that of y
	sx, sy := x.LeadingZeros(), y.LeadingZeros()
	x, y = x.Lsh(sx), y.Lsh(sy)
	e := uint64(1023 - 64 + sy - sx)
	a4, a3, a2, a1, a0 := x.L3, x.L2, x.L1, x.L0, uint64(0)
	if x.L3 >= y.L3 {
		a4, a3, a2, a1, a0 = x.L3>>1, x.L3<<63|x.L2>>1, x.L2<<63|x.L1>>1, x.L1<<63|x.L0>>1, x.L0<<63
		e++
	}
	q, r3, r2, r1, r0 := knuthStep4m(a4, a3, a2, a1, a0, y.L3, y.L2, y.L1, y.L0, reciprocal3by2m(y.L3, y.L2))
	r := r3 | r2 | r1 | r0
	return float64(q|(r|-r)>>63) * math.Float64frombits(e<<52|neg)
}

// divFloatNear ends DivToFloat64x for a digit q next to c, a multiple of 2^10.
// x * 2^d against c * y tells the side of c the quotient lies on,
// which rounds it, and whether it is c itself.
// d is the leading zeros of x less those of y plus the 64 or 63 of the digit,
// e the exponent of the digit and neg the sign bit
func divFloatNear(x, y Uint256, q, c, d, e, neg uint64) (float64, bool) {
	if c == 0 {
		// a zero numerator, else c is 2^64 with q all ones under it
		if q == 0 {
			return 0, true
		}
		return float64(q) * math.Float64frombits(e<<52|neg), false
	}
	h0, p0 := bits.Mul64(c, y.L0)
	h1, l1 := bits.Mul64(c, y.L1)
	h2, l2 := bits.Mul64(c, y.L2)
	h3, l3 := bits.Mul64(c, y.L3)
	p1, cy := bits.Add64(h0, l1, 0)
	p2, cy := bits.Add64(h1, l2, cy)
	p3, cy := bits.Add64(h2, l3, cy)
	p4 := h3 + cy
	// only one side is shifted, x up by d or the product up by -d:
	// either stays within 5 limbs
	a4, a3, a2, a1, a0 := uint64(0), x.L3, x.L2, x.L1, x.L0
	b4, b3, b2, b1, b0 := p4, p3, p2, p1, p0
	n := d
	if int64(d) < 0 {
		a4, a3, a2, a1, a0, b4, b3, b2, b1, b0, n = p4, p3, p2, p1, p0, 0, x.L3, x.L2, x.L1, x.L0, -d
	}
	// bits within limbs, then whole limbs by conditional moves, as Uint256.lsh
	s, t := n&63, ^n&63
	a4 = a4<<s | a3>>1>>t
	a3 = a3<<s | a2>>1>>t
	a2 = a2<<s | a1>>1>>t
	a1 = a1<<s | a0>>1>>t
	a0 <<= s
	if n&64 != 0 {
		a4, a3, a2, a1, a0 = a3, a2, a1, a0, 0
	}
	if n&128 != 0 {
		a4, a3, a2, a1, a0 = a2, a1, a0, 0, 0
	}
	if n > 255 {
		a4, a3, a2, a1, a0 = a0, 0, 0, 0, 0
	}
	_, b := bits.Sub64(a0, b0, 0)
	_, b = bits.Sub64(a1, b1, b)
	_, b = bits.Sub64(a2, b2, b)
	_, b = bits.Sub64(a3, b3, b)
	_, b = bits.Sub64(a4, b4, b)
	if (a0^b0)|(a1^b1)|(a2^b2)|(a3^b3)|(a4^b4) == 0 {
		// the quotient is c: a float when its 53 bits hold it,
		// else a tie that the conversion takes to the even one
		return float64(c) * math.Float64frombits(e<<52|neg), bits.Len64(c)-bits.TrailingZeros64(c) <= 53
	}
	// one off c toward the quotient rounds as it does:
	// the shifted side below the other is x below when it is x
	b ^= d >> 63
	return float64(c+1-2*b) * math.Float64frombits(e<<52|neg), false
}

// gcdOdd256 is gcdOdd128 on four limbs while one value is past 128 bits.
// For an odd b and a nonzero a only
func gcdOdd256(a, b Uint256) Uint256 {
	for a.L2|a.L3|b.L2|b.L3 != 0 {
		f0, g0, f1, g1 := gcdSteps31(approx256(a, b))
		a, b = linAbs256(a, b, f0, g0), linAbs256(a, b, f1, g1)
		if a.IsZero() {
			return b
		}
	}
	// a stays nonzero: the loop returns on a zero one
	x := Uint128{a.L0, a.L1}
	r := gcdOdd128(x.Rsh(x.TrailingZeros()), Uint128{b.L0, b.L1})
	return Uint256{r.L0, r.L1, 0, 0}
}

// invOdd256 returns 1/x mod m and true, or false where x and m share a factor,
// as invOdd128 on four limbs: a and b go on in two limbs, then in one, once
// they fit. For an odd m and x < m only
func invOdd256(x, m Uint256) (Uint256, bool) {
	if m.L2|m.L3 == 0 {
		r, ok := invOdd128(Uint128{x.L0, x.L1}, Uint128{m.L0, m.L1})
		return Uint256{r.L0, r.L1, 0, 0}, ok
	}
	mp := montInv(m.L0)
	a, b, u, v := x, m, Uint256{L0: 1}, Uint256{}
	var p0, q0, p1, q1 int64
	odd := false
	for a.L2|a.L3|b.L2|b.L3 != 0 {
		// a zero a leaves b, the GCD, past 128 bits
		if a.IsZero() {
			return Uint256{}, false
		}
		f0, g0, f1, g1 := gcdSteps31(approx256(a, b))
		na, sa := linOne256(a, b, f0, g0)
		nb, sb := linOne256(a, b, f1, g1)
		a, b = na, nb
		f0, g0, f1, g1 = foldSigns(f0, g0, f1, g1, sa, sb)
		if odd {
			f0, g0, f1, g1 = matMul(f0, g0, f1, g1, p0, q0, p1, q1)
			u, v = modLin256(u, v, f0, g0, 62, m, mp), modLin256(u, v, f1, g1, 62, m, mp)
		}
		p0, q0, p1, q1 = f0, g0, f1, g1
		odd = !odd
	}
	e, h := Uint128{a.L0, a.L1}, Uint128{b.L0, b.L1}
	for e.L1|h.L1 != 0 {
		if e.L0|e.L1 == 0 {
			return Uint256{}, false
		}
		f0, g0, f1, g1 := gcdSteps31(approx128(e, h))
		ne, se := linOne128(e, h, f0, g0)
		nh, sh := linOne128(e, h, f1, g1)
		e, h = ne, nh
		f0, g0, f1, g1 = foldSigns(f0, g0, f1, g1, se, sh)
		if odd {
			f0, g0, f1, g1 = matMul(f0, g0, f1, g1, p0, q0, p1, q1)
			u, v = modLin256(u, v, f0, g0, 62, m, mp), modLin256(u, v, f1, g1, 62, m, mp)
		}
		p0, q0, p1, q1 = f0, g0, f1, g1
		odd = !odd
	}
	c, d := e.L0, h.L0
	for c != 0 {
		f0, g0, f1, g1 := gcdSteps31(c, d)
		nc, sc := linOne64(c, d, f0, g0)
		nd, sd := linOne64(c, d, f1, g1)
		c, d = nc, nd
		f0, g0, f1, g1 = foldSigns(f0, g0, f1, g1, sc, sd)
		if odd {
			f0, g0, f1, g1 = matMul(f0, g0, f1, g1, p0, q0, p1, q1)
			u, v = modLin256(u, v, f0, g0, 62, m, mp), modLin256(u, v, f1, g1, 62, m, mp)
		}
		p0, q0, p1, q1 = f0, g0, f1, g1
		odd = !odd
	}
	if d != 1 {
		return Uint256{}, false
	}
	if odd {
		v = modLin256(u, v, p1, q1, 31, m, mp)
	}
	return v, true
}

// approx256 is approx128 on four limbs, the window from bit 31..223
func approx256(a, b Uint256) (uint64, uint64) {
	s := uint(max(223-a.Or(b).LeadingZeros(), 31))
	r, q := s&63, s>>6
	// the limb at s and the one above it by conditional moves on q
	a0, a1, b0, b1 := a.L0, a.L1, b.L0, b.L1
	if q&1 != 0 {
		a0, a1, b0, b1 = a.L1, a.L2, b.L1, b.L2
	}
	a2, a3, b2, b3 := a.L2, a.L3, b.L2, b.L3
	if q&1 != 0 {
		a2, a3, b2, b3 = a.L3, 0, b.L3, 0
	}
	if q&2 != 0 {
		a0, a1, b0, b1 = a2, a3, b2, b3
	}
	// <<1<<(^r&63) is <<(64-r), 0 at r = 0
	wa := a0>>r | a1<<1<<(^r&63)
	wb := b0>>r | b1<<1<<(^r&63)
	return wa<<31 | a.L0&(1<<31-1), wb<<31 | b.L0&(1<<31-1)
}

// mulS256 returns a * f as five limbs of two's complement
func mulS256(a Uint256, f int64) (p0, p1, p2, p3, p4 uint64) {
	h0, p0 := bits.Mul64(a.L0, uint64(f))
	h1, l1 := bits.Mul64(a.L1, uint64(f))
	h2, l2 := bits.Mul64(a.L2, uint64(f))
	h3, l3 := bits.Mul64(a.L3, uint64(f))
	p1, c := bits.Add64(h0, l1, 0)
	p2, c = bits.Add64(h1, l2, c)
	p3, c = bits.Add64(h2, l3, c)
	p4 = h3 + c
	// a negative f is uint64(f) - 2^64
	mf := uint64(f >> 63)
	p1, c = bits.Sub64(p1, a.L0&mf, 0)
	p2, c = bits.Sub64(p2, a.L1&mf, c)
	p3, c = bits.Sub64(p3, a.L2&mf, c)
	p4 -= a.L3&mf + c
	return
}

// linAbs256 is linOne256 without the sign, for the GCD
func linAbs256(a, b Uint256, f, g int64) Uint256 {
	r, _ := linOne256(a, b, f, g)
	return r
}

// linOne256 returns |a*f + b*g| / 2^31 and ^0 for a negative sum. For a sum
// that is a multiple of 2^31 and |f| + |g| <= 2^31 only
func linOne256(a, b Uint256, f, g int64) (Uint256, uint64) {
	p0, p1, p2, p3, p4 := mulS256(a, f)
	q0, q1, q2, q3, q4 := mulS256(b, g)
	p0, c := bits.Add64(p0, q0, 0)
	p1, c = bits.Add64(p1, q1, c)
	p2, c = bits.Add64(p2, q2, c)
	p3, c = bits.Add64(p3, q3, c)
	p4 += q4 + c
	// below 2^287 in size: the shifted value and its sign fit 257 bits
	n := uint64(int64(p4) >> 63)
	r0, r1, r2, r3 := p0>>31|p1<<33, p1>>31|p2<<33, p2>>31|p3<<33, p3>>31|p4<<33
	r0, c = bits.Sub64(r0^n, n, 0)
	r1, c = bits.Sub64(r1^n, n, c)
	r2, c = bits.Sub64(r2^n, n, c)
	r3, _ = bits.Sub64(r3^n, n, c)
	return Uint256{r0, r1, r2, r3}, n
}

// modLin256 is modLin128 on four limbs
func modLin256(u, v Uint256, f, g int64, s uint, m Uint256, mp uint64) Uint256 {
	p0, p1, p2, p3, p4 := mulS256(u, f)
	q0, q1, q2, q3, q4 := mulS256(v, g)
	p0, c := bits.Add64(p0, q0, 0)
	p1, c = bits.Add64(p1, q1, c)
	p2, c = bits.Add64(p2, q2, c)
	p3, c = bits.Add64(p3, q3, c)
	p4 += q4 + c
	// plus j*m, a multiple of 2^s, below (2^(s+1) - 1) * m: over 2^s it lies in -m..2m
	j := p0 * mp & (1<<s - 1)
	h0, l0 := bits.Mul64(j, m.L0)
	h1, l1 := bits.Mul64(j, m.L1)
	h2, l2 := bits.Mul64(j, m.L2)
	h3, l3 := bits.Mul64(j, m.L3)
	l1, c = bits.Add64(l1, h0, 0)
	l2, c = bits.Add64(l2, h1, c)
	l3, c = bits.Add64(l3, h2, c)
	h3 += c
	p0, c = bits.Add64(p0, l0, 0)
	p1, c = bits.Add64(p1, l1, c)
	p2, c = bits.Add64(p2, l2, c)
	p3, c = bits.Add64(p3, l3, c)
	p4 += h3 + c
	r0, r1, r2, r3 := p0>>s|p1<<(64-s), p1>>s|p2<<(64-s), p2>>s|p3<<(64-s), p3>>s|p4<<(64-s)
	r4 := uint64(int64(p4) >> s)
	// m back for a negative r, then m off unless that borrows
	mn := uint64(int64(r4) >> 63)
	r0, c = bits.Add64(r0, m.L0&mn, 0)
	r1, c = bits.Add64(r1, m.L1&mn, c)
	r2, c = bits.Add64(r2, m.L2&mn, c)
	r3, c = bits.Add64(r3, m.L3&mn, c)
	r4 += c
	t0, b := bits.Sub64(r0, m.L0, 0)
	t1, b := bits.Sub64(r1, m.L1, b)
	t2, b := bits.Sub64(r2, m.L2, b)
	t3, b := bits.Sub64(r3, m.L3, b)
	_, b = bits.Sub64(r4, 0, b)
	keep := -b
	return Uint256{t0 ^ (r0^t0)&keep, t1 ^ (r1^t1)&keep, t2 ^ (r2^t2)&keep, t3 ^ (r3^t3)&keep}
}
