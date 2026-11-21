package fat

import (
	"math/big"
	"testing"

	"github.com/avpetkun/fat/internal/require"
	. "github.com/avpetkun/fat/internal/testutil"
)

// TestUint256Convert checks conversions to the other types and big.Int
func TestUint256Convert(t *testing.T) {
	t.Parallel()
	require.Big(t, new(big.Int).Sub(Two256, Big1), RefU256(MaxUint256()))

	Values256(func(x Uint256) {
		xb := RefU256(x)
		t := On(t, xb)

		CheckConvert(t, "Uint256.Uint64", BigFits(xb, 64, false), false, x.L0, x.Uint64)
		CheckConvertUO(t, "Uint256.Uint64", BigFits(xb, 64, false), x.L0, x.Uint64u, x.Uint64o)
		CheckConvert(t, "Uint256.Int64", BigFits(xb, 64, true), false, int64(x.L0), x.Int64)
		CheckConvertUO(t, "Uint256.Int64", BigFits(xb, 64, true), int64(x.L0), x.Int64u, x.Int64o)
		CheckConvert(t, "Uint256.Uint128", BigFits(xb, 128, false), false, Uint128{x.L0, x.L1}, x.Uint128)
		CheckConvertUO(t, "Uint256.Uint128", BigFits(xb, 128, false), Uint128{x.L0, x.L1}, x.Uint128u, x.Uint128o)
		CheckConvert(t, "Uint256.Int128", BigFits(xb, 128, true), false, Int128{x.L0, x.L1}, x.Int128)
		CheckConvertUO(t, "Uint256.Int128", BigFits(xb, 128, true), Int128{x.L0, x.L1}, x.Int128u, x.Int128o)
		CheckConvert(t, "Uint256.Int256", BigFits(xb, 256, true), false, Int256(x), x.Int256)
		CheckConvertUO(t, "Uint256.Int256", BigFits(xb, 256, true), Int256(x), x.Int256u, x.Int256o)
		require.Equal(t, BigFits(xb, 64, false), x.IsUint64())
		require.Equal(t, BigFits(xb, 128, false), x.IsUint128())
		require.Equal(t, BigFits(xb, 64, true), x.IsInt64())
		require.Equal(t, BigFits(xb, 128, true), x.IsInt128())
		require.Equal(t, BigFits(xb, 256, true), x.IsInt256())
		require.Equal(t, xb.Sign() == 0, x.IsZero())

		// the constructor and the widening zero the limbs above
		require.Equal(t, Uint256{x.L0, 0, 0, 0}, NewUint256(x.L0))
		require.Equal(t, Uint256{x.L0, x.L1, 0, 0}, Uint128{x.L0, x.L1}.Uint256())
	})
}

// TestUint256From128 checks that Uint128 widens by zeros, also in binary
func TestUint256From128(t *testing.T) {
	t.Parallel()
	Values128(func(x Uint128) {
		t := On(t, RefU128(x))
		require.Equal(t, Uint256{x.L0, x.L1, 0, 0}, x.Uint256())
	})
}
