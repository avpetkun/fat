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

		// the constructor sign-extends into every limb
		v64 := int64(x.L0)
		require.Big(t, big.NewInt(v64), RefI128(NewInt128(v64)))
	})
}
