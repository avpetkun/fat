package fat

import (
	"math/big"
	"testing"

	"github.com/avpetkun/fat/internal/require"
	. "github.com/avpetkun/fat/internal/testutil"
)

// TestUint128Convert checks the conversions to other types and big.Int
func TestUint128Convert(t *testing.T) {
	t.Parallel()
	require.Big(t, new(big.Int).Sub(Two128, Big1), RefU128(MaxUint128()))

	Values128(func(x Uint128) {
		xb := RefU128(x)
		t := On(t, xb)

		CheckConvert(t, "Uint128.Uint64", BigFits(xb, 64, false), false, x.L0, x.Uint64)
		CheckConvertUO(t, "Uint128.Uint64", BigFits(xb, 64, false), x.L0, x.Uint64u, x.Uint64o)
		CheckConvert(t, "Uint128.Int64", BigFits(xb, 64, true), false, int64(x.L0), x.Int64)
		CheckConvertUO(t, "Uint128.Int64", BigFits(xb, 64, true), int64(x.L0), x.Int64u, x.Int64o)
		CheckConvert(t, "Uint128.Int128", BigFits(xb, 128, true), false, Int128(x), x.Int128)
		CheckConvertUO(t, "Uint128.Int128", BigFits(xb, 128, true), Int128(x), x.Int128u, x.Int128o)
		require.Equal(t, Uint256{x.L0, x.L1, 0, 0}, x.Uint256())
		require.Equal(t, Int256{x.L0, x.L1, 0, 0}, x.Int256())
		require.Equal(t, BigFits(xb, 64, false), x.IsUint64())
		require.Equal(t, BigFits(xb, 64, true), x.IsInt64())
		require.Equal(t, BigFits(xb, 128, true), x.IsInt128())
		require.Equal(t, xb.Sign() == 0, x.IsZero())

		// big.Int both ways, reusing dirty values
		require.Big(t, xb, x.Big())
		require.Big(t, xb, x.PutBig(big.NewInt(-12345)))
		// no words to reuse, and small capacity words from SetBits
		require.Big(t, xb, x.PutBig(new(big.Int)))
		require.Equal(t, x, MustUint128FromBig(BigClip(xb)))
		require.Big(t, xb, x.PutBig(new(big.Int).Neg(new(big.Int).Lsh(Big1, 200))))
		require.Equal(t, x, MustUint128FromBig(xb))
		y, err := Uint128FromBig(xb)
		require.NoError(t, err)
		require.Equal(t, x, y)

		// negative and oversized big.Int are out of range and zero the value
		for _, b := range []*big.Int{BigSub(xb, Two128), BigAdd(xb, Two128), BigAdd(xb, new(big.Int).Lsh(Two128, 3))} {
			t := On(t, b)
			v, err := Uint128FromBig(b)
			want := "fat.Uint128FromBig: out of range"
			if b.Sign() < 0 {
				want = "fat.Uint128FromBig: negative value"
			}
			require.Equal(t, want, err.Error())
			require.True(t, v.IsZero())
			require.Equal(t, "fat.Must"+want[len("fat."):], PanicValue(func() { MustUint128FromBig(b) }))
		}

		// the constructor sets every limb
		require.Equal(t, Uint128{x.L0, 0}, NewUint128(x.L0))
	})

	// an error with the input quotes it
	require.Equal(t, `fat.Uint128FromHex: negative value "-1"`, ErrNegative{"Uint128FromHex", "-1"}.Error())
}
