package fat

import (
	"math/big"
	"sync"
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

		// big.Int both ways, reusing dirty values
		require.Big(t, xb, x.Big())
		require.Big(t, xb, x.PutBig(big.NewInt(-12345)))
		require.Big(t, xb, x.PutBig(new(big.Int)))
		// reused words and the sign of a wider value
		require.Big(t, xb, x.PutBig(new(big.Int).Neg(Two512)))
		require.Equal(t, x, MustUint256FromBig(xb))
		// small capacity words from SetBits
		require.Equal(t, x, MustUint256FromBig(BigClip(xb)))
		y, err := Uint256FromBig(xb)
		require.NoError(t, err)
		require.Equal(t, x, y)

		// negative and oversized big.Int are out of range and zero the value
		for _, b := range []*big.Int{BigSub(xb, Two256), BigAdd(xb, Two256), BigAdd(xb, new(big.Int).Lsh(Two256, 3))} {
			t := On(t, b)
			v, err := Uint256FromBig(b)
			want := "fat.Uint256FromBig: out of range"
			if b.Sign() < 0 {
				want = "fat.Uint256FromBig: negative value"
			}
			require.Equal(t, want, err.Error())
			require.True(t, v.IsZero())
			require.Equal(t, "fat.Must"+want[len("fat."):], PanicValue(func() { MustUint256FromBig(b) }))
		}

		// the constructor and the widening zero the limbs above
		require.Equal(t, Uint256{x.L0, 0, 0, 0}, NewUint256(x.L0))
		require.Equal(t, Uint256{x.L0, x.L1, 0, 0}, Uint128{x.L0, x.L1}.Uint256())
	})
}

// TestUint256Arith checks Add, Sub, Mul and the bit ops against big.Int,
// with 256- and 128-bit operands in the panicking, u and o forms
func TestUint256Arith(t *testing.T) {
	t.Parallel()
	RunArith(t, kU256, []Arith[Uint256]{
		{Name: "Add", Width: 256, Fn: Plain(Uint256.Add), Ref: BigAdd, Panic: "fat.Uint256.Add: overflow"},
		{Name: "Addu", Width: 256, Fn: Plain(Uint256.Addu), Ref: BigAdd},
		{Name: "Addo", Width: 256, Fn: Uint256.Addo, Ref: BigAdd, Over: true},
		{Name: "Add128", Width: 128, Fn: Plain(func(x, y Uint256) Uint256 { return x.Add128(Uint128{y.L0, y.L1}) }), Ref: BigAdd, Panic: "fat.Uint256.Add128: overflow"},
		{Name: "Add128u", Width: 128, Fn: Plain(func(x, y Uint256) Uint256 { return x.Add128u(Uint128{y.L0, y.L1}) }), Ref: BigAdd},
		{Name: "Add128o", Width: 128, Fn: func(x, y Uint256) (Uint256, bool) { return x.Add128o(Uint128{y.L0, y.L1}) }, Ref: BigAdd, Over: true},

		{Name: "Sub", Width: 256, Fn: Plain(Uint256.Sub), Ref: BigSub, Panic: "fat.Uint256.Sub: negative value"},
		{Name: "Subu", Width: 256, Fn: Plain(Uint256.Subu), Ref: BigSub},
		{Name: "Subo", Width: 256, Fn: Uint256.Subo, Ref: BigSub, Over: true},
		{Name: "Sub128", Width: 128, Fn: Plain(func(x, y Uint256) Uint256 { return x.Sub128(Uint128{y.L0, y.L1}) }), Ref: BigSub, Panic: "fat.Uint256.Sub128: negative value"},
		{Name: "Sub128u", Width: 128, Fn: Plain(func(x, y Uint256) Uint256 { return x.Sub128u(Uint128{y.L0, y.L1}) }), Ref: BigSub},
		{Name: "Sub128o", Width: 128, Fn: func(x, y Uint256) (Uint256, bool) { return x.Sub128o(Uint128{y.L0, y.L1}) }, Ref: BigSub, Over: true},

		{Name: "And", Width: 256, Fn: Plain(Uint256.And), Ref: BigAnd},
		{Name: "Or", Width: 256, Fn: Plain(Uint256.Or), Ref: BigOr},
		{Name: "Xor", Width: 256, Fn: Plain(Uint256.Xor), Ref: BigXor},
		{Name: "AndNot", Width: 256, Fn: Plain(Uint256.AndNot), Ref: BigAndNot},
	})
}

