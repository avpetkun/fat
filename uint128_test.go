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

// TestUint128Convert checks the conversions to other types and big.Int
func TestUint128Convert(t *testing.T) {
	t.Parallel()
	require.Equal(t, "340282366920938463463374607431768211455", MaxUint128().String())

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
}

// TestUint128Encoding checks the 16-byte binary form, gob and JSON
func TestUint128Encoding(t *testing.T) {
	t.Parallel()
	Values128(func(x Uint128) {
		t := On(t, RefU128(x))

		b, err := x.MarshalBinary()
		require.NoError(t, err)
		require.Equal(t, RefBE(RefU128(x), 16), b)
		ab, err := x.AppendBinary([]byte("ab"))
		require.NoError(t, err)
		require.Equal(t, append([]byte("ab"), b...), ab)

		y := Uint128{7, 7}
		require.NoError(t, y.UnmarshalBinary(b))
		require.Equal(t, x, y)
	})

	require.Equal(t, Be(16, 0), BinaryOf(Uint128{}))
	require.Equal(t, Be(16, 0, 0x01, 0x2c), BinaryOf(NewUint128(300)))
	require.Equal(t, Be(16, 0, 8, 7, 6, 5, 4, 3, 2, 1), BinaryOf(NewUint128(0x0807060504030201)))
	require.Equal(t, Be(16, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1), BinaryOf(Uint128{1, 1}))
	require.Equal(t, 16, len(BinaryOf(MaxUint128())))
	RunBinaryLengths(t, kU128)
	// a failed decode leaves the value as it was
	y := Uint128{7, 7}
	require.Equal(t, "fat.Uint128.UnmarshalBinary: cannot parse 17 bytes, want 16", y.UnmarshalBinary(make([]byte, 17)).Error())
	require.Equal(t, "fat.Uint128.GobDecode: cannot parse 0 bytes, want 16", y.GobDecode(nil).Error())
	require.Equal(t, Uint128{7, 7}, y)

	// gob and json pick up the interfaces
	type doc struct{ A, B Uint128 }
	in := doc{MaxUint128(), Uint128{1, 2}}

	var buf bytes.Buffer
	require.NoError(t, gob.NewEncoder(&buf).Encode(in))
	var out doc
	require.NoError(t, gob.NewDecoder(&buf).Decode(&out))
	require.Equal(t, in, out)

	js, err := json.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, `{"A":"`+MaxUint128().String()+`","B":"`+RefU128(in.B).String()+`"}`, string(js))
	out = doc{}
	require.NoError(t, json.Unmarshal(js, &out))
	require.Equal(t, in, out)

	RunJSON(t, kU128)
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

		{Name: "Mul", Width: 128, Fn: Plain(Uint128.Mul), Ref: BigMul, Panic: "fat.Uint128.Mul: overflow"},
		{Name: "Mulu", Width: 128, Fn: Plain(Uint128.Mulu), Ref: BigMul},
		{Name: "Mulo", Width: 128, Fn: Uint128.Mulo, Ref: BigMul, Over: true},

		{Name: "And", Width: 128, Fn: Plain(Uint128.And), Ref: BigAnd},
		{Name: "Or", Width: 128, Fn: Plain(Uint128.Or), Ref: BigOr},
		{Name: "Xor", Width: 128, Fn: Plain(Uint128.Xor), Ref: BigXor},
		{Name: "AndNot", Width: 128, Fn: Plain(Uint128.AndNot), Ref: BigAndNot},
	})
}

