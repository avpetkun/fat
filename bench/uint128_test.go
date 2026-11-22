package bench

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/avpetkun/fat"
	"lukechampine.com/uint128"

	"github.com/avpetkun/fat/internal/require"
	. "github.com/avpetkun/fat/internal/testutil"
)

// TestAgreeUint128 checks that the sides of every Uint128 benchmark give the same values
func TestAgreeUint128(t *testing.T) {
	in := inputsU128()
	nb := func() *big.Int { return new(big.Int) }
	s := func(v fat.Uint128) string { return v.String() }

	// arithmetic
	for _, set := range []struct {
		name string
		x, y u128
	}{{"fit", in.fx, in.fy}, {"small", in.small, in.small2}} {
		x, y := set.x, set.y
		agree(t, "Add/"+set.name, func(i int) []string {
			return []string{s(x.s[i].Add(y.s[i])), lDec(x.l[i].Add(y.l[i])), nb().Add(x.b[i], y.b[i]).String()}
		})
		agree(t, "Mul/"+set.name, func(i int) []string {
			return []string{s(x.s[i].Mul(y.s[i])), lDec(x.l[i].Mul(y.l[i])), nb().Mul(x.b[i], y.b[i]).String()}
		})
	}
	agree(t, "Sub/fit", func(i int) []string {
		return []string{s(in.fx.s[i].Sub(in.fy.s[i])), lDec(in.fx.l[i].Sub(in.fy.l[i])), nb().Sub(in.fx.b[i], in.fy.b[i]).String()}
	})
	agree(t, "Addu", func(i int) []string {
		return []string{s(in.x.s[i].Addu(in.y.s[i])), lDec(in.x.l[i].AddWrap(in.y.l[i]))}
	})
	agree(t, "Subu", func(i int) []string {
		return []string{s(in.x.s[i].Subu(in.y.s[i])), lDec(in.x.l[i].SubWrap(in.y.l[i]))}
	})
	agree(t, "Mulu", func(i int) []string {
		return []string{s(in.x.s[i].Mulu(in.y.s[i])), lDec(in.x.l[i].MulWrap(in.y.l[i]))}
	})
	agree(t, "Add64u", func(i int) []string {
		return []string{s(in.x.s[i].Add64u(in.k[i])), lDec(in.x.l[i].AddWrap64(in.k[i]))}
	})
	agree(t, "Sub64u", func(i int) []string {
		return []string{s(in.x.s[i].Sub64u(in.k[i])), lDec(in.x.l[i].SubWrap64(in.k[i]))}
	})
	agree(t, "Mul64u", func(i int) []string {
		return []string{s(in.x.s[i].Mul64u(in.k[i])), lDec(in.x.l[i].MulWrap64(in.k[i]))}
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
	agree(t, "Mule", func(i int) []string {
		return []string{in.x.s[i].Mule(in.y.s[i]).String(), fmt.Sprint(new(big.Int).Mul(in.x.b[i], in.y.b[i]))}
	})

	// division
	for _, set := range []struct {
		name string
		x, y u128
	}{{"mixed", in.x, in.y}, {"div64", in.x, in.small}, {"full", in.full, in.y}} {
		x, y := set.x, set.y
		agree(t, "Div/"+set.name, func(i int) []string {
			return []string{s(x.s[i].Div(y.s[i])), lDec(x.l[i].Div(y.l[i])), nb().Quo(x.b[i], y.b[i]).String()}
		})
		agree(t, "Mod/"+set.name, func(i int) []string {
			return []string{s(x.s[i].Mod(y.s[i])), lDec(x.l[i].Mod(y.l[i])), nb().Rem(x.b[i], y.b[i]).String()}
		})
		agree(t, "DivMod/"+set.name, func(i int) []string {
			q, r := x.s[i].DivMod(y.s[i])
			lq, lr := x.l[i].QuoRem(y.l[i])
			bq, br := nb().QuoRem(x.b[i], y.b[i], nb())
			return []string{s(q) + " " + s(r), lDec(lq) + " " + lDec(lr), bq.String() + " " + br.String()}
		})
	}
	agree(t, "Div64", func(i int) []string { return []string{s(in.x.s[i].Div64(in.k[i])), lDec(in.x.l[i].Div64(in.k[i]))} })
	agree(t, "Mod64", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Mod64(in.k[i])), fmt.Sprint(in.x.l[i].Mod64(in.k[i]))}
	})
	agree(t, "DivMod64", func(i int) []string {
		q, r := in.x.s[i].DivMod64(in.k[i])
		lq, lr := in.x.l[i].QuoRem64(in.k[i])
		return []string{fmt.Sprint(s(q), r), fmt.Sprint(lDec(lq), lr)}
	})
	agree(t, "Div64/big", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Div64(in.k[i])), fmt.Sprint(new(big.Int).Quo(in.x.b[i], in.kb[i]))}
	})
	agree(t, "Mod64/big", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Mod64(in.k[i])), fmt.Sprint(new(big.Int).Rem(in.x.b[i], in.kb[i]))}
	})
	agree(t, "DivMod64/big", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].DivMod64(in.k[i])), fmt.Sprint(new(big.Int).QuoRem(in.x.b[i], in.kb[i], new(big.Int)))}
	})

	// modular forms and MulDiv
	agree(t, "ExpMod", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].ExpMod(in.ye.s[i], in.z.s[i])), fmt.Sprint(new(big.Int).Exp(in.x.b[i], in.ye.b[i], in.z.b[i]))}
	})
	agree(t, "MulDiv", func(i int) []string {
		return []string{fmt.Sprint(in.fx.s[i].MulDiv(in.fy.s[i], in.y.s[i])), fmt.Sprint(new(big.Int).Quo(new(big.Int).Mul(in.fx.b[i], in.fy.b[i]), in.y.b[i]))}
	})
	agree(t, "MulDivMod", func(i int) []string {
		return []string{fmt.Sprint(in.fx.s[i].MulDivMod(in.fy.s[i], in.y.s[i])), fmt.Sprint(new(big.Int).QuoRem(new(big.Int).Mul(in.fx.b[i], in.fy.b[i]), in.y.b[i], new(big.Int)))}
	})
	agree(t, "AddMod", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].AddMod(in.y.s[i], in.z.s[i])), fmt.Sprint(new(big.Int).Mod(new(big.Int).Add(in.x.b[i], in.y.b[i]), in.z.b[i]))}
	})
	agree(t, "MulMod", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].MulMod(in.y.s[i], in.z.s[i])), fmt.Sprint(new(big.Int).Mod(new(big.Int).Mul(in.x.b[i], in.y.b[i]), in.z.b[i]))}
	})

	// powers and roots
	agree(t, "Exp", func(i int) []string {
		return []string{fmt.Sprint(in.ex.s[i].Exp(in.ey.s[i])), fmt.Sprint(new(big.Int).Exp(in.ex.b[i], in.ey.b[i], nil))}
	})
	agree(t, "Sqrt", func(i int) []string {
		return []string{fmt.Sprint(in.sq.s[i].Sqrt()), fmt.Sprint(new(big.Int).Sqrt(in.sq.b[i]))}
	})

	// comparisons
	agree(t, "Cmp", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Cmp(in.y.s[i])), fmt.Sprint(in.x.l[i].Cmp(in.y.l[i])), fmt.Sprint(in.x.b[i].Cmp(in.y.b[i]))}
	})
	agree(t, "Cmp64", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Cmp64(in.k[i])), fmt.Sprint(in.x.l[i].Cmp64(in.k[i]))}
	})
	agree(t, "Eq", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i] == in.y.s[i]), fmt.Sprint(in.x.l[i].Equals(in.y.l[i]))}
	})
	agree(t, "IsZero", func(i int) []string { return []string{fmt.Sprint(in.x.s[i].IsZero()), fmt.Sprint(in.x.l[i].IsZero())} })
	agree(t, "Cmp64/big", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Cmp64(in.k[i])), fmt.Sprint(in.x.b[i].Cmp(in.kb[i]))}
	})
	agree(t, "Sign", func(i int) []string { return []string{fmt.Sprint(in.x.s[i].Sign()), fmt.Sprint(in.x.b[i].Sign())} })
	agree(t, "IsUint64", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].IsUint64()), fmt.Sprint(in.x.b[i].IsUint64())}
	})
	agree(t, "IsInt64", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].IsInt64()), fmt.Sprint(in.x.b[i].IsInt64())}
	})

	// shifts, logic and bits
	agree(t, "Lsh", func(i int) []string { return []string{s(in.x.s[i].Lsh(in.n[i])), lDec(in.x.l[i].Lsh(in.nu[i]))} })
	agree(t, "Rsh", func(i int) []string {
		return []string{s(in.x.s[i].Rsh(in.n[i])), lDec(in.x.l[i].Rsh(in.nu[i])), nb().Rsh(in.x.b[i], in.nu[i]).String()}
	})
	agree(t, "RotateLeft", func(i int) []string {
		return []string{s(in.x.s[i].RotateLeft(in.n[i])), lDec(in.x.l[i].RotateLeft(in.n[i]))}
	})
	agree(t, "And", func(i int) []string {
		return []string{s(in.x.s[i].And(in.y.s[i])), lDec(in.x.l[i].And(in.y.l[i])), nb().And(in.x.b[i], in.y.b[i]).String()}
	})
	agree(t, "Xor", func(i int) []string {
		return []string{s(in.x.s[i].Xor(in.y.s[i])), lDec(in.x.l[i].Xor(in.y.l[i])), nb().Xor(in.x.b[i], in.y.b[i]).String()}
	})
	agree(t, "BitLen", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].BitLen()), fmt.Sprint(in.x.l[i].Len()), fmt.Sprint(in.x.b[i].BitLen())}
	})
	agree(t, "LeadingZeros", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].LeadingZeros()), fmt.Sprint(in.x.l[i].LeadingZeros())}
	})
	agree(t, "TrailingZeros", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].TrailingZeros()), fmt.Sprint(in.x.l[i].TrailingZeros()), fmt.Sprint(in.x.b[i].TrailingZeroBits())}
	})
	agree(t, "OnesCount", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].OnesCount()), fmt.Sprint(in.x.l[i].OnesCount())}
	})
	agree(t, "Reverse", func(i int) []string { return []string{s(in.x.s[i].Reverse()), lDec(in.x.l[i].Reverse())} })
	agree(t, "ReverseBytes", func(i int) []string { return []string{s(in.x.s[i].ReverseBytes()), lDec(in.x.l[i].ReverseBytes())} })
	agree(t, "Lsh/big", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Lsh(in.ln[i])), fmt.Sprint(new(big.Int).Lsh(in.x.b[i], uint(in.ln[i])))}
	})
	agree(t, "Or", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Or(in.y.s[i])), fmt.Sprint(new(big.Int).Or(in.x.b[i], in.y.b[i]))}
	})
	agree(t, "AndNot", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].AndNot(in.y.s[i])), fmt.Sprint(new(big.Int).AndNot(in.x.b[i], in.y.b[i]))}
	})
	agree(t, "Bit", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Bit(in.bi[i])), fmt.Sprint(in.x.b[i].Bit(in.bi[i]))}
	})
	agree(t, "WithBit", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].WithBit(in.bi[i], in.bv[i])), fmt.Sprint(new(big.Int).SetBit(in.x.b[i], in.bi[i], in.bv[i]))}
	})

	// bytes
	agree(t, "PutBytes16", func(i int) []string {
		var a, w [16]byte
		in.x.s[i].PutBytes16(a[:])
		in.x.l[i].PutBytesBE(w[:])
		return []string{fmt.Sprint(a), fmt.Sprint(w)}
	})
	agree(t, "FromBytes16", func(i int) []string {
		var v fat.Uint128
		return []string{fmt.Sprint(v.UnmarshalBinary(in.be16[i]) == nil, s(v)), fmt.Sprint(true, lDec(uint128.FromBytesBE(in.be16[i]))), fmt.Sprint(true, nb().SetBytes(in.be16[i]).String())}
	})
	agree(t, "Bytes", func(i int) []string { return []string{fmt.Sprint(in.x.s[i].Bytes()), fmt.Sprint(in.x.b[i].Bytes())} })
	agree(t, "FromBytes", func(i int) []string {
		return []string{fmt.Sprint(fat.Uint128FromBytes(in.be[i])), fmt.Sprint(new(big.Int).SetBytes(in.be[i]), true)}
	})
	agree(t, "PutBytes32", func(i int) []string {
		return []string{func() string { var a [32]byte; in.x.s[i].PutBytes32(a[:]); return fmt.Sprint(a) }(), func() string { var a [32]byte; in.x.b[i].FillBytes(a[:]); return fmt.Sprint(a) }()}
	})

	// conversions
	agree(t, "Big", func(i int) []string { return []string{in.x.s[i].Big().String(), in.x.l[i].Big().String()} })
	agree(t, "FromBig", func(i int) []string {
		v, err := fat.Uint128FromBig(in.x.b[i])
		return []string{fmt.Sprint(s(v), err == nil), fmt.Sprint(lDec(uint128.FromBig(nb().Set(in.x.b[i]))), true)}
	})
	agree(t, "Uint64", func(i int) []string {
		return []string{fmt.Sprint(in.small.s[i].Uint64()), fmt.Sprint(in.small.b[i].Uint64())}
	})
	agree(t, "Int64", func(i int) []string {
		return []string{fmt.Sprint(in.small.s[i].Int64()), fmt.Sprint(in.small.b[i].Int64())}
	})
	agree(t, "Float64", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Float64()), func() string { f, _ := in.x.b[i].Float64(); return fmt.Sprint(f) }()}
	})

	// text and encodings
	agree(t, "String", func(i int) []string { return []string{in.x.s[i].String(), in.x.l[i].String(), in.x.b[i].String()} })
	agree(t, "FromDec", func(i int) []string {
		v, err := fat.Uint128FromDec(in.dec[i])
		lv, lerr := uint128.FromString(in.dec[i])
		bv, ok := nb().SetString(in.dec[i], 10)
		return []string{fmt.Sprint(s(v), err == nil), fmt.Sprint(lDec(lv), lerr == nil), fmt.Sprint(bv.String(), ok)}
	})
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
	agree(t, "FromHex", func(i int) []string {
		return []string{func() string { v, err := fat.Uint128FromHex(in.hex[i]); return fmt.Sprint(v, err == nil) }(), func() string { v, ok := new(big.Int).SetString(in.hex[i], 0); return fmt.Sprint(v, ok) }()}
	})
	agree(t, "MarshalText", func(i int) []string {
		return []string{func() string { b, _ := in.x.s[i].MarshalText(); return string(b) }(), func() string { b, _ := in.x.b[i].MarshalText(); return string(b) }()}
	})
	agree(t, "UnmarshalText", func(i int) []string {
		var v fat.Uint128
		err := v.UnmarshalText(in.decb[i])
		var lv uint128.Uint128
		lerr := lv.UnmarshalText(in.decb[i])
		bv := new(big.Int)
		berr := bv.UnmarshalText(in.decb[i])
		return []string{fmt.Sprint(v.String(), err == nil), fmt.Sprint(lDec(lv), lerr == nil), fmt.Sprint(bv.String(), berr == nil)}
	})
	agree(t, "MarshalJSON", func(i int) []string {
		return []string{func() string { b, _ := in.x.s[i].MarshalJSON(); return strings.Trim(string(b), `"`) }(), func() string { b, _ := in.x.b[i].MarshalJSON(); return string(b) }()}
	})
	agree(t, "UnmarshalJSON", func(i int) []string {
		return []string{func() string { var v fat.Uint128; err := v.UnmarshalJSON(in.js[i]); return fmt.Sprint(v, err) }(), func() string { v := new(big.Int); err := v.UnmarshalJSON(in.jsb[i]); return fmt.Sprint(v, err) }()}
	})
	agree(t, "Format", func(i int) []string { return []string{fmt.Sprintf("%d", in.x.s[i]), fmt.Sprintf("%d", in.x.b[i])} })
	agree(t, "GobEncode", func(i int) []string { return []string{"same", "same"} })
	agree(t, "GobDecode", func(i int) []string {
		return []string{func() string { var v fat.Uint128; err := v.GobDecode(in.gob[i]); return fmt.Sprint(v, err) }(), func() string { v := new(big.Int); err := v.GobDecode(in.gobb[i]); return fmt.Sprint(v, err) }()}
	})

	// chains of chainN sums, each on the one before; the first 256 of them
	t.Run("Chain", func(t *testing.T) {
		for i := range 256 {
			k := chainAt(i)
			bz := new(big.Int)
			chainBig(bz, in.cx.b[k:])
			require.Equal(On(t, i), bz.String(), s(chainFat128((*[chainN + 1]fat.Uint128)(in.cx.s[k:]))))
			require.Equal(On(t, i), bz.String(), lDec(chainLuke((*[chainN + 1]uint128.Uint128)(in.cx.l[k:]))))
		}
	})

	// no side may change its inputs, as FromBig of lukechampine would
	agree(t, "inputs", func(i int) []string { return []string{s(in.x.s[i]), lDec(in.x.l[i]), in.x.b[i].String()} })
}

