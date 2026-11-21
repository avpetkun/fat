package testutil

import (
	"math/big"
	"testing"

	"github.com/avpetkun/fat/internal/require"
)

// RunPanicText checks that every panic names the func or the method
func RunPanicText[T Num[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	big256 := new(big.Int).Lsh(Big1, 256)
	require.Equal(t, "fat.Must"+k.Name+"FromBig: out of range", PanicValue(func() { k.FromBig(big256) }))
	if !k.Signed {
		// a negative big.Int is told apart from a wide one, wide or not itself
		for _, b := range []*big.Int{big.NewInt(-1), new(big.Int).Neg(big256)} {
			require.Equal(On(t, b), "fat.Must"+k.Name+"FromBig: negative value", PanicValue(func() { k.FromBig(b) }))
		}
	}
}
