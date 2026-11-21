package fat

import (
	"math/big"
	"sync"
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

		{Name: "And", Width: 128, Fn: Plain(Uint128.And), Ref: BigAnd},
		{Name: "Or", Width: 128, Fn: Plain(Uint128.Or), Ref: BigOr},
		{Name: "Xor", Width: 128, Fn: Plain(Uint128.Xor), Ref: BigXor},
		{Name: "AndNot", Width: 128, Fn: Plain(Uint128.AndNot), Ref: BigAndNot},
	})
}

// TestUint128BitCounts checks LeadingZeros, TrailingZeros and OnesCount
func TestUint128BitCounts(t *testing.T) {
	t.Parallel()
	Values128(func(x Uint128) {
		xb := RefU128(x)
		t := On(t, xb)
		require.Equal(t, 128-xb.BitLen(), x.LeadingZeros())
		require.Equal(t, BigTrailingZeros(xb, 128), x.TrailingZeros())
		require.Equal(t, BigOnesCount(xb), x.OnesCount())
	})
}

// TestUint128Rotate checks the rotations and reversals with big.Int
func TestUint128Rotate(t *testing.T) {
	t.Parallel()
	Values128(func(x Uint128) {
		xb := RefU128(x)
		t := On(t, xb)
		for _, k := range Rotations {
			t := On(t, k)
			require.Big(t, BigRotateLeft(xb, k, 128), RefU128(x.RotateLeft(k)))
			require.Big(t, BigRotateLeft(xb, -k, 128), RefU128(x.RotateRight(k)))
		}
		require.Big(t, BigReverse(xb, 128), RefU128(x.Reverse()))
		require.Big(t, BigReverseBytes(xb, 128), RefU128(x.ReverseBytes()))
	})
}

// BenchmarkUint128 measures the methods on 2^18 values of random bit length.
// Add, Sub, Mul and DivMod/fit take pairs whose results fit
func BenchmarkUint128(b *testing.B) {
	in := benchU128()
	b.Run("op=Cmp", func(b *testing.B) {
		x, y := in.u128, in.v128
		for i := range b.N {
			SinkInt = x[i&BenchMask].Cmp(y[i&BenchMask])
		}
	})
	b.Run("op=LessThan", func(b *testing.B) {
		x, y := in.u128, in.v128
		for i := range b.N {
			SinkBool = x[i&BenchMask].LessThan(y[i&BenchMask])
		}
	})
	b.Run("op=Max", func(b *testing.B) {
		x, y := in.u128, in.v128
		for i := range b.N {
			sinkU128 = x[i&BenchMask].Max(y[i&BenchMask])
		}
	})
	b.Run("op=Min", func(b *testing.B) {
		x, y := in.u128, in.v128
		for i := range b.N {
			sinkU128 = x[i&BenchMask].Min(y[i&BenchMask])
		}
	})
	b.Run("op=Cmp64", func(b *testing.B) {
		x, y := in.u128, in.u64
		for i := range b.N {
			SinkInt = x[i&BenchMask].Cmp64(y[i&BenchMask])
		}
	})
	b.Run("op=Add", func(b *testing.B) {
		x, y := in.fx, in.fy
		for i := range b.N {
			sinkU128 = x[i&BenchMask].Add(y[i&BenchMask])
		}
	})
	b.Run("op=Addo", func(b *testing.B) {
		x, y := in.u128, in.v128
		for i := range b.N {
			sinkU128, SinkBool = x[i&BenchMask].Addo(y[i&BenchMask])
		}
	})
	b.Run("op=Addu", func(b *testing.B) {
		x, y := in.u128, in.v128
		for i := range b.N {
			sinkU128 = x[i&BenchMask].Addu(y[i&BenchMask])
		}
	})
	b.Run("op=Subo", func(b *testing.B) {
		x, y := in.u128, in.v128
		for i := range b.N {
			sinkU128, SinkBool = x[i&BenchMask].Subo(y[i&BenchMask])
		}
	})
	b.Run("op=Subu", func(b *testing.B) {
		x, y := in.u128, in.v128
		for i := range b.N {
			sinkU128 = x[i&BenchMask].Subu(y[i&BenchMask])
		}
	})
	b.Run("op=Lsh", func(b *testing.B) {
		x, y := in.u128, in.u64
		for i := range b.N {
			sinkU128 = x[i&BenchMask].Lsh(int(y[i&BenchMask] & 127))
		}
	})
	b.Run("op=Rsh", func(b *testing.B) {
		x, y := in.u128, in.u64
		for i := range b.N {
			sinkU128 = x[i&BenchMask].Rsh(int(y[i&BenchMask] & 127))
		}
	})
	b.Run("op=RotateLeft", func(b *testing.B) {
		x, y := in.u128, in.u64
		for i := range b.N {
			sinkU128 = x[i&BenchMask].RotateLeft(int(y[i&BenchMask] & 127))
		}
	})
	b.Run("op=RotateRight", func(b *testing.B) {
		x, y := in.u128, in.u64
		for i := range b.N {
			sinkU128 = x[i&BenchMask].RotateRight(int(y[i&BenchMask] & 127))
		}
	})
	b.Run("op=Reverse", func(b *testing.B) {
		x := in.u128
		for i := range b.N {
			sinkU128 = x[i&BenchMask].Reverse()
		}
	})
	b.Run("op=ReverseBytes", func(b *testing.B) {
		x := in.u128
		for i := range b.N {
			sinkU128 = x[i&BenchMask].ReverseBytes()
		}
	})
	b.Run("op=LessOrEqual", func(b *testing.B) {
		x, y := in.u128, in.v128
		for i := range b.N {
			SinkBool = x[i&BenchMask].LessOrEqual(y[i&BenchMask])
		}
	})
	b.Run("op=Sub", func(b *testing.B) {
		x, y := in.fx, in.fy
		for i := range b.N {
			sinkU128 = x[i&BenchMask].Sub(y[i&BenchMask])
		}
	})
	b.Run("op=Add64o", func(b *testing.B) {
		x, y := in.u128, in.u64
		for i := range b.N {
			sinkU128, SinkBool = x[i&BenchMask].Add64o(y[i&BenchMask])
		}
	})
	b.Run("op=Sub64o", func(b *testing.B) {
		x, y := in.u128, in.u64
		for i := range b.N {
			sinkU128, SinkBool = x[i&BenchMask].Sub64o(y[i&BenchMask])
		}
	})
	b.Run("op=BitLen", func(b *testing.B) {
		x := in.u128
		for i := range b.N {
			SinkInt = x[i&BenchMask].BitLen()
		}
	})
	b.Run("op=IsZero", func(b *testing.B) {
		x := in.u128
		for i := range b.N {
			// the top limb alone, zero for about half of the values
			SinkBool = Uint128{x[i&BenchMask].L1, 0}.IsZero()
		}
	})
	b.Run("op=Bit", func(b *testing.B) {
		x, y := in.u128, in.u64
		for i := range b.N {
			SinkUint = x[i&BenchMask].Bit(int(y[i&BenchMask] & 127))
		}
	})
	b.Run("op=WithBit", func(b *testing.B) {
		x, y := in.u128, in.u64
		for i := range b.N {
			v, k := x[i&BenchMask], y[i&BenchMask]
			sinkU128 = v.WithBit(int(k&127), uint(k>>7&1))
		}
	})
}

