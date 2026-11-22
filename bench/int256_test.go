package bench

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/avpetkun/fat"
	"github.com/holiman/uint256"

	. "github.com/avpetkun/fat/internal/testutil"
)

// TestAgreeInt256 checks that the sides of every Int256 benchmark give the same values
func TestAgreeInt256(t *testing.T) {
	in := inputsI256()
	var h uint256.Int
	nb := func() *big.Int { return new(big.Int) }

	// arithmetic
	agree(t, "Abs", func(i int) []string {
		return []string{iDec(in.x.s[i].Abs()), hsDec(h.Abs(&in.x.h[i])), bDec(nb().Abs(in.x.b[i]))}
	})
	agree(t, "Neg", func(i int) []string {
		return []string{iDec(in.x.s[i].Neg()), hsDec(h.Neg(&in.x.h[i])), bDec(nb().Neg(in.x.b[i]))}
	})
	agree(t, "Add", func(i int) []string {
		return []string{fmt.Sprint(in.fx.s[i].Add(in.fy.s[i])), fmt.Sprint(new(big.Int).Add(in.fx.b[i], in.fy.b[i]))}
	})
	agree(t, "Sub", func(i int) []string {
		return []string{fmt.Sprint(in.fx.s[i].Sub(in.fy.s[i])), fmt.Sprint(new(big.Int).Sub(in.fx.b[i], in.fy.b[i]))}
	})
	agree(t, "Mul", func(i int) []string {
		return []string{fmt.Sprint(in.fx.s[i].Mul(in.fy.s[i])), fmt.Sprint(new(big.Int).Mul(in.fx.b[i], in.fy.b[i]))}
	})
	agree(t, "Add64", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Add64(in.k[i])), fmt.Sprint(new(big.Int).Add(in.x.b[i], in.kb[i]))}
	})
	agree(t, "Sub64", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Sub64(in.ks[i])), fmt.Sprint(new(big.Int).Sub(in.x.b[i], in.ksb[i]))}
	})
	agree(t, "Mul64", func(i int) []string {
		return []string{fmt.Sprint(in.x64.s[i].Mul64(in.k[i])), fmt.Sprint(new(big.Int).Mul(in.x64.b[i], in.kb[i]))}
	})
	agree(t, "Add128", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Add128(in.w[i])), fmt.Sprint(new(big.Int).Add(in.x.b[i], in.wb[i]))}
	})
	agree(t, "Sub128", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Sub128(in.ws[i])), fmt.Sprint(new(big.Int).Sub(in.x.b[i], in.wsb[i]))}
	})
	agree(t, "Mul128", func(i int) []string {
		return []string{fmt.Sprint(in.x128.s[i].Mul128(in.w[i])), fmt.Sprint(new(big.Int).Mul(in.x128.b[i], in.wb[i]))}
	})
	agree(t, "Mule", func(i int) []string {
		return []string{func() string {
			hi, lo := in.x.s[i].Mule(in.y.s[i])
			return new(big.Int).Add(new(big.Int).Lsh(hi.Big(), 256), lo.Big()).String()
		}(), fmt.Sprint(new(big.Int).Mul(in.x.b[i], in.y.b[i]))}
	})

	// division
	agree(t, "SDiv", func(i int) []string {
		return []string{iDec(in.x.s[i].Divu(in.y.s[i])), hsDec(h.SDiv(&in.x.h[i], &in.y.h[i])), bDec(nb().Quo(in.x.b[i], in.y.b[i]))}
	})
	agree(t, "SMod", func(i int) []string {
		return []string{iDec(in.x.s[i].Mod(in.y.s[i])), hsDec(h.SMod(&in.x.h[i], &in.y.h[i])), bDec(nb().Rem(in.x.b[i], in.y.b[i]))}
	})
	agree(t, "Div", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Div(in.y.s[i])), fmt.Sprint(new(big.Int).Quo(in.x.b[i], in.y.b[i]))}
	})
	agree(t, "DivMod", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].DivMod(in.y.s[i])), fmt.Sprint(new(big.Int).QuoRem(in.x.b[i], in.y.b[i], new(big.Int)))}
	})
	agree(t, "Div64", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Div64(in.k[i])), fmt.Sprint(new(big.Int).Quo(in.x.b[i], in.kb[i]))}
	})
	agree(t, "Mod64", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Mod64(in.k[i])), fmt.Sprint(new(big.Int).Rem(in.x.b[i], in.kb[i]))}
	})
	agree(t, "DivMod64", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].DivMod64(in.k[i])), fmt.Sprint(new(big.Int).QuoRem(in.x.b[i], in.kb[i], new(big.Int)))}
	})
	agree(t, "Div128", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Div128(in.w[i])), fmt.Sprint(new(big.Int).Quo(in.x.b[i], in.wb[i]))}
	})
	agree(t, "Mod128", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Mod128(in.w[i])), fmt.Sprint(new(big.Int).Rem(in.x.b[i], in.wb[i]))}
	})
	agree(t, "DivMod128", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].DivMod128(in.w[i])), fmt.Sprint(new(big.Int).QuoRem(in.x.b[i], in.wb[i], new(big.Int)))}
	})

	// modular forms and MulDiv
	agree(t, "ExpMod", func(i int) []string {
		return []string{func() string {
			r := in.x.s[i].ExpMod(in.ye.s[i], in.z.s[i]).Big()
			return r.Mod(r, new(big.Int).Abs(in.z.b[i])).String()
		}(), fmt.Sprint(new(big.Int).Exp(in.x.b[i], in.ye.b[i], in.z.b[i]))}
	})
	agree(t, "MulDiv", func(i int) []string {
		return []string{fmt.Sprint(in.fx.s[i].MulDiv(in.fy.s[i], in.y.s[i])), fmt.Sprint(new(big.Int).Quo(new(big.Int).Mul(in.fx.b[i], in.fy.b[i]), in.y.b[i]))}
	})
	agree(t, "MulDivMod", func(i int) []string {
		return []string{fmt.Sprint(in.fx.s[i].MulDivMod(in.fy.s[i], in.y.s[i])), fmt.Sprint(new(big.Int).QuoRem(new(big.Int).Mul(in.fx.b[i], in.fy.b[i]), in.y.b[i], new(big.Int)))}
	})
	agree(t, "AddMod", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].AddMod(in.y.s[i], in.z.s[i])), fmt.Sprint(new(big.Int).Rem(new(big.Int).Add(in.x.b[i], in.y.b[i]), in.z.b[i]))}
	})
	agree(t, "MulMod", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].MulMod(in.y.s[i], in.z.s[i])), fmt.Sprint(new(big.Int).Rem(new(big.Int).Mul(in.x.b[i], in.y.b[i]), in.z.b[i]))}
	})

	// powers and roots
	agree(t, "Exp", func(i int) []string {
		return []string{fmt.Sprint(in.ex.s[i].Exp(in.ey.s[i])), fmt.Sprint(new(big.Int).Exp(in.ex.b[i], in.ey.b[i], nil))}
	})
	agree(t, "Sqrt", func(i int) []string {
		return []string{fmt.Sprint(in.sq.s[i].Sqrt()), fmt.Sprint(new(big.Int).Sqrt(in.sq.b[i]))}
	})

	// comparisons
	agree(t, "Slt", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].LessThan(in.y.s[i])), fmt.Sprint(in.x.h[i].Slt(&in.y.h[i])), fmt.Sprint(in.x.b[i].Cmp(in.y.b[i]) < 0)}
	})
	agree(t, "Sign", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Sign()), fmt.Sprint(in.x.h[i].Sign()), fmt.Sprint(in.x.b[i].Sign())}
	})
	agree(t, "Cmp", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Cmp(in.y.s[i])), fmt.Sprint(in.x.b[i].Cmp(in.y.b[i]))}
	})
	agree(t, "Cmp64", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Cmp64(in.k[i])), fmt.Sprint(in.x.b[i].Cmp(in.kb[i]))}
	})
	agree(t, "Cmp128", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Cmp128(in.w[i])), fmt.Sprint(in.x.b[i].Cmp(in.wb[i]))}
	})
	agree(t, "IsUint64", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].IsUint64()), fmt.Sprint(in.x.b[i].IsUint64())}
	})
	agree(t, "IsInt64", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].IsInt64()), fmt.Sprint(in.x.b[i].IsInt64())}
	})

	// shifts, logic and bits
	agree(t, "SRsh", func(i int) []string {
		return []string{iDec(in.x.s[i].Rsh(in.n[i])), hsDec(h.SRsh(&in.x.h[i], in.nu[i])), bDec(nb().Rsh(in.x.b[i], in.nu[i]))}
	})
	agree(t, "Lsh", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Lsh(in.ln[i])), fmt.Sprint(new(big.Int).Lsh(in.x.b[i], uint(in.ln[i])))}
	})
	agree(t, "And", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].And(in.y.s[i])), fmt.Sprint(new(big.Int).And(in.x.b[i], in.y.b[i]))}
	})
	agree(t, "Or", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Or(in.y.s[i])), fmt.Sprint(new(big.Int).Or(in.x.b[i], in.y.b[i]))}
	})
	agree(t, "Xor", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Xor(in.y.s[i])), fmt.Sprint(new(big.Int).Xor(in.x.b[i], in.y.b[i]))}
	})
	agree(t, "AndNot", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].AndNot(in.y.s[i])), fmt.Sprint(new(big.Int).AndNot(in.x.b[i], in.y.b[i]))}
	})
	agree(t, "Not", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Not()), fmt.Sprint(new(big.Int).Not(in.x.b[i]))}
	})
	agree(t, "Bit", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Bit(in.bi[i])), fmt.Sprint(in.x.b[i].Bit(in.bi[i]))}
	})
	agree(t, "WithBit", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].WithBit(in.bi[i], in.bv[i])), fmt.Sprint(new(big.Int).SetBit(in.x.b[i], in.bi[i], in.bv[i]))}
	})
	agree(t, "BitLen", func(i int) []string { return []string{fmt.Sprint(in.x.s[i].BitLen()), fmt.Sprint(in.x.b[i].BitLen())} })

	// conversions
	agree(t, "Int64", func(i int) []string {
		return []string{fmt.Sprint(in.small.s[i].Int64()), fmt.Sprint(in.small.b[i].Int64())}
	})
	agree(t, "Float64", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Float64()), func() string { f, _ := in.x.b[i].Float64(); return fmt.Sprint(f) }()}
	})

	// text and encodings
	agree(t, "String", func(i int) []string { return []string{fmt.Sprint(in.x.s[i].String()), fmt.Sprint(in.x.b[i].String())} })
	agree(t, "AppendDec", func(i int) []string {
		return []string{string(in.x.s[i].AppendDec(nil)), string(in.x.b[i].Append(nil, 10))}
	})
	agree(t, "AppendText", func(i int) []string {
		return []string{func() string { b, err := in.x.s[i].AppendText(nil); return fmt.Sprint(string(b), err) }(), func() string { b, err := in.x.b[i].AppendText(nil); return fmt.Sprint(string(b), err) }()}
	})
	agree(t, "Hex", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Hex()), func() string {
			if in.x.b[i].Sign() < 0 {
				return "-0x" + new(big.Int).Neg(in.x.b[i]).Text(16)
			}
			return "0x" + in.x.b[i].Text(16)
		}()}
	})
	agree(t, "AppendHex", func(i int) []string {
		return []string{string(in.x.s[i].AppendHex(nil)), func() string {
			if in.x.b[i].Sign() < 0 {
				return "-0x" + new(big.Int).Neg(in.x.b[i]).Text(16)
			}
			return "0x" + in.x.b[i].Text(16)
		}()}
	})
	agree(t, "FromDec", func(i int) []string {
		return []string{func() string { v, err := fat.Int256FromDec(in.dec[i]); return fmt.Sprint(v, err == nil) }(), func() string { v, ok := new(big.Int).SetString(in.dec[i], 10); return fmt.Sprint(v, ok) }()}
	})
	agree(t, "FromHex", func(i int) []string {
		return []string{func() string { v, err := fat.Int256FromHex(in.hex[i]); return fmt.Sprint(v, err == nil) }(), func() string { v, ok := new(big.Int).SetString(in.hex[i], 0); return fmt.Sprint(v, ok) }()}
	})
	agree(t, "MarshalText", func(i int) []string {
		return []string{func() string { b, _ := in.x.s[i].MarshalText(); return string(b) }(), func() string { b, _ := in.x.b[i].MarshalText(); return string(b) }()}
	})
	agree(t, "UnmarshalText", func(i int) []string {
		return []string{func() string { var v fat.Int256; err := v.UnmarshalText(in.decb[i]); return fmt.Sprint(v, err) }(), func() string { v := new(big.Int); err := v.UnmarshalText(in.decb[i]); return fmt.Sprint(v, err) }()}
	})
	agree(t, "MarshalJSON", func(i int) []string {
		return []string{func() string { b, _ := in.x.s[i].MarshalJSON(); return strings.Trim(string(b), `"`) }(), func() string { b, _ := in.x.b[i].MarshalJSON(); return string(b) }()}
	})
	agree(t, "UnmarshalJSON", func(i int) []string {
		return []string{func() string { var v fat.Int256; err := v.UnmarshalJSON(in.js[i]); return fmt.Sprint(v, err) }(), func() string { v := new(big.Int); err := v.UnmarshalJSON(in.jsb[i]); return fmt.Sprint(v, err) }()}
	})
	agree(t, "Format", func(i int) []string { return []string{fmt.Sprintf("%d", in.x.s[i]), fmt.Sprintf("%d", in.x.b[i])} })
	agree(t, "GobEncode", func(i int) []string { return []string{"same", "same"} })
	agree(t, "GobDecode", func(i int) []string {
		return []string{func() string { var v fat.Int256; err := v.GobDecode(in.gob[i]); return fmt.Sprint(v, err) }(), func() string { v := new(big.Int); err := v.GobDecode(in.gobb[i]); return fmt.Sprint(v, err) }()}
	})

	// no side may change its inputs
	agree(t, "inputs", func(i int) []string { return []string{iDec(in.x.s[i]), hsDec(&in.x.h[i]), bDec(in.x.b[i])} })
}

