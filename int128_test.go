package fat

import (
	"math/big"
	"sync"
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

		{Name: "Mul", Width: 128, Fn: Plain(Int128.Mul), Ref: BigMul, Panic: "fat.Int128.Mul: overflow"},
		{Name: "Mulu", Width: 128, Fn: Plain(Int128.Mulu), Ref: BigMul},
		{Name: "Mulo", Width: 128, Fn: Int128.Mulo, Ref: BigMul, Over: true},

		{Name: "And", Width: 128, Fn: Plain(Int128.And), Ref: BigAnd},
		{Name: "Or", Width: 128, Fn: Plain(Int128.Or), Ref: BigOr},
		{Name: "Xor", Width: 128, Fn: Plain(Int128.Xor), Ref: BigXor},
		{Name: "AndNot", Width: 128, Fn: Plain(Int128.AndNot), Ref: BigAndNot},
	})
}

// TestInt128Mule checks the full 256-bit product with big.Int
func TestInt128Mule(t *testing.T) {
	t.Parallel()
	Pairs128(func(x, y Int128) {
		xb, yb := RefI128(x), RefI128(y)
		require.Big(On(t, xb, yb), BigMul(xb, yb), RefI256(x.Mule(y)))
	})
}

// BenchmarkInt128 measures the methods on 2^18 values, random length and sign.
// Add, Mul and DivMod/fit take pairs whose results fit
func BenchmarkInt128(b *testing.B) {
	in := benchI128()
	b.Run("op=Cmp", func(b *testing.B) {
		x, y := in.i128, in.j128
		for i := range b.N {
			SinkInt = x[i&BenchMask].Cmp(y[i&BenchMask])
		}
	})
	b.Run("op=LessThan", func(b *testing.B) {
		x, y := in.i128, in.j128
		for i := range b.N {
			SinkBool = x[i&BenchMask].LessThan(y[i&BenchMask])
		}
	})
	b.Run("op=Max", func(b *testing.B) {
		x, y := in.i128, in.j128
		for i := range b.N {
			sinkI128 = x[i&BenchMask].Max(y[i&BenchMask])
		}
	})
	b.Run("op=Min", func(b *testing.B) {
		x, y := in.i128, in.j128
		for i := range b.N {
			sinkI128 = x[i&BenchMask].Min(y[i&BenchMask])
		}
	})
	b.Run("op=Cmp64", func(b *testing.B) {
		x, y := in.i128, in.i64
		for i := range b.N {
			SinkInt = x[i&BenchMask].Cmp64(y[i&BenchMask])
		}
	})
	b.Run("op=Add", func(b *testing.B) {
		x, y := in.fx, in.fy
		for i := range b.N {
			sinkI128 = x[i&BenchMask].Add(y[i&BenchMask])
		}
	})
	b.Run("op=Addo", func(b *testing.B) {
		x, y := in.i128, in.j128
		for i := range b.N {
			sinkI128, SinkBool = x[i&BenchMask].Addo(y[i&BenchMask])
		}
	})
	b.Run("op=Addu", func(b *testing.B) {
		x, y := in.i128, in.j128
		for i := range b.N {
			sinkI128 = x[i&BenchMask].Addu(y[i&BenchMask])
		}
	})
	b.Run("op=Subo", func(b *testing.B) {
		x, y := in.i128, in.j128
		for i := range b.N {
			sinkI128, SinkBool = x[i&BenchMask].Subo(y[i&BenchMask])
		}
	})
	b.Run("op=Mul", func(b *testing.B) {
		x, y := in.fx, in.fy
		for i := range b.N {
			sinkI128 = x[i&BenchMask].Mul(y[i&BenchMask])
		}
	})
	b.Run("op=Mulo", func(b *testing.B) {
		x, y := in.i128, in.j128
		for i := range b.N {
			sinkI128, SinkBool = x[i&BenchMask].Mulo(y[i&BenchMask])
		}
	})
	b.Run("op=Mulu", func(b *testing.B) {
		x, y := in.i128, in.j128
		for i := range b.N {
			sinkI128 = x[i&BenchMask].Mulu(y[i&BenchMask])
		}
	})
	b.Run("op=Mule", func(b *testing.B) {
		x, y := in.i128, in.j128
		for i := range b.N {
			sinkI256 = x[i&BenchMask].Mule(y[i&BenchMask])
		}
	})
	b.Run("op=Mul64", func(b *testing.B) {
		x, y := in.fx64, in.i64
		for i := range b.N {
			sinkI128 = x[i&BenchMask].Mul64(y[i&BenchMask])
		}
	})
	b.Run("op=Mul64o", func(b *testing.B) {
		x, y := in.i128, in.i64
		for i := range b.N {
			sinkI128, SinkBool = x[i&BenchMask].Mul64o(y[i&BenchMask])
		}
	})
	b.Run("op=Mul64u", func(b *testing.B) {
		x, y := in.i128, in.i64
		for i := range b.N {
			sinkI128 = x[i&BenchMask].Mul64u(y[i&BenchMask])
		}
	})
	b.Run("op=Rsh", func(b *testing.B) {
		x, y := in.i128, in.u64
		for i := range b.N {
			sinkI128 = x[i&BenchMask].Rsh(int(y[i&BenchMask] & 127))
		}
	})
	b.Run("op=Lsh", func(b *testing.B) {
		x, y := in.i128, in.u64
		for i := range b.N {
			sinkI128 = x[i&BenchMask].Lsh(int(y[i&BenchMask] & 127))
		}
	})
	b.Run("op=Sign", func(b *testing.B) {
		x := in.i128
		for i := range b.N {
			SinkInt = x[i&BenchMask].Sign()
		}
	})
	b.Run("op=IsPositive", func(b *testing.B) {
		x := in.i128
		for i := range b.N {
			SinkBool = x[i&BenchMask].IsPositive()
		}
	})
	b.Run("op=Abs", func(b *testing.B) {
		x := in.i128
		for i := range b.N {
			sinkI128 = x[i&BenchMask].Abs()
		}
	})
	b.Run("op=UnsignedAbs", func(b *testing.B) {
		x := in.i128
		for i := range b.N {
			sinkU128 = x[i&BenchMask].UnsignedAbs()
		}
	})
	b.Run("op=BitLen", func(b *testing.B) {
		x := in.i128
		for i := range b.N {
			SinkInt = x[i&BenchMask].BitLen()
		}
	})
	b.Run("op=Add64o", func(b *testing.B) {
		x, y := in.i128, in.i64
		for i := range b.N {
			sinkI128, SinkBool = x[i&BenchMask].Add64o(y[i&BenchMask])
		}
	})
	b.Run("op=Sub64o", func(b *testing.B) {
		x, y := in.i128, in.i64
		for i := range b.N {
			sinkI128, SinkBool = x[i&BenchMask].Sub64o(y[i&BenchMask])
		}
	})
	b.Run("op=Bit", func(b *testing.B) {
		x, y := in.i128, in.u64
		for i := range b.N {
			SinkUint = x[i&BenchMask].Bit(int(y[i&BenchMask] & 127))
		}
	})
	b.Run("op=WithBit", func(b *testing.B) {
		x, y := in.i128, in.u64
		for i := range b.N {
			v, k := x[i&BenchMask], y[i&BenchMask]
			sinkI128 = v.WithBit(int(k&127), uint(k>>7&1))
		}
	})
	b.Run("op=PutBig", func(b *testing.B) {
		x := in.i128
		r := new(big.Int)
		for i := range b.N {
			SinkBig = x[i&BenchMask].PutBig(r)
		}
	})
	b.Run("op=Big", func(b *testing.B) {
		x := in.i128
		for i := range b.N {
			SinkBig = x[i&BenchMask].Big()
		}
	})
	b.Run("op=FromBig", func(b *testing.B) {
		x := benchBigs128(in.i128)
		b.ResetTimer()
		for i := range b.N {
			sinkI128, _ = Int128FromBig(x[i&BenchMask])
		}
	})
	b.Run("op=CmpBig", func(b *testing.B) {
		x, y := in.i128, benchBigs128(in.j128)
		b.ResetTimer()
		for i := range b.N {
			SinkInt = x[i&BenchMask].CmpBig(y[i&BenchMask])
		}
	})
}