// BenchmarkUint128Arith compares addition, subtraction and multiplication with lukechampine and
// math/big. lukechampine checks as fat does: Add, Sub and Mul on pairs that fit, the wrapping
// forms on random lengths. The checked forms meet math/big where they do not overflow, a uint64
// operand is a big.Int built before the timer
func BenchmarkUint128Arith(b *testing.B) {
	in := inputsU128()
	for _, set := range []struct {
		name string
		x, y u128
	}{{"fit", in.fx, in.fy}, {"small", in.small, in.small2}} {
		x, y := set.x, set.y
		b.Run("op=Add/in="+set.name+"/impl=fat", func(b *testing.B) {
			for i := range b.N {
				sU128 = x.s[i&m].Add(y.s[i&m])
			}
		})
		b.Run("op=Add/in="+set.name+"/impl=lukechampine", func(b *testing.B) {
			for i := range b.N {
				lz = x.l[i&m].Add(y.l[i&m])
			}
		})
		b.Run("op=Add/in="+set.name+"/impl=big", func(b *testing.B) {
			for i := range b.N {
				bz.Add(x.b[i&m], y.b[i&m])
			}
		})
		b.Run("op=Mul/in="+set.name+"/impl=fat", func(b *testing.B) {
			for i := range b.N {
				sU128 = x.s[i&m].Mul(y.s[i&m])
			}
		})
		b.Run("op=Mul/in="+set.name+"/impl=lukechampine", func(b *testing.B) {
			for i := range b.N {
				lz = x.l[i&m].Mul(y.l[i&m])
			}
		})
		b.Run("op=Mul/in="+set.name+"/impl=big", func(b *testing.B) {
			for i := range b.N {
				bz.Mul(x.b[i&m], y.b[i&m])
			}
		})
	}
	b.Run("op=Sub/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.fx.s[i&m].Sub(in.fy.s[i&m])
		}
	})
	b.Run("op=Sub/in=fit/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			lz = in.fx.l[i&m].Sub(in.fy.l[i&m])
		}
	})
	b.Run("op=Sub/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Sub(in.fx.b[i&m], in.fy.b[i&m])
		}
	})
	b.Run("op=Addu/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].Addu(in.y.s[i&m])
		}
	})
	b.Run("op=Addu/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			lz = in.x.l[i&m].AddWrap(in.y.l[i&m])
		}
	})
	b.Run("op=Subu/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].Subu(in.y.s[i&m])
		}
	})
	b.Run("op=Subu/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			lz = in.x.l[i&m].SubWrap(in.y.l[i&m])
		}
	})
	b.Run("op=Mulu/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].Mulu(in.y.s[i&m])
		}
	})
	b.Run("op=Mulu/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			lz = in.x.l[i&m].MulWrap(in.y.l[i&m])
		}
	})
	b.Run("op=Add64u/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].Add64u(in.k[i&m])
		}
	})
	b.Run("op=Add64u/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			lz = in.x.l[i&m].AddWrap64(in.k[i&m])
		}
	})
	b.Run("op=Sub64u/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].Sub64u(in.k[i&m])
		}
	})
	b.Run("op=Sub64u/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			lz = in.x.l[i&m].SubWrap64(in.k[i&m])
		}
	})
	b.Run("op=Mul64u/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].Mul64u(in.k[i&m])
		}
	})
	b.Run("op=Mul64u/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			lz = in.x.l[i&m].MulWrap64(in.k[i&m])
		}
	})
	b.Run("op=Add64/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].Add64(in.k[i&m])
		}
	})
	b.Run("op=Add64/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Add(in.x.b[i&m], in.kb[i&m])
		}
	})
	b.Run("op=Sub64/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].Sub64(in.ks[i&m])
		}
	})
	b.Run("op=Sub64/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Sub(in.x.b[i&m], in.ksb[i&m])
		}
	})
	b.Run("op=Mul64/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x64.s[i&m].Mul64(in.k[i&m])
		}
	})
	b.Run("op=Mul64/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Mul(in.x64.b[i&m], in.kb[i&m])
		}
	})
	b.Run("op=Mule/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.x.s[i&m].Mule(in.y.s[i&m])
		}
	})
	b.Run("op=Mule/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Mul(in.x.b[i&m], in.y.b[i&m])
		}
	})
}