// BenchmarkInt256Arith compares addition, subtraction and multiplication, Abs and Neg with holiman
// and math/big. holiman reads the same limbs as two's complement, on random lengths and signs; no
// input is MinInt256, where Abs and Neg would differ. The checked forms meet math/big where they
// do not overflow, a uint64 or 128-bit operand is a big.Int built before the timer
func BenchmarkInt256Arith(b *testing.B) {
	in := inputsI256()
	b.Run("op=Abs/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x.s[i&m].Abs()
		}
	})
	b.Run("op=Abs/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			hz.Abs(&in.x.h[i&m])
		}
	})
	b.Run("op=Abs/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Abs(in.x.b[i&m])
		}
	})
	b.Run("op=Neg/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x.s[i&m].Neg()
		}
	})
	b.Run("op=Neg/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			hz.Neg(&in.x.h[i&m])
		}
	})
	b.Run("op=Neg/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Neg(in.x.b[i&m])
		}
	})
	b.Run("op=Add/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.fx.s[i&m].Add(in.fy.s[i&m])
		}
	})
	b.Run("op=Add/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Add(in.fx.b[i&m], in.fy.b[i&m])
		}
	})
	b.Run("op=Sub/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.fx.s[i&m].Sub(in.fy.s[i&m])
		}
	})
	b.Run("op=Sub/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Sub(in.fx.b[i&m], in.fy.b[i&m])
		}
	})
	b.Run("op=Mul/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.fx.s[i&m].Mul(in.fy.s[i&m])
		}
	})
	b.Run("op=Mul/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Mul(in.fx.b[i&m], in.fy.b[i&m])
		}
	})
	b.Run("op=Add64/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x.s[i&m].Add64(in.k[i&m])
		}
	})
	b.Run("op=Add64/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Add(in.x.b[i&m], in.kb[i&m])
		}
	})
	b.Run("op=Sub64/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x.s[i&m].Sub64(in.ks[i&m])
		}
	})
	b.Run("op=Sub64/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Sub(in.x.b[i&m], in.ksb[i&m])
		}
	})
	b.Run("op=Mul64/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x64.s[i&m].Mul64(in.k[i&m])
		}
	})
	b.Run("op=Mul64/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Mul(in.x64.b[i&m], in.kb[i&m])
		}
	})
	b.Run("op=Add128/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x.s[i&m].Add128(in.w[i&m])
		}
	})
	b.Run("op=Add128/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Add(in.x.b[i&m], in.wb[i&m])
		}
	})
	b.Run("op=Sub128/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x.s[i&m].Sub128(in.ws[i&m])
		}
	})
	b.Run("op=Sub128/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Sub(in.x.b[i&m], in.wsb[i&m])
		}
	})
	b.Run("op=Mul128/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x128.s[i&m].Mul128(in.w[i&m])
		}
	})
	b.Run("op=Mul128/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Mul(in.x128.b[i&m], in.wb[i&m])
		}
	})
	b.Run("op=Mule/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256, sU256 = in.x.s[i&m].Mule(in.y.s[i&m])
		}
	})
	b.Run("op=Mule/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Mul(in.x.b[i&m], in.y.b[i&m])
		}
	})
}

