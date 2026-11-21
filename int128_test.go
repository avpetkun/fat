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

		// big.Int both ways, reusing dirty values
		require.Big(t, xb, x.Big())
		require.Big(t, xb, x.PutBig(big.NewInt(-12345)))
		// no words to reuse, and small capacity words from SetBits
		require.Big(t, xb, x.PutBig(new(big.Int)))
		require.Equal(t, x, MustInt128FromBig(BigClip(xb)))
		require.Equal(t, x, MustInt128FromBig(xb))
		y, err := Int128FromBig(xb)
		require.NoError(t, err)
		require.Equal(t, x, y)

		// out of range big.Int zeroes the value
		for _, b := range []*big.Int{BigSub(xb, Two128), BigAdd(xb, Two128), BigAdd(xb, new(big.Int).Lsh(Two128, 3))} {
			t := On(t, b)
			v, err := Int128FromBig(b)
			require.Equal(t, "fat.Int128FromBig: out of range", err.Error())
			require.True(t, v.IsZero())
			require.Equal(t, "fat.MustInt128FromBig: out of range", PanicValue(func() { MustInt128FromBig(b) }))
		}

		// the constructor sign-extends into every limb
		v64 := int64(x.L0)
		require.Big(t, big.NewInt(v64), RefI128(NewInt128(v64)))
	})
}

// TestInt128Arith checks Add, Sub, Mul and the bitwise ops with big.Int
func TestInt128Arith(t *testing.T) {
	t.Parallel()
	RunArith(t, kI128, []Arith[Int128]{
		{Name: "Add", Width: 128, Fn: Plain(Int128.Add), Ref: BigAdd, Panic: "fat.Int128.Add: overflow"},
		{Name: "Addu", Width: 128, Fn: Plain(Int128.Addu), Ref: BigAdd},
		{Name: "Addo", Width: 128, Fn: Int128.Addo, Ref: BigAdd, Over: true},

		{Name: "Sub", Width: 128, Fn: Plain(Int128.Sub), Ref: BigSub, Panic: "fat.Int128.Sub: overflow"},
		{Name: "Subu", Width: 128, Fn: Plain(Int128.Subu), Ref: BigSub},
		{Name: "Subo", Width: 128, Fn: Int128.Subo, Ref: BigSub, Over: true},
	})
}

// kI128 describes Int128 to the shared checks of testutil
var kI128 = Kind[Int128]{
	Name: "Int128", Bits: 128, Signed: true,
	Ref: RefI128, Wrap: WrapI128, Fits: FitsI128,
	FromBig: MustInt128FromBig,
	Max:     MaxInt128(), Min: MinInt128(),
	Values: Values128, Pairs: Pairs128,
	Narrow: narrowInt128,
	Ops64: Ops64[Int128]{
		New: func(v uint64) Int128 { return NewInt128(int64(v)) }, Cmp: Signed64(Int128.Cmp64),
		Add: Signed64(Int128.Add64), Sub: Signed64(Int128.Sub64),
		Addo: Signed64Pair(Int128.Add64o), Subo: Signed64Pair(Int128.Sub64o), Addu: Signed64(Int128.Add64u), Subu: Signed64(Int128.Sub64u),
	},
}

// TestInt128Cmp checks Cmp, the comparisons and CmpBig with big.Int
func TestInt128Cmp(t *testing.T) { RunCmp(t, kI128) }

// TestInt128Ops64 checks the methods with an int64 operand with big.Int
func TestInt128Ops64(t *testing.T) { RunOps64(t, kI128) }

// TestInt128Sign checks Sign and IsZero with big.Int
func TestInt128Sign(t *testing.T) { RunSign(t, kI128) }

// TestInt128Signed checks IsNegative, IsPositive, Neg, Abs and UnsignedAbs with big.Int
func TestInt128Signed(t *testing.T) {
	RunSigned(t, kI128, func(x Int128) *big.Int { return RefU128(x.UnsignedAbs()) })
}

// TestInt128PanicText checks that every panic names the method
func TestInt128PanicText(t *testing.T) { RunPanicText(t, kI128) }

// TestInt128Constants checks untyped constants as 64-bit operands and shifts
func TestInt128Constants(t *testing.T) {
	t.Parallel()
	x := NewInt128(-5)
	require.Equal(t, NewInt128(-3), x.Add64(2))
	require.Equal(t, NewInt128(-7), x.Sub64(2))
	require.Equal(t, -1, x.Cmp64(-4))
	x = x.Add64(1).Sub64(-3)
	require.Equal(t, NewInt128(-1), x)
}

// narrowInt128 cuts y to width bits, sign-extending into the limb above
func narrowInt128(y Int128, width int) Int128 {
	if width == 64 {
		y.L1 = uint64(int64(y.L0) >> 63)
	}
	return y
}
