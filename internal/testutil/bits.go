package testutil

import (
	"math"
	"math/big"
	"testing"

	"github.com/avpetkun/fat/internal/require"
)

// RunBits checks BitLen, Not, Lsh, Rsh, Bit and WithBit against math/big.
// Bad arguments of Bit and WithBit must panic
func RunBits[T Num[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	k.Values(func(x T) {
		xb := k.Ref(x)
		t := On(t, xb)

		require.Equal(t, xb.BitLen(), x.BitLen())
		require.Big(t, k.Wrap(new(big.Int).Not(xb)), k.Ref(x.Not()))

		// right shift is arithmetic for the signed types, like big.Int
		for _, n := range k.Shifts {
			require.Big(t, k.Wrap(new(big.Int).Lsh(xb, uint(n))), k.Ref(x.Lsh(n)))
			require.Big(t, new(big.Int).Rsh(xb, uint(n)), k.Ref(x.Rsh(n)))
		}

		for _, i := range k.BitIdx {
			t := On(t, i)
			require.Equal(t, xb.Bit(i), x.Bit(i))
			// a bit past the width drops out, as when big.Int wraps to it
			require.Big(t, k.Wrap(new(big.Int).SetBit(xb, i, 1)), k.Ref(x.WithBit(i, 1)))
			require.Big(t, k.Wrap(new(big.Int).SetBit(xb, i, 0)), k.Ref(x.WithBit(i, 0)))
		}
		// far bits, too wide for a big.Int to hold
		for _, i := range []int{1 << 20, math.MaxInt} {
			t := On(t, i)
			require.Equal(t, xb.Bit(i), x.Bit(i))
			require.Equal(t, x, x.WithBit(i, 1))
		}
	})

	require.Equal(t, "fat."+k.Name+".WithBit: bit not 0 or 1", PanicValue(func() { k.Zero.WithBit(0, 2) }))
	// a negative index panics as in big.Int
	for _, i := range []int{-1, -k.Bits, math.MinInt} {
		require.Equal(On(t, i), "fat."+k.Name+".Bit: negative index", PanicValue(func() { k.Zero.Bit(i) }))
		require.Equal(On(t, i), "fat."+k.Name+".WithBit: negative index", PanicValue(func() { k.Zero.WithBit(i, 0) }))
	}
}

// RunBitArgs checks that a negative bit index panics.
// A huge index is past the end: Bit reads the sign, WithBit sets nothing
func RunBitArgs[T Num[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	k.Values(func(x T) {
		for _, n := range []int{1 << 30, math.MaxInt} {
			t := On(t, x, n)
			want := uint(0)
			if k.Signed {
				want = x.Bit(k.Bits - 1)
			}
			require.Equal(t, want, x.Bit(n))
			require.Equal(t, x, x.WithBit(n, 1))
		}
	})
	for _, n := range []int{-1, math.MinInt} {
		require.Equal(On(t, n), "fat."+k.Name+".Bit: negative index", PanicValue(func() { k.Zero.Bit(n) }))
		require.Equal(On(t, n), "fat."+k.Name+".WithBit: negative index", PanicValue(func() { k.Zero.WithBit(n, 1) }))
	}
}

// hugeShifts are shift counts too big for a big.Int reference
var hugeShifts = []int{1 << 20, 1<<30 + 64, 1<<30 + 128, math.MaxInt - 255, math.MaxInt - 127, math.MaxInt - 63, math.MaxInt}

// RunHugeShifts checks that a huge shift leaves zero, or the sign for Rsh
func RunHugeShifts[T Num[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	k.Values(func(x T) {
		fill := k.Zero
		if k.Signed && k.Ref(x).Sign() < 0 {
			fill = k.Zero.Not()
		}
		for _, n := range hugeShifts {
			t := On(t, x, n)
			require.Equal(t, k.Zero, x.Lsh(n))
			require.Equal(t, fill, x.Rsh(n))
		}
	})
}
