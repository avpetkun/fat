package fat

import (
	"math/big"
	"sync"
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

// TestInt256Arith checks Add, Sub, Mul and the bit ops against big.Int,
// with 256- and 128-bit operands in the panicking, u and o forms
func TestInt256Arith(t *testing.T) {
	t.Parallel()
	low128 := func(y Int256) Int128 { return Int128{y.L0, y.L1} }
	RunArith(t, kI256, []Arith[Int256]{
		{Name: "Add", Width: 256, Fn: Plain(Int256.Add), Ref: BigAdd, Panic: "fat.Int256.Add: overflow"},
		{Name: "Addu", Width: 256, Fn: Plain(Int256.Addu), Ref: BigAdd},
		{Name: "Addo", Width: 256, Fn: Int256.Addo, Ref: BigAdd, Over: true},
		{Name: "Add128", Width: 128, Fn: Plain(func(x, y Int256) Int256 { return x.Add128(low128(y)) }), Ref: BigAdd, Panic: "fat.Int256.Add128: overflow"},
		{Name: "Add128u", Width: 128, Fn: Plain(func(x, y Int256) Int256 { return x.Add128u(low128(y)) }), Ref: BigAdd},
		{Name: "Add128o", Width: 128, Fn: func(x, y Int256) (Int256, bool) { return x.Add128o(low128(y)) }, Ref: BigAdd, Over: true},

		{Name: "Sub", Width: 256, Fn: Plain(Int256.Sub), Ref: BigSub, Panic: "fat.Int256.Sub: overflow"},
		{Name: "Subu", Width: 256, Fn: Plain(Int256.Subu), Ref: BigSub},
		{Name: "Subo", Width: 256, Fn: Int256.Subo, Ref: BigSub, Over: true},
		{Name: "Sub128", Width: 128, Fn: Plain(func(x, y Int256) Int256 { return x.Sub128(low128(y)) }), Ref: BigSub, Panic: "fat.Int256.Sub128: overflow"},
		{Name: "Sub128u", Width: 128, Fn: Plain(func(x, y Int256) Int256 { return x.Sub128u(low128(y)) }), Ref: BigSub},
		{Name: "Sub128o", Width: 128, Fn: func(x, y Int256) (Int256, bool) { return x.Sub128o(low128(y)) }, Ref: BigSub, Over: true},

		{Name: "And", Width: 256, Fn: Plain(Int256.And), Ref: BigAnd},
		{Name: "Or", Width: 256, Fn: Plain(Int256.Or), Ref: BigOr},
		{Name: "Xor", Width: 256, Fn: Plain(Int256.Xor), Ref: BigXor},
		{Name: "AndNot", Width: 256, Fn: Plain(Int256.AndNot), Ref: BigAndNot},
	})
}

// BenchmarkInt256 measures the methods on 2^18 values of random bit length.
// Add, Mul and DivMod/fit take pairs whose results fit
func BenchmarkInt256(b *testing.B) {
	in := benchI256()
	b.Run("op=Cmp", func(b *testing.B) {
		x, y := in.i256, in.j256
		for i := range b.N {
			SinkInt = x[i&BenchMask].Cmp(y[i&BenchMask])
		}
	})
	b.Run("op=LessThan", func(b *testing.B) {
		x, y := in.i256, in.j256
		for i := range b.N {
			SinkBool = x[i&BenchMask].LessThan(y[i&BenchMask])
		}
	})
	b.Run("op=Max", func(b *testing.B) {
		x, y := in.i256, in.j256
		for i := range b.N {
			sinkI256 = x[i&BenchMask].Max(y[i&BenchMask])
		}
	})
	b.Run("op=Min", func(b *testing.B) {
		x, y := in.i256, in.j256
		for i := range b.N {
			sinkI256 = x[i&BenchMask].Min(y[i&BenchMask])
		}
	})
	b.Run("op=LessOrEqual", func(b *testing.B) {
		x, y := in.i256, in.j256
		for i := range b.N {
			SinkBool = x[i&BenchMask].LessOrEqual(y[i&BenchMask])
		}
	})
	b.Run("op=CmpBig", func(b *testing.B) {
		x := in.i256
		y := make([]*big.Int, len(in.j256))
		for i, v := range in.j256 {
			y[i] = v.Big()
		}
		b.ResetTimer()
		for i := range b.N {
			SinkInt = x[i&BenchMask].CmpBig(y[i&BenchMask])
		}
	})
	b.Run("op=Cmp128", func(b *testing.B) {
		x, y := in.i256, in.i128
		for i := range b.N {
			SinkInt = x[i&BenchMask].Cmp128(y[i&BenchMask])
		}
	})
	b.Run("op=Cmp64", func(b *testing.B) {
		x, y := in.i256, in.i64
		for i := range b.N {
			SinkInt = x[i&BenchMask].Cmp64(y[i&BenchMask])
		}
	})
	b.Run("op=Add", func(b *testing.B) {
		x, y := in.fx, in.fy
		for i := range b.N {
			sinkI256 = x[i&BenchMask].Add(y[i&BenchMask])
		}
	})
	b.Run("op=Addo", func(b *testing.B) {
		x, y := in.i256, in.j256
		for i := range b.N {
			sinkI256, SinkBool = x[i&BenchMask].Addo(y[i&BenchMask])
		}
	})
	b.Run("op=Addu", func(b *testing.B) {
		x, y := in.i256, in.j256
		for i := range b.N {
			sinkI256 = x[i&BenchMask].Addu(y[i&BenchMask])
		}
	})
	b.Run("op=Add64o", func(b *testing.B) {
		x, y := in.i256, in.i64
		for i := range b.N {
			sinkI256, SinkBool = x[i&BenchMask].Add64o(y[i&BenchMask])
		}
	})
	b.Run("op=Add128o", func(b *testing.B) {
		x, y := in.i256, in.i128
		for i := range b.N {
			sinkI256, SinkBool = x[i&BenchMask].Add128o(y[i&BenchMask])
		}
	})
	b.Run("op=Subo", func(b *testing.B) {
		x, y := in.i256, in.j256
		for i := range b.N {
			sinkI256, SinkBool = x[i&BenchMask].Subo(y[i&BenchMask])
		}
	})
	b.Run("op=Sub128o", func(b *testing.B) {
		x, y := in.i256, in.i128
		for i := range b.N {
			sinkI256, SinkBool = x[i&BenchMask].Sub128o(y[i&BenchMask])
		}
	})
	b.Run("op=Sub64o", func(b *testing.B) {
		x, y := in.i256, in.i64
		for i := range b.N {
			sinkI256, SinkBool = x[i&BenchMask].Sub64o(y[i&BenchMask])
		}
	})
	b.Run("op=Lsh", func(b *testing.B) {
		x, y := in.i256, in.u64
		for i := range b.N {
			sinkI256 = x[i&BenchMask].Lsh(int(y[i&BenchMask] & 255))
		}
	})
	b.Run("op=Rsh", func(b *testing.B) {
		x, y := in.i256, in.u64
		for i := range b.N {
			sinkI256 = x[i&BenchMask].Rsh(int(y[i&BenchMask] & 255))
		}
	})
	b.Run("op=Abs", func(b *testing.B) {
		x := in.i256
		for i := range b.N {
			sinkI256 = x[i&BenchMask].Abs()
		}
	})
	b.Run("op=UnsignedAbs", func(b *testing.B) {
		x := in.i256
		for i := range b.N {
			sinkU256 = x[i&BenchMask].UnsignedAbs()
		}
	})
	b.Run("op=Neg", func(b *testing.B) {
		x := in.i256
		for i := range b.N {
			sinkI256 = x[i&BenchMask].Neg()
		}
	})
	b.Run("op=Bit", func(b *testing.B) {
		x, y := in.i256, in.u64
		for i := range b.N {
			SinkUint = x[i&BenchMask].Bit(int(y[i&BenchMask] & 255))
		}
	})
	b.Run("op=WithBit", func(b *testing.B) {
		x, y := in.i256, in.u64
		for i := range b.N {
			v := y[i&BenchMask]
			sinkI256 = x[i&BenchMask].WithBit(int(v&255), uint(v>>8&1))
		}
	})
	b.Run("op=BitLen", func(b *testing.B) {
		x := in.i256
		for i := range b.N {
			SinkInt = x[i&BenchMask].BitLen()
		}
	})
}

// BenchmarkInt256Convert measures Sign, big.Int, float64 and binary forms
func BenchmarkInt256Convert(b *testing.B) {
	in := benchI256()
	b.Run("op=Sign", func(b *testing.B) {
		x := in.i256
		for i := range b.N {
			SinkInt = x[i&BenchMask].Sign()
		}
	})
	b.Run("op=IsPositive", func(b *testing.B) {
		x := in.i256
		for i := range b.N {
			SinkBool = x[i&BenchMask].IsPositive()
		}
	})
	b.Run("op=Big", func(b *testing.B) {
		x := in.i256
		for i := range b.N {
			SinkBig = x[i&BenchMask].Big()
		}
	})
	b.Run("op=PutBig", func(b *testing.B) {
		x := in.i256
		z := new(big.Int)
		for i := range b.N {
			SinkBig = x[i&BenchMask].PutBig(z)
		}
		SinkInt = z.Sign()
	})
	b.Run("op=FromBig", func(b *testing.B) {
		x := make([]*big.Int, len(in.i256))
		for i, v := range in.i256 {
			x[i] = v.Big()
		}
		b.ResetTimer()
		for i := range b.N {
			sinkI256, _ = Int256FromBig(x[i&BenchMask])
		}
	})
}

// kI256 describes Int256 for the shared tests
var kI256 = Kind[Int256]{
	Name: "Int256", Bits: 256, Signed: true,
	Ref: RefI256, Wrap: WrapI256, Fits: FitsI256,
	FromBig: MustInt256FromBig,
	Max:     MaxInt256(), Min: MinInt256(),
	Values: Values256, Pairs: Pairs256,
	Narrow: narrowInt256,
	Shifts: Shifts256,
	BitIdx: []int{0, 1, 63, 64, 127, 128, 191, 192, 254, 255, 256, 319, 320, 511, 512},
	CmpMore: func(t testing.TB, x, y Int256, xb *big.Int) {
		y128 := narrowInt256(y, 128)
		require.Equal(t, xb.Cmp(RefI256(y128)), x.Cmp128(Int128{y128.L0, y128.L1}))
	},
	Ops64: Ops64[Int256]{
		New: func(v uint64) Int256 { return NewInt256(int64(v)) }, Cmp: Signed64(Int256.Cmp64),
		Add: Signed64(Int256.Add64), Sub: Signed64(Int256.Sub64),
		Addo: Signed64Pair(Int256.Add64o), Subo: Signed64Pair(Int256.Sub64o), Addu: Signed64(Int256.Add64u), Subu: Signed64(Int256.Sub64u),
	},
}

// TestInt256Bits checks BitLen, Not, shifts, Bit and WithBit against big.Int
func TestInt256Bits(t *testing.T) { RunBits(t, kI256) }

// TestInt256Cmp checks the comparisons and CmpBig against big.Int
func TestInt256Cmp(t *testing.T) { RunCmp(t, kI256) }

// TestInt256Ops64 checks the methods with an int64 operand against big.Int
func TestInt256Ops64(t *testing.T) { RunOps64(t, kI256) }

// TestInt256Sign checks Sign and IsZero against big.Int
func TestInt256Sign(t *testing.T) { RunSign(t, kI256) }

// TestInt256Signed checks IsNegative, IsPositive, Neg, Abs and UnsignedAbs against big.Int
func TestInt256Signed(t *testing.T) {
	RunSigned(t, kI256, func(x Int256) *big.Int { return RefU256(x.UnsignedAbs()) })
}

// TestInt256BitArgs checks Bit and WithBit at negative and huge indexes
func TestInt256BitArgs(t *testing.T) { RunBitArgs(t, kI256) }

// TestInt256HugeShifts checks that shifts past the width leave 0 or the sign
func TestInt256HugeShifts(t *testing.T) { RunHugeShifts(t, kI256) }

// TestInt256PanicText checks that every panic names its func or method
func TestInt256PanicText(t *testing.T) { RunPanicText(t, kI256) }

// TestInt256Constants checks that untyped constants of either sign compile
func TestInt256Constants(t *testing.T) {
	t.Parallel()
	x := NewInt256(-5)
	require.Equal(t, NewInt256(-3), x.Add64(2))
	require.Equal(t, NewInt256(-7), x.Sub64(2))
	require.Equal(t, -1, x.Cmp64(-4))
	x = x.Add64(1).Sub64(-3)
	require.Equal(t, NewInt256(-1), x)
	require.Equal(t, NewInt256(-8), x.Lsh(3))
	require.Equal(t, NewInt256(-1), x.Rsh(1))
	require.Equal(t, uint(1), x.Bit(1))
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

// --- benchmark inputs -----------------------------------

// sinkI256 and sink2I256 take a benchmark result on each iteration so the work is kept
var sinkI256, sink2I256 Int256

// benchI256 views the shared benchmark inputs as Int256
var benchI256 = sync.OnceValue(func() (in struct {
	i256, j256 []Int256
	i128       []Int128
	i64        []int64
	u64        []uint64
	i256s      []string
	// pairs whose results fit, x shifted for Mul64 and Mul128
	fx, fy []Int256
	fx64   []Int256
	fx128  []Int256
}) {
	b := BenchIn()
	in.fx, in.fy = Slice256[Int256](b.FI256), Slice256[Int256](b.FJ256)
	for _, x := range in.fx {
		in.fx64 = append(in.fx64, x.Rsh(64))
	}
	for _, x := range in.fx {
		in.fx128 = append(in.fx128, x.Rsh(128))
	}
	in.i256, in.j256 = Slice256[Int256](b.I256), Slice256[Int256](b.J256)
	in.i128 = Slice128[Int128](b.I128)
	in.i64 = b.I64
	in.u64 = b.U64
	in.i256s = b.I256s
	return
})
