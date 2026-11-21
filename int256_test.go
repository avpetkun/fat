package fat

import (
	"math/big"
	"testing"

	"github.com/avpetkun/fat/internal/require"
	. "github.com/avpetkun/fat/internal/testutil"
)

// TestInt256Convert checks conversions to the other types and big.Int
func TestInt256Convert(t *testing.T) {
	t.Parallel()
	require.Big(t, BigMaxInt256, RefI256(MaxInt256()))
	require.Big(t, BigMinInt256, RefI256(MinInt256()))

	Values256(func(x Int256) {
		xb := RefI256(x)
		require.Big(t, big.NewInt(int64(x.L0)), RefI256(NewInt256(int64(x.L0))))
		require.Equal(t, xb.Sign() == 0, x.IsZero())
	})
}