// TestUint128Mule checks the full 256-bit product with big.Int
func TestUint128Mule(t *testing.T) {
	t.Parallel()
	Pairs128(func(x, y Uint128) {
		xb, yb := RefU128(x), RefU128(y)
		require.Big(On(t, xb, yb), BigMul(xb, yb), RefU256(x.Mule(y)))
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

// TestUint128DivMod64 checks division by a uint64 with big.Int and by zero
func TestUint128DivMod64(t *testing.T) {
	t.Parallel()
	check := func(x Uint128, y uint64) {
		xb, yb := RefU128(x), new(big.Int).SetUint64(y)
		t := On(t, xb, yb)
		qb, rb := new(big.Int).QuoRem(xb, yb, new(big.Int))
		q, r := x.DivMod64(y)
		require.Big(t, qb, RefU128(q))
		require.Equal(t, rb.Uint64(), r)
		require.Equal(t, q, x.Div64(y))
		require.Equal(t, r, x.Mod64(y))
	}
	Values128(func(x Uint128) {
		for _, y := range Divisors64 {
			check(x, y)
		}
	})
	for _, x := range Edge128 {
		for _, y := range SeedDivisors64 {
			check(Limbs128[Uint128](x), y)
		}
	}
	require.Equal(t, "fat.Uint128.Div64: division by zero", PanicValue(func() { MaxUint128().Div64(0) }))
	require.Equal(t, "fat.Uint128.Mod64: division by zero", PanicValue(func() { MaxUint128().Mod64(0) }))
	require.Equal(t, "fat.Uint128.DivMod64: division by zero", PanicValue(func() { MaxUint128().DivMod64(0) }))
}

// divPairs128 calls fn with pairs at the borders of the division paths.
// Limbs lie near 0, 2^32, 2^33, 2^63 and 2^64, x within 2 of a multiple of y
func divPairs128(fn func(x, y Uint128)) {
	limbs := []uint64{0, 1, 2, 1<<32 - 1, 1 << 32, 1<<33 - 1, 1 << 33, 1<<33 + 1, 1<<63 - 1, 1 << 63,
		math.MaxUint64 - 1, math.MaxUint64}
	for _, y1 := range limbs {
		for _, y0 := range limbs {
			y := Uint128{y0, y1}
			for _, k := range limbs {
				fn(Uint128{y0, k}, y)
				fn(Uint128{^y0, k}, y)
				m := y.Mulu(Uint128{k, 0})
				for d := range uint64(5) {
					fn(m.Add64u(d).Sub64u(2), y)
				}
			}
		}
	}
}

// TestUint128DivMod checks Div, Mod and DivMod with big.Int and by zero
func TestUint128DivMod(t *testing.T) {
	t.Parallel()
	check := func(x, y Uint128) {
		xb, yb := RefU128(x), RefU128(y)
		t := On(t, xb, yb)
		if y.IsZero() {
			return
		}
		qb, rb := new(big.Int).QuoRem(xb, yb, new(big.Int))
		q, r := x.DivMod(y)
		require.Big(t, qb, RefU128(q))
		require.Big(t, rb, RefU128(r))
		require.Equal(t, q, x.Div(y))
		require.Equal(t, r, x.Mod(y))
	}
	Pairs128(check)
	divPairs128(check)
	require.Equal(t, "fat.Uint128.Div: division by zero", PanicValue(func() { MaxUint128().Div(Uint128{}) }))
	require.Equal(t, "fat.Uint128.Mod: division by zero", PanicValue(func() { MaxUint128().Mod(Uint128{}) }))
	require.Equal(t, "fat.Uint128.DivMod: division by zero", PanicValue(func() { MaxUint128().DivMod(Uint128{}) }))
}

// TestUint128MulDiv checks MulDiv, MulDivCeil, MulMod, AddMod and SubMod with big.Int and by zero
func TestUint128MulDiv(t *testing.T) {
	t.Parallel()
	var ds []Uint128
	Values128(func(d Uint128) {
		if !d.IsZero() {
			ds = append(ds, d)
		}
	})
	i := 0
	Pairs128(func(x, y Uint128) {
		i++
		xb, yb := RefU128(x), RefU128(y)
		pb := new(big.Int).Mul(xb, yb)
		for _, d := range []Uint128{ds[i%len(ds)], ds[(i*7+3)%len(ds)], y, x} {
			if d.IsZero() {
				continue
			}
			db := RefU128(d)
			t := On(t, xb, yb, db)
			qb, rb := new(big.Int).QuoRem(pb, db, new(big.Int))
			over := !BigFits(qb, 128, false)
			q, o := x.MulDivo(y, d)
			require.Big(t, WrapU128(qb), RefU128(q))
			require.Equal(t, over, o)
			require.Equal(t, q, x.MulDivu(y, d))
			if over {
				require.Equal(t, "fat.Uint128.MulDiv: overflow", PanicValue(func() { x.MulDiv(y, d) }))
				require.Equal(t, "fat.Uint128.MulDivMod: overflow", PanicValue(func() { x.MulDivMod(y, d) }))
			} else {
				require.Big(t, qb, RefU128(x.MulDiv(y, d)))
				q, r := x.MulDivMod(y, d)
				require.Big(t, qb, RefU128(q))
				require.Big(t, rb, RefU128(r))
				// MulDiv is Mul then Div where Mul does not overflow
				if p, o := x.Mulo(y); !o {
					require.Equal(t, p.Div(d), q)
				}
			}
			require.Big(t, new(big.Int).Mod(pb, db), RefU128(x.MulMod(y, d)))
			require.Big(t, new(big.Int).Mod(new(big.Int).Add(xb, yb), db), RefU128(x.AddMod(y, d)))
			require.Big(t, new(big.Int).Mod(new(big.Int).Sub(xb, yb), db), RefU128(x.SubMod(y, d)))
			// MulDivCeil is the quotient plus one where x * y / d has a positive fraction
			cb := new(big.Int).Set(qb)
			if rb.Sign() != 0 && rb.Sign() == db.Sign() {
				cb.Add(cb, Big1)
			}
			if BigFits(cb, 128, false) {
				require.Big(t, cb, RefU128(x.MulDivCeil(y, d)))
			} else {
				require.Equal(t, "fat.Uint128.MulDivCeil: overflow", PanicValue(func() { x.MulDivCeil(y, d) }))
			}
		}
	})
	// a carried sum by small moduli, 1 included
	for _, m := range []Uint128{NewUint128(1), NewUint128(2), NewUint128(3), MaxUint128(), MaxUint128().Sub64(1)} {
		for _, x := range []Uint128{MaxUint128(), MaxUint128().Sub64(1), MaxUint128().Rsh(1).Add64(1)} {
			want := new(big.Int).Mod(new(big.Int).Add(RefU128(x), RefU128(MaxUint128())), RefU128(m))
			require.Big(On(t, x, m), want, RefU128(x.AddMod(MaxUint128(), m)))
		}
	}
	// operands below m with and without a carry,
	// a carried sum with m shifted into it once and twice
	divPairs128(func(x, m Uint128) {
		if m.IsZero() {
			return
		}
		for _, y := range []Uint128{x, x.Not(), m.Sub64u(1), MaxUint128()} {
			want := new(big.Int).Mod(new(big.Int).Add(RefU128(x), RefU128(y)), RefU128(m))
			require.Big(On(t, x, y, m), want, RefU128(x.AddMod(y, m)))
		}
	})
	// differences at -2m, -m, 0, m and 2m and one off, with operands below m or not
	for _, m := range ds {
		for _, x := range []Uint128{{}, m.Rsh(1), m.Sub64u(1), m, MaxUint128()} {
			for _, k := range []Uint128{{}, m, m.Addu(m)} {
				for _, y := range []Uint128{x.Subu(k), x.Subu(k).Add64u(1), x.Subu(k).Sub64u(1), x.Addu(k), x.Addu(k).Add64u(1), x.Addu(k).Sub64u(1)} {
					want := new(big.Int).Mod(new(big.Int).Sub(RefU128(x), RefU128(y)), RefU128(m))
					require.Big(On(t, x, y, m), want, RefU128(x.SubMod(y, m)))
				}
			}
		}
	}
	// (2^65 - 1)(2^65 + 1) / 4 is the maximum and 3/4: it fits truncated, not rounded up
	{
		h := NewUint128(1).Lsh(65)
		x, y, d := h.Sub64(1), h.Add64(1), NewUint128(4)
		require.Equal(t, MaxUint128(), x.MulDiv(y, d))
		require.Equal(t, "fat.Uint128.MulDivCeil: overflow", PanicValue(func() { x.MulDivCeil(y, d) }))
	}
	// a product past 128 bits and one within them take different paths to d
	for _, x := range []Uint128{MaxUint128(), {L0: 3}} {
		for name, fn := range map[string]func(){
			"MulDiv":     func() { x.MulDiv(x, Uint128{}) },
			"MulDivo":    func() { x.MulDivo(x, Uint128{}) },
			"MulDivu":    func() { x.MulDivu(x, Uint128{}) },
			"MulDivMod":  func() { x.MulDivMod(x, Uint128{}) },
			"MulDivCeil": func() { x.MulDivCeil(x, Uint128{}) },
			"AddMod":     func() { x.AddMod(x, Uint128{}) },
			"SubMod":     func() { x.SubMod(x, Uint128{}) },
			"MulMod":     func() { x.MulMod(x, Uint128{}) },
		} {
			require.Equal(On(t, x, name), "fat.Uint128."+name+": division by zero", PanicValue(fn))
		}
	}
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
	b.Run("op=Mul", func(b *testing.B) {
		x, y := in.fx, in.fy
		for i := range b.N {
			sinkU128 = x[i&BenchMask].Mul(y[i&BenchMask])
		}
	})
	b.Run("op=Mulo", func(b *testing.B) {
		x, y := in.u128, in.v128
		for i := range b.N {
			sinkU128, SinkBool = x[i&BenchMask].Mulo(y[i&BenchMask])
		}
	})
	b.Run("op=Mulu", func(b *testing.B) {
		x, y := in.u128, in.v128
		for i := range b.N {
			sinkU128 = x[i&BenchMask].Mulu(y[i&BenchMask])
		}
	})
	b.Run("op=Mule", func(b *testing.B) {
		x, y := in.u128, in.v128
		for i := range b.N {
			sinkU256 = x[i&BenchMask].Mule(y[i&BenchMask])
		}
	})
	b.Run("op=Mul64", func(b *testing.B) {
		x, y := in.fx64, in.u64
		for i := range b.N {
			sinkU128 = x[i&BenchMask].Mul64(y[i&BenchMask])
		}
	})
	b.Run("op=Mul64o", func(b *testing.B) {
		x, y := in.u128, in.u64
		for i := range b.N {
			sinkU128, SinkBool = x[i&BenchMask].Mul64o(y[i&BenchMask])
		}
	})
	b.Run("op=Mul64u", func(b *testing.B) {
		x, y := in.u128, in.u64
		for i := range b.N {
			sinkU128 = x[i&BenchMask].Mul64u(y[i&BenchMask])
		}
	})
	b.Run("op=DivMod", func(b *testing.B) {
		x, y := in.u128, in.v128
		for i := range b.N {
			sinkU128, sink2U128 = x[i&BenchMask].DivMod(y[i&BenchMask])
		}
	})
	b.Run("op=DivMod/fit", func(b *testing.B) {
		x, y := in.fx, in.fy
		for i := range b.N {
			sinkU128, sink2U128 = x[i&BenchMask].DivMod(y[i&BenchMask])
		}
	})
	b.Run("op=Div", func(b *testing.B) {
		x, y := in.u128, in.v128
		for i := range b.N {
			sinkU128 = x[i&BenchMask].Div(y[i&BenchMask])
		}
	})
	b.Run("op=Mod", func(b *testing.B) {
		x, y := in.u128, in.v128
		for i := range b.N {
			sinkU128 = x[i&BenchMask].Mod(y[i&BenchMask])
		}
	})
	b.Run("op=MulDivo", func(b *testing.B) {
		x, y := in.u128, in.v128
		for i := range b.N {
			sinkU128, SinkBool = x[i&BenchMask].MulDivo(y[i&BenchMask], y[(i+1)&BenchMask])
		}
	})
	// |d| >= |y| keeps x * y / d within range: y with its low bit set away from zero
	mulDivD := sync.OnceValue(func() []Uint128 {
		d := make([]Uint128, len(in.v128))
		for i, v := range in.v128 {
			d[i] = v.Or(NewUint128(1))
		}
		return d
	})
	b.Run("op=MulDiv", func(b *testing.B) {
		x, y, d := in.u128, in.v128, mulDivD()
		for i := range b.N {
			sinkU128 = x[i&BenchMask].MulDiv(y[i&BenchMask], d[i&BenchMask])
		}
	})
	b.Run("op=MulDivCeil", func(b *testing.B) {
		x, y, d := in.u128, in.v128, mulDivD()
		for i := range b.N {
			sinkU128 = x[i&BenchMask].MulDivCeil(y[i&BenchMask], d[i&BenchMask])
		}
	})
	b.Run("op=MulMod", func(b *testing.B) {
		x, y := in.u128, in.v128
		for i := range b.N {
			sinkU128 = x[i&BenchMask].MulMod(y[i&BenchMask], y[(i+1)&BenchMask])
		}
	})
	b.Run("op=AddMod", func(b *testing.B) {
		x, y := in.u128, in.v128
		for i := range b.N {
			sinkU128 = x[i&BenchMask].AddMod(y[i&BenchMask], y[(i+1)&BenchMask])
		}
	})
	b.Run("op=SubMod", func(b *testing.B) {
		x, y := in.u128, in.v128
		for i := range b.N {
			sinkU128 = x[i&BenchMask].SubMod(y[i&BenchMask], y[(i+1)&BenchMask])
		}
	})
	b.Run("op=GCD", func(b *testing.B) {
		x, y := in.u128, in.v128
		for i := range b.N {
			sinkU128 = x[i&BenchMask].GCD(y[i&BenchMask])
		}
	})
	// nonzero moduli of either parity: y with bit 1 set
	invM := sync.OnceValue(func() []Uint128 {
		m := make([]Uint128, len(in.v128))
		for i, v := range in.v128 {
			m[i] = v.Or(NewUint128(2))
		}
		return m
	})
	b.Run("op=ModInverse", func(b *testing.B) {
		x, m := in.u128, invM()
		for i := range b.N {
			sinkU128, SinkBool = x[i&BenchMask].ModInverse(m[i&BenchMask])
		}
	})
	b.Run("op=Div64", func(b *testing.B) {
		x, y := in.u128, in.u64
		for i := range b.N {
			sinkU128 = x[i&BenchMask].Div64(y[i&BenchMask])
		}
	})
	b.Run("op=Mod64", func(b *testing.B) {
		x, y := in.u128, in.u64
		for i := range b.N {
			SinkUint64 = x[i&BenchMask].Mod64(y[i&BenchMask])
		}
	})
	b.Run("op=DivMod64", func(b *testing.B) {
		x, y := in.u128, in.u64
		for i := range b.N {
			sinkU128, SinkUint64 = x[i&BenchMask].DivMod64(y[i&BenchMask])
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
	bins := make([][]byte, len(in.u128))
	for i, x := range in.u128 {
		bins[i], _ = x.MarshalBinary()
	}
	b.Run("op=AppendBinary", func(b *testing.B) {
		x := in.u128
		buf := make([]byte, 0, 32)
		for i := range b.N {
			buf, _ = x[i&BenchMask].AppendBinary(buf[:0])
		}
		SinkBytes = buf
	})
	b.Run("op=UnmarshalBinary", func(b *testing.B) {
		for i := range b.N {
			_ = sinkU128.UnmarshalBinary(bins[i&BenchMask])
		}
	})
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
	b.Run("op=Float64", func(b *testing.B) {
		x := in.u128
		for i := range b.N {
			SinkFloat = x[i&BenchMask].Float64()
		}
	})
	b.Run("op=FromFloat64", func(b *testing.B) {
		x := make([]float64, len(in.u128))
		for i, v := range in.u128 {
			x[i] = v.Float64()
		}
		b.ResetTimer()
		for i := range b.N {
			sinkU128, SinkBool = Uint128FromFloat64(x[i&BenchMask])
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

// BenchmarkUint128Decimal measures decimal output and parsing, mixed lengths
func BenchmarkUint128Decimal(b *testing.B) {
	in := benchU128()
	b.Run("op=AppendText/digits=mixed", func(b *testing.B) {
		x := in.u128
		buf := make([]byte, 0, 100)
		for i := range b.N {
			buf, _ = x[i&BenchMask].AppendText(buf[:0])
		}
		SinkBytes = buf
	})
	b.Run("op=FromDec/digits=mixed", func(b *testing.B) {
		x := in.u128s
		for i := range b.N {
			sinkU128, _ = Uint128FromDec(x[i&BenchMask])
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
	FromBig: MustUint128FromBig, FromBytes: Uint128FromBytes, MustFromBytes: MustUint128FromBytes, FromFloat64: Uint128FromFloat64, MustFromFloat64: MustUint128FromFloat64, Small: NewUint128,
	FromDec: Uint128FromDec, FromHex: Uint128FromHex, MustFromDec: MustUint128FromDec, MustFromHex: MustUint128FromHex,
	Dirty: Uint128{7, 7}, Max: MaxUint128(),
	Values: Values128, Pairs: Pairs128,
	Narrow: narrowUint128,
	Shifts: Shifts128,
	BitIdx: []int{0, 1, 63, 64, 126, 127, 128, 191, 192, 255, 256},
	Ops64: Ops64[Uint128]{
		New: NewUint128, Cmp: Uint128.Cmp64,
		Add: Uint128.Add64, Sub: Uint128.Sub64, Mul: Uint128.Mul64,
		Addo: Uint128.Add64o, Subo: Uint128.Sub64o, Mulo: Uint128.Mul64o, Addu: Uint128.Add64u, Subu: Uint128.Sub64u, Mulu: Uint128.Mul64u,
	},
}

// TestUint128Fmt checks the fmt verbs and flags with big.Int
func TestUint128Fmt(t *testing.T) { RunFmt(t, kU128) }

// TestUint128ParseErrors checks FromDec on malformed and out of range input
func TestUint128ParseErrors(t *testing.T) { RunParseErrors(t, kU128) }

// TestUint128Bits checks BitLen, Not, shifts, Bit and WithBit with big.Int
func TestUint128Bits(t *testing.T) { RunBits(t, kU128) }

// TestUint128Cmp checks Cmp, the comparisons and CmpBig with big.Int
func TestUint128Cmp(t *testing.T) { RunCmp(t, kU128) }

// TestUint128GCD checks GCD with big.Int
func TestUint128GCD(t *testing.T) {
	RunGCD(t, kU128, func(x, y Uint128) *big.Int { return RefU128(x.GCD(y)) })
}

// TestUint128ModInverse checks ModInverse with big.Int
func TestUint128ModInverse(t *testing.T) { RunModInverse(t, kU128, Uint128.ModInverse) }

// TestUint128Decimal checks decimal output and parsing with big.Int
func TestUint128Decimal(t *testing.T) { RunDecimal(t, kU128) }

// TestUint128Sci checks the form m e n of FromDec
func TestUint128Sci(t *testing.T) { RunSci(t, kU128) }

// TestUint128Hex checks hex output and parsing with big.Int
func TestUint128Hex(t *testing.T) { RunHex(t, kU128) }

// TestUint128Ops64 checks the methods with a uint64 operand with big.Int
func TestUint128Ops64(t *testing.T) { RunOps64(t, kU128) }

// TestUint128Sign checks Sign and IsZero with big.Int
func TestUint128Sign(t *testing.T) { RunSign(t, kU128) }

// TestUint128BitArgs checks Bit and WithBit at a negative and a huge index
func TestUint128BitArgs(t *testing.T) { RunBitArgs(t, kU128) }

// TestUint128HugeShifts checks that a shift past 128 bits leaves zero
func TestUint128HugeShifts(t *testing.T) { RunHugeShifts(t, kU128) }

// TestUint128ErrorText checks that parse errors name the type and the input
func TestUint128ErrorText(t *testing.T) { RunErrorText(t, kU128) }

// TestUint128PanicText checks that every panic names the method
func TestUint128PanicText(t *testing.T) { RunPanicText(t, kU128) }

// TestUint128Exp checks Exp, Expo and Expu with big.Int
func TestUint128Exp(t *testing.T) { RunExp(t, kU128) }

// TestUint128ExpMod checks ExpMod against big.Int
func TestUint128ExpMod(t *testing.T) { RunExpMod(t, kU128) }

// TestUint128Sqrt checks Sqrt with big.Int at squares and next to them
func TestUint128Sqrt(t *testing.T) { RunSqrt(t, kU128) }

// TestUint128Log10 checks Log10 at every power of ten and bit length
func TestUint128Log10(t *testing.T) { RunLog10(t, kU128) }

// TestUint128Bytes checks the byte API with big.Int
func TestUint128Bytes(t *testing.T) { RunBytes(t, kU128) }

// TestUint128Bytes16 checks Bytes16 and PutBytes16 with big.Int
func TestUint128Bytes16(t *testing.T) { RunBytes16(t, kU128) }

// TestUint128Float checks Float64 and FromFloat64 with big.Float
func TestUint128Float(t *testing.T) { RunFloat(t, kU128) }

// TestUint128DivFloat checks the rounding of DivToFloat64 with big.Rat
func TestUint128DivFloat(t *testing.T) { RunDivFloat(t, kU128) }

// TestUint128Constants checks untyped constants as 64-bit operands and shifts
func TestUint128Constants(t *testing.T) {
	t.Parallel()
	x := NewUint128(5)
	require.Equal(t, NewUint128(7), x.Add64(2))
	require.Equal(t, NewUint128(3), x.Sub64(2))
	require.Equal(t, NewUint128(10), x.Mul64(2))
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

// BenchmarkUint128Pow measures Exp, Expo, Expu, Sqrt and Log10.
// Exp, Expo and Expu take powers that fit, Expu/full random pairs
func BenchmarkUint128Pow(b *testing.B) {
	in := benchU128()
	l, e := ExpIn(128, false)
	ex, ey := Slice128[Uint128](Low128(l)), make([]Uint128, len(e))
	for i, v := range e {
		ey[i] = NewUint128(v)
	}
	b.Run("op=Exp", func(b *testing.B) {
		for i := range b.N {
			sinkU128 = ex[i&BenchMask].Exp(ey[i&BenchMask])
		}
	})
	b.Run("op=Expo", func(b *testing.B) {
		for i := range b.N {
			sinkU128, SinkBool = ex[i&BenchMask].Expo(ey[i&BenchMask])
		}
	})
	b.Run("op=Expu", func(b *testing.B) {
		for i := range b.N {
			sinkU128 = ex[i&BenchMask].Expu(ey[i&BenchMask])
		}
	})
	b.Run("op=Expu/full", func(b *testing.B) {
		x, y := in.u128, in.v128
		for i := range b.N {
			sinkU128 = x[i&BenchMask].Expu(y[i&BenchMask])
		}
	})
	a := in.u128
	b.Run("op=Sqrt", func(b *testing.B) {
		for i := range b.N {
			sinkU128 = a[i&BenchMask].Sqrt()
		}
	})
	b.Run("op=Log10", func(b *testing.B) {
		for i := range b.N {
			SinkInt = a[i&BenchMask].Log10()
		}
	})
}

// BenchmarkUint128Bytes measures the byte API on random values and indexes
func BenchmarkUint128Bytes(b *testing.B) {
	x := benchU128().u128
	r := rand.New(rand.NewSource(4))
	idx, ext := make([]int, len(x)), make([]int, len(x))
	enc, word := make([][]byte, len(x)), make([][]byte, len(x))
	for i, v := range x {
		idx[i], ext[i] = r.Intn(16), r.Intn(16-1)
		enc[i] = v.Bytes()
		w := v.Bytes32()
		word[i] = w[16:]
	}
	b.Run("op=ByteLen", func(b *testing.B) {
		for i := range b.N {
			SinkInt = x[i&BenchMask].ByteLen()
		}
	})
	b.Run("op=Byte", func(b *testing.B) {
		for i := range b.N {
			SinkByte = x[i&BenchMask].Byte(idx[i&BenchMask])
		}
	})
	b.Run("op=ExtendSign", func(b *testing.B) {
		for i := range b.N {
			sinkU128 = x[i&BenchMask].ExtendSign(ext[i&BenchMask])
		}
	})
	b.Run("op=Bytes32", func(b *testing.B) {
		for i := range b.N {
			SinkBytes32 = x[i&BenchMask].Bytes32()
		}
	})
	b.Run("op=Bytes20", func(b *testing.B) {
		for i := range b.N {
			SinkBytes20 = x[i&BenchMask].Bytes20()
		}
	})
	b.Run("op=PutBytes32", func(b *testing.B) {
		w := SinkBytes32[:]
		for i := range b.N {
			x[i&BenchMask].PutBytes32(w)
		}
	})
	b.Run("op=PutBytes20", func(b *testing.B) {
		w := SinkBytes20[:]
		for i := range b.N {
			x[i&BenchMask].PutBytes20(w)
		}
	})
	b.Run("op=Bytes16", func(b *testing.B) {
		for i := range b.N {
			SinkBytes16 = x[i&BenchMask].Bytes16()
		}
	})
	b.Run("op=PutBytes16", func(b *testing.B) {
		w := SinkBytes16[:]
		for i := range b.N {
			x[i&BenchMask].PutBytes16(w)
		}
	})
	b.Run("op=Bytes", func(b *testing.B) {
		for i := range b.N {
			SinkBytes = x[i&BenchMask].Bytes()
		}
	})
	b.Run("op=AppendBytes", func(b *testing.B) {
		buf := make([]byte, 0, 32)
		for i := range b.N {
			SinkBytes = x[i&BenchMask].AppendBytes(buf)
		}
	})
	b.Run("op=FromBytes", func(b *testing.B) {
		for i := range b.N {
			sinkU128, SinkBool = Uint128FromBytes(enc[i&BenchMask])
		}
	})
	b.Run("op=FromBytes/full", func(b *testing.B) {
		for i := range b.N {
			sinkU128, SinkBool = Uint128FromBytes(word[i&BenchMask])
		}
	})
}

// BenchmarkUint128DivToFloat64 measures DivToFloat64 and DivToFloat64x.
// big.Rat.Float64 of the same pairs is the baseline, compare by /impl
func BenchmarkUint128DivToFloat64(b *testing.B) {
	for _, s := range DivFloatIn(128, false) {
		x, y := Slice128[Uint128](Low128(s.X)), Slice128[Uint128](Low128(s.Y))
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
		b.Run("bits="+s.Name+"/impl=big.Rat", func(b *testing.B) { BenchRatFloat(b, RefU128, x, y) })
	}
}
