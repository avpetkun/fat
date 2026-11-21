package fat

import (
	"math/big"
	"testing"

	"github.com/avpetkun/fat/internal/require"
	. "github.com/avpetkun/fat/internal/testutil"
)

// TestInt128Convert checks the conversions to other types and big.Int
func TestInt128Convert(t *testing.T) {
	t.Parallel()
	require.Big(t, BigMaxInt128, RefI128(MaxInt128()))
	require.Big(t, BigMinInt128, RefI128(MinInt128()))

	Values128(func(x Int128) {
		xb := RefI128(x)
		require.Equal(t, xb.Sign() == 0, x.IsZero())
		t := On(t, xb)

		// sign extended above 128 bits, a negative x has no unsigned value
		m := uint64(int64(x.L1) >> 63)
		neg := xb.Sign() < 0
		CheckConvert(t, "Int128.Int64", BigFits(xb, 64, true), false, int64(x.L0), x.Int64)
		CheckConvertUO(t, "Int128.Int64", BigFits(xb, 64, true), int64(x.L0), x.Int64u, x.Int64o)
		CheckConvert(t, "Int128.Uint64", BigFits(xb, 64, false), neg, x.L0, x.Uint64)
		CheckConvertUO(t, "Int128.Uint64", BigFits(xb, 64, false), x.L0, x.Uint64u, x.Uint64o)
		CheckConvert(t, "Int128.Uint128", !neg, true, Uint128(x), x.Uint128)
		CheckConvertUO(t, "Int128.Uint128", !neg, Uint128(x), x.Uint128u, x.Uint128o)
		CheckConvert(t, "Int128.Uint256", !neg, true, Uint256{x.L0, x.L1, 0, 0}, x.Uint256)
		CheckConvertUO(t, "Int128.Uint256", !neg, Uint256{x.L0, x.L1, m, m}, x.Uint256u, x.Uint256o)
		require.Equal(t, Int256{x.L0, x.L1, m, m}, x.Int256())
		require.Equal(t, BigFits(xb, 64, true), x.IsInt64())
		require.Equal(t, BigFits(xb, 64, false), x.IsUint64())
		require.Equal(t, BigFits(xb, 128, false), x.IsUint128())
		require.Equal(t, BigFits(xb, 256, false), x.IsUint256())

		// the constructor sign-extends into every limb
		v64 := int64(x.L0)
		require.Big(t, big.NewInt(v64), RefI128(NewInt128(v64)))
	})
}