// BenchmarkUint128Div compares division with lukechampine and math/big, the divisors never zero: a
// zero one panics in fat. A uint64 operand is a big.Int built before the timer
func BenchmarkUint128Div(b *testing.B) {
	in := inputsU128()
	for _, set := range []struct {
		name string
		x, y u128
	}{{"mixed", in.x, in.y}, {"div64", in.x, in.small}, {"full", in.full, in.y}} {
		x, y := set.x, set.y
		b.Run("op=Div/in="+set.name+"/impl=fat", func(b *testing.B) {
			for i := range b.N {
				sU128 = x.s[i&m].Div(y.s[i&m])
			}
		})
		b.Run("op=Div/in="+set.name+"/impl=lukechampine", func(b *testing.B) {
			for i := range b.N {
				lz = x.l[i&m].Div(y.l[i&m])
			}
		})
		b.Run("op=Div/in="+set.name+"/impl=big", func(b *testing.B) {
			for i := range b.N {
				bz.Quo(x.b[i&m], y.b[i&m])
			}
		})
		b.Run("op=Mod/in="+set.name+"/impl=fat", func(b *testing.B) {
			for i := range b.N {
				sU128 = x.s[i&m].Mod(y.s[i&m])
			}
		})
		b.Run("op=Mod/in="+set.name+"/impl=lukechampine", func(b *testing.B) {
			for i := range b.N {
				lz = x.l[i&m].Mod(y.l[i&m])
			}
		})
		b.Run("op=Mod/in="+set.name+"/impl=big", func(b *testing.B) {
			for i := range b.N {
				bz.Rem(x.b[i&m], y.b[i&m])
			}
		})
		b.Run("op=DivMod/in="+set.name+"/impl=fat", func(b *testing.B) {
			for i := range b.N {
				sU128, sU128b = x.s[i&m].DivMod(y.s[i&m])
			}
		})
		b.Run("op=DivMod/in="+set.name+"/impl=lukechampine", func(b *testing.B) {
			for i := range b.N {
				lz, lm = x.l[i&m].QuoRem(y.l[i&m])
			}
		})
		b.Run("op=DivMod/in="+set.name+"/impl=big", func(b *testing.B) {
			for i := range b.N {
				bz.QuoRem(x.b[i&m], y.b[i&m], bm)
			}
		})
	}
	b.Run("op=Div64/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].Div64(in.k[i&m])
		}
	})
	b.Run("op=Div64/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			lz = in.x.l[i&m].Div64(in.k[i&m])
		}
	})
	b.Run("op=Mod64/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU64 = in.x.s[i&m].Mod64(in.k[i&m])
		}
	})
	b.Run("op=Mod64/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			sU64 = in.x.l[i&m].Mod64(in.k[i&m])
		}
	})
	b.Run("op=DivMod64/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128, sU64 = in.x.s[i&m].DivMod64(in.k[i&m])
		}
	})
	b.Run("op=DivMod64/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			lz, sU64 = in.x.l[i&m].QuoRem64(in.k[i&m])
		}
	})
	b.Run("op=Div64/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Quo(in.x.b[i&m], in.kb[i&m])
		}
	})
	b.Run("op=Mod64/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Rem(in.x.b[i&m], in.kb[i&m])
		}
	})
	b.Run("op=DivMod64/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.QuoRem(in.x.b[i&m], in.kb[i&m], bm)
		}
	})
}

