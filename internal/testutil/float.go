package testutil

import (
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/avpetkun/fat/internal/require"
)

// RunFloat checks Float64 and FromFloat64 with big.Float on both signs.
// Float64 gets up to 65 top bits at every place, over tails around a tie.
// FromFloat64 gets every exponent from below 1 to past the width, Inf and NaN
func RunFloat[T Num[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	toFloat := func(xb *big.Int) {
		want, _ := new(big.Float).SetInt(xb).Float64()
		got := k.FromBig(xb).Float64()
		require.Same(On(t, xb), math.Float64bits(want), math.Float64bits(got))
	}
	fromFloat := func(v float64) {
		t := On(t, v)
		got, ok := k.FromFloat64(v)
		fits := false
		var want T
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			// toward zero
			ib, _ := new(big.Float).SetFloat64(v).Int(nil)
			if fits = k.Fits(ib); fits {
				want = k.FromBig(ib)
			}
		}
		require.Same(t, fits, ok)
		require.Same(t, want, got)
		if fits {
			require.Same(t, want, k.MustFromFloat64(v))
		} else {
			require.Equal(t, "fat.Must"+k.Name+"FromFloat64: out of range", PanicValue(func() { k.MustFromFloat64(v) }))
		}
	}
	k.Values(func(x T) {
		toFloat(k.Ref(x))
		fromFloat(x.Float64())
	})
	r := rand.New(rand.NewSource(11))
	// n bits: random top bits over a tail of 0, 1, around the half or all ones
	for n := 1; n <= k.Bits; n++ {
		for _, bitsKept := range []int{1, 2, 52, 53, 54, 55, 56, 64, 65} {
			if bitsKept > n {
				continue
			}
			for range 8 {
				top := new(big.Int).SetUint64(r.Uint64()>>(64-min(bitsKept, 64)) | 1<<(min(bitsKept, 64)-1))
				if bitsKept == 65 {
					top.Lsh(top, 1).Or(top, big.NewInt(int64(r.Intn(2))))
				}
				if r.Intn(4) == 0 {
					// all ones round up into the next exponent
					top.Sub(new(big.Int).Lsh(Big1, uint(bitsKept)), Big1)
				}
				rest := uint(n - bitsKept)
				top.Lsh(top, rest)
				tails := []*big.Int{new(big.Int)}
				if rest > 0 {
					half := new(big.Int).Lsh(Big1, rest-1)
					tails = append(tails, big.NewInt(1), half, new(big.Int).Sub(new(big.Int).Lsh(Big1, rest), Big1))
					if rest > 1 {
						tails = append(tails, new(big.Int).Sub(half, Big1), new(big.Int).Add(half, Big1))
					}
				}
				for _, tail := range tails {
					v := new(big.Int).Or(top, tail)
					for _, s := range []*big.Int{v, new(big.Int).Neg(v)} {
						if k.Fits(s) {
							toFloat(s)
						}
					}
				}
			}
		}
	}
	// NaNs, infinities, zeros, fractions and the ends of float64
	for _, v := range []float64{math.NaN(), math.Float64frombits(0xFFF8000000000001), math.Inf(1), math.Inf(-1),
		0, math.Copysign(0, -1), math.SmallestNonzeroFloat64, 0.5, math.Nextafter(1, 0), 1, 1.5, math.Nextafter(2, 0),
		math.MaxFloat64, 1 << 53, 1<<53 + 2, 1<<63 - 512, 1 << 63, 1 << 64} {
		fromFloat(v)
		fromFloat(-v)
	}
	// 2^e for e from -4 to past the width, its neighbours and random mantissas
	for e := -4; e <= k.Bits+4; e++ {
		p := math.Ldexp(1, e)
		vs := []float64{p, math.Nextafter(p, 0), math.Nextafter(p, math.Inf(1)), math.Nextafter(2*p, 0)}
		for range 40 {
			vs = append(vs, math.Ldexp(1+r.Float64(), e))
		}
		for _, v := range vs {
			fromFloat(v)
			fromFloat(-v)
		}
	}
}

