package fat

import (
	"math/big"
	"math/bits"

	"github.com/avpetkun/fat/internal/arch"
)

// --- big.Int --------------------------------------------

// limbWords is the number of big.Word in a limb, 1 or 2
const limbWords = 64 / bits.UintSize

// bigNeg returns all ones for a negative v, else 0.
// w0 must be the low word of |v|; Sign would branch on the sign
func bigNeg(v *big.Int, w0 uint64) uint64 {
	if w0<<1 == 0 {
		// 0 and 1<<63 are their own negation, rare enough for Sign
		return uint64(v.Sign() >> 1)
	}
	// Int64 negates by a select, so it differs from w0 only for v < 0
	d := uint64(v.Int64()) ^ w0
	return -((d | -d) >> 63)
}

// bigIsNeg reports whether v < 0.
// w0 must be the low word of |v|; Sign would branch on the sign
func bigIsNeg(v *big.Int, w0 uint64) bool {
	// Int64 negates by a select, so it differs from w0 only for v < 0
	neg := uint64(v.Int64()) != w0
	if w0<<1 == 0 {
		// 0 and 1<<63 are their own negation, rare enough for Sign
		neg = v.Sign() < 0
	}
	return neg
}

// div2by1 returns the quotient and remainder of <u1, u0> / d by v,
// the reciprocal of d: DIV2BY1 of "Improved division by invariant integers".
// It needs u1 < d and the top bit of d set, other inputs give garbage
func div2by1(u1, u0, d, v uint64) (uint64, uint64) {
	q1, q0 := bits.Mul64(v, u1)
	q0, c := bits.Add64(q0, u0, 0)
	q1, _ = bits.Add64(q1, u1, c)
	// the candidate quotient is q1+1, u0-d is ready before the product
	r := u0 - d - q1*d

	// one if over both values: amd64 keeps an if-else as a branch,
	// and the branch follows no pattern
	q := q1 + 1
	if r > q0 {
		q = q1
		r += d
	}
	q1 = q

	if r >= d {
		q1++
		r -= d
	}

	return q1, r
}

// quo2by1 is div2by1 without the remainder, with the same preconditions
func quo2by1(u1, u0, d, v uint64) uint64 {
	q1, q0 := bits.Mul64(v, u1)
	q0, c := bits.Add64(q0, u0, 0)
	q1, _ = bits.Add64(q1, u1, c)
	r := u0 - d - q1*d
	q := q1 + 1
	if r > q0 {
		q = q1
		r += d
	}
	q1 = q
	if r >= d {
		q1++
	}
	return q1
}

// reciprocal2by1 returns (2^128-1)/d - 2^64; d must have its top bit set.
// A hardware division gives 2^87/d less than 3 short.
// One Newton step with its second order term gets within 1 of the result,
// and the last step of Algorithm 3 of the paper above makes it exact
func reciprocal2by1(d uint64) uint64 {
	// a 24-bit seed and not 32: a divider that takes a cycle per 4 bits
	// of the quotient is 2 cycles shorter
	v := (1<<47 - 1) / (d>>40 + 1)
	// e = (2^87 - v*d) / 2 < 2^64, halved through d/2 rounded up
	d0 := d & 1
	e := (v>>1)&-d0 - v*(d>>1+d0)
	// 2^128/d = 2^41*v + v*E/2^46 + v*E^2/2^133 + less than 0.22
	// for E = 2^87 - v*d < 1.5*2^64 + 2^40
	hi, _ := bits.Mul64(e, e)
	e += hi >> 22
	hi, _ = bits.Mul64(v<<19, e)
	v = v<<41 + hi
	// one more when (v + 2^64 + 1) * d still fits 2^128
	hi, lo := bits.Mul64(v, d)
	_, b := bits.Sub64(^d, lo, 0)
	v, _ = bits.Sub64(v, hi+d, b)
	return v
}

// --- roots ----------------------------------------------

// sqrtSeed holds ceil(sqrt((t+1) * 2^55)) / 2^17 rounded up by the top 9 bits t
// of a limb >= 2^62: a root from above within 2^-7.9.
// 512 entries keep the index in range without a check
var sqrtSeed = func() (s [512]uint16) {
	for t := 128; t < 512; t++ {
		// ceil(sqrt((t+1) * 2^21)) by integer Newton steps from above
		v := uint64(t+1) << 21
		z := v
		for {
			n := (z + v/z) >> 1
			if n >= z {
				break
			}
			z = n
		}
		if z*z < v {
			z++
		}
		s[t] = uint16(z)
	}
	return
}()