// BenchmarkUint128Modular compares the modular forms, MulDiv and ExpMod with math/big. math/big
// does a modular form as two calls, as code on it would
func BenchmarkUint128Modular(b *testing.B) {
	in := inputsU128()
	b.Run("op=ExpMod/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].ExpMod(in.ye.s[i&m], in.z.s[i&m])
		}
	})
	b.Run("op=ExpMod/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Exp(in.x.b[i&m], in.ye.b[i&m], in.z.b[i&m])
		}
	})
	b.Run("op=MulDiv/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.fx.s[i&m].MulDiv(in.fy.s[i&m], in.y.s[i&m])
		}
	})
	b.Run("op=MulDiv/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Quo(bz.Mul(in.fx.b[i&m], in.fy.b[i&m]), in.y.b[i&m])
		}
	})
	b.Run("op=MulDivMod/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128, sU128b = in.fx.s[i&m].MulDivMod(in.fy.s[i&m], in.y.s[i&m])
		}
	})
	b.Run("op=MulDivMod/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.QuoRem(bz.Mul(in.fx.b[i&m], in.fy.b[i&m]), in.y.b[i&m], bm)
		}
	})
	b.Run("op=AddMod/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].AddMod(in.y.s[i&m], in.z.s[i&m])
		}
	})
	b.Run("op=AddMod/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Mod(bz.Add(in.x.b[i&m], in.y.b[i&m]), in.z.b[i&m])
		}
	})
	b.Run("op=MulMod/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].MulMod(in.y.s[i&m], in.z.s[i&m])
		}
	})
	b.Run("op=MulMod/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Mod(bz.Mul(in.x.b[i&m], in.y.b[i&m]), in.z.b[i&m])
		}
	})
}