// BenchmarkInt256Div compares division with holiman and math/big, the divisors never zero: a zero
// one panics in fat, gives 0 in holiman. A uint64 or 128-bit operand is a big.Int built before
// the timer
func BenchmarkInt256Div(b *testing.B) {
	in := inputsI256()
	b.Run("op=SDiv/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x.s[i&m].Divu(in.y.s[i&m])
		}
	})
	b.Run("op=SDiv/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			hz.SDiv(&in.x.h[i&m], &in.y.h[i&m])
		}
	})
	b.Run("op=SDiv/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Quo(in.x.b[i&m], in.y.b[i&m])
		}
	})
	b.Run("op=SMod/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x.s[i&m].Mod(in.y.s[i&m])
		}
	})
	b.Run("op=SMod/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			hz.SMod(&in.x.h[i&m], &in.y.h[i&m])
		}
	})
	b.Run("op=SMod/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Rem(in.x.b[i&m], in.y.b[i&m])
		}
	})
	b.Run("op=Div/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x.s[i&m].Div(in.y.s[i&m])
		}
	})
	b.Run("op=Div/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Quo(in.x.b[i&m], in.y.b[i&m])
		}
	})
	b.Run("op=DivMod/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256, sI256b = in.x.s[i&m].DivMod(in.y.s[i&m])
		}
	})
	b.Run("op=DivMod/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.QuoRem(in.x.b[i&m], in.y.b[i&m], bm)
		}
	})
	b.Run("op=Div64/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x.s[i&m].Div64(in.k[i&m])
		}
	})
	b.Run("op=Div64/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Quo(in.x.b[i&m], in.kb[i&m])
		}
	})
	b.Run("op=Mod64/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI64 = in.x.s[i&m].Mod64(in.k[i&m])
		}
	})
	b.Run("op=Mod64/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Rem(in.x.b[i&m], in.kb[i&m])
		}
	})
	b.Run("op=DivMod64/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256, sI64 = in.x.s[i&m].DivMod64(in.k[i&m])
		}
	})
	b.Run("op=DivMod64/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.QuoRem(in.x.b[i&m], in.kb[i&m], bm)
		}
	})
	b.Run("op=Div128/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x.s[i&m].Div128(in.w[i&m])
		}
	})
	b.Run("op=Div128/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Quo(in.x.b[i&m], in.wb[i&m])
		}
	})
	b.Run("op=Mod128/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI128 = in.x.s[i&m].Mod128(in.w[i&m])
		}
	})
	b.Run("op=Mod128/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Rem(in.x.b[i&m], in.wb[i&m])
		}
	})
	b.Run("op=DivMod128/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256, sI128 = in.x.s[i&m].DivMod128(in.w[i&m])
		}
	})
	b.Run("op=DivMod128/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.QuoRem(in.x.b[i&m], in.wb[i&m], bm)
		}
	})
}