// TestUint256BitCounts checks LeadingZeros, TrailingZeros and OnesCount
func TestUint256BitCounts(t *testing.T) {
	t.Parallel()
	Values256(func(x Uint256) {
		xb := RefU256(x)
		t := On(t, xb)
		require.Equal(t, 256-xb.BitLen(), x.LeadingZeros())
		require.Equal(t, BigTrailingZeros(xb, 256), x.TrailingZeros())
		require.Equal(t, BigOnesCount(xb), x.OnesCount())
	})
}

// TestUint256Rotate checks RotateLeft, RotateRight, Reverse and ReverseBytes
func TestUint256Rotate(t *testing.T) {
	t.Parallel()
	Values256(func(x Uint256) {
		xb := RefU256(x)
		t := On(t, xb)
		for _, k := range Rotations {
			t := On(t, k)
			require.Big(t, BigRotateLeft(xb, k, 256), RefU256(x.RotateLeft(k)))
			require.Big(t, BigRotateLeft(xb, -k, 256), RefU256(x.RotateRight(k)))
		}
		require.Big(t, BigReverse(xb, 256), RefU256(x.Reverse()))
		require.Big(t, BigReverseBytes(xb, 256), RefU256(x.ReverseBytes()))
	})
}

// BenchmarkUint256 measures the methods on 2^18 values of random bit length.
// Add, Sub, Mul and DivMod/fit take pairs whose results fit
func BenchmarkUint256(b *testing.B) {
	in := benchU256()
	b.Run("op=Cmp", func(b *testing.B) {
		x, y := in.u256, in.v256
		for i := range b.N {
			SinkInt = x[i&BenchMask].Cmp(y[i&BenchMask])
		}
	})
	b.Run("op=LessThan", func(b *testing.B) {
		x, y := in.u256, in.v256
		for i := range b.N {
			SinkBool = x[i&BenchMask].LessThan(y[i&BenchMask])
		}
	})
	b.Run("op=Max", func(b *testing.B) {
		x, y := in.u256, in.v256
		for i := range b.N {
			sinkU256 = x[i&BenchMask].Max(y[i&BenchMask])
		}
	})
	b.Run("op=Min", func(b *testing.B) {
		x, y := in.u256, in.v256
		for i := range b.N {
			sinkU256 = x[i&BenchMask].Min(y[i&BenchMask])
		}
	})
	b.Run("op=LessOrEqual", func(b *testing.B) {
		x, y := in.u256, in.v256
		for i := range b.N {
			SinkBool = x[i&BenchMask].LessOrEqual(y[i&BenchMask])
		}
	})
	b.Run("op=Cmp128", func(b *testing.B) {
		x, y := in.u256, in.u128
		for i := range b.N {
			SinkInt = x[i&BenchMask].Cmp128(y[i&BenchMask])
		}
	})
	b.Run("op=Cmp64", func(b *testing.B) {
		x, y := in.u256, in.u64
		for i := range b.N {
			SinkInt = x[i&BenchMask].Cmp64(y[i&BenchMask])
		}
	})
	b.Run("op=Add", func(b *testing.B) {
		x, y := in.fx, in.fy
		for i := range b.N {
			sinkU256 = x[i&BenchMask].Add(y[i&BenchMask])
		}
	})
	b.Run("op=Addo", func(b *testing.B) {
		x, y := in.u256, in.v256
		for i := range b.N {
			sinkU256, SinkBool = x[i&BenchMask].Addo(y[i&BenchMask])
		}
	})
	b.Run("op=Addu", func(b *testing.B) {
		x, y := in.u256, in.v256
		for i := range b.N {
			sinkU256 = x[i&BenchMask].Addu(y[i&BenchMask])
		}
	})
	b.Run("op=Sub", func(b *testing.B) {
		x, y := in.fx, in.fy
		for i := range b.N {
			sinkU256 = x[i&BenchMask].Sub(y[i&BenchMask])
		}
	})
	b.Run("op=Subo", func(b *testing.B) {
		x, y := in.u256, in.v256
		for i := range b.N {
			sinkU256, SinkBool = x[i&BenchMask].Subo(y[i&BenchMask])
		}
	})
	b.Run("op=Subu", func(b *testing.B) {
		x, y := in.u256, in.v256
		for i := range b.N {
			sinkU256 = x[i&BenchMask].Subu(y[i&BenchMask])
		}
	})
	b.Run("op=Lsh", func(b *testing.B) {
		x, y := in.u256, in.u64
		for i := range b.N {
			sinkU256 = x[i&BenchMask].Lsh(int(y[i&BenchMask] & 255))
		}
	})
	b.Run("op=Rsh", func(b *testing.B) {
		x, y := in.u256, in.u64
		for i := range b.N {
			sinkU256 = x[i&BenchMask].Rsh(int(y[i&BenchMask] & 255))
		}
	})
	b.Run("op=RotateLeft", func(b *testing.B) {
		x, y := in.u256, in.u64
		for i := range b.N {
			sinkU256 = x[i&BenchMask].RotateLeft(int(y[i&BenchMask] & 255))
		}
	})
	b.Run("op=RotateRight", func(b *testing.B) {
		x, y := in.u256, in.u64
		for i := range b.N {
			sinkU256 = x[i&BenchMask].RotateRight(int(y[i&BenchMask] & 255))
		}
	})
	b.Run("op=Reverse", func(b *testing.B) {
		x := in.u256
		for i := range b.N {
			sinkU256 = x[i&BenchMask].Reverse()
		}
	})
	b.Run("op=ReverseBytes", func(b *testing.B) {
		x := in.u256
		for i := range b.N {
			sinkU256 = x[i&BenchMask].ReverseBytes()
		}
	})
	b.Run("op=BitLen", func(b *testing.B) {
		x := in.u256
		for i := range b.N {
			SinkInt = x[i&BenchMask].BitLen()
		}
	})
	b.Run("op=Bit", func(b *testing.B) {
		x, y := in.u256, in.u64
		for i := range b.N {
			SinkUint = x[i&BenchMask].Bit(int(y[i&BenchMask] & 255))
		}
	})
	b.Run("op=WithBit", func(b *testing.B) {
		x, y, v := in.u256, in.u64, in.v256
		for i := range b.N {
			sinkU256 = x[i&BenchMask].WithBit(int(y[i&BenchMask]&255), uint(v[i&BenchMask].L0&1))
		}
	})
	b.Run("op=CmpBig", func(b *testing.B) {
		x := in.u256
		// v256 with random signs, a few wider than 256 bits
		y := make([]*big.Int, len(in.v256))
		for i, v := range in.v256 {
			y[i] = RefU256(v)
			if i&1 == 0 {
				y[i].Neg(y[i])
			}
			if i&63 == 1 {
				y[i].Lsh(y[i], 256)
			}
		}
		b.ResetTimer()
		for i := range b.N {
			SinkInt = x[i&BenchMask].CmpBig(y[i&BenchMask])
		}
	})
}

