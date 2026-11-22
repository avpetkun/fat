package fat

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"math"
	"math/big"
	"math/rand"
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

// TestInt256Encoding checks the binary, gob and JSON forms and round trips
func TestInt256Encoding(t *testing.T) {
	t.Parallel()
	Values256(func(x Int256) {
		t := On(t, RefI256(x))

		b, err := x.MarshalBinary()
		require.NoError(t, err)
		require.Equal(t, RefBE(RefI256(x), 32), b)
		ab, err := x.AppendBinary([]byte("ab"))
		require.NoError(t, err)
		require.Equal(t, append([]byte("ab"), b...), ab)

		y := Int256{7, 7, 7, 7}
		require.NoError(t, y.UnmarshalBinary(b))
		require.Equal(t, x, y)
	})

	require.Equal(t, Be(32, 0), BinaryOf(Int256{}))
	require.Equal(t, Be(32, 0, 0x7f), BinaryOf(NewInt256(127)))
	require.Equal(t, Be(32, 0xff, 0x80), BinaryOf(NewInt256(-128)))
	require.Equal(t, Be(32, 0xff), BinaryOf(NewInt256(-1)))
	require.Equal(t, 32, len(BinaryOf(MaxInt256())))
	RunBinaryLengths(t, kI256)

	// gob and json pick up the interfaces
	type doc struct{ A, B Int256 }
	in := doc{MinInt256(), MaxInt256()}

	var buf bytes.Buffer
	require.NoError(t, gob.NewEncoder(&buf).Encode(in))
	var out doc
	require.NoError(t, gob.NewDecoder(&buf).Decode(&out))
	require.Equal(t, in, out)

	js, err := json.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, `{"A":"`+BigMinInt256.String()+`","B":"`+BigMaxInt256.String()+`"}`, string(js))
	out = doc{}
	require.NoError(t, json.Unmarshal(js, &out))
	require.Equal(t, in, out)

	RunJSON(t, kI256)
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

		{Name: "Mul", Width: 256, Fn: Plain(Int256.Mul), Ref: BigMul, Panic: "fat.Int256.Mul: overflow"},
		{Name: "Mulu", Width: 256, Fn: Plain(Int256.Mulu), Ref: BigMul},
		{Name: "Mulo", Width: 256, Fn: Int256.Mulo, Ref: BigMul, Over: true},
		{Name: "Mul128", Width: 128, Fn: Plain(func(x, y Int256) Int256 { return x.Mul128(low128(y)) }), Ref: BigMul, Panic: "fat.Int256.Mul128: overflow"},
		{Name: "Mul128u", Width: 128, Fn: Plain(func(x, y Int256) Int256 { return x.Mul128u(low128(y)) }), Ref: BigMul},
		{Name: "Mul128o", Width: 128, Fn: func(x, y Int256) (Int256, bool) { return x.Mul128o(low128(y)) }, Ref: BigMul, Over: true},

		{Name: "And", Width: 256, Fn: Plain(Int256.And), Ref: BigAnd},
		{Name: "Or", Width: 256, Fn: Plain(Int256.Or), Ref: BigOr},
		{Name: "Xor", Width: 256, Fn: Plain(Int256.Xor), Ref: BigXor},
		{Name: "AndNot", Width: 256, Fn: Plain(Int256.AndNot), Ref: BigAndNot},
	})
}

// TestInt256Mule checks the full 512-bit product against big.Int
func TestInt256Mule(t *testing.T) {
	t.Parallel()
	Pairs256(func(x, y Int256) {
		xb, yb := RefI256(x), RefI256(y)
		hi, lo := x.Mule(y)
		// hi is signed, lo the unsigned low half
		require.Big(On(t, xb, yb), BigMul(xb, yb), BigAdd(new(big.Int).Lsh(RefI256(hi), 256), RefU256(lo)))
	})
}