// BenchmarkInt256Modular compares the modular forms, MulDiv and ExpMod with math/big. math/big
// does a modular form as two calls, as code on it would
func BenchmarkInt256Modular(b *testing.B) {
	in := inputsI256()
	b.Run("op=ExpMod/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x.s[i&m].ExpMod(in.ye.s[i&m], in.z.s[i&m])
		}
	})
	b.Run("op=ExpMod/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Exp(in.x.b[i&m], in.ye.b[i&m], in.z.b[i&m])
		}
	})
	b.Run("op=MulDiv/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.fx.s[i&m].MulDiv(in.fy.s[i&m], in.y.s[i&m])
		}
	})
	b.Run("op=MulDiv/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Quo(bz.Mul(in.fx.b[i&m], in.fy.b[i&m]), in.y.b[i&m])
		}
	})
	b.Run("op=MulDivMod/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256, sI256b = in.fx.s[i&m].MulDivMod(in.fy.s[i&m], in.y.s[i&m])
		}
	})
	b.Run("op=MulDivMod/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.QuoRem(bz.Mul(in.fx.b[i&m], in.fy.b[i&m]), in.y.b[i&m], bm)
		}
	})
	b.Run("op=AddMod/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x.s[i&m].AddMod(in.y.s[i&m], in.z.s[i&m])
		}
	})
	b.Run("op=AddMod/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Rem(bz.Add(in.x.b[i&m], in.y.b[i&m]), in.z.b[i&m])
		}
	})
	b.Run("op=MulMod/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x.s[i&m].MulMod(in.y.s[i&m], in.z.s[i&m])
		}
	})
	b.Run("op=MulMod/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Rem(bz.Mul(in.x.b[i&m], in.y.b[i&m]), in.z.b[i&m])
		}
	})
}