// sqrtInvSeed holds 2^15 / sqrt((t+0.5) / 512) rounded by the top 9 bits t
// of a limb >= 2^62: the inverse root within 2^-9, for roots with no division.
// The entries below 128 stay zero and give the root 0 for a limb 0
var sqrtInvSeed = func() (s [512]uint16) {
	for t := 128; t < 512; t++ {
		// floor(sqrt(2^42 / (2t+1))) is twice the seed, by Newton steps
		v := uint64(1<<42) / uint64(2*t+1)
		z := v
		for {
			n := (z + v/z) >> 1
			if n >= z {
				break
			}
			z = n
		}
		s[t] = uint16((z + 1) >> 1)
	}
	return
}()

// sqrt64n returns the square root of a rounded down.
// a must be at least 2^62, a smaller a gives garbage
func sqrt64n(a uint64) uint64 {
	if !arch.IsAmd64 {
		// no division, amd64 keeps the division-based root below: on Graviton
		// UDIV is slow and not pipelined, so this was faster in our benchmarks.
		// y = 2^63/sqrt(a) by one Newton step on the inverse root from a seed
		// within 2^-9: y0*(3 - a*y0^2)/2 is within 2^-17.4.
		// 2^-29 is taken off to stay below.
		// Then s = a*y and one step on the root, s + (a - s^2)*y/2,
		// gives the root or one less
		y0 := uint64(sqrtInvSeed[a>>55])
		a32 := a >> 32
		y, _ := bits.Mul64(y0<<17, 3<<62-1<<34-a32*(y0*y0))
		s := a32 * y >> 31
		d, _ := bits.Mul64(a-s*s, y)
		s += d
		// one more when the remainder is past 2s, by a conditional increment
		if a-s*s > 2*s {
			s++
		}
		return s
	}
	z := uint64(sqrtSeed[a>>55]) << 17
	// Newton steps from above: 2^-7.9 off the root, then 2^-16.8,
	// then less than 1/8 past it
	z = (z + a/z) >> 1
	z = (z + a/z) >> 1
	// z is the root or one more, z*z-1 wraps to 2^64-1 for z = 2^32
	_, b := bits.Sub64(a, z*z-1, 1)
	return z - b
}

// sqrtrem64n is the amd64 sqrt64n with the remainder a - s^2 <= 2s.
// a must be at least 2^62
func sqrtrem64n(a uint64) (s, r uint64) {
	z := uint64(sqrtSeed[a>>55]) << 17
	z = (z + a/z) >> 1
	z = (z + a/z) >> 1
	// a - z^2 is exact as int64 for z = 2^32 too, negative for a root too big
	r = a - z*z
	m := uint64(int64(r) >> 63)
	return z + m, r + (2*z-1)&m
}

// sqrt64 returns the square root of x rounded down
func sqrt64(x uint64) uint64 {
	if x == 0 {
		return 0
	}
	k := uint(bits.LeadingZeros64(x)) &^ 1
	return sqrt64n(x<<k) >> (k >> 1)
}

// sqrt128n returns the square root of <a1, a0> rounded down.
// a1 must be at least 2^62, a smaller a1 gives garbage.
// The root s1 of a1 with its remainder r1, then 32 more bits by a division
func sqrt128n(a1, a0 uint64) uint64 {
	s1, r1 := sqrtrem64n(a1)
	// q = (r1*2^32 + a0>>32) / (2*s1) as (r1*2^31 + a0>>33) / s1,
	// 2^32 at most and then one less
	n := r1<<31 | a0>>33
	q := n / s1
	q -= q >> 32
	u := n - q*s1
	// the remainder (2u + bit 32 of a0)*2^32 + low 32 bits of a0 - q^2
	// has u>>31 above 2^64, and is negative for a root one too big
	_, b := bits.Sub64(u<<33|a0&(1<<33-1), q*q, 0)
	return (s1<<32 | q) - b&((u>>31-1)>>63)
}

