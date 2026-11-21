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
		require.Equal(t, xb.Sign() == 0, x.IsZero())

		// the constructor and the widening zero the limbs above
		require.Equal(t, Uint256{x.L0, 0, 0, 0}, NewUint256(x.L0))
	})
}