// BenchmarkInt256Pow compares Exp and Sqrt with math/big, Exp on bases and exponents with x^y in
// 255 bits
func BenchmarkInt256Pow(b *testing.B) {
	in := inputsI256()
	b.Run("op=Exp/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.ex.s[i&m].Exp(in.ey.s[i&m])
		}
	})
	b.Run("op=Exp/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Exp(in.ex.b[i&m], in.ey.b[i&m], nil)
		}
	})
	b.Run("op=Sqrt/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.sq.s[i&m].Sqrt()
		}
	})
	b.Run("op=Sqrt/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Sqrt(in.sq.b[i&m])
		}
	})
}

// BenchmarkInt256Cmp compares the comparisons and tests with holiman and math/big on random
// lengths
func BenchmarkInt256Cmp(b *testing.B) {
	in := inputsI256()
	b.Run("op=Slt/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sBool = in.x.s[i&m].LessThan(in.y.s[i&m])
		}
	})
	b.Run("op=Slt/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sBool = in.x.h[i&m].Slt(&in.y.h[i&m])
		}
	})
	b.Run("op=Slt/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sBool = in.x.b[i&m].Cmp(in.y.b[i&m]) < 0
		}
	})
	b.Run("op=Sign/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.s[i&m].Sign()
		}
	})
	b.Run("op=Sign/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.h[i&m].Sign()
		}
	})
	b.Run("op=Sign/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.b[i&m].Sign()
		}
	})
	b.Run("op=Cmp/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.s[i&m].Cmp(in.y.s[i&m])
		}
	})
	b.Run("op=Cmp/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.b[i&m].Cmp(in.y.b[i&m])
		}
	})
	b.Run("op=Cmp64/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.s[i&m].Cmp64(in.k[i&m])
		}
	})
	b.Run("op=Cmp64/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.b[i&m].Cmp(in.kb[i&m])
		}
	})
	b.Run("op=Cmp128/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.s[i&m].Cmp128(in.w[i&m])
		}
	})
	b.Run("op=Cmp128/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.b[i&m].Cmp(in.wb[i&m])
		}
	})
	b.Run("op=IsUint64/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sBool = in.x.s[i&m].IsUint64()
		}
	})
	b.Run("op=IsUint64/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sBool = in.x.b[i&m].IsUint64()
		}
	})
	b.Run("op=IsInt64/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sBool = in.x.s[i&m].IsInt64()
		}
	})
	b.Run("op=IsInt64/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sBool = in.x.b[i&m].IsInt64()
		}
	})
}