// sqrtrem128n is sqrt128n with the remainder <rh, rl> <= 2s.
// a1 must be at least 2^62
func sqrtrem128n(a1, a0 uint64) (s, rh, rl uint64) {
	if !arch.IsAmd64 {
		// the root of a1 as in sqrt64n, written out as its remainder is needed.
		// amd64 keeps the division-based root below.
		// The division stays: a quotient from y*(2 - s1*y) is as long a chain.
		// s1 has bit 31 set, so the division needs no zero check;
		// nothing here calls, so there is no frame
		y0 := uint64(sqrtInvSeed[a1>>55])
		a32 := a1 >> 32
		y, _ := bits.Mul64(y0<<17, 3<<62-1<<34-a32*(y0*y0))
		s1 := a32 * y >> 31
		d, _ := bits.Mul64(a1-s1*s1, y)
		s1 += d
		r1 := a1 - s1*s1
		if s2, r2 := s1+1, r1-2*s1-1; int64(r2) >= 0 {
			s1, r1 = s2, r2
		}
		n := r1<<31 | a0>>33
		q := n / (s1 | 1<<31)
		q -= q >> 32
		u := n - q*s1
		s = s1<<32 | q
		var b, c uint64
		rl, b = bits.Sub64(u<<33|a0&(1<<33-1), q*q, 0)
		rh = u>>31 - b
		m := uint64(int64(rh) >> 63)
		rl, c = bits.Add64(rl, s&m, 0)
		rh += c
		s += m
		rl, c = bits.Add64(rl, s&m, 0)
		rh += c
		return
	}
	s1, r1 := sqrtrem64n(a1)
	n := r1<<31 | a0>>33
	q := n / s1
	q -= q >> 32
	u := n - q*s1
	s = s1<<32 | q
	rl, b := bits.Sub64(u<<33|a0&(1<<33-1), q*q, 0)
	rh = u>>31 - b
	// one less root for a negative remainder, r + 2s - 1 as r + s + (s-1).
	// A mask, not a branch: a quarter of the roots are one too big at random
	m := uint64(int64(rh) >> 63)
	rl, c := bits.Add64(rl, s&m, 0)
	rh += c
	s += m
	rl, c = bits.Add64(rl, s&m, 0)
	rh += c
	return
}

// --- log10 ----------------------------------------------

// pow10x128 holds 10^i for every i that fits a Uint128
var pow10x128 = func() (p [39]Uint128) {
	p[0] = Uint128{1, 0}
	for i := 1; i < len(p); i++ {
		p[i] = p[i-1].Mul64u(10)
	}
	return
}()

// pow10x256 holds 10^i for every i that fits a Uint256
var pow10x256 = func() (p [78]Uint256) {
	p[0] = Uint256{1, 0, 0, 0}
	for i := 1; i < len(p); i++ {
		p[i] = p[i-1].Mul64u(10)
	}
	return
}()

// --- Montgomery -----------------------------------------

// nonzero returns 1 for v != 0, else 0, without a branch
func nonzero(v uint64) uint {
	return uint((v | -v) >> 63)
}

// madd returns a*b + c + d as hi, lo, which never overflows 128 bits
func madd(a, b, c, d uint64) (hi, lo uint64) {
	hi, lo = bits.Mul64(a, b)
	var k uint64
	lo, k = bits.Add64(lo, c, 0)
	hi += k
	lo, k = bits.Add64(lo, d, 0)
	return hi + k, lo
}

// montInv returns -1/m0 mod 2^64 for an odd m0: each Newton step doubles
// the low bits that are right, 3 of them in m0 itself
func montInv(m0 uint64) uint64 {
	v := m0
	v *= 2 - m0*v
	v *= 2 - m0*v
	v *= 2 - m0*v
	v *= 2 - m0*v
	v *= 2 - m0*v
	return -v
}

// montMul64 returns a * b / 2^64 % m by Montgomery's product, montMul on
// one limb. m must be odd and a, b below it, mp = -1/m mod 2^64, else it
// is wrong
func montMul64(a, b, m, mp uint64) uint64 {
	h, l := bits.Mul64(a, b)
	hu, _ := bits.Mul64(l*mp, m)
	// the low half of l + u*m is 0, with a carry for l != 0
	t, c := bits.Add64(h, hu, uint64(nonzero(l)))
	// t + c*2^64 < 2m
	r, br := bits.Sub64(t, m, 0)
	keep := -(br &^ c)
	return r&^keep | t&keep
}

// expModMont64 is expModMont for an odd m in 2..2^64-1, y > 0 and b < m,
// by Montgomery products modulo R = 2^64
func expModMont64(b uint64, y Uint256, m uint64) uint64 {
	mp := montInv(m)
	// R % m, then R^2 % m to take b to R*b % m
	r := -m % m
	h, l := bits.Mul64(r, r)
	r2 := bits.Rem64(h, l, m)
	var odd [8]uint64
	odd[0] = montMul64(b, r2, m, mp)
	n, w := y.BitLen(), expWindow(y.BitLen())
	if w > 1 {
		b2 := montMul64(odd[0], odd[0], m, mp)
		for k := 1; k < 1<<(w-1); k++ {
			odd[k] = montMul64(odd[k-1], b2, m, mp)
		}
	}
	// the top bit of y is set, so the first window starts r
	started := false
	for i := n - 1; i >= 0; {
		if y.Bit(i) == 0 {
			r = montMul64(r, r, m, mp)
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
				r = montMul64(r, r, m, mp)
			}
		}
		if started {
			r = montMul64(r, odd[v>>1], m, mp)
		} else {
			r, started = odd[v>>1], true
		}
		i = j - 1
	}
	// out of the Montgomery form
	return montMul64(r, 1, m, mp)
}
