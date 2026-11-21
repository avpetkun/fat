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

// TestUint128Arith checks Add, Sub, Mul and the bitwise ops with big.Int
func TestUint128Arith(t *testing.T) {
	t.Parallel()
	RunArith(t, kU128, []Arith[Uint128]{
		{Name: "Add", Width: 128, Fn: Plain(Uint128.Add), Ref: BigAdd, Panic: "fat.Uint128.Add: overflow"},
		{Name: "Addu", Width: 128, Fn: Plain(Uint128.Addu), Ref: BigAdd},
		{Name: "Addo", Width: 128, Fn: Uint128.Addo, Ref: BigAdd, Over: true},

		{Name: "Sub", Width: 128, Fn: Plain(Uint128.Sub), Ref: BigSub, Panic: "fat.Uint128.Sub: negative value"},
		{Name: "Subu", Width: 128, Fn: Plain(Uint128.Subu), Ref: BigSub},
		{Name: "Subo", Width: 128, Fn: Uint128.Subo, Ref: BigSub, Over: true},
	})
}

// narrowUint128 cuts y to width bits, zeroing the limb above
func narrowUint128(y Uint128, width int) Uint128 {
	if width == 64 {
		y.L1 = 0
	}
	return y
}

// kU128 describes Uint128 to the shared checks of testutil
var kU128 = Kind[Uint128]{
	Name: "Uint128", Bits: 128,
	Ref: RefU128, Wrap: WrapU128, Fits: FitsU128,
	FromBig: MustUint128FromBig,
	Max:     MaxUint128(),
	Values:  Values128, Pairs: Pairs128,
	Narrow: narrowUint128,
	Ops64: Ops64[Uint128]{
		New: NewUint128, Cmp: Uint128.Cmp64,
		Add: Uint128.Add64, Sub: Uint128.Sub64,
		Addo: Uint128.Add64o, Subo: Uint128.Sub64o, Addu: Uint128.Add64u, Subu: Uint128.Sub64u,
	},
}

// TestUint128Cmp checks Cmp, the comparisons and CmpBig with big.Int
func TestUint128Cmp(t *testing.T) { RunCmp(t, kU128) }

// TestUint128Ops64 checks the methods with a uint64 operand with big.Int
func TestUint128Ops64(t *testing.T) { RunOps64(t, kU128) }

// TestUint128Sign checks Sign and IsZero with big.Int
func TestUint128Sign(t *testing.T) { RunSign(t, kU128) }

// TestUint128PanicText checks that every panic names the method
func TestUint128PanicText(t *testing.T) { RunPanicText(t, kU128) }

// TestUint128Constants checks untyped constants as 64-bit operands and shifts
func TestUint128Constants(t *testing.T) {
	t.Parallel()
	x := NewUint128(5)
	require.Equal(t, NewUint128(7), x.Add64(2))
	require.Equal(t, NewUint128(3), x.Sub64(2))
	require.Equal(t, 1, x.Cmp64(4))
	x = x.Add64(1).Sub64(3)
	require.Equal(t, NewUint128(3), x)
}