// BenchmarkUint128Latency measures chains of 4 dependent calls.
// A branch on the result would cut the chain, and b.Loop would store
// each link to the stack and load it back
func BenchmarkUint128Latency(b *testing.B) {
	in := benchU128()
	b.Run("op=Cmp", func(b *testing.B) {
		x, y := in.u128, in.v128
		n := 0
		for i := range b.N {
			v, w := x[i&BenchMask], y[i&BenchMask]
			n = v.Cmp(w.Add64(uint64(n)))
			n = v.Cmp(w.Add64(uint64(n)))
			n = v.Cmp(w.Add64(uint64(n)))
			n = v.Cmp(w.Add64(uint64(n)))
		}
		SinkInt = n
	})
	b.Run("op=Cmp64", func(b *testing.B) {
		x, y := in.u128, in.u64
		n := 0
		for i := range b.N {
			v, k := x[i&BenchMask], y[i&BenchMask]
			n = v.Cmp64(k + uint64(n))
			n = v.Cmp64(k + uint64(n))
			n = v.Cmp64(k + uint64(n))
			n = v.Cmp64(k + uint64(n))
		}
		SinkInt = n
	})
	b.Run("op=Add64o", func(b *testing.B) {
		x, y := in.u128, in.u64
		var c bool
		var s Uint128
		for i := range b.N {
			v, k := x[i&BenchMask], y[i&BenchMask]
			v, c = v.Add64o(k + uint64(BoolInt(c)))
			v, c = v.Add64o(k + uint64(BoolInt(c)))
			v, c = v.Add64o(k + uint64(BoolInt(c)))
			v, c = v.Add64o(k + uint64(BoolInt(c)))
			s = s.Xor(v)
		}
		sinkU128 = s
		SinkInt = BoolInt(c)
	})
	b.Run("op=Sub64o", func(b *testing.B) {
		x, y := in.u128, in.u64
		var c bool
		var s Uint128
		for i := range b.N {
			v, k := x[i&BenchMask], y[i&BenchMask]
			v, c = v.Sub64o(k + uint64(BoolInt(c)))
			v, c = v.Sub64o(k + uint64(BoolInt(c)))
			v, c = v.Sub64o(k + uint64(BoolInt(c)))
			v, c = v.Sub64o(k + uint64(BoolInt(c)))
			s = s.Xor(v)
		}
		sinkU128 = s
		SinkInt = BoolInt(c)
	})
	b.Run("op=Bit", func(b *testing.B) {
		x, y := in.u128, in.u64
		var n uint64
		for i := range b.N {
			v, k := x[i&BenchMask], y[i&BenchMask]
			n = uint64(v.Bit(int((k + n) & 127)))
			n = uint64(v.Bit(int((k>>8 + n) & 127)))
			n = uint64(v.Bit(int((k>>16 + n) & 127)))
			n = uint64(v.Bit(int((k>>24 + n) & 127)))
		}
		SinkInt = int(n)
	})
}

