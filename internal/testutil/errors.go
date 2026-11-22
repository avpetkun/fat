package testutil

import (
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/avpetkun/fat/internal/require"
)

// RunErrorText checks that a parse error names the func, input and reason
func RunErrorText[T Num[T], PT NumPtr[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
}

// RunPanicText checks that every panic names the func or the method
func RunPanicText[T Num[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	typ := "fat." + k.Name
	// the parse panics carry the error itself
	must := "fat.Must" + k.Name
	require.Equal(t, must+`FromDec: cannot parse "x": invalid digit 'x' at offset 0`, fmt.Sprint(PanicValue(func() { k.MustFromDec("x") })))
	require.Equal(t, must+`FromDec: cannot parse "0x5": 0x in decimal text`, fmt.Sprint(PanicValue(func() { k.MustFromDec("0x5") })))
	require.Equal(t, must+`FromHex: cannot parse "x": invalid hex digit 'x' at offset 0`, fmt.Sprint(PanicValue(func() { k.MustFromHex("x") })))
	require.Equal(t, must+`FromHex: cannot parse "0x": no digits after 0x`, fmt.Sprint(PanicValue(func() { k.MustFromHex("0x") })))
	if !k.Signed {
		require.Equal(t, must+`FromDec: negative value "-5"`, fmt.Sprint(PanicValue(func() { k.MustFromDec("-5") })))
		require.Equal(t, must+`FromHex: negative value "-0x5"`, fmt.Sprint(PanicValue(func() { k.MustFromHex("-0x5") })))
	}
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
