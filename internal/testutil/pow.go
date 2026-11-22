package testutil

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/avpetkun/fat/internal/require"
)

// RunExp checks Exp, Expo and Expu with big.Int and the overflow at the edge
func RunExp[T Num[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	typ := "fat." + k.Name
	mod := new(big.Int).Lsh(Big1, uint(k.Bits))
	w := big.NewInt(int64(k.Bits))
	check := func(x, y T) {
		xb, yb := k.Ref(x), k.Ref(y)
		t := On(t, xb, yb)
		if yb.Sign() < 0 {
			require.Equal(t, typ+".Exp: negative exponent", PanicValue(func() { x.Exp(y) }))
			require.Equal(t, typ+".Expo: negative exponent", PanicValue(func() { x.Expo(y) }))
			require.Equal(t, typ+".Expu: negative exponent", PanicValue(func() { x.Expu(y) }))
			return
		}
		want := k.Wrap(new(big.Int).Exp(new(big.Int).Mod(xb, mod), yb, mod))
		require.Big(t, want, k.Ref(x.Expu(y)))
		r, o := x.Expo(y)
		require.Big(t, want, k.Ref(r))
		ax := new(big.Int).Abs(xb)
		var over bool
		switch {
		case ax.Cmp(Big1) <= 0:
		case yb.Cmp(w) >= 0 || int64(ax.BitLen()-1)*yb.Int64() >= int64(k.Bits):
			// |x|^y >= 2^((len-1)*y)
			over = true
		default:
			over = !k.Fits(new(big.Int).Exp(xb, yb, nil))
		}
		require.Same(t, over, o)
		if over {
			require.Equal(t, typ+".Exp: overflow", PanicValue(func() { x.Exp(y) }))
		} else {
			require.Big(t, want, k.Ref(x.Exp(y)))
		}
	}
	r := rand.New(rand.NewSource(5))
	var ys []T
	for _, v := range []uint64{0, 1, 2, 3, 4, 5, 7, 8, 63, 64, 65, 127, 128, 129, 1<<32 - 1, 1 << 32, 1<<32 + 1, 1<<64 - 1} {
		ys = append(ys, k.Small(v))
	}
	// odd x takes the binomial sum from y = 2^32 on 128 bits, 2^64 on 256
	for _, d := range []int64{0, 1, 2, 3} {
		ys = append(ys, k.FromBig(BigAdd(new(big.Int).Lsh(Big1, 64), big.NewInt(d))))
	}
	for _, d := range []int64{-2, -1, 0, 1} {
		ys = append(ys, k.FromBig(big.NewInt(int64(k.Bits)+d)))
	}
	// odd x^(2^(w-2)) is 1, the exponents past it wrap
	quarter := new(big.Int).Rsh(mod, 2)
	for _, d := range []int64{-1, 0, 1} {
		for _, v := range []*big.Int{BigAdd(quarter, big.NewInt(d)), BigAdd(BigMul(quarter, big.NewInt(3)), big.NewInt(d))} {
			if k.Fits(v) {
				ys = append(ys, k.FromBig(v))
			}
		}
	}
	ys = append(ys, k.Max, k.Min)
	k.Values(func(x T) {
		for _, y := range ys {
			check(x, y)
		}
		check(x, k.Small(uint64(r.Intn(k.Bits))))
		// the largest n with |x|^n below 2^w and the exponents around it
		if ax := new(big.Int).Abs(k.Ref(x)); ax.Cmp(Big1) > 0 {
			n := 0
			for p := new(big.Int).Set(ax); p.Cmp(mod) < 0; p.Mul(p, ax) {
				n++
			}
			for d := -1; d <= 2; d++ {
				if n+d >= 0 {
					check(x, k.Small(uint64(n+d)))
				}
			}
		}
	})
	k.Pairs(check)
	// small bases with every exponent up to past the width
	for _, b := range []int64{-10, -7, -3, -2, -1, 0, 1, 2, 3, 7, 10} {
		if b < 0 && !k.Signed {
			continue
		}
		x := k.FromBig(big.NewInt(b))
		for n := range k.Bits + 2 {
			check(x, k.Small(uint64(n)))
		}
	}
	// powers of two at the top of the range, signed ones down to -2^(w-1)
	for e := 1; e < k.Bits; e++ {
		for _, v := range []*big.Int{new(big.Int).Lsh(Big1, uint(e)), new(big.Int).Neg(new(big.Int).Lsh(Big1, uint(e)))} {
			if !k.Fits(v) {
				continue
			}
			x := k.FromBig(v)
			n := (k.Bits - 1) / e
			for d := -1; d <= 2; d++ {
				if n+d >= 0 {
					check(x, k.Small(uint64(n+d)))
				}
			}
		}
	}
}

// RunSqrt checks Sqrt with big.Int next to squares and at every bit length
func RunSqrt[T Num[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	typ := "fat." + k.Name
	check := func(x T) {
		xb := k.Ref(x)
		t := On(t, xb)
		if xb.Sign() < 0 {
			require.Equal(t, typ+".Sqrt: negative value", PanicValue(func() { x.Sqrt() }))
			return
		}
		require.Big(t, new(big.Int).Sqrt(xb), k.Ref(x.Sqrt()))
	}
	checkBig := func(v *big.Int) {
		if v.Sign() >= 0 && k.Fits(v) {
			check(k.FromBig(v))
		}
	}
	// squares, one off them and the largest value with the same root
	around := func(s *big.Int) {
		sq := BigMul(s, s)
		for _, v := range []*big.Int{sq, BigSub(sq, Big1), BigAdd(sq, Big1), BigAdd(sq, s), BigAdd(sq, BigAdd(s, s))} {
			checkBig(v)
		}
	}
	k.Values(check)
	for v := range int64(5000) {
		checkBig(big.NewInt(v))
	}
	for b := 1; b <= k.Bits; b++ {
		for _, v := range []*big.Int{new(big.Int).Lsh(Big1, uint(b-1)), BigSub(new(big.Int).Lsh(Big1, uint(b)), Big1)} {
			checkBig(v)
			around(new(big.Int).Sqrt(v))
		}
	}
	r := rand.New(rand.NewSource(6))
	half := k.Bits / 2
	for range 20000 {
		n := 1 + r.Intn(half)
		s := new(big.Int).Rand(r, new(big.Int).Lsh(Big1, uint(n)))
		around(s)
		around(BigSub(new(big.Int).Lsh(Big1, uint(n)), BigAdd(s, Big1)))
	}
	// a top half s1^2 + 2*s1 gives the largest remainder and next root digit,
	// for every half size from 64 bits up
	for h := 64; h <= half; h *= 2 {
		lo := new(big.Int).Lsh(Big1, uint(h))
		for range 5000 {
			s1 := new(big.Int).Rand(r, new(big.Int).Lsh(Big1, uint(h/2-1)))
			s1.SetBit(s1, h/2-1, 1)
			for _, top := range []*big.Int{BigMul(s1, s1), BigAdd(BigMul(s1, s1), BigAdd(s1, s1)), BigAdd(BigMul(s1, s1), s1)} {
				for _, low := range []*big.Int{new(big.Int), BigSub(lo, Big1), new(big.Int).Rand(r, lo)} {
					v := BigAdd(new(big.Int).Lsh(top, uint(h)), low)
					// and shifted down by an even count
					for sh := 0; sh <= 6; sh += 2 {
						checkBig(new(big.Int).Rsh(v, uint(sh)))
					}
				}
			}
		}
	}
}

// RunLog10 checks Log10 with big.Int at every power of ten and bit length
func RunLog10[T Num[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	typ := "fat." + k.Name
	check := func(x T) {
		xb := k.Ref(x)
		t := On(t, xb)
		if xb.Sign() <= 0 {
			require.Equal(t, typ+".Log10: non-positive value", PanicValue(func() { x.Log10() }))
			return
		}
		require.Equal(t, len(xb.String())-1, x.Log10())
	}
	checkBig := func(v *big.Int) {
		if k.Fits(v) {
			check(k.FromBig(v))
		}
	}
	k.Values(check)
	check(k.Zero)
	for p := big.NewInt(1); k.Fits(p); p = BigMul(p, big.NewInt(10)) {
		for _, d := range []int64{-1, 0, 1} {
			checkBig(BigAdd(p, big.NewInt(d)))
		}
	}
	for b := 1; b <= k.Bits; b++ {
		checkBig(new(big.Int).Lsh(Big1, uint(b-1)))
		checkBig(BigSub(new(big.Int).Lsh(Big1, uint(b)), Big1))
	}
}

// RunExpMod checks ExpMod with big.Int: x^y % m with the sign of x^y,
// moduli of every length and both signs, and the panics
func RunExpMod[T Num[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	typ := "fat." + k.Name
	r := rand.New(rand.NewSource(7))
	rnd := func(n int) *big.Int {
		v := new(big.Int).Rand(r, new(big.Int).Lsh(Big1, uint(n)))
		return v.SetBit(v, n-1, 1)
	}
	var ms []T
	for _, v := range []uint64{1, 2, 3, 4, 7, 10, 1<<32 + 15, 1<<64 - 1} {
		ms = append(ms, k.Small(v))
	}
	for n := 2; n < k.Bits; n += 1 + r.Intn(13) {
		if v := rnd(n); k.Fits(v) {
			ms = append(ms, k.FromBig(v))
		}
	}
	ms = append(ms, k.Max)
	if k.Signed {
		for _, m := range ms[:len(ms):len(ms)] {
			ms = append(ms, k.FromBig(new(big.Int).Neg(k.Ref(m))))
		}
		ms = append(ms, k.Min)
	}
	ys := []T{k.Small(0), k.Small(1), k.Small(2), k.Small(3), k.Small(64), k.Max}
	for range 4 {
		ys = append(ys, k.FromBig(rnd(1+r.Intn(k.Bits-1))))
	}
	check := func(x, y, m T) {
		xb, yb, mb := k.Ref(x), k.Ref(y), k.Ref(m)
		t := On(t, xb, yb, mb)
		am := new(big.Int).Abs(mb)
		want := new(big.Int).Exp(new(big.Int).Abs(xb), yb, am)
		if xb.Sign() < 0 && yb.Bit(0) == 1 {
			want.Neg(want)
		}
		require.Big(t, want, k.Ref(x.ExpMod(y, m)))
	}
	i := 0
	k.Values(func(x T) {
		// every other modulus with a few exponents, not the whole cross product;
		// the next x takes the moduli between
		for j := i & 1; j < len(ms); j += 2 {
			check(x, ys[(i+j)%len(ys)], ms[j])
		}
		for _, y := range ys {
			check(x, y, ms[(i+len(ys))%len(ms)])
		}
		i++
	})
	require.Equal(t, typ+".ExpMod: division by zero", PanicValue(func() { k.Small(2).ExpMod(k.Small(3), k.Zero) }))
	if k.Signed {
		neg := k.FromBig(big.NewInt(-1))
		require.Equal(t, typ+".ExpMod: negative exponent", PanicValue(func() { k.Small(2).ExpMod(neg, k.Small(3)) }))
		require.Equal(t, typ+".ExpMod: negative exponent", PanicValue(func() { k.Small(2).ExpMod(neg, k.Zero) }))
	}
}
