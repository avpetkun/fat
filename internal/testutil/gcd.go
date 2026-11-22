package testutil

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/avpetkun/fat/internal/require"
)

// randGCD returns a random value of k of a random length, of either sign for
// a signed k, with a shared factor of a random length now and then
func randGCD[T Num[T]](r *rand.Rand, k Kind[T]) *big.Int {
	n := k.Bits
	if k.Signed {
		n--
	}
	l := RandBits(r, 1+r.Intn(n))
	v := new(big.Int)
	for i := len(l) - 1; i >= 0; i-- {
		v.Lsh(v, 64).Or(v, new(big.Int).SetUint64(l[i]))
	}
	if k.Signed && r.Intn(2) == 0 {
		v.Neg(v)
	}
	return v
}

// RunGCD checks GCD against math/big on the pairs, on random pairs of mixed
// lengths with and without a shared factor, and with zero; gcd gives GCD(x, y)
// as a big.Int, its result is unsigned on the signed types
func RunGCD[T Num[T]](t *testing.T, k Kind[T], gcd func(x, y T) *big.Int) {
	t.Parallel()
	check := func(x, y T) {
		xb, yb := k.Ref(x), k.Ref(y)
		require.Big(On(t, xb, yb), new(big.Int).GCD(nil, nil, xb, yb), gcd(x, y))
	}
	k.Pairs(func(x, y T) {
		check(x, y)
		check(x, k.Zero)
		check(k.Zero, x)
		check(x, x)
	})
	r := rand.New(rand.NewSource(int64(k.Bits)))
	for range 20 * RandomCount {
		x, y := randGCD(r, k), randGCD(r, k)
		// a shared factor of up to half the width keeps long cofactors
		if f := randGCD(r, k); r.Intn(2) == 0 {
			f.Rsh(f, uint(r.Intn(k.Bits/2)+k.Bits/2))
			x.Mul(x, f)
			y.Mul(y, f)
		}
		if k.Fits(x) && k.Fits(y) {
			check(k.FromBig(x), k.FromBig(y))
		}
	}
	// values whose difference keeps many factors of 2 for several rounds
	for _, s := range []int{31, 32, 62, 63, 64, 65, 95, 96, 127} {
		if s+1 >= k.Bits {
			continue
		}
		p := new(big.Int).Lsh(Big1, uint(s))
		for _, x := range []*big.Int{BigAdd(p, Big1), BigSub(p, Big1), BigAdd(BigMul(p, big.NewInt(3)), Big1)} {
			for _, y := range []*big.Int{Big1, big.NewInt(3), BigSub(p, big.NewInt(3)), BigAdd(p, BigAdd(p, Big1))} {
				if k.Fits(x) && k.Fits(y) {
					check(k.FromBig(x), k.FromBig(y))
				}
			}
		}
	}
}

// RunModInverse checks ModInverse against math/big on the pairs and on random
// pairs of mixed lengths, odd and even moduli among them, and its panic for
// m == 0. For an unsigned k only
func RunModInverse[T Num[T]](t *testing.T, k Kind[T], inv func(x, m T) (T, bool)) {
	t.Parallel()
	check := func(x, m T) {
		xb, mb := k.Ref(x), k.Ref(m)
		t := On(t, xb, mb)
		if mb.Sign() == 0 {
			require.Equal(t, "fat."+k.Name+".ModInverse: division by zero", PanicValue(func() { inv(x, m) }))
			return
		}
		y, ok := inv(x, m)
		// everything is 1/x mod 1, 0 among them
		if mb.Cmp(Big1) == 0 {
			require.True(t, ok)
			require.True(t, y.IsZero())
			return
		}
		want := new(big.Int).ModInverse(xb, mb)
		require.Equal(t, want != nil, ok)
		if want == nil {
			require.True(t, y.IsZero())
			return
		}
		require.Big(t, want, k.Ref(y))
	}
	k.Pairs(check)
	r := rand.New(rand.NewSource(int64(k.Bits) + 1))
	for range 20 * RandomCount {
		x, m := randGCD(r, k), randGCD(r, k)
		check(k.FromBig(x), k.FromBig(m))
		// x past m, and m even with an odd x
		check(k.FromBig(x), k.FromBig(m.Rsh(m, uint(r.Intn(k.Bits)))))
		if m.Bit(0) == 1 {
			m.Xor(m, Big1)
		}
		check(k.FromBig(x.SetBit(x, 0, 1)), k.FromBig(m))
	}
	// 1 is its own inverse modulo an even m, and 0 has none
	for _, m := range []T{k.Small(2), k.Small(6), k.Max.Rsh(1).Lsh(1), k.Max} {
		check(k.Small(1), m)
		check(k.Zero, m)
		check(m, m)
	}
}