// BenchmarkUint128Pow compares Exp and Sqrt with math/big, Exp on bases and exponents with x^y in
// 127 bits
func BenchmarkUint128Pow(b *testing.B) {
	in := inputsU128()
	b.Run("op=Exp/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.ex.s[i&m].Exp(in.ey.s[i&m])
		}
	})
	b.Run("op=Exp/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Exp(in.ex.b[i&m], in.ey.b[i&m], nil)
		}
	})
	b.Run("op=Sqrt/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.sq.s[i&m].Sqrt()
		}
	})
	b.Run("op=Sqrt/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Sqrt(in.sq.b[i&m])
		}
	})
}

// BenchmarkUint128Cmp compares the comparisons and tests with lukechampine and math/big on random
// lengths
func BenchmarkUint128Cmp(b *testing.B) {
	in := inputsU128()
	b.Run("op=Cmp/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.s[i&m].Cmp(in.y.s[i&m])
		}
	})
	b.Run("op=Cmp/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.l[i&m].Cmp(in.y.l[i&m])
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
	b.Run("op=Cmp64/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.l[i&m].Cmp64(in.k[i&m])
		}
	})
	b.Run("op=Eq/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sBool = in.x.s[i&m] == in.y.s[i&m]
		}
	})
	b.Run("op=Eq/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			sBool = in.x.l[i&m].Equals(in.y.l[i&m])
		}
	})
	b.Run("op=IsZero/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sBool = in.x.s[i&m].IsZero()
		}
	})
	b.Run("op=IsZero/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			sBool = in.x.l[i&m].IsZero()
		}
	})
	b.Run("op=Cmp64/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.b[i&m].Cmp(in.kb[i&m])
		}
	})
	b.Run("op=Sign/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.s[i&m].Sign()
		}
	})
	b.Run("op=Sign/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.b[i&m].Sign()
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

// BenchmarkUint128Bits compares shifts, logic and bit counts with lukechampine and math/big on
// random lengths. math/big has no shift within 128 bits: its Lsh runs on counts that keep x in
// them
func BenchmarkUint128Bits(b *testing.B) {
	in := inputsU128()
	b.Run("op=Lsh/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].Lsh(in.n[i&m])
		}
	})
	b.Run("op=Lsh/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			lz = in.x.l[i&m].Lsh(in.nu[i&m])
		}
	})
	b.Run("op=Rsh/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].Rsh(in.n[i&m])
		}
	})
	b.Run("op=Rsh/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			lz = in.x.l[i&m].Rsh(in.nu[i&m])
		}
	})
	b.Run("op=Rsh/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Rsh(in.x.b[i&m], in.nu[i&m])
		}
	})
	b.Run("op=RotateLeft/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].RotateLeft(in.n[i&m])
		}
	})
	b.Run("op=RotateLeft/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			lz = in.x.l[i&m].RotateLeft(in.n[i&m])
		}
	})
	b.Run("op=And/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].And(in.y.s[i&m])
		}
	})
	b.Run("op=And/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			lz = in.x.l[i&m].And(in.y.l[i&m])
		}
	})
	b.Run("op=And/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.And(in.x.b[i&m], in.y.b[i&m])
		}
	})
	b.Run("op=Xor/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].Xor(in.y.s[i&m])
		}
	})
	b.Run("op=Xor/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			lz = in.x.l[i&m].Xor(in.y.l[i&m])
		}
	})
	b.Run("op=Xor/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Xor(in.x.b[i&m], in.y.b[i&m])
		}
	})
	b.Run("op=BitLen/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.s[i&m].BitLen()
		}
	})
	b.Run("op=BitLen/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.l[i&m].Len()
		}
	})
	b.Run("op=BitLen/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.b[i&m].BitLen()
		}
	})
	b.Run("op=LeadingZeros/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.s[i&m].LeadingZeros()
		}
	})
	b.Run("op=LeadingZeros/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.l[i&m].LeadingZeros()
		}
	})
	b.Run("op=TrailingZeros/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.s[i&m].TrailingZeros()
		}
	})
	b.Run("op=TrailingZeros/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.l[i&m].TrailingZeros()
		}
	})
	b.Run("op=TrailingZeros/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sUint = in.x.b[i&m].TrailingZeroBits()
		}
	})
	b.Run("op=OnesCount/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.s[i&m].OnesCount()
		}
	})
	b.Run("op=OnesCount/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.l[i&m].OnesCount()
		}
	})
	b.Run("op=Reverse/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].Reverse()
		}
	})
	b.Run("op=Reverse/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			lz = in.x.l[i&m].Reverse()
		}
	})
	b.Run("op=ReverseBytes/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].ReverseBytes()
		}
	})
	b.Run("op=ReverseBytes/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			lz = in.x.l[i&m].ReverseBytes()
		}
	})
	b.Run("op=Lsh/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].Lsh(in.ln[i&m])
		}
	})
	b.Run("op=Lsh/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Lsh(in.x.b[i&m], uint(in.ln[i&m]))
		}
	})
	b.Run("op=Or/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].Or(in.y.s[i&m])
		}
	})
	b.Run("op=Or/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Or(in.x.b[i&m], in.y.b[i&m])
		}
	})
	b.Run("op=AndNot/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].AndNot(in.y.s[i&m])
		}
	})
	b.Run("op=AndNot/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.AndNot(in.x.b[i&m], in.y.b[i&m])
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
			sU128 = in.x.s[i&m].WithBit(in.bi[i&m], in.bv[i&m])
		}
	})
	b.Run("op=WithBit/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.SetBit(in.x.b[i&m], in.bi[i&m], in.bv[i&m])
		}
	})
}