// BenchmarkInt256Bits compares shifts, logic and bit counts with holiman and math/big on random
// lengths. math/big has no shift within 256 bits: its Lsh runs on counts that keep x in them
func BenchmarkInt256Bits(b *testing.B) {
	in := inputsI256()
	b.Run("op=SRsh/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x.s[i&m].Rsh(in.n[i&m])
		}
	})
	b.Run("op=SRsh/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			hz.SRsh(&in.x.h[i&m], in.nu[i&m])
		}
	})
	b.Run("op=SRsh/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Rsh(in.x.b[i&m], in.nu[i&m])
		}
	})
	b.Run("op=Lsh/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x.s[i&m].Lsh(in.ln[i&m])
		}
	})
	b.Run("op=Lsh/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Lsh(in.x.b[i&m], uint(in.ln[i&m]))
		}
	})
	b.Run("op=And/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x.s[i&m].And(in.y.s[i&m])
		}
	})
	b.Run("op=And/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.And(in.x.b[i&m], in.y.b[i&m])
		}
	})
	b.Run("op=Or/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x.s[i&m].Or(in.y.s[i&m])
		}
	})
	b.Run("op=Or/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Or(in.x.b[i&m], in.y.b[i&m])
		}
	})
	b.Run("op=Xor/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x.s[i&m].Xor(in.y.s[i&m])
		}
	})
	b.Run("op=Xor/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Xor(in.x.b[i&m], in.y.b[i&m])
		}
	})
	b.Run("op=AndNot/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x.s[i&m].AndNot(in.y.s[i&m])
		}
	})
	b.Run("op=AndNot/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.AndNot(in.x.b[i&m], in.y.b[i&m])
		}
	})
	b.Run("op=Not/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x.s[i&m].Not()
		}
	})
	b.Run("op=Not/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Not(in.x.b[i&m])
		}
	})
	b.Run("op=Bit/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sUint = in.x.s[i&m].Bit(in.bi[i&m])
		}
	})
	b.Run("op=Bit/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sUint = in.x.b[i&m].Bit(in.bi[i&m])
		}
	})
	b.Run("op=WithBit/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256 = in.x.s[i&m].WithBit(in.bi[i&m], in.bv[i&m])
		}
	})
	b.Run("op=WithBit/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.SetBit(in.x.b[i&m], in.bi[i&m], in.bv[i&m])
		}
	})
	b.Run("op=BitLen/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.s[i&m].BitLen()
		}
	})
	b.Run("op=BitLen/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.b[i&m].BitLen()
		}
	})
}

