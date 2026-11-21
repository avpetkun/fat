package testutil

import (
	"math"
	"math/big"
	"testing"

	"github.com/avpetkun/fat/internal/require"
)

// RunPanicText checks that every panic names the func or the method
func RunPanicText[T Num[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	typ := "fat." + k.Name
	big256 := new(big.Int).Lsh(Big1, 256)
	require.Equal(t, "fat.Must"+k.Name+"FromBig: out of range", PanicValue(func() { k.FromBig(big256) }))
	if !k.Signed {
		// a negative big.Int is told apart from a wide one, wide or not itself
		for _, b := range []*big.Int{big.NewInt(-1), new(big.Int).Neg(big256)} {
			require.Equal(On(t, b), "fat.Must"+k.Name+"FromBig: negative value", PanicValue(func() { k.FromBig(b) }))
		}
	}
	require.Equal(t, typ+".Bit: negative index", PanicValue(func() { k.Zero.Bit(-1) }))
	require.Equal(t, typ+".WithBit: negative index", PanicValue(func() { k.Zero.WithBit(-1, 0) }))
	require.Equal(t, typ+".WithBit: bit not 0 or 1", PanicValue(func() { k.Zero.WithBit(0, 2) }))
	// a negative shift panics as the << operator does
	for _, n := range []int{-1, math.MinInt} {
		require.Equal(On(t, n), typ+".Lsh: negative shift", PanicValue(func() { k.Zero.Lsh(n) }))
		require.Equal(On(t, n), typ+".Rsh: negative shift", PanicValue(func() { k.Zero.Rsh(n) }))
	}
}