// BenchmarkUint256Big measures the big.Int and float64 conversions both ways
func BenchmarkUint256Big(b *testing.B) {
	in := benchU256()
	b.Run("op=PutBig", func(b *testing.B) {
		x := in.u256
		z := new(big.Int)
		for i := range b.N {
			SinkBig = x[i&BenchMask].PutBig(z)
		}
	})
	b.Run("op=FromBig", func(b *testing.B) {
		x := make([]*big.Int, len(in.u256))
		for i, v := range in.u256 {
			x[i] = v.Big()
		}
		b.ResetTimer()
		for i := range b.N {
			sinkU256, _ = Uint256FromBig(x[i&BenchMask])
		}
	})
}

// narrowUint256 zeroes the limbs of y above width bits
func narrowUint256(y Uint256, width int) Uint256 {
	l := [4]uint64{y.L0, y.L1, y.L2, y.L3}
	for i := width / 64; i < 4; i++ {
		l[i] = 0
	}
	return Uint256{l[0], l[1], l[2], l[3]}
}

// kU256 describes Uint256 for the shared tests
var kU256 = Kind[Uint256]{
	Name: "Uint256", Bits: 256,
	Ref: RefU256, Wrap: WrapU256, Fits: FitsU256,
	FromBig: MustUint256FromBig,
	Max:     MaxUint256(),
	Values:  Values256, Pairs: Pairs256,
	Narrow: narrowUint256,
	Shifts: Shifts256,
	BitIdx: []int{0, 1, 63, 64, 127, 128, 191, 192, 254, 255, 256, 319, 320, 511, 512},
	CmpMore: func(t testing.TB, x, y Uint256, xb *big.Int) {
		require.Equal(t, xb.Cmp(LimbsBig(y.L0, y.L1)), x.Cmp128(Uint128{y.L0, y.L1}))
	},
	Ops64: Ops64[Uint256]{
		New: NewUint256, Cmp: Uint256.Cmp64,
		Add: Uint256.Add64, Sub: Uint256.Sub64,
		Addo: Uint256.Add64o, Subo: Uint256.Sub64o, Addu: Uint256.Add64u, Subu: Uint256.Sub64u,
	},
}