// BenchmarkInt256Convert compares the conversions with math/big on random lengths
func BenchmarkInt256Convert(b *testing.B) {
	in := inputsI256()
	b.Run("op=Int64/in=small/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI64 = in.small.s[i&m].Int64()
		}
	})
	b.Run("op=Int64/in=small/impl=big", func(b *testing.B) {
		for i := range b.N {
			sI64 = in.small.b[i&m].Int64()
		}
	})
	b.Run("op=Float64/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sFloat = in.x.s[i&m].Float64()
		}
	})
	b.Run("op=Float64/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sFloat, _ = in.x.b[i&m].Float64()
		}
	})
}

// BenchmarkInt256Text compares text and encodings with math/big on random lengths: math/big writes
// JSON as a bare number and its own gob, each side reads its own
func BenchmarkInt256Text(b *testing.B) {
	in := inputsI256()
	b.Run("op=String/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sString = in.x.s[i&m].String()
		}
	})
	b.Run("op=String/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sString = in.x.b[i&m].String()
		}
	})
	b.Run("op=AppendDec/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sBytes = in.x.s[i&m].AppendDec(sBytes[:0])
		}
	})
	b.Run("op=AppendDec/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sBytes = in.x.b[i&m].Append(sBytes[:0], 10)
		}
	})
	b.Run("op=AppendText/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sBytes, sErr = in.x.s[i&m].AppendText(sBytes[:0])
		}
	})
	b.Run("op=AppendText/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sBytes, sErr = in.x.b[i&m].AppendText(sBytes[:0])
		}
	})
	b.Run("op=Hex/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sString = in.x.s[i&m].Hex()
		}
	})
	b.Run("op=Hex/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sString = in.x.b[i&m].Text(16)
		}
	})
	b.Run("op=AppendHex/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sBytes = in.x.s[i&m].AppendHex(sBytes[:0])
		}
	})
	b.Run("op=AppendHex/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sBytes = in.x.b[i&m].Append(sBytes[:0], 16)
		}
	})
	b.Run("op=FromDec/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256, sErr = fat.Int256FromDec(in.dec[i&m])
		}
	})
	b.Run("op=FromDec/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			_, sBool = bz.SetString(in.dec[i&m], 10)
		}
	})
	b.Run("op=FromHex/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI256, sErr = fat.Int256FromHex(in.hex[i&m])
		}
	})
	b.Run("op=FromHex/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			_, sBool = bz.SetString(in.hex[i&m], 0)
		}
	})
	b.Run("op=MarshalText/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sBytes, sErr = in.x.s[i&m].MarshalText()
		}
	})
	b.Run("op=MarshalText/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sBytes, sErr = in.x.b[i&m].MarshalText()
		}
	})
	b.Run("op=UnmarshalText/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sErr = sI256.UnmarshalText(in.decb[i&m])
		}
	})
	b.Run("op=UnmarshalText/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sErr = bz.UnmarshalText(in.decb[i&m])
		}
	})
	b.Run("op=MarshalJSON/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sBytes, sErr = in.x.s[i&m].MarshalJSON()
		}
	})
	b.Run("op=MarshalJSON/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sBytes, sErr = in.x.b[i&m].MarshalJSON()
		}
	})
	b.Run("op=UnmarshalJSON/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sErr = sI256.UnmarshalJSON(in.js[i&m])
		}
	})
	b.Run("op=UnmarshalJSON/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sErr = bz.UnmarshalJSON(in.jsb[i&m])
		}
	})
	b.Run("op=Format/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sBytes = fmt.Appendf(sBytes[:0], "%d", in.x.s[i&m])
		}
	})
	b.Run("op=Format/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sBytes = fmt.Appendf(sBytes[:0], "%d", in.x.b[i&m])
		}
	})
	b.Run("op=GobEncode/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sBytes, sErr = in.x.s[i&m].GobEncode()
		}
	})
	b.Run("op=GobEncode/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sBytes, sErr = in.x.b[i&m].GobEncode()
		}
	})
	b.Run("op=GobDecode/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sErr = sI256.GobDecode(in.gob[i&m])
		}
	})
	b.Run("op=GobDecode/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sErr = bz.GobDecode(in.gobb[i&m])
		}
	})
}

// BenchmarkInt256DivToFloat64 compares DivToFloat64 with big.Rat.Float64 of x/y, the rationals
// built before the timer
func BenchmarkInt256DivToFloat64(b *testing.B) {
	for _, set := range DivFloatIn(256, true) {
		x, y := newI256(set.X), newI256(set.Y)
		b.Run("in="+set.Name+"/impl=fat", func(b *testing.B) {
			for i := range b.N {
				sFloat = x.s[i&m].DivToFloat64(y.s[i&m])
			}
		})
		b.Run("in="+set.Name+"/impl=big.Rat", func(b *testing.B) {
			rats := make([]*big.Rat, len(set.X))
			for i := range rats {
				rats[i] = new(big.Rat).SetFrac(x.b[i], y.b[i])
			}
			b.ResetTimer()
			for i := range b.N {
				sFloat, sBool = rats[i&m].Float64()
			}
		})
	}
}