// BenchmarkUint128Bytes compares the big-endian forms with lukechampine and math/big on random
// lengths
func BenchmarkUint128Bytes(b *testing.B) {
	in := inputsU128()
	b.Run("op=PutBytes16/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			in.x.s[i&m].PutBytes16(sB32[:16])
		}
	})
	b.Run("op=PutBytes16/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			in.x.l[i&m].PutBytesBE(sB32[:16])
		}
	})
	b.Run("op=FromBytes16/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sErr = sU128.UnmarshalBinary(in.be16[i&m])
		}
	})
	b.Run("op=FromBytes16/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			lz = uint128.FromBytesBE(in.be16[i&m])
		}
	})
	b.Run("op=FromBytes16/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.SetBytes(in.be16[i&m])
		}
	})
	b.Run("op=Bytes/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sBytes = in.x.s[i&m].Bytes()
		}
	})
	b.Run("op=Bytes/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sBytes = in.x.b[i&m].Bytes()
		}
	})
	b.Run("op=FromBytes/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128, sBool = fat.Uint128FromBytes(in.be[i&m])
		}
	})
	b.Run("op=FromBytes/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.SetBytes(in.be[i&m])
		}
	})
	b.Run("op=PutBytes32/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			in.x.s[i&m].PutBytes32(sB32[:])
		}
	})
	b.Run("op=PutBytes32/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			in.x.b[i&m].FillBytes(sB32[:])
		}
	})
}

