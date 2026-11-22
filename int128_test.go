package fat

import (
	"math"
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
	b.Run("op=DivMod", func(b *testing.B) {
		x, y := in.i128, in.j128
		for i := range b.N {
			sinkI128, sink2I128 = x[i&BenchMask].DivMod(y[i&BenchMask])
		}
	})
	b.Run("op=DivMod/fit", func(b *testing.B) {
		x, y := in.fx, in.fy
		for i := range b.N {
			sinkI128, sink2I128 = x[i&BenchMask].DivMod(y[i&BenchMask])
		}
	})
	b.Run("op=DivMod64", func(b *testing.B) {
		x, y := in.i128, in.i64
		for i := range b.N {
			sinkI128, SinkInt64 = x[i&BenchMask].DivMod64(y[i&BenchMask])
		}
	})
	b.Run("op=MulDivo", func(b *testing.B) {
		x, y := in.i128, in.j128
		for i := range b.N {
			sinkI128, SinkBool = x[i&BenchMask].MulDivo(y[i&BenchMask], y[(i+1)&BenchMask])
		}
	})
	// |d| >= |y| keeps x * y / d within range: y with its low bit set away from zero
	mulDivD := sync.OnceValue(func() []Int128 {
		d := make([]Int128, len(in.j128))
		for i, v := range in.j128 {
			if v.IsNegative() {
				d[i] = v.AndNot(NewInt128(1))
			} else {
				d[i] = v.Or(NewInt128(1))
			}
		}
		return d
	})
	b.Run("op=MulDiv", func(b *testing.B) {
		x, y, d := in.i128, in.j128, mulDivD()
		for i := range b.N {
			sinkI128 = x[i&BenchMask].MulDiv(y[i&BenchMask], d[i&BenchMask])
		}
	})
	b.Run("op=MulDivCeil", func(b *testing.B) {
		x, y, d := in.i128, in.j128, mulDivD()
		for i := range b.N {
			sinkI128 = x[i&BenchMask].MulDivCeil(y[i&BenchMask], d[i&BenchMask])
		}
	})
	b.Run("op=MulMod", func(b *testing.B) {
		x, y := in.i128, in.j128
		for i := range b.N {
			sinkI128 = x[i&BenchMask].MulMod(y[i&BenchMask], y[(i+1)&BenchMask])
		}
	})
	b.Run("op=AddMod", func(b *testing.B) {
		x, y := in.i128, in.j128
		for i := range b.N {
			sinkI128 = x[i&BenchMask].AddMod(y[i&BenchMask], y[(i+1)&BenchMask])
		}
	})
	b.Run("op=SubMod", func(b *testing.B) {
		x, y := in.i128, in.j128
		for i := range b.N {
			sinkI128 = x[i&BenchMask].SubMod(y[i&BenchMask], y[(i+1)&BenchMask])
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
	b.Run("op=Float64", func(b *testing.B) {
		x := in.i128
		for i := range b.N {
			SinkFloat = x[i&BenchMask].Float64()
		}
	})
	b.Run("op=FromFloat64", func(b *testing.B) {
		x := make([]float64, len(in.i128))
		for i, v := range in.i128 {
			x[i] = v.Float64()
		}
		b.ResetTimer()
		for i := range b.N {
			sinkI128, SinkBool = Int128FromFloat64(x[i&BenchMask])
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
	FromBig: MustInt128FromBig, FromFloat64: Int128FromFloat64, MustFromFloat64: MustInt128FromFloat64, Small: func(v uint64) Int128 { return NewInt128(int64(v)) },
	Max: MaxInt128(), Min: MinInt128(),
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

// TestInt128DivMod checks the truncated division forms with big.Int
func TestInt128DivMod(t *testing.T) {
	t.Parallel()
	// check compares one division form with big.Int, q and r truncated.
	// Div and DivMod panic where Divo reports an overflow
	check := func(t testing.TB, name string, xb, yb *big.Int, div func() Int128, divo func() (Int128, bool), divu func() Int128,
		mod func() *big.Int, divmod func() (Int128, *big.Int)) {
		qb, rb := new(big.Int).QuoRem(xb, yb, new(big.Int))
		over := !BigFits(qb, 128, true)
		wrap := WrapI128(qb)
		q, o := divo()
		require.Big(t, wrap, RefI128(q))
		require.Equal(t, over, o)
		require.Big(t, wrap, RefI128(divu()))
		require.Big(t, rb, mod())
		if over {
			require.Equal(t, "fat.Int128."+name+": overflow", PanicValue(func() { div() }))
			require.Equal(t, "fat.Int128.DivMod"+name[3:]+": overflow", PanicValue(func() { divmod() }))
			return
		}
		require.Big(t, qb, RefI128(div()))
		q, r := divmod()
		require.Big(t, qb, RefI128(q))
		require.Big(t, rb, r)
	}
	check64 := func(x Int128, v int64) {
		xb, vb := RefI128(x), big.NewInt(v)
		check(On(t, xb, vb), "Div64", xb, vb, func() Int128 { return x.Div64(v) }, func() (Int128, bool) { return x.Div64o(v) },
			func() Int128 { return x.Div64u(v) }, func() *big.Int { return big.NewInt(x.Mod64(v)) },
			func() (Int128, *big.Int) { q, r := x.DivMod64(v); return q, big.NewInt(r) })
	}
	pair := func(x, y Int128) {
		xb := RefI128(x)
		if !y.IsZero() {
			yb := RefI128(y)
			check(On(t, xb, yb), "Div", xb, yb, func() Int128 { return x.Div(y) }, func() (Int128, bool) { return x.Divo(y) },
				func() Int128 { return x.Divu(y) }, func() *big.Int { return RefI128(x.Mod(y)) },
				func() (Int128, *big.Int) { q, r := x.DivMod(y); return q, RefI128(r) })
		}
		if v := int64(y.L0); v != 0 {
			check64(x, v)
		}
	}
	Pairs128(pair)
	// the borders of the division paths with every pair of signs
	divPairs128(func(x, y Uint128) {
		pair(Int128(x), Int128(y))
		pair(Int128(x), Int128(y).Not())
		pair(Int128(x).Not(), Int128(y))
	})
	// every step of the reciprocal with a divisor of either sign:
	// a limb one under the divisor shows a reciprocal that is one short
	for _, d := range Divisors64 {
		v := int64(d >> 1)
		for _, l := range []uint64{0, 1, 1<<63 - 1, 1 << 63, math.MaxUint64 - 1, math.MaxUint64} {
			if v != 0 {
				check64(Int128{l, uint64(v) - 1}, v)
				check64(Int128{l, uint64(v) - 1}, -v)
			}
		}
	}
	// the one overflow, the edges have it too
	q, over := MinInt128().Divo(NewInt128(-1))
	require.Equal(t, MinInt128(), q)
	require.True(t, over)
	require.Equal(t, MinInt128(), MinInt128().Div64u(-1))
	require.Equal(t, int64(0), MinInt128().Mod64(-1))
	for name, fn := range map[string]func(){
		"Div":      func() { MaxInt128().Div(Int128{}) },
		"Divo":     func() { MaxInt128().Divo(Int128{}) },
		"Divu":     func() { MaxInt128().Divu(Int128{}) },
		"Mod":      func() { MaxInt128().Mod(Int128{}) },
		"DivMod":   func() { MaxInt128().DivMod(Int128{}) },
		"Div64":    func() { MaxInt128().Div64(0) },
		"Div64o":   func() { MaxInt128().Div64o(0) },
		"Div64u":   func() { MaxInt128().Div64u(0) },
		"Mod64":    func() { MaxInt128().Mod64(0) },
		"DivMod64": func() { MaxInt128().DivMod64(0) },
	} {
		require.Equal(On(t, name), "fat.Int128."+name+": division by zero", PanicValue(fn))
	}
	// y == 0 is division by zero on every path, not an overflow for MinInt128
	for _, x := range []Int128{{}, NewInt128(7), NewInt128(-7), MinInt128()} {
		t := On(t, x)
		require.Equal(t, "fat.Int128.Div: division by zero", PanicValue(func() { x.Div(Int128{}) }))
		require.Equal(t, "fat.Int128.Divo: division by zero", PanicValue(func() { x.Divo(Int128{}) }))
		require.Equal(t, "fat.Int128.DivMod: division by zero", PanicValue(func() { x.DivMod(Int128{}) }))
	}
}

// TestInt128MulDiv checks MulDiv, MulDivCeil, MulMod, AddMod and SubMod with big.Int and by zero
func TestInt128MulDiv(t *testing.T) {
	t.Parallel()
	var ds []Int128
	Values128(func(d Int128) {
		if !d.IsZero() {
			ds = append(ds, d)
		}
	})
	i := 0
	Pairs128(func(x, y Int128) {
		i++
		xb, yb := RefI128(x), RefI128(y)
		pb := new(big.Int).Mul(xb, yb)
		for _, d := range []Int128{ds[i%len(ds)], ds[(i*7+3)%len(ds)], y, x} {
			if d.IsZero() {
				continue
			}
			db := RefI128(d)
			t := On(t, xb, yb, db)
			qb, rb := new(big.Int).QuoRem(pb, db, new(big.Int))
			over := !BigFits(qb, 128, true)
			q, o := x.MulDivo(y, d)
			require.Big(t, WrapI128(qb), RefI128(q))
			require.Equal(t, over, o)
			require.Equal(t, q, x.MulDivu(y, d))
			if over {
				require.Equal(t, "fat.Int128.MulDiv: overflow", PanicValue(func() { x.MulDiv(y, d) }))
				require.Equal(t, "fat.Int128.MulDivMod: overflow", PanicValue(func() { x.MulDivMod(y, d) }))
			} else {
				require.Big(t, qb, RefI128(x.MulDiv(y, d)))
				q, r := x.MulDivMod(y, d)
				require.Big(t, qb, RefI128(q))
				require.Big(t, rb, RefI128(r))
				// MulDiv is Mul then Div where Mul does not overflow
				if p, o := x.Mulo(y); !o {
					require.Equal(t, p.Div(d), q)
				}
			}
			// the sign of the dividend as Go's %
			require.Big(t, new(big.Int).Rem(pb, db), RefI128(x.MulMod(y, d)))
			require.Big(t, new(big.Int).Rem(new(big.Int).Add(xb, yb), db), RefI128(x.AddMod(y, d)))
			require.Big(t, new(big.Int).Rem(new(big.Int).Sub(xb, yb), db), RefI128(x.SubMod(y, d)))
			// MulDivCeil is the quotient plus one where x * y / d has a positive fraction
			cb := new(big.Int).Set(qb)
			if rb.Sign() != 0 && rb.Sign() == db.Sign() {
				cb.Add(cb, Big1)
			}
			if BigFits(cb, 128, true) {
				require.Big(t, cb, RefI128(x.MulDivCeil(y, d)))
			} else {
				require.Equal(t, "fat.Int128.MulDivCeil: overflow", PanicValue(func() { x.MulDivCeil(y, d) }))
			}
		}
	})
	// an overflowing sum of one sign by small, negative and extreme moduli
	for _, m := range []Int128{NewInt128(1), NewInt128(-1), NewInt128(3), NewInt128(-3), MaxInt128(), MinInt128()} {
		for _, x := range []Int128{MaxInt128(), MaxInt128().Sub64(1), MinInt128(), MinInt128().Add64(1)} {
			want := new(big.Int).Rem(new(big.Int).Add(RefI128(x), RefI128(x)), RefI128(m))
			require.Big(On(t, x, m), want, RefI128(x.AddMod(x, m)))
		}
	}
	// sums at the borders of the division paths, with and without an overflow
	divPairs128(func(a, b Uint128) {
		x, m := Int128(a), Int128(b)
		if m.IsZero() {
			return
		}
		for _, y := range []Int128{x, x.Not(), m, MinInt128(), MaxInt128()} {
			want := new(big.Int).Rem(new(big.Int).Add(RefI128(x), RefI128(y)), RefI128(m))
			require.Big(On(t, x, y, m), want, RefI128(x.AddMod(y, m)))
		}
	})
	// differences at -2m, -m, 0, m and 2m and one off, with operands below m or not
	for _, m := range ds {
		for _, x := range []Int128{{}, m.Rsh(1), m, m.Not(), MaxInt128(), MinInt128()} {
			for _, k := range []Int128{{}, m, m.Addu(m)} {
				for _, y := range []Int128{x.Subu(k), x.Subu(k).Add64u(1), x.Subu(k).Sub64u(1), x.Addu(k), x.Addu(k).Add64u(1), x.Addu(k).Sub64u(1)} {
					want := new(big.Int).Rem(new(big.Int).Sub(RefI128(x), RefI128(y)), RefI128(m))
					require.Big(On(t, x, y, m), want, RefI128(x.SubMod(y, m)))
				}
			}
		}
	}
	// (2^65 - 1)(2^65 + 1) / 8 is the maximum and 7/8: it fits truncated, not rounded up.
	// Negated it rounds up to -maximum
	{
		h := NewInt128(1).Lsh(65)
		x, y, d := h.Sub64(1), h.Add64(1), NewInt128(8)
		require.Equal(t, MaxInt128(), x.MulDiv(y, d))
		require.Equal(t, "fat.Int128.MulDivCeil: overflow", PanicValue(func() { x.MulDivCeil(y, d) }))
		require.Equal(t, "fat.Int128.MulDivCeil: overflow", PanicValue(func() { x.Neg().MulDivCeil(y, d.Neg()) }))
		require.Equal(t, MaxInt128().Neg(), x.Neg().MulDivCeil(y, d))
		require.Equal(t, MaxInt128().Neg(), x.MulDivCeil(y, d.Neg()))
	}
	// a product past 128 bits and one within them take different paths to d
	for _, x := range []Int128{MaxInt128(), NewInt128(-3)} {
		for name, fn := range map[string]func(){
			"MulDiv":     func() { x.MulDiv(x, Int128{}) },
			"MulDivo":    func() { x.MulDivo(x, Int128{}) },
			"MulDivu":    func() { x.MulDivu(x, Int128{}) },
			"MulDivMod":  func() { x.MulDivMod(x, Int128{}) },
			"MulDivCeil": func() { x.MulDivCeil(x, Int128{}) },
			"AddMod":     func() { x.AddMod(x, Int128{}) },
			"SubMod":     func() { x.SubMod(x, Int128{}) },
			"MulMod":     func() { x.MulMod(x, Int128{}) },
		} {
			require.Equal(On(t, x, name), "fat.Int128."+name+": division by zero", PanicValue(fn))
		}
	}
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

// TestInt128Exp checks Exp, Expo and Expu with big.Int
func TestInt128Exp(t *testing.T) { RunExp(t, kI128) }

// TestInt128ExpMod checks ExpMod against big.Int
func TestInt128ExpMod(t *testing.T) { RunExpMod(t, kI128) }

// TestInt128Sqrt checks Sqrt with big.Int at squares and next to them
func TestInt128Sqrt(t *testing.T) { RunSqrt(t, kI128) }

// TestInt128Log10 checks Log10 at every power of ten and bit length
func TestInt128Log10(t *testing.T) { RunLog10(t, kI128) }

// TestInt128Float checks Float64 and FromFloat64 with big.Float
func TestInt128Float(t *testing.T) { RunFloat(t, kI128) }

// TestInt128DivFloat checks the rounding of DivToFloat64 with big.Rat
func TestInt128DivFloat(t *testing.T) { RunDivFloat(t, kI128) }

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

// BenchmarkInt128Pow measures Exp, Expo, Expu, Sqrt and Log10.
// Exp, Expo and Expu take powers that fit, Expu/full random pairs.
// Sqrt and Log10 take |x|
func BenchmarkInt128Pow(b *testing.B) {
	in := benchI128()
	l, e := ExpIn(128, true)
	ex, ey := Slice128[Int128](Low128(l)), make([]Int128, len(e))
	for i, v := range e {
		ey[i] = NewInt128(int64(v))
	}
	b.Run("op=Exp", func(b *testing.B) {
		for i := range b.N {
			sinkI128 = ex[i&BenchMask].Exp(ey[i&BenchMask])
		}
	})
	b.Run("op=Expo", func(b *testing.B) {
		for i := range b.N {
			sinkI128, SinkBool = ex[i&BenchMask].Expo(ey[i&BenchMask])
		}
	})
	b.Run("op=Expu", func(b *testing.B) {
		for i := range b.N {
			sinkI128 = ex[i&BenchMask].Expu(ey[i&BenchMask])
		}
	})
	b.Run("op=Expu/full", func(b *testing.B) {
		x, y := in.i128, in.j128
		for i := range b.N {
			sinkI128 = x[i&BenchMask].Expu(y[i&BenchMask].Abs())
		}
	})
	a := make([]Int128, len(in.i128))
	for i, v := range in.i128 {
		a[i] = v.Abs()
	}
	b.Run("op=Sqrt", func(b *testing.B) {
		for i := range b.N {
			sinkI128 = a[i&BenchMask].Sqrt()
		}
	})
	b.Run("op=Log10", func(b *testing.B) {
		for i := range b.N {
			SinkInt = a[i&BenchMask].Log10()
		}
	})
}

// BenchmarkInt128DivToFloat64 measures DivToFloat64 and DivToFloat64x.
// big.Rat.Float64 of the same pairs is the baseline, compare by /impl
func BenchmarkInt128DivToFloat64(b *testing.B) {
	for _, s := range DivFloatIn(128, true) {
		x, y := Slice128[Int128](Low128(s.X)), Slice128[Int128](Low128(s.Y))
		b.Run("bits="+s.Name+"/impl=fat", func(b *testing.B) {
			for i := range b.N {
				SinkFloat = x[i&BenchMask].DivToFloat64(y[i&BenchMask])
			}
		})
		b.Run("bits="+s.Name+"/impl=fatx", func(b *testing.B) {
			for i := range b.N {
				SinkFloat, SinkBool = x[i&BenchMask].DivToFloat64x(y[i&BenchMask])
			}
		})
		b.Run("bits="+s.Name+"/impl=big.Rat", func(b *testing.B) { BenchRatFloat(b, RefI128, x, y) })
	}
}
