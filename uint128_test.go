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
		require.Equal(t, xb.Sign() == 0, x.IsZero())

		// the constructor sets every limb
		require.Equal(t, Uint128{x.L0, 0}, NewUint128(x.L0))
	})
}