// BenchmarkUint128Convert compares the conversions with lukechampine and math/big on random
// lengths
func BenchmarkUint128Convert(b *testing.B) {
	in := inputsU128()
	b.Run("op=Big/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			bm = in.x.s[i&m].Big()
		}
	})
	b.Run("op=Big/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			bm = in.x.l[i&m].Big()
		}
	})
	b.Run("op=FromBig/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128, sErr = fat.Uint128FromBig(in.x.b[i&m])
		}
	})
	// lukechampine shifts the big.Int it reads, i.Rsh(i, 64), so it takes
	// a copy each time and the copy counts in its time
	b.Run("op=FromBig/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			lz = uint128.FromBig(bm.Set(in.x.b[i&m]))
		}
	})
	b.Run("op=Uint64/in=small/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU64 = in.small.s[i&m].Uint64()
		}
	})
	b.Run("op=Uint64/in=small/impl=big", func(b *testing.B) {
		for i := range b.N {
			sU64 = in.small.b[i&m].Uint64()
		}
	})
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

// BenchmarkUint128Text compares text and encodings with lukechampine and math/big on random
// lengths: math/big writes JSON as a bare number and its own gob, each side reads its own
func BenchmarkUint128Text(b *testing.B) {
	in := inputsU128()
	b.Run("op=String/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sString = in.x.s[i&m].String()
		}
	})
	b.Run("op=String/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			sString = in.x.l[i&m].String()
		}
	})
	b.Run("op=String/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sString = in.x.b[i&m].String()
		}
	})
	b.Run("op=FromDec/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128, sErr = fat.Uint128FromDec(in.dec[i&m])
		}
	})
	b.Run("op=FromDec/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			lz, sErr = uint128.FromString(in.dec[i&m])
		}
	})
	b.Run("op=FromDec/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			_, sBool = bz.SetString(in.dec[i&m], 10)
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
	b.Run("op=FromHex/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128, sErr = fat.Uint128FromHex(in.hex[i&m])
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
			sErr = sU128.UnmarshalText(in.decb[i&m])
		}
	})
	b.Run("op=UnmarshalText/in=mixed/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			sErr = lz.UnmarshalText(in.decb[i&m])
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
			sErr = sU128.UnmarshalJSON(in.js[i&m])
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
			sErr = sU128.GobDecode(in.gob[i&m])
		}
	})
	b.Run("op=GobDecode/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sErr = bz.GobDecode(in.gobb[i&m])
		}
	})
}

