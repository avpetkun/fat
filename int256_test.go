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
		require.Equal(t, xb.Sign() == 0, x.IsZero())
		t := On(t, xb)

		// a negative x has no unsigned value
		neg := xb.Sign() < 0
		CheckConvert(t, "Int256.Int64", BigFits(xb, 64, true), false, int64(x.L0), x.Int64)
		CheckConvertUO(t, "Int256.Int64", BigFits(xb, 64, true), int64(x.L0), x.Int64u, x.Int64o)
		CheckConvert(t, "Int256.Int128", BigFits(xb, 128, true), false, Int128{x.L0, x.L1}, x.Int128)
		CheckConvertUO(t, "Int256.Int128", BigFits(xb, 128, true), Int128{x.L0, x.L1}, x.Int128u, x.Int128o)
		CheckConvert(t, "Int256.Uint64", BigFits(xb, 64, false), neg, x.L0, x.Uint64)
		CheckConvertUO(t, "Int256.Uint64", BigFits(xb, 64, false), x.L0, x.Uint64u, x.Uint64o)
		CheckConvert(t, "Int256.Uint128", BigFits(xb, 128, false), neg, Uint128{x.L0, x.L1}, x.Uint128)
		CheckConvertUO(t, "Int256.Uint128", BigFits(xb, 128, false), Uint128{x.L0, x.L1}, x.Uint128u, x.Uint128o)
		CheckConvert(t, "Int256.Uint256", !neg, true, Uint256(x), x.Uint256)
		CheckConvertUO(t, "Int256.Uint256", !neg, Uint256(x), x.Uint256u, x.Uint256o)
		require.Equal(t, BigFits(xb, 64, true), x.IsInt64())
		require.Equal(t, BigFits(xb, 128, true), x.IsInt128())
		require.Equal(t, BigFits(xb, 64, false), x.IsUint64())
		require.Equal(t, BigFits(xb, 128, false), x.IsUint128())
		require.Equal(t, BigFits(xb, 256, false), x.IsUint256())
		require.Big(t, new(big.Int).Abs(xb), RefU256(x.UnsignedAbs()))

		// big.Int both ways, reusing dirty values
		require.Big(t, xb, x.Big())
		require.Big(t, xb, x.PutBig(big.NewInt(-12345)))
		// no words to reuse, and small capacity words from SetBits
		require.Big(t, xb, x.PutBig(new(big.Int)))
		require.Equal(t, x, MustInt256FromBig(BigClip(xb)))
		require.Equal(t, x, MustInt256FromBig(xb))
		y, err := Int256FromBig(xb)
		require.NoError(t, err)
		require.Equal(t, x, y)

		// out of range big.Int zeroes the value
		for _, b := range []*big.Int{BigSub(xb, Two256), BigAdd(xb, Two256), BigAdd(xb, new(big.Int).Lsh(Two256, 3))} {
			t := On(t, b)
			v, err := Int256FromBig(b)
			require.Equal(t, "fat.Int256FromBig: out of range", err.Error())
			require.True(t, v.IsZero())
			require.Equal(t, "fat.MustInt256FromBig: out of range", PanicValue(func() { MustInt256FromBig(b) }))
		}

		// the constructor and the widening sign-extend into every limb
		v64, v128 := int64(x.L0), Int128{x.L0, x.L1}
		require.Big(t, big.NewInt(v64), RefI256(NewInt256(v64)))
		require.Big(t, RefI256(narrowInt256(x, 128)), RefI256(v128.Int256()))
	})
}

// narrowInt256 cuts y to width bits and extends its sign
func narrowInt256(y Int256, width int) Int256 {
	l := [4]uint64{y.L0, y.L1, y.L2, y.L3}
	ext := uint64(int64(l[width/64-1]) >> 63)
	for i := width / 64; i < 4; i++ {
		l[i] = ext
	}
	return Int256{l[0], l[1], l[2], l[3]}
}

// TestInt256From128 checks that Int128 widens by its sign, also in binary
func TestInt256From128(t *testing.T) {
	t.Parallel()
	Values128(func(x Int128) {
		xb := RefI128(x)
		t := On(t, xb)
		require.Big(t, xb, RefI256(x.Int256()))
	})
}