// TestUint256Bits checks BitLen, Not, shifts, Bit and WithBit against big.Int
func TestUint256Bits(t *testing.T) { RunBits(t, kU256) }

// TestUint256Cmp checks the comparisons and CmpBig against big.Int
func TestUint256Cmp(t *testing.T) { RunCmp(t, kU256) }

// TestUint256Ops64 checks the methods with a uint64 operand against big.Int
func TestUint256Ops64(t *testing.T) { RunOps64(t, kU256) }

// TestUint256Sign checks Sign and IsZero against big.Int
func TestUint256Sign(t *testing.T) { RunSign(t, kU256) }

// TestUint256BitArgs checks Bit and WithBit at negative and huge indexes
func TestUint256BitArgs(t *testing.T) { RunBitArgs(t, kU256) }

// TestUint256HugeShifts checks that shifts past the width leave zero
func TestUint256HugeShifts(t *testing.T) { RunHugeShifts(t, kU256) }

// TestUint256PanicText checks that every panic names its func or method
func TestUint256PanicText(t *testing.T) { RunPanicText(t, kU256) }

// TestUint256Constants checks that untyped constants compile as operands
func TestUint256Constants(t *testing.T) {
	t.Parallel()
	x := NewUint256(5)
	require.Equal(t, NewUint256(7), x.Add64(2))
	require.Equal(t, NewUint256(3), x.Sub64(2))
	require.Equal(t, 1, x.Cmp64(4))
	x = x.Add64(1).Sub64(3)
	require.Equal(t, NewUint256(3), x)
	require.Equal(t, NewUint256(24), x.Lsh(3))
	require.Equal(t, NewUint256(1), x.Rsh(1))
	require.Equal(t, uint(1), x.Bit(1))
}

// TestUint256From128 checks that Uint128 widens by zeros, also in binary
func TestUint256From128(t *testing.T) {
	t.Parallel()
	Values128(func(x Uint128) {
		t := On(t, RefU128(x))
		require.Equal(t, Uint256{x.L0, x.L1, 0, 0}, x.Uint256())
	})
}

// --- benchmark inputs -----------------------------------

// sinkU256 and sink2U256 take a benchmark result on each iteration so the work is kept
var sinkU256, sink2U256 Uint256

// benchU256 views the shared benchmark inputs as Uint256
var benchU256 = sync.OnceValue(func() (in struct {
	u256, v256 []Uint256
	u128       []Uint128
	u64        []uint64
	u256s      []string
	// pairs whose results fit, x shifted for Mul64 and Mul128
	fx, fy []Uint256
	fx64   []Uint256
	fx128  []Uint256
	// 2^n for n at random in 0..255
	pow2 []Uint256
}) {
	b := BenchIn()
	in.fx, in.fy = Slice256[Uint256](b.FU256), Slice256[Uint256](b.FV256)
	for _, x := range in.fx {
		in.fx64 = append(in.fx64, x.Rsh(64))
	}
	for _, x := range in.fx {
		in.fx128 = append(in.fx128, x.Rsh(128))
	}
	in.u256, in.v256 = Slice256[Uint256](b.U256), Slice256[Uint256](b.V256)
	in.u128 = Slice128[Uint128](b.U128)
	in.u64 = b.U64
	in.u256s = b.U256s
	for _, v := range in.u64 {
		in.pow2 = append(in.pow2, Uint256{L0: 1}.Lsh(int(v&255)))
	}
	return
})