// BenchmarkUint128Convert measures the binary form and big.Int and float64
func BenchmarkUint128Convert(b *testing.B) {
	in := benchU128()
	bigs := make([]*big.Int, len(in.u128))
	for i, x := range in.u128 {
		bigs[i] = x.Big()
		if in.u64[i]&1 != 0 {
			// a random half takes the two's complement path
			bigs[i].Neg(bigs[i])
		}
	}
	b.Run("op=PutBig", func(b *testing.B) {
		x := in.u128
		z := new(big.Int)
		for i := range b.N {
			SinkBig = x[i&BenchMask].PutBig(z)
		}
	})
	b.Run("op=Big", func(b *testing.B) {
		x := in.u128
		for i := range b.N {
			SinkBig = x[i&BenchMask].Big()
		}
	})
	b.Run("op=FromBig", func(b *testing.B) {
		x := make([]*big.Int, len(in.u128))
		for i, v := range in.u128 {
			x[i] = v.Big()
		}
		b.ResetTimer()
		for i := range b.N {
			sinkU128, _ = Uint128FromBig(x[i&BenchMask])
		}
	})
	b.Run("op=CmpBig", func(b *testing.B) {
		x := in.u128
		for i := range b.N {
			// a random half is negative
			SinkInt = x[i&BenchMask].CmpBig(bigs[i&BenchMask])
		}
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
	Shifts: Shifts128,
	BitIdx: []int{0, 1, 63, 64, 126, 127, 128, 191, 192, 255, 256},
	Ops64: Ops64[Uint128]{
		New: NewUint128, Cmp: Uint128.Cmp64,
		Add: Uint128.Add64, Sub: Uint128.Sub64,
		Addo: Uint128.Add64o, Subo: Uint128.Sub64o, Addu: Uint128.Add64u, Subu: Uint128.Sub64u,
	},
}

// TestUint128Bits checks BitLen, Not, shifts, Bit and WithBit with big.Int
func TestUint128Bits(t *testing.T) { RunBits(t, kU128) }

// TestUint128Cmp checks Cmp, the comparisons and CmpBig with big.Int
func TestUint128Cmp(t *testing.T) { RunCmp(t, kU128) }

// TestUint128Ops64 checks the methods with a uint64 operand with big.Int
func TestUint128Ops64(t *testing.T) { RunOps64(t, kU128) }

// TestUint128Sign checks Sign and IsZero with big.Int
func TestUint128Sign(t *testing.T) { RunSign(t, kU128) }

// TestUint128BitArgs checks Bit and WithBit at a negative and a huge index
func TestUint128BitArgs(t *testing.T) { RunBitArgs(t, kU128) }

// TestUint128HugeShifts checks that a shift past 128 bits leaves zero
func TestUint128HugeShifts(t *testing.T) { RunHugeShifts(t, kU128) }

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
	require.Equal(t, NewUint128(24), x.Lsh(3))
	require.Equal(t, NewUint128(1), x.Rsh(1))
	require.Equal(t, uint(1), x.Bit(1))
}

// --- benchmark inputs -----------------------------------

// sinkU128 and sink2U128 take a benchmark result on each iteration so the work is kept
var sinkU128, sink2U128 Uint128

// benchU128 is BenchIn as Uint128
var benchU128 = sync.OnceValue(func() (in struct {
	u128, v128 []Uint128
	u64        []uint64
	u128s      []string
	// sum, difference and product fit, the shifted x for the narrow Mul forms
	fx, fy []Uint128
	fx64   []Uint128
}) {
	b := BenchIn()
	in.fx, in.fy = Slice128[Uint128](b.FU128), Slice128[Uint128](b.FV128)
	for _, x := range in.fx {
		in.fx64 = append(in.fx64, x.Rsh(64))
	}
	in.u128, in.v128 = Slice128[Uint128](b.U128), Slice128[Uint128](b.V128)
	in.u64 = b.U64
	in.u128s = b.U128s
	return
})