// TestInt256MuloLimbs checks Mulo and Mul on every limb pattern and sign.
// mulFit branches on the zero limbs of |x| and |y|
func TestInt256MuloLimbs(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(3))
	limbs := func(mask int) Int256 {
		var l [4]uint64
		for i := range l {
			if mask>>i&1 != 0 {
				// small, full and random limbs, the top one within MaxInt256
				switch r.Intn(3) {
				case 0:
					l[i] = 1
				case 1:
					l[i] = math.MaxUint64
				default:
					l[i] = r.Uint64() | 1
				}
			}
		}
		l[3] &= math.MaxInt64
		x := Int256{l[0], l[1], l[2], l[3]}
		if r.Intn(2) == 0 {
			x = x.Neg()
		}
		return x
	}
	for range 50 {
		for mx := range 16 {
			for my := range 16 {
				x, y := limbs(mx), limbs(my)
				xb, yb := RefI256(x), RefI256(y)
				want := BigMul(xb, yb)
				t := On(t, xb, yb)
				got, c := x.Mulo(y)
				require.Big(t, WrapI256(want), RefI256(got))
				require.Equal(t, !FitsI256(want), c)
				if FitsI256(want) {
					require.Big(t, want, RefI256(x.Mul(y)))
				} else {
					require.Equal(t, "fat.Int256.Mul: overflow", PanicValue(func() { x.Mul(y) }))
				}
			}
		}
	}
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
	b.Run("op=Mul", func(b *testing.B) {
		x, y := in.fx, in.fy
		for i := range b.N {
			sinkI256 = x[i&BenchMask].Mul(y[i&BenchMask])
		}
	})
	b.Run("op=Mulo", func(b *testing.B) {
		x, y := in.i256, in.j256
		for i := range b.N {
			sinkI256, SinkBool = x[i&BenchMask].Mulo(y[i&BenchMask])
		}
	})
	b.Run("op=Mulu", func(b *testing.B) {
		x, y := in.i256, in.j256
		for i := range b.N {
			sinkI256 = x[i&BenchMask].Mulu(y[i&BenchMask])
		}
	})
	b.Run("op=Mule", func(b *testing.B) {
		x, y := in.i256, in.j256
		for i := range b.N {
			sinkI256, sinkU256 = x[i&BenchMask].Mule(y[i&BenchMask])
		}
	})
	b.Run("op=Mul128", func(b *testing.B) {
		x, y := in.fx128, in.i128
		for i := range b.N {
			sinkI256 = x[i&BenchMask].Mul128(y[i&BenchMask])
		}
	})
	b.Run("op=Mul128o", func(b *testing.B) {
		x, y := in.i256, in.i128
		for i := range b.N {
			sinkI256, SinkBool = x[i&BenchMask].Mul128o(y[i&BenchMask])
		}
	})
	b.Run("op=Mul128u", func(b *testing.B) {
		x, y := in.i256, in.i128
		for i := range b.N {
			sinkI256 = x[i&BenchMask].Mul128u(y[i&BenchMask])
		}
	})
	b.Run("op=Mul64", func(b *testing.B) {
		x, y := in.fx64, in.i64
		for i := range b.N {
			sinkI256 = x[i&BenchMask].Mul64(y[i&BenchMask])
		}
	})
	b.Run("op=DivMod", func(b *testing.B) {
		x, y := in.i256, in.j256
		for i := range b.N {
			sinkI256, sink2I256 = x[i&BenchMask].DivMod(y[i&BenchMask])
		}
	})
	b.Run("op=DivMod/fit", func(b *testing.B) {
		x, y := in.fx, in.fy
		for i := range b.N {
			sinkI256, sink2I256 = x[i&BenchMask].DivMod(y[i&BenchMask])
		}
	})
	b.Run("op=DivMod64", func(b *testing.B) {
		x, y := in.i256, in.i64
		for i := range b.N {
			sinkI256, SinkInt64 = x[i&BenchMask].DivMod64(y[i&BenchMask])
		}
	})
	b.Run("op=DivMod128", func(b *testing.B) {
		x, y := in.i256, in.i128
		for i := range b.N {
			sinkI256, sinkI128 = x[i&BenchMask].DivMod128(y[i&BenchMask])
		}
	})
	b.Run("op=MulDivo", func(b *testing.B) {
		x, y := in.i256, in.j256
		for i := range b.N {
			sinkI256, SinkBool = x[i&BenchMask].MulDivo(y[i&BenchMask], y[(i+1)&BenchMask])
		}
	})
	// |d| >= |y| keeps x * y / d within range: y with its low bit set away from zero
	mulDivD := sync.OnceValue(func() []Int256 {
		d := make([]Int256, len(in.j256))
		for i, v := range in.j256 {
			if v.IsNegative() {
				d[i] = v.AndNot(NewInt256(1))
			} else {
				d[i] = v.Or(NewInt256(1))
			}
		}
		return d
	})
	b.Run("op=MulDiv", func(b *testing.B) {
		x, y, d := in.i256, in.j256, mulDivD()
		for i := range b.N {
			sinkI256 = x[i&BenchMask].MulDiv(y[i&BenchMask], d[i&BenchMask])
		}
	})
	b.Run("op=MulDivCeil", func(b *testing.B) {
		x, y, d := in.i256, in.j256, mulDivD()
		for i := range b.N {
			sinkI256 = x[i&BenchMask].MulDivCeil(y[i&BenchMask], d[i&BenchMask])
		}
	})
	b.Run("op=MulMod", func(b *testing.B) {
		x, y := in.i256, in.j256
		for i := range b.N {
			sinkI256 = x[i&BenchMask].MulMod(y[i&BenchMask], y[(i+1)&BenchMask])
		}
	})
	b.Run("op=AddMod", func(b *testing.B) {
		x, y := in.i256, in.j256
		for i := range b.N {
			sinkI256 = x[i&BenchMask].AddMod(y[i&BenchMask], y[(i+1)&BenchMask])
		}
	})
	b.Run("op=SubMod", func(b *testing.B) {
		x, y := in.i256, in.j256
		for i := range b.N {
			sinkI256 = x[i&BenchMask].SubMod(y[i&BenchMask], y[(i+1)&BenchMask])
		}
	})
	b.Run("op=GCD", func(b *testing.B) {
		x, y := in.i256, in.j256
		for i := range b.N {
			sinkU256 = x[i&BenchMask].GCD(y[i&BenchMask])
		}
	})
	b.Run("op=Mul64o", func(b *testing.B) {
		x, y := in.i256, in.i64
		for i := range b.N {
			sinkI256, SinkBool = x[i&BenchMask].Mul64o(y[i&BenchMask])
		}
	})
	b.Run("op=Mul64u", func(b *testing.B) {
		x, y := in.i256, in.i64
		for i := range b.N {
			sinkI256 = x[i&BenchMask].Mul64u(y[i&BenchMask])
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

// BenchmarkInt256Decimal measures decimal text both ways on mixed lengths
func BenchmarkInt256Decimal(b *testing.B) {
	in := benchI256()
	b.Run("op=AppendText/digits=mixed", func(b *testing.B) {
		x := in.i256
		buf := make([]byte, 0, 100)
		for i := range b.N {
			buf, _ = x[i&BenchMask].AppendText(buf[:0])
		}
		SinkBytes = buf
	})
	b.Run("op=FromDec/digits=mixed", func(b *testing.B) {
		x := in.i256s
		for i := range b.N {
			sinkI256, _ = Int256FromDec(x[i&BenchMask])
		}
	})
	b.Run("op=String/alloc", func(b *testing.B) {
		x := in.i256
		for i := range b.N {
			SinkString = x[i&BenchMask].String()
		}
	})
	b.Run("op=MarshalText", func(b *testing.B) {
		x := in.i256
		var buf []byte
		for i := range b.N {
			buf, _ = x[i&BenchMask].MarshalText()
		}
		SinkBytes = buf
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
	b.Run("op=Float64", func(b *testing.B) {
		x := in.i256
		for i := range b.N {
			SinkFloat = x[i&BenchMask].Float64()
		}
	})
	b.Run("op=FromFloat64", func(b *testing.B) {
		x := make([]float64, len(in.i256))
		for i, v := range in.i256 {
			x[i] = v.Float64()
		}
		b.ResetTimer()
		for i := range b.N {
			sinkI256, SinkBool = Int256FromFloat64(x[i&BenchMask])
		}
	})
	b.Run("op=AppendBinary", func(b *testing.B) {
		x := in.i256
		buf := make([]byte, 0, 32)
		for i := range b.N {
			buf, _ = x[i&BenchMask].AppendBinary(buf[:0])
		}
		SinkBytes = buf
	})
	b.Run("op=MarshalBinary", func(b *testing.B) {
		x := in.i256
		var buf []byte
		for i := range b.N {
			buf, _ = x[i&BenchMask].MarshalBinary()
		}
		SinkBytes = buf
	})
	b.Run("op=UnmarshalBinary", func(b *testing.B) {
		x := make([][]byte, len(in.i256))
		for i, v := range in.i256 {
			x[i], _ = v.MarshalBinary()
		}
		b.ResetTimer()
		for i := range b.N {
			_ = sinkI256.UnmarshalBinary(x[i&BenchMask])
		}
	})
}

// BenchmarkInt256Binary measures AppendBinary and UnmarshalBinary
func BenchmarkInt256Binary(b *testing.B) {
	in := benchI256()
	b.Run("op=Append", func(b *testing.B) {
		x := in.i256
		buf := make([]byte, 0, 64)
		for i := range b.N {
			buf, _ = x[i&BenchMask].AppendBinary(buf[:0])
		}
		SinkBytes = buf
	})
	b.Run("op=Unmarshal", func(b *testing.B) {
		x := make([][]byte, len(in.i256))
		for i, v := range in.i256 {
			x[i], _ = v.MarshalBinary()
		}
		b.ResetTimer()
		for i := range b.N {
			_ = sinkI256.UnmarshalBinary(x[i&BenchMask])
		}
	})
}

// kI256 describes Int256 for the shared tests
var kI256 = Kind[Int256]{
	Name: "Int256", Bits: 256, Signed: true,
	Ref: RefI256, Wrap: WrapI256, Fits: FitsI256,
	FromBig: MustInt256FromBig, FromFloat64: Int256FromFloat64, MustFromFloat64: MustInt256FromFloat64, Small: func(v uint64) Int256 { return NewInt256(int64(v)) },
	FromDec: Int256FromDec, FromHex: Int256FromHex, MustFromDec: MustInt256FromDec, MustFromHex: MustInt256FromHex,
	Dirty: Int256{7, 7, 7, 7}, Max: MaxInt256(), Min: MinInt256(),
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
		Add: Signed64(Int256.Add64), Sub: Signed64(Int256.Sub64), Mul: Signed64(Int256.Mul64),
		Addo: Signed64Pair(Int256.Add64o), Subo: Signed64Pair(Int256.Sub64o), Mulo: Signed64Pair(Int256.Mul64o), Addu: Signed64(Int256.Add64u), Subu: Signed64(Int256.Sub64u), Mulu: Signed64(Int256.Mul64u),
	},
}

// checkInt256DivSigns checks Div, Mod and DivMod of +-x by +-y with big.Int.
// It skips a zero y and magnitudes of 2^255 and more
func checkInt256DivSigns(t *testing.T) func(x, y Uint256) {
	return func(x, y Uint256) {
		if (x.L3|y.L3)>>63 != 0 || y.IsZero() {
			return
		}
		for i := range 4 {
			xi, yi := x.withSign(-uint64(i&1)), y.withSign(-uint64(i>>1))
			xb, yb := RefI256(xi), RefI256(yi)
			t := On(t, xb, yb)
			qb, rb := new(big.Int).QuoRem(xb, yb, new(big.Int))
			q, r := xi.DivMod(yi)
			require.Big(t, qb, RefI256(q))
			require.Big(t, rb, RefI256(r))
			require.Equal(t, q, xi.Div(yi))
			require.Equal(t, r, xi.Mod(yi))
		}
	}
}

// TestInt256DivMod checks every Div, Mod and DivMod form against big.Int.
// The divisors are 256, 128 and 64 bits wide, of either sign
func TestInt256DivMod(t *testing.T) {
	t.Parallel()
	// check compares one division form with big.Int, the overflow included
	check := func(t testing.TB, name string, xb, yb *big.Int, div func() Int256, divo func() (Int256, bool), divu func() Int256,
		mod func() *big.Int, divmod func() (Int256, *big.Int)) {
		qb, rb := new(big.Int).QuoRem(xb, yb, new(big.Int))
		over := !BigFits(qb, 256, true)
		wrap := WrapI256(qb)
		q, o := divo()
		require.Big(t, wrap, RefI256(q))
		require.Equal(t, over, o)
		require.Big(t, wrap, RefI256(divu()))
		require.Big(t, rb, mod())
		if over {
			require.Equal(t, "fat.Int256."+name+": overflow", PanicValue(func() { div() }))
			require.Equal(t, "fat.Int256.DivMod"+name[3:]+": overflow", PanicValue(func() { divmod() }))
			return
		}
		require.Big(t, qb, RefI256(div()))
		q, r := divmod()
		require.Big(t, qb, RefI256(q))
		require.Big(t, rb, r)
	}
	check64 := func(x Int256, v int64) {
		xb, vb := RefI256(x), big.NewInt(v)
		check(On(t, xb, vb), "Div64", xb, vb, func() Int256 { return x.Div64(v) }, func() (Int256, bool) { return x.Div64o(v) },
			func() Int256 { return x.Div64u(v) }, func() *big.Int { return big.NewInt(x.Mod64(v)) },
			func() (Int256, *big.Int) { q, r := x.DivMod64(v); return q, big.NewInt(r) })
	}
	Pairs256(func(x, y Int256) {
		xb := RefI256(x)
		if !y.IsZero() {
			yb := RefI256(y)
			check(On(t, xb, yb), "Div", xb, yb, func() Int256 { return x.Div(y) }, func() (Int256, bool) { return x.Divo(y) },
				func() Int256 { return x.Divu(y) }, func() *big.Int { return RefI256(x.Mod(y)) },
				func() (Int256, *big.Int) { q, r := x.DivMod(y); return q, RefI256(r) })
		}
		if v := int64(y.L0); v != 0 {
			check64(x, v)
		}
		if v := (Int128{y.L0, y.L1}); !v.IsZero() {
			vb := RefI128(v)
			check(On(t, xb, vb), "Div128", xb, vb, func() Int256 { return x.Div128(v) }, func() (Int256, bool) { return x.Div128o(v) },
				func() Int256 { return x.Div128u(v) }, func() *big.Int { return RefI128(x.Mod128(v)) },
				func() (Int256, *big.Int) { q, r := x.DivMod128(v); return q, RefI128(r) })
		}
	})
	// every step of the reciprocal with a divisor of either sign:
	// a limb one under the divisor shows a reciprocal that is one short
	for _, d := range Divisors64 {
		v := int64(d >> 1)
		for _, l := range []uint64{0, 1, 1<<63 - 1, 1 << 63, math.MaxUint64 - 1, math.MaxUint64} {
			if v != 0 {
				check64(Int256{l, uint64(v) - 1, l, uint64(v) - 1}, v)
				check64(Int256{l, l, uint64(v) - 1, 0}, -v)
			}
		}
	}
	// unsigned pairs as magnitudes reach the rare steps of the Int256 kernels
	Pairs256(checkInt256DivSigns(t))
	// 2-limb divisors: the first digit one too big,
	// the rare second adjustment of the last 3-by-2 step
	for _, p := range [][2]Uint256{
		{{1, 2, 4, 7}, {5, 7, 0, 0}},
		{{0xff4, 1 << 63, 1 << 63, 0}, {4, 0x414f5b3ec, 0, 0}},
		{{2, MaxUint256().L1, MaxUint256().L2, 10}, {1 << 63, 0x8ac, 0, 0}},
	} {
		checkInt256DivSigns(t)(p[0], p[1])
	}
	// MinInt256 / -1 is the one overflow, the u and 64-bit forms have it too
	q, over := MinInt256().Divo(NewInt256(-1))
	require.Equal(t, MinInt256(), q)
	require.True(t, over)
	require.Equal(t, MinInt256(), MinInt256().Div64u(-1))
	require.Equal(t, int64(0), MinInt256().Mod64(-1))
	for name, fn := range map[string]func(){
		"Div":       func() { MaxInt256().Div(Int256{}) },
		"Divo":      func() { MaxInt256().Divo(Int256{}) },
		"Divu":      func() { MaxInt256().Divu(Int256{}) },
		"Mod":       func() { MaxInt256().Mod(Int256{}) },
		"DivMod":    func() { MaxInt256().DivMod(Int256{}) },
		"Div64":     func() { MaxInt256().Div64(0) },
		"Div64o":    func() { MaxInt256().Div64o(0) },
		"Div64u":    func() { MaxInt256().Div64u(0) },
		"Mod64":     func() { MaxInt256().Mod64(0) },
		"DivMod64":  func() { MaxInt256().DivMod64(0) },
		"Div128":    func() { MaxInt256().Div128(Int128{}) },
		"Div128o":   func() { MaxInt256().Div128o(Int128{}) },
		"Div128u":   func() { MaxInt256().Div128u(Int128{}) },
		"Mod128":    func() { MaxInt256().Mod128(Int128{}) },
		"DivMod128": func() { MaxInt256().DivMod128(Int128{}) },
	} {
		require.Equal(On(t, name), "fat.Int256."+name+": division by zero", PanicValue(fn))
	}
}

// TestInt256MulDiv checks MulDiv in all forms, MulDivCeil, MulMod, AddMod and SubMod with big.Int.
// AddMod also gets sums at m and 2m and sums that overflow
func TestInt256MulDiv(t *testing.T) {
	t.Parallel()
	var ds []Int256
	Values256(func(d Int256) {
		if !d.IsZero() {
			ds = append(ds, d)
		}
	})
	i := 0
	Pairs256(func(x, y Int256) {
		i++
		xb, yb := RefI256(x), RefI256(y)
		pb := new(big.Int).Mul(xb, yb)
		for _, d := range []Int256{ds[i%len(ds)], ds[(i*7+3)%len(ds)], y, x} {
			if d.IsZero() {
				continue
			}
			db := RefI256(d)
			t := On(t, xb, yb, db)
			qb, rb := new(big.Int).QuoRem(pb, db, new(big.Int))
			over := !BigFits(qb, 256, true)
			q, o := x.MulDivo(y, d)
			require.Big(t, WrapI256(qb), RefI256(q))
			require.Equal(t, over, o)
			require.Equal(t, q, x.MulDivu(y, d))
			if over {
				require.Equal(t, "fat.Int256.MulDiv: overflow", PanicValue(func() { x.MulDiv(y, d) }))
				require.Equal(t, "fat.Int256.MulDivMod: overflow", PanicValue(func() { x.MulDivMod(y, d) }))
			} else {
				require.Big(t, qb, RefI256(x.MulDiv(y, d)))
				q, r := x.MulDivMod(y, d)
				require.Big(t, qb, RefI256(q))
				require.Big(t, rb, RefI256(r))
				// MulDiv is Mul then Div where Mul does not overflow
				if p, o := x.Mulo(y); !o {
					require.Equal(t, p.Div(d), q)
				}
			}
			// the sign of the dividend as Go's %
			require.Big(t, new(big.Int).Rem(pb, db), RefI256(x.MulMod(y, d)))
			require.Big(t, new(big.Int).Rem(new(big.Int).Add(xb, yb), db), RefI256(x.AddMod(y, d)))
			require.Big(t, new(big.Int).Rem(new(big.Int).Sub(xb, yb), db), RefI256(x.SubMod(y, d)))
			// MulDivCeil is the quotient plus one where x * y / d has a positive fraction
			cb := new(big.Int).Set(qb)
			if rb.Sign() != 0 && rb.Sign() == db.Sign() {
				cb.Add(cb, Big1)
			}
			if BigFits(cb, 256, true) {
				require.Big(t, cb, RefI256(x.MulDivCeil(y, d)))
			} else {
				require.Equal(t, "fat.Int256.MulDivCeil: overflow", PanicValue(func() { x.MulDivCeil(y, d) }))
			}
		}
	})
	// sums at m and 2m of either sign need no division
	Values256(func(m Int256) {
		if m.IsZero() {
			return
		}
		for _, x := range []Int256{{}, m.Rsh(1), m, m.Not(), MaxInt256(), MinInt256()} {
			for _, sum := range []Int256{m.Sub64u(1), m, m.Add64u(1), m.Addu(m).Sub64u(1), m.Addu(m), m.Addu(m).Add64u(1)} {
				for _, y := range []Int256{sum.Subu(x), Int256{}.Subu(sum).Subu(x)} {
					want := new(big.Int).Rem(new(big.Int).Add(RefI256(x), RefI256(y)), RefI256(m))
					require.Big(On(t, x, y, m), want, RefI256(x.AddMod(y, m)))
				}
			}
		}
	})
	// an overflowing sum of one sign by small, negative and extreme moduli
	for _, m := range []Int256{NewInt256(1), NewInt256(-1), NewInt256(3), NewInt256(-3), MaxInt256(), MinInt256()} {
		for _, x := range []Int256{MaxInt256(), MaxInt256().Sub64(1), MinInt256(), MinInt256().Add64(1)} {
			want := new(big.Int).Rem(new(big.Int).Add(RefI256(x), RefI256(x)), RefI256(m))
			require.Big(On(t, x, m), want, RefI256(x.AddMod(x, m)))
		}
	}
	// differences at -2m, -m, 0, m and 2m and one off, with operands below m or not
	for _, m := range ds {
		for _, x := range []Int256{{}, m.Rsh(1), m, m.Not(), MaxInt256(), MinInt256()} {
			for _, k := range []Int256{{}, m, m.Addu(m)} {
				for _, y := range []Int256{x.Subu(k), x.Subu(k).Add64u(1), x.Subu(k).Sub64u(1), x.Addu(k), x.Addu(k).Add64u(1), x.Addu(k).Sub64u(1)} {
					want := new(big.Int).Rem(new(big.Int).Sub(RefI256(x), RefI256(y)), RefI256(m))
					require.Big(On(t, x, y, m), want, RefI256(x.SubMod(y, m)))
				}
			}
		}
	}
	// (2^129 - 1)(2^129 + 1) / 8 is the maximum and 7/8: it fits truncated, not rounded up.
	// Negated it rounds up to -maximum
	{
		h := NewInt256(1).Lsh(129)
		x, y, d := h.Sub64(1), h.Add64(1), NewInt256(8)
		require.Equal(t, MaxInt256(), x.MulDiv(y, d))
		require.Equal(t, "fat.Int256.MulDivCeil: overflow", PanicValue(func() { x.MulDivCeil(y, d) }))
		require.Equal(t, "fat.Int256.MulDivCeil: overflow", PanicValue(func() { x.Neg().MulDivCeil(y, d.Neg()) }))
		require.Equal(t, MaxInt256().Neg(), x.Neg().MulDivCeil(y, d))
		require.Equal(t, MaxInt256().Neg(), x.MulDivCeil(y, d.Neg()))
	}
	// a product past 256 bits and one within them take different paths to d
	for _, x := range []Int256{MaxInt256(), NewInt256(-3)} {
		for name, fn := range map[string]func(){
			"MulDiv":     func() { x.MulDiv(x, Int256{}) },
			"MulDivo":    func() { x.MulDivo(x, Int256{}) },
			"MulDivu":    func() { x.MulDivu(x, Int256{}) },
			"MulDivMod":  func() { x.MulDivMod(x, Int256{}) },
			"MulDivCeil": func() { x.MulDivCeil(x, Int256{}) },
			"AddMod":     func() { x.AddMod(x, Int256{}) },
			"SubMod":     func() { x.SubMod(x, Int256{}) },
			"MulMod":     func() { x.MulMod(x, Int256{}) },
		} {
			require.Equal(On(t, x, name), "fat.Int256."+name+": division by zero", PanicValue(fn))
		}
	}
}

// TestInt256Fmt checks Format against big.Int on the fmt verbs and flags
func TestInt256Fmt(t *testing.T) { RunFmt(t, kI256) }

// TestInt256ParseErrors checks FromDec on malformed and out of range text
func TestInt256ParseErrors(t *testing.T) { RunParseErrors(t, kI256) }

// TestInt256Bits checks BitLen, Not, shifts, Bit and WithBit against big.Int
func TestInt256Bits(t *testing.T) { RunBits(t, kI256) }

// TestInt256Cmp checks the comparisons and CmpBig against big.Int
func TestInt256Cmp(t *testing.T) { RunCmp(t, kI256) }

// TestInt256GCD checks GCD with big.Int, MinInt256 with 0 among the pairs
func TestInt256GCD(t *testing.T) {
	RunGCD(t, kI256, func(x, y Int256) *big.Int { return RefU256(x.GCD(y)) })
}

// TestInt256Decimal checks decimal text both ways against big.Int
func TestInt256Decimal(t *testing.T) { RunDecimal(t, kI256) }

// TestInt256Sci checks the form m e n of FromDec
func TestInt256Sci(t *testing.T) { RunSci(t, kI256) }

// TestInt256Hex checks hex text both ways against big.Int
func TestInt256Hex(t *testing.T) { RunHex(t, kI256) }

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

// TestInt256ErrorText checks that parse errors name the type and the input
func TestInt256ErrorText(t *testing.T) { RunErrorText(t, kI256) }

// TestInt256PanicText checks that every panic names its func or method
func TestInt256PanicText(t *testing.T) { RunPanicText(t, kI256) }

// TestInt256Exp checks Exp, Expo and Expu against big.Int
func TestInt256Exp(t *testing.T) { RunExp(t, kI256) }

// TestInt256ExpMod checks ExpMod against big.Int
func TestInt256ExpMod(t *testing.T) { RunExpMod(t, kI256) }

// TestInt256Sqrt checks Sqrt against big.Int at and next to squares
func TestInt256Sqrt(t *testing.T) { RunSqrt(t, kI256) }

// TestInt256Log10 checks Log10 at every power of ten and bit length
func TestInt256Log10(t *testing.T) { RunLog10(t, kI256) }

// TestInt256Float checks Float64 and FromFloat64 against big.Float
func TestInt256Float(t *testing.T) { RunFloat(t, kI256) }

// TestInt256DivFloat checks DivToFloat64 against big.Rat at rounding ties
func TestInt256DivFloat(t *testing.T) { RunDivFloat(t, kI256) }

// TestInt256Constants checks that untyped constants of either sign compile
func TestInt256Constants(t *testing.T) {
	t.Parallel()
	x := NewInt256(-5)
	require.Equal(t, NewInt256(-3), x.Add64(2))
	require.Equal(t, NewInt256(-7), x.Sub64(2))
	require.Equal(t, NewInt256(10), x.Mul64(-2))
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
		b, err := x.MarshalBinary()
		require.NoError(t, err)
		w, err := x.Int256().MarshalBinary()
		require.NoError(t, err)
		ext := byte(0)
		if x.IsNegative() {
			ext = 0xff
		}
		require.Equal(t, append(Be(16, ext), b...), w)
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

// BenchmarkInt256Pow measures Exp with results that fit, Sqrt and Log10 of |x|.
// Expu/full takes full-width operands that wrap
func BenchmarkInt256Pow(b *testing.B) {
	in := benchI256()
	l, e := ExpIn(256, true)
	ex, ey := Slice256[Int256](l), make([]Int256, len(e))
	for i, v := range e {
		ey[i] = NewInt256(int64(v))
	}
	b.Run("op=Exp", func(b *testing.B) {
		for i := range b.N {
			sinkI256 = ex[i&BenchMask].Exp(ey[i&BenchMask])
		}
	})
	b.Run("op=Expo", func(b *testing.B) {
		for i := range b.N {
			sinkI256, SinkBool = ex[i&BenchMask].Expo(ey[i&BenchMask])
		}
	})
	b.Run("op=Expu", func(b *testing.B) {
		for i := range b.N {
			sinkI256 = ex[i&BenchMask].Expu(ey[i&BenchMask])
		}
	})
	b.Run("op=Expu/full", func(b *testing.B) {
		x, y := in.i256, in.j256
		for i := range b.N {
			sinkI256 = x[i&BenchMask].Expu(y[i&BenchMask].Abs())
		}
	})
	a := make([]Int256, len(in.i256))
	for i, v := range in.i256 {
		a[i] = v.Abs()
	}
	b.Run("op=Sqrt", func(b *testing.B) {
		for i := range b.N {
			sinkI256 = a[i&BenchMask].Sqrt()
		}
	})
	b.Run("op=Log10", func(b *testing.B) {
		for i := range b.N {
			SinkInt = a[i&BenchMask].Log10()
		}
	})
}

// BenchmarkInt256DivToFloat64 measures DivToFloat64 and DivToFloat64x
// next to big.Rat.Float64 on the same pairs by bit length.
// Compare with benchstat -col /impl
func BenchmarkInt256DivToFloat64(b *testing.B) {
	for _, s := range DivFloatIn(256, true) {
		x, y := Slice256[Int256](s.X), Slice256[Int256](s.Y)
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
		b.Run("bits="+s.Name+"/impl=big.Rat", func(b *testing.B) { BenchRatFloat(b, RefI256, x, y) })
	}
}