// BenchmarkUint128DivToFloat64 compares DivToFloat64 with big.Rat.Float64 of x/y, the rationals
// built before the timer
func BenchmarkUint128DivToFloat64(b *testing.B) {
	for _, set := range DivFloatIn(128, false) {
		x, y := newU128(Low128(set.X)), newU128(Low128(set.Y))
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

// BenchmarkUint128Chain compares a chain of chainN sums, x0 + x1 + ... written
// out in one expression, each sum on the one before, as each API writes it:
// fat and lukechampine on values, math/big on a *big.Int. The values are
// below 2^125, so the sum never wraps and math/big gives the same; ns/add is
// the time of one sum
func BenchmarkUint128Chain(b *testing.B) {
	in := inputsU128()
	b.Run("op=Add/in=chain/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = chainFat128((*[chainN + 1]fat.Uint128)(in.cx.s[chainAt(i):]))
		}
		b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*chainN), "ns/add")
	})
	b.Run("op=Add/in=chain/impl=lukechampine", func(b *testing.B) {
		for i := range b.N {
			lz = chainLuke((*[chainN + 1]uint128.Uint128)(in.cx.l[chainAt(i):]))
		}
		b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*chainN), "ns/add")
	})
	b.Run("op=Add/in=chain/impl=big", func(b *testing.B) {
		for i := range b.N {
			chainBig(bz, in.cx.b[chainAt(i):])
		}
		b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*chainN), "ns/add")
	})
}

// chainFat128 and chainLuke add the chainN+1 values of w as one chain
func chainFat128(w *[chainN + 1]fat.Uint128) fat.Uint128 {
	return w[0].Addu(w[1]).Addu(w[2]).Addu(w[3]).Addu(w[4]).Addu(w[5])
}

func chainLuke(w *[chainN + 1]uint128.Uint128) uint128.Uint128 {
	return w[0].AddWrap(w[1]).AddWrap(w[2]).AddWrap(w[3]).AddWrap(w[4]).AddWrap(w[5])
}