// benchBigs128 converts x to big.Int values
func benchBigs128(x []Int128) []*big.Int {
	r := make([]*big.Int, len(x))
	for i, v := range x {
		r[i] = v.Big()
	}
	return r
}

// kI128 describes Int128 to the shared checks of testutil
var kI128 = Kind[Int128]{
	Name: "Int128", Bits: 128, Signed: true,
	Ref: RefI128, Wrap: WrapI128, Fits: FitsI128,
	FromBig: MustInt128FromBig,
	Max:     MaxInt128(), Min: MinInt128(),
	Values: Values128, Pairs: Pairs128,
	Narrow: narrowInt128,
	Shifts: Shifts128,
	BitIdx: []int{0, 1, 63, 64, 126, 127, 128, 191, 192, 255, 256},
	Ops64: Ops64[Int128]{
		New: func(v uint64) Int128 { return NewInt128(int64(v)) }, Cmp: Signed64(Int128.Cmp64),
		Add: Signed64(Int128.Add64), Sub: Signed64(Int128.Sub64), Mul: Signed64(Int128.Mul64),
		Addo: Signed64Pair(Int128.Add64o), Subo: Signed64Pair(Int128.Sub64o), Mulo: Signed64Pair(Int128.Mul64o), Addu: Signed64(Int128.Add64u), Subu: Signed64(Int128.Sub64u), Mulu: Signed64(Int128.Mul64u),
	},
}

