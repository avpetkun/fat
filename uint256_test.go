package fat

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"math"
	"math/big"
	"math/bits"
	"math/rand"
	"strings"
	"sync"
	"testing"

	"github.com/avpetkun/fat/internal/require"
	. "github.com/avpetkun/fat/internal/testutil"
)

// TestUint256Convert checks conversions to the other types and big.Int
func TestUint256Convert(t *testing.T) {
	t.Parallel()
	require.Equal(t, "115792089237316195423570985008687907853269984665640564039457584007913129639935", MaxUint256().String())

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

// TestUint256ParseLengths checks FromDec against big.Int at 1 to 80 digits,
// with a bad symbol at a random place too
func TestUint256ParseLengths(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(4))
	for n := 1; n <= 80; n++ {
		for range 200 {
			d := make([]byte, n)
			for i := range d {
				d[i] = '0' + byte(r.Intn(10))
			}
			// nines and zeros at the window edges
			switch r.Intn(4) {
			case 0:
				copy(d[n/2:], strings.Repeat("9", n))
			case 1:
				copy(d[n/2:], strings.Repeat("0", n))
			}
			d[0] = '1' + byte(r.Intn(9))
			s := string(d)
			t := On(t, s)
			want, _ := new(big.Int).SetString(s, 10)
			v, err := Uint256FromDec(s)
			require.Equal(t, FitsU256(want), err == nil)
			if err == nil {
				require.Big(t, want, RefU256(v))
			} else {
				require.True(t, v.IsZero())
			}
			if n > 78 {
				continue
			}
			// a bad symbol fails at any length
			for _, c := range []byte{'/', ':', 'x', 0, 0xFF} {
				i := r.Intn(n)
				d[i], c = c, d[i]
				v, err := Uint256FromDec(string(d))
				require.Error(On(t, i), err)
				require.True(t, v.IsZero())
				d[i] = c
			}
		}
	}
}

// TestUint256Encoding checks the binary, gob and JSON forms and round trips
func TestUint256Encoding(t *testing.T) {
	t.Parallel()
	Values256(func(x Uint256) {
		t := On(t, RefU256(x))

		b, err := x.MarshalBinary()
		require.NoError(t, err)
		require.Equal(t, RefBE(RefU256(x), 32), b)
		ab, err := x.AppendBinary([]byte("ab"))
		require.NoError(t, err)
		require.Equal(t, append([]byte("ab"), b...), ab)

		y := Uint256{7, 7, 7, 7}
		require.NoError(t, y.UnmarshalBinary(b))
		require.Equal(t, x, y)
	})

	require.Equal(t, Be(32, 0), BinaryOf(Uint256{}))
	require.Equal(t, Be(32, 0, 0xff), BinaryOf(NewUint256(255)))
	require.Equal(t, 32, len(BinaryOf(MaxUint256())))
	RunBinaryLengths(t, kU256)

	// gob and json pick up the interfaces
	type doc struct{ A, B Uint256 }
	in := doc{MaxUint256(), Uint256{1, 2, 3, 4}}

	var buf bytes.Buffer
	require.NoError(t, gob.NewEncoder(&buf).Encode(in))
	var out doc
	require.NoError(t, gob.NewDecoder(&buf).Decode(&out))
	require.Equal(t, in, out)

	js, err := json.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, `{"A":"`+MaxUint256().String()+`","B":"`+RefU256(in.B).String()+`"}`, string(js))
	out = doc{}
	require.NoError(t, json.Unmarshal(js, &out))
	require.Equal(t, in, out)

	RunJSON(t, kU256)
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

		{Name: "Mul", Width: 256, Fn: Plain(Uint256.Mul), Ref: BigMul, Panic: "fat.Uint256.Mul: overflow"},
		{Name: "Mulu", Width: 256, Fn: Plain(Uint256.Mulu), Ref: BigMul},
		{Name: "Mulo", Width: 256, Fn: Uint256.Mulo, Ref: BigMul, Over: true},
		{Name: "Mul128", Width: 128, Fn: Plain(func(x, y Uint256) Uint256 { return x.Mul128(Uint128{y.L0, y.L1}) }), Ref: BigMul, Panic: "fat.Uint256.Mul128: overflow"},
		{Name: "Mul128u", Width: 128, Fn: Plain(func(x, y Uint256) Uint256 { return x.Mul128u(Uint128{y.L0, y.L1}) }), Ref: BigMul},
		{Name: "Mul128o", Width: 128, Fn: func(x, y Uint256) (Uint256, bool) { return x.Mul128o(Uint128{y.L0, y.L1}) }, Ref: BigMul, Over: true},

		{Name: "And", Width: 256, Fn: Plain(Uint256.And), Ref: BigAnd},
		{Name: "Or", Width: 256, Fn: Plain(Uint256.Or), Ref: BigOr},
		{Name: "Xor", Width: 256, Fn: Plain(Uint256.Xor), Ref: BigXor},
		{Name: "AndNot", Width: 256, Fn: Plain(Uint256.AndNot), Ref: BigAndNot},
	})
}