// RunDivFloat checks DivToFloat64 and DivToFloat64x with big.Rat on all signs.
// It covers quotients next to a tie, hard divisors, powers of two, small values
func RunDivFloat[T Num[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	check := func(xb, yb *big.Int) {
		if yb.Sign() == 0 || !k.Fits(xb) || !k.Fits(yb) {
			return
		}
		want, exact := new(big.Rat).SetFrac(xb, yb).Float64()
		got := k.FromBig(xb).DivToFloat64(k.FromBig(yb))
		// a zero has no sign in either
		require.Same(On(t, xb, yb), math.Float64bits(want), math.Float64bits(got))
		gotx, isExact := k.FromBig(xb).DivToFloat64x(k.FromBig(yb))
		require.Same(On(t, xb, yb), math.Float64bits(want), math.Float64bits(gotx))
		require.Same(On(t, xb, yb), exact, isExact)
	}
	signs := func(xb, yb *big.Int) {
		check(xb, yb)
		if k.Signed {
			nx, ny := new(big.Int).Neg(xb), new(big.Int).Neg(yb)
			check(nx, yb)
			check(xb, ny)
			check(nx, ny)
		}
	}
	k.Pairs(func(x, y T) { check(k.Ref(x), k.Ref(y)) })
	require.Equal(t, "fat."+k.Name+".DivToFloat64: division by zero", PanicValue(func() { k.Max.DivToFloat64(k.Zero) }))
	require.Equal(t, "fat."+k.Name+".DivToFloat64x: division by zero", PanicValue(func() { k.Max.DivToFloat64x(k.Zero) }))

	r := rand.New(rand.NewSource(12))
	// n bits: random, all ones, or a power of two with a few low bits
	randBig := func(n int) *big.Int {
		top := new(big.Int).Lsh(Big1, uint(n-1))
		switch r.Intn(4) {
		case 0:
			return top.Sub(top.Lsh(top, 1), Big1)
		case 1:
			return top.Or(top, big.NewInt(int64(r.Intn(3))))
		}
		return top.Or(top, new(big.Int).Rand(r, top))
	}
	deltas := []*big.Int{new(big.Int), Big1, big.NewInt(-1), big.NewInt(2), big.NewInt(-2)}
	// w is the bit length of the largest magnitude, check skips pairs past it
	w := k.Bits
	if k.Signed {
		w--
	}
	// x = y * m for m of 53 to 56 bits, shifted up at times:
	// an odd m of 54 bits is a tie, its low bits move it to either side.
	// Then x off by 1, 2 and y, and the same with y = x * m
	for n := 1; n <= w-53; n++ {
		for _, mb := range []int{53, 54, 55, 56} {
			if n+mb > w {
				continue
			}
			for range 6 {
				a := randBig(n)
				m := new(big.Int).SetUint64(r.Uint64()>>(64-mb) | 1<<(mb-1))
				// a tie on an even or odd last place, or next to one
				m.Xor(m, new(big.Int).And(m, big.NewInt(7))).Or(m, big.NewInt(int64(r.Intn(8))))
				p := new(big.Int).Mul(a, m)
				up := new(big.Int).Lsh(p, uint(r.Intn(w-p.BitLen()+1)))
				for _, d := range deltas {
					signs(new(big.Int).Add(up, d), a)
					signs(a, new(big.Int).Add(p, d))
				}
				signs(new(big.Int).Add(up, a), a)
				signs(new(big.Int).Sub(up, a), a)
			}
		}
	}
	// divisors with the hardest reciprocals, alone, shifted and with low bits
	for i, d := range append(Divisors64, SeedDivisors64...) {
		y := new(big.Int).SetUint64(d)
		switch i % 3 {
		case 1:
			y.Lsh(y, uint(r.Intn(w-y.BitLen()+1)))
		case 2:
			n := uint(r.Intn(w - y.BitLen() + 1))
			y.Lsh(y, n).Or(y, new(big.Int).Rand(r, new(big.Int).Lsh(Big1, n)))
		}
		for _, mb := range []int{54, 55} {
			m := new(big.Int).SetUint64(r.Uint64()>>(64-mb) | 1<<(mb-1) | 1)
			p := new(big.Int).Mul(y, m)
			for _, dl := range deltas {
				signs(new(big.Int).Add(p, dl), y)
			}
			signs(m, y)
			signs(y, m)
		}
	}
	// x = y * (m<<10 + j + c/4) for m of 54 bits and |j| <= 8:
	// quotients just off a tie or a float, where an estimated digit decides.
	// y has zeros, ones or random bits under its top limb.
	// The last 64 divisors, a top limb near 2^63 over ones, are the worst case
	near := append(Divisors64, SeedDivisors64...)
	for i := range 64 {
		near = append(near, 1<<63|r.Uint64()>>uint(2+i%32))
	}
	for i, d := range near {
		far := i >= len(near)-64
		for j := -8; j <= 8; j++ {
			y := new(big.Int).SetUint64(d)
			kind := (i + j) & 3
			if far {
				kind = 1
			}
			if kind != 0 {
				n := uint(r.Intn(w - y.BitLen() + 1))
				low := new(big.Int).Lsh(Big1, n)
				y.Lsh(y, n)
				switch kind {
				case 1:
					y.Or(y, low.Sub(low, Big1))
				case 2:
					y.Or(y, low.Rand(r, low))
				}
			}
			m := r.Uint64()>>10 | 1<<53
			if far || j&1 == 0 {
				m |= (1<<54 - 1) &^ (1<<uint(r.Intn(54)) - 1)
			}
			x := new(big.Int).SetUint64(m << 10)
			x.Add(x, big.NewInt(int64(j))).Lsh(x, 2).Or(x, big.NewInt(int64(r.Intn(4))))
			x.Mul(x, y).Rsh(x, uint(max(y.BitLen()+64-w, 0))+2)
			signs(x, y)
			signs(new(big.Int).Add(x, Big1), y)
		}
	}
	// powers of two and their neighbours: top limbs equal or one apart
	for i := 0; i < k.Bits; i++ {
		for _, j := range []int{0, 1, 2, 31, 32, 33, 52, 53, 54, 62, 63, 64, 65, 66, 96, 126, 127, 128, 129, 190, 191, 192, 193, 254, 255} {
			if j >= k.Bits {
				continue
			}
			x, y := new(big.Int).Lsh(Big1, uint(i)), new(big.Int).Lsh(Big1, uint(j))
			for _, dx := range deltas[:3] {
				for _, dy := range deltas[:3] {
					signs(new(big.Int).Add(x, dx), new(big.Int).Add(y, dy))
				}
			}
		}
	}
	// small operands and those around 2^53, exact as a float64 or just past it
	var small []*big.Int
	for v := int64(0); v <= 20; v++ {
		small = append(small, big.NewInt(v), big.NewInt(1<<53-10+v))
	}
	for range 200 {
		small = append(small, randBig(1+r.Intn(min(56, k.Bits-1))))
	}
	for _, x := range small {
		for _, y := range small {
			signs(x, y)
		}
	}
}