// TestInt128Bits checks BitLen, Not, shifts, Bit and WithBit with big.Int
func TestInt128Bits(t *testing.T) { RunBits(t, kI128) }

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

// TestInt128BitArgs checks Bit and WithBit at a negative and a huge index
func TestInt128BitArgs(t *testing.T) { RunBitArgs(t, kI128) }

// TestInt128HugeShifts checks shifts past 128 bits: zero, or the sign for Rsh
func TestInt128HugeShifts(t *testing.T) { RunHugeShifts(t, kI128) }

// TestInt128PanicText checks that every panic names the method
func TestInt128PanicText(t *testing.T) { RunPanicText(t, kI128) }

// TestInt128Constants checks untyped constants as 64-bit operands and shifts
func TestInt128Constants(t *testing.T) {
	t.Parallel()
	x := NewInt128(-5)
	require.Equal(t, NewInt128(-3), x.Add64(2))
	require.Equal(t, NewInt128(-7), x.Sub64(2))
	require.Equal(t, NewInt128(10), x.Mul64(-2))
	require.Equal(t, -1, x.Cmp64(-4))
	x = x.Add64(1).Sub64(-3)
	require.Equal(t, NewInt128(-1), x)
	require.Equal(t, NewInt128(-8), x.Lsh(3))
	require.Equal(t, NewInt128(-1), x.Rsh(1))
	require.Equal(t, uint(1), x.Bit(1))
}

// narrowInt128 cuts y to width bits, sign-extending into the limb above
func narrowInt128(y Int128, width int) Int128 {
	if width == 64 {
		y.L1 = uint64(int64(y.L0) >> 63)
	}
	return y
}

// --- benchmark inputs -----------------------------------

// sinkI128 and sink2I128 take a benchmark result on each iteration so the work is kept
var sinkI128, sink2I128 Int128

// benchI128 is BenchIn as Int128
var benchI128 = sync.OnceValue(func() (in struct {
	i128, j128 []Int128
	i64        []int64
	u64        []uint64
	i128s      []string
	// sum, difference and product fit, the shifted x for the narrow Mul forms
	fx, fy []Int128
	fx64   []Int128
}) {
	b := BenchIn()
	in.fx, in.fy = Slice128[Int128](b.FI128), Slice128[Int128](b.FJ128)
	for _, x := range in.fx {
		in.fx64 = append(in.fx64, x.Rsh(64))
	}
	in.i128, in.j128 = Slice128[Int128](b.I128), Slice128[Int128](b.J128)
	in.i64 = b.I64
	in.u64 = b.U64
	in.i128s = b.I128s
	return
})