// TestUint256Mule checks the full 512-bit product against big.Int
func TestUint256Mule(t *testing.T) {
	t.Parallel()
	Pairs256(func(x, y Uint256) {
		xb, yb := RefU256(x), RefU256(y)
		hi, lo := x.Mule(y)
		require.Big(On(t, xb, yb), BigMul(xb, yb), BigAdd(new(big.Int).Lsh(RefU256(hi), 256), RefU256(lo)))
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

// TestUint256DivMod64 checks Div64, Mod64 and DivMod64 against big.Int
// and their panic on a zero divisor
func TestUint256DivMod64(t *testing.T) {
	t.Parallel()
	check := func(x Uint256, y uint64) {
		xb, yb := RefU256(x), new(big.Int).SetUint64(y)
		t := On(t, xb, yb)
		qb, rb := new(big.Int).QuoRem(xb, yb, new(big.Int))
		q, r := x.DivMod64(y)
		require.Big(t, qb, RefU256(q))
		require.Equal(t, rb.Uint64(), r)
		require.Equal(t, q, x.Div64(y))
		require.Equal(t, r, x.Mod64(y))
	}
	Values256(func(x Uint256) {
		for _, y := range Divisors64 {
			check(x, y)
		}
	})
	for _, x := range Edge256 {
		for _, y := range SeedDivisors64 {
			check(Limbs256[Uint256](x), y)
		}
	}
	require.Equal(t, "fat.Uint256.Div64: division by zero", PanicValue(func() { MaxUint256().Div64(0) }))
	require.Equal(t, "fat.Uint256.Mod64: division by zero", PanicValue(func() { MaxUint256().Mod64(0) }))
	require.Equal(t, "fat.Uint256.DivMod64: division by zero", PanicValue(func() { MaxUint256().DivMod64(0) }))
}

// TestUint256DivMod checks Div, Mod, DivMod and the 128-bit forms with big.Int
func TestUint256DivMod(t *testing.T) {
	t.Parallel()
	Pairs256(func(x, y Uint256) {
		xb, yb := RefU256(x), RefU256(y)
		t := On(t, xb, yb)
		if y.L1|y.L0 != 0 {
			yl := Uint128{y.L0, y.L1}
			yb := RefU128(yl)
			qb, rb := new(big.Int).QuoRem(xb, yb, new(big.Int))
			q, r := x.DivMod128(yl)
			require.Big(On(t, yb), qb, RefU256(q))
			require.Big(On(t, yb), rb, RefU128(r))
			require.Equal(t, q, x.Div128(yl))
			require.Equal(t, r, x.Mod128(yl))
		}
		if y.IsZero() {
			return
		}
		qb, rb := new(big.Int).QuoRem(xb, yb, new(big.Int))
		q, r := x.DivMod(y)
		require.Big(t, qb, RefU256(q))
		require.Big(t, rb, RefU256(r))
		require.Equal(t, q, x.Div(y))
		require.Equal(t, r, x.Mod(y))
	})
	require.Equal(t, "fat.Uint256.Div: division by zero", PanicValue(func() { MaxUint256().Div(Uint256{}) }))
	require.Equal(t, "fat.Uint256.Mod: division by zero", PanicValue(func() { MaxUint256().Mod(Uint256{}) }))
	require.Equal(t, "fat.Uint256.DivMod: division by zero", PanicValue(func() { MaxUint256().DivMod(Uint256{}) }))
	require.Equal(t, "fat.Uint256.Div128: division by zero", PanicValue(func() { MaxUint256().Div128(Uint128{}) }))
	require.Equal(t, "fat.Uint256.Mod128: division by zero", PanicValue(func() { MaxUint256().Mod128(Uint128{}) }))
	require.Equal(t, "fat.Uint256.DivMod128: division by zero", PanicValue(func() { MaxUint256().DivMod128(Uint128{}) }))
}

// TestReciprocal2by1 checks reciprocal2by1 against bits.Div64
// on the normalized test divisors
func TestReciprocal2by1(t *testing.T) {
	t.Parallel()
	for _, ds := range [][]uint64{Divisors64, SeedDivisors64} {
		for _, d := range ds {
			d <<= uint(bits.LeadingZeros64(d))
			want, _ := bits.Div64(^d, math.MaxUint64, d)
			require.Equal(On(t, d), want, reciprocal2by1(d))
		}
	}
}

// TestReciprocal2by1Seed checks reciprocal2by1 against bits.Div64
// at both ends of each class of the top 24 bits, where the seed is worst,
// and on 2^24 random divisors.
// The value before the last step must be exact or one short for the copies
// that skip that step, the final result would hide a value two short
func TestReciprocal2by1Seed(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(6))
	short := 0
	check := func(d uint64) {
		want, _ := bits.Div64(^d, math.MaxUint64, d)
		if got := reciprocal2by1(d); got != want {
			require.Equal(On(t, d), want, got)
		}
		// reciprocal2by1 without its last step
		v := (1<<47 - 1) / (d>>40 + 1)
		d0 := d & 1
		e := (v>>1)&-d0 - v*(d>>1+d0)
		hi, _ := bits.Mul64(e, e)
		e += hi >> 22
		hi, _ = bits.Mul64(v<<19, e)
		v = v<<41 + hi
		if want-v > 1 {
			require.Same(On(t, d), want, v)
		}
		short += int(want - v)
	}
	for k := uint64(1 << 23); k < 1<<24; k++ {
		lo := k << 40
		check(lo)
		check(lo + 1)
		check(lo + 2)
		check(lo | (1<<40 - 2))
		check(lo | (1<<40 - 1))
		check(lo | r.Uint64()>>24)
	}
	for range 1 << 24 {
		check(r.Uint64() | 1<<63)
	}
	// 1 in 30 of these are one short
	require.True(t, short > 0)
}

// TestUint256DivModKnuth checks DivMod on the rare steps of a 3-limb divisor.
// A remainder with the divisor's top limbs makes the digit 2^64-1,
// one of q*<d2, d1> over a zero limb makes it one too big
func TestUint256DivModKnuth(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(5))
	for range RandomCount {
		// d normalized with d0 > 0, x and d go right by s, low s bits clear
		s := r.Intn(64)
		low := ^uint64(0) << s
		d := Uint256{(r.Uint64() | 1<<s) & low, r.Uint64(), r.Uint64() | 1<<63, 0}
		var rem Uint256
		if r.Intn(2) == 0 {
			// <d2, d1, r0> with r0 < d0
			rem = Uint256{(r.Uint64() % d.L0) & low, d.L1, d.L2, 0}
		} else {
			// q*<d2, d1> as three limbs
			p := Uint256{d.L1, d.L2, 0, 0}.Mul64u(r.Uint64() | 1)
			rem = Uint256{p.L0 & low, p.L1, p.L2, 0}
		}
		// x = (first*d + rem) << 64, with a zero limb under the remainder.
		// The first digit is 0, 1 or any below 2^s that fits,
		// one past the short top limb of y skips the estimate
		first := uint64(r.Intn(2))
		if r.Intn(2) == 0 {
			first = r.Uint64() >> (64 - s) >> 1
		}
		y, rem := d.Rsh(s), rem.Rsh(s)
		top := y.Mul64u(first).Addu(rem)
		x := Uint256{0, top.L0, top.L1, top.L2}
		xb, yb := RefU256(x), RefU256(y)
		qb, rb := new(big.Int).QuoRem(xb, yb, new(big.Int))
		q, m := x.DivMod(y)
		require.Big(On(t, xb, yb), qb, RefU256(q))
		require.Big(On(t, xb, yb), rb, RefU256(m))
		// the same steps in the Int256 kernels
		checkInt256DivSigns(t)(x, y)
	}
}

// TestUint256MulDiv checks MulDiv in all forms, MulDivCeil, MulMod, AddMod and SubMod with big.Int.
// AddMod also gets sums at m and 2m and sums that carry past 2^256
func TestUint256MulDiv(t *testing.T) {
	t.Parallel()
	var ds []Uint256
	Values256(func(d Uint256) {
		if !d.IsZero() {
			ds = append(ds, d)
		}
	})
	i := 0
	Pairs256(func(x, y Uint256) {
		i++
		xb, yb := RefU256(x), RefU256(y)
		pb := new(big.Int).Mul(xb, yb)
		for _, d := range []Uint256{ds[i%len(ds)], ds[(i*7+3)%len(ds)], y, x} {
			if d.IsZero() {
				continue
			}
			db := RefU256(d)
			t := On(t, xb, yb, db)
			qb, rb := new(big.Int).QuoRem(pb, db, new(big.Int))
			over := !BigFits(qb, 256, false)
			q, o := x.MulDivo(y, d)
			require.Big(t, WrapU256(qb), RefU256(q))
			require.Equal(t, over, o)
			require.Equal(t, q, x.MulDivu(y, d))
			if over {
				require.Equal(t, "fat.Uint256.MulDiv: overflow", PanicValue(func() { x.MulDiv(y, d) }))
				require.Equal(t, "fat.Uint256.MulDivMod: overflow", PanicValue(func() { x.MulDivMod(y, d) }))
			} else {
				require.Big(t, qb, RefU256(x.MulDiv(y, d)))
				q, r := x.MulDivMod(y, d)
				require.Big(t, qb, RefU256(q))
				require.Big(t, rb, RefU256(r))
				// MulDiv is Mul then Div where Mul does not overflow
				if p, o := x.Mulo(y); !o {
					require.Equal(t, p.Div(d), q)
				}
			}
			require.Big(t, new(big.Int).Mod(pb, db), RefU256(x.MulMod(y, d)))
			require.Big(t, new(big.Int).Mod(new(big.Int).Add(xb, yb), db), RefU256(x.AddMod(y, d)))
			require.Big(t, new(big.Int).Mod(new(big.Int).Sub(xb, yb), db), RefU256(x.SubMod(y, d)))
			// MulDivCeil is the quotient plus one where x * y / d has a positive fraction
			cb := new(big.Int).Set(qb)
			if rb.Sign() != 0 && rb.Sign() == db.Sign() {
				cb.Add(cb, Big1)
			}
			if BigFits(cb, 256, false) {
				require.Big(t, cb, RefU256(x.MulDivCeil(y, d)))
			} else {
				require.Equal(t, "fat.Uint256.MulDivCeil: overflow", PanicValue(func() { x.MulDivCeil(y, d) }))
			}
		}
	})
	// sums near m and 2m, with operands below m or a carried sum
	for _, m := range ds {
		for _, x := range []Uint256{{}, m.Rsh(1), m.Sub64(1), m, MaxUint256()} {
			for _, sum := range []Uint256{m.Sub64u(1), m, m.Add64u(1), m.Addu(m).Sub64u(1), m.Addu(m), m.Addu(m).Add64u(1)} {
				y := sum.Subu(x)
				want := new(big.Int).Mod(new(big.Int).Add(RefU256(x), RefU256(y)), RefU256(m))
				require.Big(On(t, x, y, m), want, RefU256(x.AddMod(y, m)))
			}
		}
	}
	// a carried sum by small moduli, 1 included
	for _, m := range []Uint256{NewUint256(1), NewUint256(2), NewUint256(3), MaxUint256(), MaxUint256().Sub64(1)} {
		for _, x := range []Uint256{MaxUint256(), MaxUint256().Sub64(1), MaxUint256().Rsh(1).Add64(1)} {
			want := new(big.Int).Mod(new(big.Int).Add(RefU256(x), RefU256(MaxUint256())), RefU256(m))
			require.Big(On(t, x, m), want, RefU256(x.AddMod(MaxUint256(), m)))
		}
	}
	// a carried sum 2^256 + s with the top limbs of a 193-bit modulus
	// makes the digit 2^64-1 in knuthStep4
	for _, m := range []Uint256{{3, 5, 7, 1}, {4, 1 << 63, MaxUint256().L0, 1}} {
		y := Uint256{10, m.L0 - 1, m.L1, m.L2}
		want := new(big.Int).Mod(new(big.Int).Add(RefU256(MaxUint256()), RefU256(y)), RefU256(m))
		require.Big(On(t, y, m), want, RefU256(MaxUint256().AddMod(y, m)))
	}
	// differences at -2m, -m, 0, m and 2m and one off, with operands below m or not
	for _, m := range ds {
		for _, x := range []Uint256{{}, m.Rsh(1), m.Sub64u(1), m, MaxUint256()} {
			for _, k := range []Uint256{{}, m, m.Addu(m)} {
				for _, y := range []Uint256{x.Subu(k), x.Subu(k).Add64u(1), x.Subu(k).Sub64u(1), x.Addu(k), x.Addu(k).Add64u(1), x.Addu(k).Sub64u(1)} {
					want := new(big.Int).Mod(new(big.Int).Sub(RefU256(x), RefU256(y)), RefU256(m))
					require.Big(On(t, x, y, m), want, RefU256(x.SubMod(y, m)))
				}
			}
		}
	}
	// (2^129 - 1)(2^129 + 1) / 4 is the maximum and 3/4: it fits truncated, not rounded up
	{
		h := NewUint256(1).Lsh(129)
		x, y, d := h.Sub64(1), h.Add64(1), NewUint256(4)
		require.Equal(t, MaxUint256(), x.MulDiv(y, d))
		require.Equal(t, "fat.Uint256.MulDivCeil: overflow", PanicValue(func() { x.MulDivCeil(y, d) }))
	}
	// a product past 256 bits and one within them take different paths to d
	for _, x := range []Uint256{MaxUint256(), {L0: 3}} {
		for name, fn := range map[string]func(){
			"MulDiv":     func() { x.MulDiv(x, Uint256{}) },
			"MulDivo":    func() { x.MulDivo(x, Uint256{}) },
			"MulDivu":    func() { x.MulDivu(x, Uint256{}) },
			"MulDivMod":  func() { x.MulDivMod(x, Uint256{}) },
			"MulDivCeil": func() { x.MulDivCeil(x, Uint256{}) },
			"AddMod":     func() { x.AddMod(x, Uint256{}) },
			"SubMod":     func() { x.SubMod(x, Uint256{}) },
			"MulMod":     func() { x.MulMod(x, Uint256{}) },
		} {
			require.Equal(On(t, x, name), "fat.Uint256."+name+": division by zero", PanicValue(fn))
		}
	}
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

// TestUint256MuloLimbs checks Mulo, Mul and Mul128o on every limb pattern.
// The overflow test branches on the zero limbs
func TestUint256MuloLimbs(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(3))
	limbs := func(mask int) Uint256 {
		var l [4]uint64
		for i := range l {
			if mask>>i&1 != 0 {
				// small, full and random limbs
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
		return Uint256{l[0], l[1], l[2], l[3]}
	}
	for range 50 {
		for mx := range 16 {
			for my := range 16 {
				x, y := limbs(mx), limbs(my)
				xb, yb := RefU256(x), RefU256(y)
				want := BigMul(xb, yb)
				t := On(t, xb, yb)
				got, c := x.Mulo(y)
				require.Big(t, WrapU256(want), RefU256(got))
				require.Equal(t, !FitsU256(want), c)
				// Mul goes through mulFit, which orders the operands by limbs
				if FitsU256(want) {
					require.Big(t, want, RefU256(x.Mul(y)))
				} else {
					require.Equal(t, "fat.Uint256.Mul: overflow", PanicValue(func() { x.Mul(y) }))
				}
				y128 := Uint128{y.L0, y.L1}
				want = BigMul(xb, LimbsBig(y.L0, y.L1))
				got, c = x.Mul128o(y128)
				require.Big(t, WrapU256(want), RefU256(got))
				require.Equal(t, !FitsU256(want), c)
			}
		}
	}
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
	b.Run("op=Mul", func(b *testing.B) {
		x, y := in.fx, in.fy
		for i := range b.N {
			sinkU256 = x[i&BenchMask].Mul(y[i&BenchMask])
		}
	})
	b.Run("op=Mulo", func(b *testing.B) {
		x, y := in.u256, in.v256
		for i := range b.N {
			sinkU256, SinkBool = x[i&BenchMask].Mulo(y[i&BenchMask])
		}
	})
	b.Run("op=Mulu", func(b *testing.B) {
		x, y := in.u256, in.v256
		for i := range b.N {
			sinkU256 = x[i&BenchMask].Mulu(y[i&BenchMask])
		}
	})
	b.Run("op=Mule", func(b *testing.B) {
		x, y := in.u256, in.v256
		for i := range b.N {
			sinkU256, sink2U256 = x[i&BenchMask].Mule(y[i&BenchMask])
		}
	})
	b.Run("op=Mul128", func(b *testing.B) {
		x, y := in.fx128, in.u128
		for i := range b.N {
			sinkU256 = x[i&BenchMask].Mul128(y[i&BenchMask])
		}
	})
	b.Run("op=Mul128o", func(b *testing.B) {
		x, y := in.u256, in.u128
		for i := range b.N {
			sinkU256, SinkBool = x[i&BenchMask].Mul128o(y[i&BenchMask])
		}
	})
	b.Run("op=Mul128u", func(b *testing.B) {
		x, y := in.u256, in.u128
		for i := range b.N {
			sinkU256 = x[i&BenchMask].Mul128u(y[i&BenchMask])
		}
	})
	b.Run("op=Mul64", func(b *testing.B) {
		x, y := in.fx64, in.u64
		for i := range b.N {
			sinkU256 = x[i&BenchMask].Mul64(y[i&BenchMask])
		}
	})
	b.Run("op=Mul64o", func(b *testing.B) {
		x, y := in.u256, in.u64
		for i := range b.N {
			sinkU256, SinkBool = x[i&BenchMask].Mul64o(y[i&BenchMask])
		}
	})
	b.Run("op=Mul64u", func(b *testing.B) {
		x, y := in.u256, in.u64
		for i := range b.N {
			sinkU256 = x[i&BenchMask].Mul64u(y[i&BenchMask])
		}
	})
	b.Run("op=DivMod", func(b *testing.B) {
		x, y := in.u256, in.v256
		for i := range b.N {
			sinkU256, sink2U256 = x[i&BenchMask].DivMod(y[i&BenchMask])
		}
	})
	b.Run("op=DivMod/fit", func(b *testing.B) {
		x, y := in.fx, in.fy
		for i := range b.N {
			sinkU256, sink2U256 = x[i&BenchMask].DivMod(y[i&BenchMask])
		}
	})
	b.Run("op=Div", func(b *testing.B) {
		x, y := in.u256, in.v256
		for i := range b.N {
			sinkU256 = x[i&BenchMask].Div(y[i&BenchMask])
		}
	})
	b.Run("op=Mod", func(b *testing.B) {
		x, y := in.u256, in.v256
		for i := range b.N {
			sinkU256 = x[i&BenchMask].Mod(y[i&BenchMask])
		}
	})
	b.Run("op=Div/pow2", func(b *testing.B) {
		x, y := in.u256, in.pow2
		for i := range b.N {
			sinkU256 = x[i&BenchMask].Div(y[i&BenchMask])
		}
	})
	b.Run("op=Mod/pow2", func(b *testing.B) {
		x, y := in.u256, in.pow2
		for i := range b.N {
			sinkU256 = x[i&BenchMask].Mod(y[i&BenchMask])
		}
	})
	b.Run("op=DivMod128", func(b *testing.B) {
		x, y := in.u256, in.u128
		for i := range b.N {
			sinkU256, sinkU128 = x[i&BenchMask].DivMod128(y[i&BenchMask])
		}
	})
	b.Run("op=MulDivo", func(b *testing.B) {
		x, y := in.u256, in.v256
		for i := range b.N {
			sinkU256, SinkBool = x[i&BenchMask].MulDivo(y[i&BenchMask], y[(i+1)&BenchMask])
		}
	})
	// |d| >= |y| keeps x * y / d within range: y with its low bit set away from zero
	mulDivD := sync.OnceValue(func() []Uint256 {
		d := make([]Uint256, len(in.v256))
		for i, v := range in.v256 {
			d[i] = v.Or(NewUint256(1))
		}
		return d
	})
	b.Run("op=MulDiv", func(b *testing.B) {
		x, y, d := in.u256, in.v256, mulDivD()
		for i := range b.N {
			sinkU256 = x[i&BenchMask].MulDiv(y[i&BenchMask], d[i&BenchMask])
		}
	})
	b.Run("op=MulDivCeil", func(b *testing.B) {
		x, y, d := in.u256, in.v256, mulDivD()
		for i := range b.N {
			sinkU256 = x[i&BenchMask].MulDivCeil(y[i&BenchMask], d[i&BenchMask])
		}
	})
	b.Run("op=MulMod", func(b *testing.B) {
		x, y := in.u256, in.v256
		for i := range b.N {
			sinkU256 = x[i&BenchMask].MulMod(y[i&BenchMask], y[(i+1)&BenchMask])
		}
	})
	b.Run("op=AddMod", func(b *testing.B) {
		x, y := in.u256, in.v256
		for i := range b.N {
			sinkU256 = x[i&BenchMask].AddMod(y[i&BenchMask], y[(i+1)&BenchMask])
		}
	})
	b.Run("op=SubMod", func(b *testing.B) {
		x, y := in.u256, in.v256
		for i := range b.N {
			sinkU256 = x[i&BenchMask].SubMod(y[i&BenchMask], y[(i+1)&BenchMask])
		}
	})
	b.Run("op=GCD", func(b *testing.B) {
		x, y := in.u256, in.v256
		for i := range b.N {
			sinkU256 = x[i&BenchMask].GCD(y[i&BenchMask])
		}
	})
	// nonzero moduli of either parity: y with bit 1 set
	invM := sync.OnceValue(func() []Uint256 {
		m := make([]Uint256, len(in.v256))
		for i, v := range in.v256 {
			m[i] = v.Or(NewUint256(2))
		}
		return m
	})
	b.Run("op=ModInverse", func(b *testing.B) {
		x, m := in.u256, invM()
		for i := range b.N {
			sinkU256, SinkBool = x[i&BenchMask].ModInverse(m[i&BenchMask])
		}
	})
	b.Run("op=Div64", func(b *testing.B) {
		x, y := in.u256, in.u64
		for i := range b.N {
			sinkU256 = x[i&BenchMask].Div64(y[i&BenchMask])
		}
	})
	b.Run("op=Mod64", func(b *testing.B) {
		x, y := in.u256, in.u64
		for i := range b.N {
			SinkUint64 = x[i&BenchMask].Mod64(y[i&BenchMask])
		}
	})
	b.Run("op=DivMod64", func(b *testing.B) {
		x, y := in.u256, in.u64
		for i := range b.N {
			sinkU256, SinkUint64 = x[i&BenchMask].DivMod64(y[i&BenchMask])
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

// BenchmarkUint256Hex measures FromHex, Hex and AppendHex on mixed lengths
func BenchmarkUint256Hex(b *testing.B) {
	in := benchU256()
	mixed := make([]string, len(in.u256))
	for i, x := range in.u256 {
		mixed[i] = HexPrefixed(RefU256(x))
	}
	b.Run("op=FromHex/digits=mixed", func(b *testing.B) {
		x := mixed
		for i := range b.N {
			sinkU256, _ = Uint256FromHex(x[i&BenchMask])
		}
	})
	b.Run("op=Hex/digits=mixed", func(b *testing.B) {
		x := make([]Uint256, len(mixed))
		for i, s := range mixed {
			x[i], _ = Uint256FromHex(s)
		}
		b.ResetTimer()
		for i := range b.N {
			SinkString = x[i&BenchMask].Hex()
		}
	})
	b.Run("op=AppendHex/digits=mixed", func(b *testing.B) {
		x := make([]Uint256, len(mixed))
		for i, s := range mixed {
			x[i], _ = Uint256FromHex(s)
		}
		buf := make([]byte, 0, 100)
		for i := range b.N {
			buf = x[i&BenchMask].AppendHex(buf[:0])
		}
		SinkBytes = buf
	})
}

// BenchmarkUint256Decimal measures AppendText and FromDec on mixed lengths
func BenchmarkUint256Decimal(b *testing.B) {
	in := benchU256()
	b.Run("op=AppendText/digits=mixed", func(b *testing.B) {
		x := in.u256
		buf := make([]byte, 0, 100)
		for i := range b.N {
			buf, _ = x[i&BenchMask].AppendText(buf[:0])
		}
		SinkBytes = buf
	})
	b.Run("op=FromDec/digits=mixed", func(b *testing.B) {
		x := in.u256s
		for i := range b.N {
			sinkU256, _ = Uint256FromDec(x[i&BenchMask])
		}
	})
}

// BenchmarkUint256Binary measures AppendBinary and UnmarshalBinary
func BenchmarkUint256Binary(b *testing.B) {
	in := benchU256()
	b.Run("op=Append", func(b *testing.B) {
		x := in.u256
		buf := make([]byte, 0, 64)
		for i := range b.N {
			buf, _ = x[i&BenchMask].AppendBinary(buf[:0])
		}
		SinkBytes = buf
	})
	b.Run("op=Unmarshal", func(b *testing.B) {
		x := make([][]byte, len(in.u256))
		for i, v := range in.u256 {
			x[i], _ = v.MarshalBinary()
		}
		b.ResetTimer()
		for i := range b.N {
			_ = sinkU256.UnmarshalBinary(x[i&BenchMask])
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
	b.Run("op=Float64", func(b *testing.B) {
		x := in.u256
		for i := range b.N {
			SinkFloat = x[i&BenchMask].Float64()
		}
	})
	b.Run("op=FromFloat64", func(b *testing.B) {
		x := make([]float64, len(in.u256))
		for i, v := range in.u256 {
			x[i] = v.Float64()
		}
		b.ResetTimer()
		for i := range b.N {
			sinkU256, SinkBool = Uint256FromFloat64(x[i&BenchMask])
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
	FromBig: MustUint256FromBig, FromBytes: Uint256FromBytes, MustFromBytes: MustUint256FromBytes, FromFloat64: Uint256FromFloat64, MustFromFloat64: MustUint256FromFloat64, Small: NewUint256,
	FromDec: Uint256FromDec, FromHex: Uint256FromHex, MustFromDec: MustUint256FromDec, MustFromHex: MustUint256FromHex,
	Dirty: Uint256{7, 7, 7, 7}, Max: MaxUint256(),
	Values: Values256, Pairs: Pairs256,
	Narrow: narrowUint256,
	Shifts: Shifts256,
	BitIdx: []int{0, 1, 63, 64, 127, 128, 191, 192, 254, 255, 256, 319, 320, 511, 512},
	CmpMore: func(t testing.TB, x, y Uint256, xb *big.Int) {
		require.Equal(t, xb.Cmp(LimbsBig(y.L0, y.L1)), x.Cmp128(Uint128{y.L0, y.L1}))
	},
	Ops64: Ops64[Uint256]{
		New: NewUint256, Cmp: Uint256.Cmp64,
		Add: Uint256.Add64, Sub: Uint256.Sub64, Mul: Uint256.Mul64,
		Addo: Uint256.Add64o, Subo: Uint256.Sub64o, Mulo: Uint256.Mul64o, Addu: Uint256.Add64u, Subu: Uint256.Sub64u, Mulu: Uint256.Mul64u,
	},
}

// TestUint256Fmt checks Format against big.Int on the fmt verbs and flags
func TestUint256Fmt(t *testing.T) { RunFmt(t, kU256) }

// TestUint256ParseErrors checks FromDec on malformed and out of range text
func TestUint256ParseErrors(t *testing.T) { RunParseErrors(t, kU256) }

// TestUint256Bits checks BitLen, Not, shifts, Bit and WithBit against big.Int
func TestUint256Bits(t *testing.T) { RunBits(t, kU256) }

// TestUint256Cmp checks the comparisons and CmpBig against big.Int
func TestUint256Cmp(t *testing.T) { RunCmp(t, kU256) }

// TestUint256GCD checks GCD with big.Int
func TestUint256GCD(t *testing.T) {
	RunGCD(t, kU256, func(x, y Uint256) *big.Int { return RefU256(x.GCD(y)) })
}

// TestUint256ModInverse checks ModInverse with big.Int
func TestUint256ModInverse(t *testing.T) { RunModInverse(t, kU256, Uint256.ModInverse) }

// TestUint256Decimal checks decimal text both ways against big.Int
func TestUint256Decimal(t *testing.T) { RunDecimal(t, kU256) }

// TestUint256Sci checks the form m e n of FromDec and Scan
func TestUint256Sci(t *testing.T) { RunSci(t, kU256) }

// TestUint256Hex checks hex text both ways against big.Int
func TestUint256Hex(t *testing.T) { RunHex(t, kU256) }

// TestUint256SQL checks Value and Scan against big.Int
func TestUint256SQL(t *testing.T) { RunSQL(t, kU256) }

// TestNullUint256 checks NullUint256 with SQL and JSON, null included
func TestNullUint256(t *testing.T) {
	RunNull(t, kU256, func(n NullUint256) (Uint256, bool) { return n.Uint256, n.Valid }, func(x Uint256, ok bool) NullUint256 { return NullUint256{x, ok} })
}

// TestUint256Ops64 checks the methods with a uint64 operand against big.Int
func TestUint256Ops64(t *testing.T) { RunOps64(t, kU256) }

// TestUint256Sign checks Sign and IsZero against big.Int
func TestUint256Sign(t *testing.T) { RunSign(t, kU256) }

// TestUint256BitArgs checks Bit and WithBit at negative and huge indexes
func TestUint256BitArgs(t *testing.T) { RunBitArgs(t, kU256) }

// TestUint256HugeShifts checks that shifts past the width leave zero
func TestUint256HugeShifts(t *testing.T) { RunHugeShifts(t, kU256) }

// TestUint256ErrorText checks that parse errors name the type and the input
func TestUint256ErrorText(t *testing.T) { RunErrorText(t, kU256) }

// TestUint256PanicText checks that every panic names its func or method
func TestUint256PanicText(t *testing.T) { RunPanicText(t, kU256) }

// TestUint256Exp checks Exp, Expo and Expu against big.Int
func TestUint256Exp(t *testing.T) { RunExp(t, kU256) }

// TestUint256ExpMod checks ExpMod against big.Int
func TestUint256ExpMod(t *testing.T) { RunExpMod(t, kU256) }

// TestUint256Sqrt checks Sqrt against big.Int at and next to squares
func TestUint256Sqrt(t *testing.T) { RunSqrt(t, kU256) }

// TestUint256Log10 checks Log10 at every power of ten and bit length
func TestUint256Log10(t *testing.T) { RunLog10(t, kU256) }

// TestUint256Bytes checks the byte methods and FromBytes against big.Int
func TestUint256Bytes(t *testing.T) { RunBytes(t, kU256) }

// TestUint256Float checks Float64 and FromFloat64 against big.Float
func TestUint256Float(t *testing.T) { RunFloat(t, kU256) }

// TestUint256DivFloat checks DivToFloat64 against big.Rat at rounding ties
func TestUint256DivFloat(t *testing.T) { RunDivFloat(t, kU256) }

// TestUint256Constants checks that untyped constants compile as operands
func TestUint256Constants(t *testing.T) {
	t.Parallel()
	x := NewUint256(5)
	require.Equal(t, NewUint256(7), x.Add64(2))
	require.Equal(t, NewUint256(3), x.Sub64(2))
	require.Equal(t, NewUint256(10), x.Mul64(2))
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
		b, err := x.MarshalBinary()
		require.NoError(t, err)
		w, err := x.Uint256().MarshalBinary()
		require.NoError(t, err)
		require.Equal(t, append(Be(16, 0), b...), w)
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

// BenchmarkUint256Pow measures Exp with results that fit, Sqrt and Log10.
// Expu/full takes full-width operands that wrap
func BenchmarkUint256Pow(b *testing.B) {
	in := benchU256()
	l, e := ExpIn(256, false)
	ex, ey := Slice256[Uint256](l), make([]Uint256, len(e))
	for i, v := range e {
		ey[i] = NewUint256(v)
	}
	b.Run("op=Exp", func(b *testing.B) {
		for i := range b.N {
			sinkU256 = ex[i&BenchMask].Exp(ey[i&BenchMask])
		}
	})
	b.Run("op=Expo", func(b *testing.B) {
		for i := range b.N {
			sinkU256, SinkBool = ex[i&BenchMask].Expo(ey[i&BenchMask])
		}
	})
	b.Run("op=Expu", func(b *testing.B) {
		for i := range b.N {
			sinkU256 = ex[i&BenchMask].Expu(ey[i&BenchMask])
		}
	})
	b.Run("op=Expu/full", func(b *testing.B) {
		x, y := in.u256, in.v256
		for i := range b.N {
			sinkU256 = x[i&BenchMask].Expu(y[i&BenchMask])
		}
	})
	a := in.u256
	b.Run("op=Sqrt", func(b *testing.B) {
		for i := range b.N {
			sinkU256 = a[i&BenchMask].Sqrt()
		}
	})
	b.Run("op=Log10", func(b *testing.B) {
		for i := range b.N {
			SinkInt = a[i&BenchMask].Log10()
		}
	})
}

// BenchmarkUint256Bytes measures the byte methods and FromBytes
func BenchmarkUint256Bytes(b *testing.B) {
	x := benchU256().u256
	r := rand.New(rand.NewSource(4))
	idx, ext := make([]int, len(x)), make([]int, len(x))
	enc, word := make([][]byte, len(x)), make([][]byte, len(x))
	for i, v := range x {
		idx[i], ext[i] = r.Intn(32), r.Intn(32-1)
		enc[i] = v.Bytes()
		w := v.Bytes32()
		word[i] = w[:]
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
			sinkU256 = x[i&BenchMask].ExtendSign(ext[i&BenchMask])
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
			sinkU256, SinkBool = Uint256FromBytes(enc[i&BenchMask])
		}
	})
	b.Run("op=FromBytes/full", func(b *testing.B) {
		for i := range b.N {
			sinkU256, SinkBool = Uint256FromBytes(word[i&BenchMask])
		}
	})
}

// BenchmarkUint256DivToFloat64 measures DivToFloat64 and DivToFloat64x
// next to big.Rat.Float64 on the same pairs by bit length.
// Compare with benchstat -col /impl
func BenchmarkUint256DivToFloat64(b *testing.B) {
	for _, s := range DivFloatIn(256, false) {
		x, y := Slice256[Uint256](s.X), Slice256[Uint256](s.Y)
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
		b.Run("bits="+s.Name+"/impl=big.Rat", func(b *testing.B) { BenchRatFloat(b, RefU256, x, y) })
	}
}
