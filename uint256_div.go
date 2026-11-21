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
