package bench

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/avpetkun/fat"
	"github.com/holiman/uint256"

	"github.com/avpetkun/fat/internal/require"
	. "github.com/avpetkun/fat/internal/testutil"
)

// TestAgreeUint256 checks that the sides of every Uint256 benchmark give the same values
func TestAgreeUint256(t *testing.T) {
	in := inputsU256()
	var h, hr uint256.Int
	nb := func() *big.Int { return new(big.Int) }

	// arithmetic
	// arithmetic on the pairs that fit
	for _, set := range []struct {
		name string
		x, y u256
	}{{"fit", in.fx, in.fy}, {"small", in.small, in.small2}} {
		x, y := set.x, set.y
		agree(t, "Add/"+set.name, func(i int) []string {
			return []string{sDec(x.s[i].Addu(y.s[i])), hDec(h.Add(&x.h[i], &y.h[i])), bDec(nb().Add(x.b[i], y.b[i]))}
		})
		agree(t, "Mul/"+set.name, func(i int) []string {
			return []string{sDec(x.s[i].Mulu(y.s[i])), hDec(h.Mul(&x.h[i], &y.h[i])), bDec(nb().Mul(x.b[i], y.b[i]))}
		})
	}
	agree(t, "Sub/fit", func(i int) []string {
		return []string{sDec(in.fx.s[i].Subu(in.fy.s[i])), hDec(h.Sub(&in.fx.h[i], &in.fy.h[i])), bDec(nb().Sub(in.fx.b[i], in.fy.b[i]))}
	})
	agree(t, "Addo", func(i int) []string {
		r, o := in.x.s[i].Addo(in.y.s[i])
		hv, ho := h.AddOverflow(&in.x.h[i], &in.y.h[i])
		return []string{fmt.Sprint(sDec(r), o), fmt.Sprint(hDec(hv), ho)}
	})
	agree(t, "Subo", func(i int) []string {
		r, o := in.x.s[i].Subo(in.y.s[i])
		hv, ho := h.SubOverflow(&in.x.h[i], &in.y.h[i])
		return []string{fmt.Sprint(sDec(r), o), fmt.Sprint(hDec(hv), ho)}
	})
	agree(t, "Mulo", func(i int) []string {
		r, o := in.x.s[i].Mulo(in.y.s[i])
		hv, ho := h.MulOverflow(&in.x.h[i], &in.y.h[i])
		return []string{fmt.Sprint(sDec(r), o), fmt.Sprint(hDec(hv), ho)}
	})
	agree(t, "Add64", func(i int) []string {
		return []string{sDec(in.x.s[i].Add64u(in.k[i])), hDec(h.AddUint64(&in.x.h[i], in.k[i]))}
	})
	agree(t, "Sub64", func(i int) []string {
		return []string{sDec(in.x.s[i].Sub64u(in.k[i])), hDec(h.SubUint64(&in.x.h[i], in.k[i]))}
	})
	agree(t, "Add64/big", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Add64(in.k[i])), fmt.Sprint(new(big.Int).Add(in.x.b[i], in.kb[i]))}
	})
	agree(t, "Sub64/big", func(i int) []string {
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
	// division
	for _, set := range []struct {
		name string
		x, y u256
	}{{"mixed", in.x, in.y}, {"div64", in.x, in.small}, {"full", in.full, in.y}} {
		x, y := set.x, set.y
		agree(t, "Div/"+set.name, func(i int) []string {
			return []string{sDec(x.s[i].Div(y.s[i])), hDec(h.Div(&x.h[i], &y.h[i])), bDec(nb().Quo(x.b[i], y.b[i]))}
		})
		agree(t, "Mod/"+set.name, func(i int) []string {
			return []string{sDec(x.s[i].Mod(y.s[i])), hDec(h.Mod(&x.h[i], &y.h[i])), bDec(nb().Rem(x.b[i], y.b[i]))}
		})
		agree(t, "DivMod/"+set.name, func(i int) []string {
			q, r := x.s[i].DivMod(y.s[i])
			h.DivMod(&x.h[i], &y.h[i], &hr)
			bq, br := nb().QuoRem(x.b[i], y.b[i], nb())
			return []string{sDec(q) + " " + sDec(r), hDec(&h) + " " + hDec(&hr), bDec(bq) + " " + bDec(br)}
		})
	}
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
	agree(t, "AddMod", func(i int) []string {
		return []string{sDec(in.x.s[i].AddMod(in.y.s[i], in.z.s[i])), hDec(h.AddMod(&in.x.h[i], &in.y.h[i], &in.z.h[i])), bDec(nb().Mod(nb().Add(in.x.b[i], in.y.b[i]), in.z.b[i]))}
	})
	agree(t, "MulMod", func(i int) []string {
		return []string{sDec(in.x.s[i].MulMod(in.y.s[i], in.z.s[i])), hDec(h.MulMod(&in.x.h[i], &in.y.h[i], &in.z.h[i])), bDec(nb().Mod(nb().Mul(in.x.b[i], in.y.b[i]), in.z.b[i]))}
	})
	// math/big keeps a quotient past 256 bits, so it joins only where none is
	agree(t, "MulDiv", func(i int) []string {
		q, o := in.x.s[i].MulDivo(in.y.s[i], in.z.s[i])
		hv, ho := h.MulDivOverflow(&in.x.h[i], &in.y.h[i], &in.z.h[i])
		r := []string{fmt.Sprint(sDec(q), o), fmt.Sprint(hDec(hv), ho)}
		if !o {
			r = append(r, fmt.Sprint(bDec(nb().Quo(nb().Mul(in.x.b[i], in.y.b[i]), in.z.b[i])), false))
		}
		return r
	})
	agree(t, "ExpMod", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].ExpMod(in.ye.s[i], in.z.s[i])), fmt.Sprint(new(big.Int).Exp(in.x.b[i], in.ye.b[i], in.z.b[i]))}
	})
	agree(t, "MulDiv/big", func(i int) []string {
		return []string{fmt.Sprint(in.fx.s[i].MulDiv(in.fy.s[i], in.y.s[i])), fmt.Sprint(new(big.Int).Quo(new(big.Int).Mul(in.fx.b[i], in.fy.b[i]), in.y.b[i]))}
	})
	agree(t, "MulDivMod", func(i int) []string {
		return []string{fmt.Sprint(in.fx.s[i].MulDivMod(in.fy.s[i], in.y.s[i])), fmt.Sprint(new(big.Int).QuoRem(new(big.Int).Mul(in.fx.b[i], in.fy.b[i]), in.y.b[i], new(big.Int)))}
	})

	// powers and roots
	for _, set := range []struct {
		name string
		x, y u256
		mod  *big.Int
	}{{"fit", in.ex, in.ey, nil}, {"full", in.x, in.y, Two256}} {
		x, y, mod := set.x, set.y, set.mod
		agree(t, "Exp/"+set.name, func(i int) []string {
			return []string{sDec(x.s[i].Expu(y.s[i])), hDec(h.Exp(&x.h[i], &y.h[i])), bDec(nb().Exp(x.b[i], y.b[i], mod))}
		})
	}
	agree(t, "Sqrt", func(i int) []string {
		return []string{sDec(in.x.s[i].Sqrt()), hDec(h.Sqrt(&in.x.h[i])), bDec(nb().Sqrt(in.x.b[i]))}
	})
	agree(t, "Log10", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Log10()), fmt.Sprint(in.x.h[i].Log10())}
	})

	// comparisons
	agree(t, "Cmp", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Cmp(in.y.s[i])), fmt.Sprint(in.x.h[i].Cmp(&in.y.h[i])), fmt.Sprint(in.x.b[i].Cmp(in.y.b[i]))}
	})
	agree(t, "Lt", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].LessThan(in.y.s[i])), fmt.Sprint(in.x.h[i].Lt(&in.y.h[i]))}
	})
	agree(t, "Eq", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i] == in.y.s[i]), fmt.Sprint(in.x.h[i].Eq(&in.y.h[i]))}
	})
	agree(t, "IsZero", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].IsZero()), fmt.Sprint(in.x.h[i].IsZero())}
	})
	agree(t, "Cmp64", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Cmp64(in.k[i])), fmt.Sprint(in.x.h[i].CmpUint64(in.k[i]))}
	})
	agree(t, "CmpBig", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].CmpBig(in.y.b[i])), fmt.Sprint(in.x.h[i].CmpBig(in.y.b[i])), fmt.Sprint(in.x.b[i].Cmp(in.y.b[i]))}
	})
	agree(t, "Cmp64/big", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Cmp64(in.k[i])), fmt.Sprint(in.x.b[i].Cmp(in.kb[i]))}
	})
	agree(t, "Cmp128", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Cmp128(in.w[i])), fmt.Sprint(in.x.b[i].Cmp(in.wb[i]))}
	})
	agree(t, "Sign", func(i int) []string { return []string{fmt.Sprint(in.x.s[i].Sign()), fmt.Sprint(in.x.b[i].Sign())} })
	agree(t, "IsUint64", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].IsUint64()), fmt.Sprint(in.x.b[i].IsUint64())}
	})
	agree(t, "IsInt64", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].IsInt64()), fmt.Sprint(in.x.b[i].IsInt64())}
	})

	// shifts, logic and bits
	agree(t, "Lsh", func(i int) []string {
		return []string{sDec(in.x.s[i].Lsh(in.n[i])), hDec(h.Lsh(&in.x.h[i], in.nu[i]))}
	})
	agree(t, "Rsh", func(i int) []string {
		return []string{sDec(in.x.s[i].Rsh(in.n[i])), hDec(h.Rsh(&in.x.h[i], in.nu[i])), bDec(nb().Rsh(in.x.b[i], in.nu[i]))}
	})
	agree(t, "And", func(i int) []string {
		return []string{sDec(in.x.s[i].And(in.y.s[i])), hDec(h.And(&in.x.h[i], &in.y.h[i])), bDec(nb().And(in.x.b[i], in.y.b[i]))}
	})
	agree(t, "Or", func(i int) []string {
		return []string{sDec(in.x.s[i].Or(in.y.s[i])), hDec(h.Or(&in.x.h[i], &in.y.h[i])), bDec(nb().Or(in.x.b[i], in.y.b[i]))}
	})
	agree(t, "Xor", func(i int) []string {
		return []string{sDec(in.x.s[i].Xor(in.y.s[i])), hDec(h.Xor(&in.x.h[i], &in.y.h[i])), bDec(nb().Xor(in.x.b[i], in.y.b[i]))}
	})
	agree(t, "Not", func(i int) []string {
		return []string{sDec(in.x.s[i].Not()), hDec(h.Not(&in.x.h[i]))}
	})
	agree(t, "BitLen", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].BitLen()), fmt.Sprint(in.x.h[i].BitLen()), fmt.Sprint(in.x.b[i].BitLen())}
	})
	agree(t, "ByteLen", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].ByteLen()), fmt.Sprint(in.x.h[i].ByteLen())}
	})
	agree(t, "Byte", func(i int) []string {
		h = in.x.h[i]
		return []string{fmt.Sprint(in.x.s[i].Byte(in.idx[i])), hDec(h.Byte(&in.hidx[i]))}
	})
	agree(t, "ExtendSign", func(i int) []string {
		return []string{sDec(in.x.s[i].ExtendSign(in.idx[i])), hDec(h.ExtendSign(&in.x.h[i], &in.hidx[i]))}
	})
	agree(t, "Lsh/big", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Lsh(in.ln[i])), fmt.Sprint(new(big.Int).Lsh(in.x.b[i], uint(in.ln[i])))}
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
	agree(t, "TrailingZeros", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].TrailingZeros()), fmt.Sprint(in.x.b[i].TrailingZeroBits())}
	})

	// bytes
	agree(t, "Bytes32", func(i int) []string {
		var w [32]byte
		in.x.b[i].FillBytes(w[:])
		return []string{fmt.Sprint(in.x.s[i].Bytes32()), fmt.Sprint(in.x.h[i].Bytes32()), fmt.Sprint(w)}
	})
	agree(t, "Bytes20", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Bytes20()), fmt.Sprint(in.x.h[i].Bytes20())}
	})
	agree(t, "PutBytes32", func(i int) []string {
		var a, c [32]byte
		in.x.s[i].PutBytes32(a[:])
		in.x.h[i].PutUint256(c[:])
		return []string{fmt.Sprint(a), fmt.Sprint(c)}
	})
	agree(t, "Bytes", func(i int) []string {
		return []string{fmt.Sprint(in.x.s[i].Bytes()), fmt.Sprint(in.x.h[i].Bytes()), fmt.Sprint(in.x.b[i].Bytes())}
	})
	agree(t, "FromBytes", func(i int) []string {
		v, ok := fat.Uint256FromBytes(in.be[i])
		return []string{fmt.Sprint(sDec(v), ok), fmt.Sprint(hDec(h.SetBytes(in.be[i])), true), fmt.Sprint(bDec(nb().SetBytes(in.be[i])), true)}
	})

	// conversions
	agree(t, "Float64", func(i int) []string {
		f, _ := in.x.b[i].Float64()
		return []string{fmt.Sprint(in.x.s[i].Float64()), fmt.Sprint(in.x.h[i].Float64()), fmt.Sprint(f)}
	})
	agree(t, "FromBig", func(i int) []string {
		v, err := fat.Uint256FromBig(in.x.b[i])
		of := h.SetFromBig(in.x.b[i])
		return []string{fmt.Sprint(sDec(v), err == nil), fmt.Sprint(hDec(&h), !of)}
	})
	agree(t, "Big", func(i int) []string {
		return []string{bDec(in.x.s[i].Big()), bDec(in.x.h[i].ToBig())}
	})
	agree(t, "PutBig", func(i int) []string {
		c := nb()
		in.x.h[i].IntoBig(&c)
		return []string{bDec(in.x.s[i].PutBig(nb())), bDec(c)}
	})
	agree(t, "Uint64", func(i int) []string {
		return []string{fmt.Sprint(in.small.s[i].Uint64()), fmt.Sprint(in.small.b[i].Uint64())}
	})
	agree(t, "Int64", func(i int) []string {
		return []string{fmt.Sprint(in.small.s[i].Int64()), fmt.Sprint(in.small.b[i].Int64())}
	})

	// text and encodings
	agree(t, "String", func(i int) []string {
		return []string{in.x.s[i].String(), in.x.h[i].Dec(), in.x.b[i].String()}
	})
	agree(t, "Hex", func(i int) []string {
		return []string{in.x.s[i].Hex(), in.x.h[i].Hex(), "0x" + in.x.b[i].Text(16)}
	})
	agree(t, "PrettyDec", func(i int) []string {
		return []string{in.x.s[i].PrettyDec(','), in.x.h[i].PrettyDec(',')}
	})
	agree(t, "FromDec", func(i int) []string {
		v, err := fat.Uint256FromDec(in.dec[i])
		herr := h.SetFromDecimal(in.dec[i])
		bv, ok := nb().SetString(in.dec[i], 10)
		return []string{fmt.Sprint(sDec(v), err == nil), fmt.Sprint(hDec(&h), herr == nil), fmt.Sprint(bDec(bv), ok)}
	})
	agree(t, "FromHex", func(i int) []string {
		v, err := fat.Uint256FromHex(in.hex[i])
		herr := h.SetFromHex(in.hex[i])
		bv, ok := nb().SetString(in.hex[i][2:], 16)
		return []string{fmt.Sprint(sDec(v), err == nil), fmt.Sprint(hDec(&h), herr == nil), fmt.Sprint(bDec(bv), ok)}
	})
	agree(t, "MarshalText", func(i int) []string {
		a, _ := in.x.s[i].MarshalText()
		c, _ := in.x.h[i].MarshalText()
		d, _ := in.x.b[i].MarshalText()
		return []string{string(a), string(c), string(d)}
	})
	agree(t, "UnmarshalText", func(i int) []string {
		var v fat.Uint256
		bv := nb()
		return []string{fmt.Sprint(v.UnmarshalText(in.decb[i]), sDec(v)), fmt.Sprint(h.UnmarshalText(in.decb[i]), hDec(&h)), fmt.Sprint(bv.UnmarshalText(in.decb[i]), bDec(bv))}
	})
	agree(t, "MarshalJSON", func(i int) []string {
		a, _ := in.x.s[i].MarshalJSON()
		c, _ := in.x.h[i].MarshalJSON()
		return []string{string(a), string(c)}
	})
	agree(t, "UnmarshalJSON", func(i int) []string {
		var v fat.Uint256
		return []string{fmt.Sprint(v.UnmarshalJSON(in.js[i]), sDec(v)), fmt.Sprint(h.UnmarshalJSON(in.js[i]), hDec(&h))}
	})
	agree(t, "AppendDec", func(i int) []string {
		return []string{string(in.x.s[i].AppendDec(nil)), string(in.x.b[i].Append(nil, 10))}
	})
	agree(t, "AppendText", func(i int) []string {
		return []string{func() string { b, err := in.x.s[i].AppendText(nil); return fmt.Sprint(string(b), err) }(), func() string { b, err := in.x.b[i].AppendText(nil); return fmt.Sprint(string(b), err) }()}
	})
	agree(t, "AppendHex", func(i int) []string {
		return []string{string(in.x.s[i].AppendHex(nil)), func() string {
			if in.x.b[i].Sign() < 0 {
				return "-0x" + new(big.Int).Neg(in.x.b[i]).Text(16)
			}
			return "0x" + in.x.b[i].Text(16)
		}()}
	})
	agree(t, "MarshalJSON/big", func(i int) []string {
		return []string{func() string { b, _ := in.x.s[i].MarshalJSON(); return strings.Trim(string(b), `"`) }(), func() string { b, _ := in.x.b[i].MarshalJSON(); return string(b) }()}
	})
	agree(t, "UnmarshalJSON/big", func(i int) []string {
		return []string{func() string { var v fat.Uint256; err := v.UnmarshalJSON(in.js[i]); return fmt.Sprint(v, err) }(), func() string { v := new(big.Int); err := v.UnmarshalJSON(in.jsb[i]); return fmt.Sprint(v, err) }()}
	})
	agree(t, "Format", func(i int) []string { return []string{fmt.Sprintf("%d", in.x.s[i]), fmt.Sprintf("%d", in.x.b[i])} })
	agree(t, "GobEncode", func(i int) []string { return []string{"same", "same"} })
	agree(t, "GobDecode", func(i int) []string {
		return []string{func() string { var v fat.Uint256; err := v.GobDecode(in.gob[i]); return fmt.Sprint(v, err) }(), func() string { v := new(big.Int); err := v.GobDecode(in.gobb[i]); return fmt.Sprint(v, err) }()}
	})

	// chains of chainN sums, each on the one before; the first 256 of them
	t.Run("Chain", func(t *testing.T) {
		for i := range 256 {
			k := chainAt(i)
			var h uint256.Int
			chainHoliman(&h, (*[chainN + 1]uint256.Int)(in.cx.h[k:]))
			bz := new(big.Int)
			chainBig(bz, in.cx.b[k:])
			require.Equal(On(t, i), bDec(bz), sDec(chainFat((*[chainN + 1]fat.Uint256)(in.cx.s[k:]))))
			require.Equal(On(t, i), bDec(bz), hDec(&h))
		}
	})

	// no side may change its inputs
	agree(t, "inputs", func(i int) []string { return []string{sDec(in.x.s[i]), hDec(&in.x.h[i]), bDec(in.x.b[i])} })
}

// BenchmarkUint256Arith compares addition, subtraction and multiplication with holiman and
// math/big. holiman wraps, so its forms pair with the u forms of fat on pairs that fit, where
// math/big gives the same, the o forms on random lengths. The checked forms meet math/big where
// they do not overflow, a uint64 or 128-bit operand is a big.Int built before the timer
func BenchmarkUint256Arith(b *testing.B) {
	in := inputsU256()
	for _, set := range []struct {
		name string
		x, y u256
	}{{"fit", in.fx, in.fy}, {"small", in.small, in.small2}} {
		x, y := set.x, set.y
		b.Run("op=Add/in="+set.name+"/impl=fat", func(b *testing.B) {
			for i := range b.N {
				sU256 = x.s[i&m].Addu(y.s[i&m])
			}
		})
		b.Run("op=Add/in="+set.name+"/impl=holiman", func(b *testing.B) {
			for i := range b.N {
				hz.Add(&x.h[i&m], &y.h[i&m])
			}
		})
		b.Run("op=Add/in="+set.name+"/impl=big", func(b *testing.B) {
			for i := range b.N {
				bz.Add(x.b[i&m], y.b[i&m])
			}
		})
		b.Run("op=Mul/in="+set.name+"/impl=fat", func(b *testing.B) {
			for i := range b.N {
				sU256 = x.s[i&m].Mulu(y.s[i&m])
			}
		})
		b.Run("op=Mul/in="+set.name+"/impl=holiman", func(b *testing.B) {
			for i := range b.N {
				hz.Mul(&x.h[i&m], &y.h[i&m])
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
			sU256 = in.fx.s[i&m].Subu(in.fy.s[i&m])
		}
	})
	b.Run("op=Sub/in=fit/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			hz.Sub(&in.fx.h[i&m], &in.fy.h[i&m])
		}
	})
	b.Run("op=Sub/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Sub(in.fx.b[i&m], in.fy.b[i&m])
		}
	})
	b.Run("op=Addo/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256, sBool = in.x.s[i&m].Addo(in.y.s[i&m])
		}
	})
	b.Run("op=Addo/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			_, sBool = hz.AddOverflow(&in.x.h[i&m], &in.y.h[i&m])
		}
	})
	b.Run("op=Subo/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256, sBool = in.x.s[i&m].Subo(in.y.s[i&m])
		}
	})
	b.Run("op=Subo/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			_, sBool = hz.SubOverflow(&in.x.h[i&m], &in.y.h[i&m])
		}
	})
	b.Run("op=Mulo/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256, sBool = in.x.s[i&m].Mulo(in.y.s[i&m])
		}
	})
	b.Run("op=Mulo/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			_, sBool = hz.MulOverflow(&in.x.h[i&m], &in.y.h[i&m])
		}
	})
	b.Run("op=Add64/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.x.s[i&m].Add64u(in.k[i&m])
		}
	})
	b.Run("op=Add64/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			hz.AddUint64(&in.x.h[i&m], in.k[i&m])
		}
	})
	b.Run("op=Sub64/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.x.s[i&m].Sub64u(in.k[i&m])
		}
	})
	b.Run("op=Sub64/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			hz.SubUint64(&in.x.h[i&m], in.k[i&m])
		}
	})
	b.Run("op=Add64/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Add(in.x.b[i&m], in.kb[i&m])
		}
	})
	b.Run("op=Sub64/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.x.s[i&m].Sub64(in.ks[i&m])
		}
	})
	b.Run("op=Sub64/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Sub(in.x.b[i&m], in.ksb[i&m])
		}
	})
	b.Run("op=Mul64/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.x64.s[i&m].Mul64(in.k[i&m])
		}
	})
	b.Run("op=Mul64/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Mul(in.x64.b[i&m], in.kb[i&m])
		}
	})
	b.Run("op=Add128/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.x.s[i&m].Add128(in.w[i&m])
		}
	})
	b.Run("op=Add128/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Add(in.x.b[i&m], in.wb[i&m])
		}
	})
	b.Run("op=Sub128/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.x.s[i&m].Sub128(in.ws[i&m])
		}
	})
	b.Run("op=Sub128/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Sub(in.x.b[i&m], in.wsb[i&m])
		}
	})
	b.Run("op=Mul128/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.x128.s[i&m].Mul128(in.w[i&m])
		}
	})
	b.Run("op=Mul128/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Mul(in.x128.b[i&m], in.wb[i&m])
		}
	})
	b.Run("op=Mule/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256, sU256b = in.x.s[i&m].Mule(in.y.s[i&m])
		}
	})
	b.Run("op=Mule/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Mul(in.x.b[i&m], in.y.b[i&m])
		}
	})
}

// BenchmarkUint256Div compares division with holiman and math/big, the divisors never zero: a zero
// one panics in fat, gives 0 in holiman. A uint64 or 128-bit operand is a big.Int built before
// the timer
func BenchmarkUint256Div(b *testing.B) {
	in := inputsU256()
	for _, set := range []struct {
		name string
		x, y u256
	}{{"mixed", in.x, in.y}, {"div64", in.x, in.small}, {"full", in.full, in.y}} {
		x, y := set.x, set.y
		b.Run("op=Div/in="+set.name+"/impl=fat", func(b *testing.B) {
			for i := range b.N {
				sU256 = x.s[i&m].Div(y.s[i&m])
			}
		})
		b.Run("op=Div/in="+set.name+"/impl=holiman", func(b *testing.B) {
			for i := range b.N {
				hz.Div(&x.h[i&m], &y.h[i&m])
			}
		})
		b.Run("op=Div/in="+set.name+"/impl=big", func(b *testing.B) {
			for i := range b.N {
				bz.Quo(x.b[i&m], y.b[i&m])
			}
		})
		b.Run("op=Mod/in="+set.name+"/impl=fat", func(b *testing.B) {
			for i := range b.N {
				sU256 = x.s[i&m].Mod(y.s[i&m])
			}
		})
		b.Run("op=Mod/in="+set.name+"/impl=holiman", func(b *testing.B) {
			for i := range b.N {
				hz.Mod(&x.h[i&m], &y.h[i&m])
			}
		})
		b.Run("op=Mod/in="+set.name+"/impl=big", func(b *testing.B) {
			for i := range b.N {
				bz.Rem(x.b[i&m], y.b[i&m])
			}
		})
		b.Run("op=DivMod/in="+set.name+"/impl=fat", func(b *testing.B) {
			for i := range b.N {
				sU256, sU256b = x.s[i&m].DivMod(y.s[i&m])
			}
		})
		b.Run("op=DivMod/in="+set.name+"/impl=holiman", func(b *testing.B) {
			for i := range b.N {
				hz.DivMod(&x.h[i&m], &y.h[i&m], &hm)
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
			sU256 = in.x.s[i&m].Div64(in.k[i&m])
		}
	})
	b.Run("op=Div64/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Quo(in.x.b[i&m], in.kb[i&m])
		}
	})
	b.Run("op=Mod64/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU64 = in.x.s[i&m].Mod64(in.k[i&m])
		}
	})
	b.Run("op=Mod64/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Rem(in.x.b[i&m], in.kb[i&m])
		}
	})
	b.Run("op=DivMod64/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256, sU64 = in.x.s[i&m].DivMod64(in.k[i&m])
		}
	})
	b.Run("op=DivMod64/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.QuoRem(in.x.b[i&m], in.kb[i&m], bm)
		}
	})
	b.Run("op=Div128/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.x.s[i&m].Div128(in.w[i&m])
		}
	})
	b.Run("op=Div128/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Quo(in.x.b[i&m], in.wb[i&m])
		}
	})
	b.Run("op=Mod128/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].Mod128(in.w[i&m])
		}
	})
	b.Run("op=Mod128/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Rem(in.x.b[i&m], in.wb[i&m])
		}
	})
	b.Run("op=DivMod128/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256, sU128 = in.x.s[i&m].DivMod128(in.w[i&m])
		}
	})
	b.Run("op=DivMod128/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.QuoRem(in.x.b[i&m], in.wb[i&m], bm)
		}
	})
}

// BenchmarkUint256Modular compares the modular forms, MulDiv and ExpMod with holiman and math/big.
// math/big does a modular form as two calls, as code on it would
func BenchmarkUint256Modular(b *testing.B) {
	in := inputsU256()
	b.Run("op=AddMod/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.x.s[i&m].AddMod(in.y.s[i&m], in.z.s[i&m])
		}
	})
	b.Run("op=AddMod/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			hz.AddMod(&in.x.h[i&m], &in.y.h[i&m], &in.z.h[i&m])
		}
	})
	b.Run("op=AddMod/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Mod(bz.Add(in.x.b[i&m], in.y.b[i&m]), in.z.b[i&m])
		}
	})
	b.Run("op=MulMod/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.x.s[i&m].MulMod(in.y.s[i&m], in.z.s[i&m])
		}
	})
	b.Run("op=MulMod/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			hz.MulMod(&in.x.h[i&m], &in.y.h[i&m], &in.z.h[i&m])
		}
	})
	b.Run("op=MulMod/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Mod(bz.Mul(in.x.b[i&m], in.y.b[i&m]), in.z.b[i&m])
		}
	})
	// the overflow flag of both is a quotient past 256 bits,
	// math/big then keeps the whole of it
	b.Run("op=MulDiv/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256, sBool = in.x.s[i&m].MulDivo(in.y.s[i&m], in.z.s[i&m])
		}
	})
	b.Run("op=MulDiv/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			_, sBool = hz.MulDivOverflow(&in.x.h[i&m], &in.y.h[i&m], &in.z.h[i&m])
		}
	})
	b.Run("op=MulDiv/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Quo(bz.Mul(in.x.b[i&m], in.y.b[i&m]), in.z.b[i&m])
		}
	})
	b.Run("op=ExpMod/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.x.s[i&m].ExpMod(in.ye.s[i&m], in.z.s[i&m])
		}
	})
	b.Run("op=ExpMod/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Exp(in.x.b[i&m], in.ye.b[i&m], in.z.b[i&m])
		}
	})
	b.Run("op=MulDiv/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.fx.s[i&m].MulDiv(in.fy.s[i&m], in.y.s[i&m])
		}
	})
	b.Run("op=MulDiv/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Quo(bz.Mul(in.fx.b[i&m], in.fy.b[i&m]), in.y.b[i&m])
		}
	})
	b.Run("op=MulDivMod/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256, sU256b = in.fx.s[i&m].MulDivMod(in.fy.s[i&m], in.y.s[i&m])
		}
	})
	b.Run("op=MulDivMod/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.QuoRem(bz.Mul(in.fx.b[i&m], in.fy.b[i&m]), in.y.b[i&m], bm)
		}
	})
}

// BenchmarkUint256Pow compares Exp, Sqrt and Log10 with holiman and math/big, Exp on bases and
// exponents with x^y in 255 bits and on exponents of random length wrapping at 2^256
func BenchmarkUint256Pow(b *testing.B) {
	in := inputsU256()
	for _, set := range []struct {
		name string
		x, y u256
		mod  *big.Int
	}{{"fit", in.ex, in.ey, nil}, {"full", in.x, in.y, Two256}} {
		x, y, mod := set.x, set.y, set.mod
		b.Run("op=Exp/in="+set.name+"/impl=fat", func(b *testing.B) {
			for i := range b.N {
				sU256 = x.s[i&m].Expu(y.s[i&m])
			}
		})
		b.Run("op=Exp/in="+set.name+"/impl=holiman", func(b *testing.B) {
			for i := range b.N {
				hz.Exp(&x.h[i&m], &y.h[i&m])
			}
		})
		b.Run("op=Exp/in="+set.name+"/impl=big", func(b *testing.B) {
			for i := range b.N {
				bz.Exp(x.b[i&m], y.b[i&m], mod)
			}
		})
	}
	b.Run("op=Sqrt/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.x.s[i&m].Sqrt()
		}
	})
	b.Run("op=Sqrt/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			hz.Sqrt(&in.x.h[i&m])
		}
	})
	b.Run("op=Sqrt/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Sqrt(in.x.b[i&m])
		}
	})
	b.Run("op=Log10/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.s[i&m].Log10()
		}
	})
	b.Run("op=Log10/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sUint = in.x.h[i&m].Log10()
		}
	})
}

// BenchmarkUint256Cmp compares the comparisons and tests with holiman and math/big on random
// lengths
func BenchmarkUint256Cmp(b *testing.B) {
	in := inputsU256()
	b.Run("op=Cmp/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.s[i&m].Cmp(in.y.s[i&m])
		}
	})
	b.Run("op=Cmp/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.h[i&m].Cmp(&in.y.h[i&m])
		}
	})
	b.Run("op=Cmp/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.b[i&m].Cmp(in.y.b[i&m])
		}
	})
	b.Run("op=Lt/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sBool = in.x.s[i&m].LessThan(in.y.s[i&m])
		}
	})
	b.Run("op=Lt/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sBool = in.x.h[i&m].Lt(&in.y.h[i&m])
		}
	})
	b.Run("op=Eq/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sBool = in.x.s[i&m] == in.y.s[i&m]
		}
	})
	b.Run("op=Eq/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sBool = in.x.h[i&m].Eq(&in.y.h[i&m])
		}
	})
	b.Run("op=IsZero/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sBool = in.x.s[i&m].IsZero()
		}
	})
	b.Run("op=IsZero/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sBool = in.x.h[i&m].IsZero()
		}
	})
	b.Run("op=Cmp64/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.s[i&m].Cmp64(in.k[i&m])
		}
	})
	b.Run("op=Cmp64/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.h[i&m].CmpUint64(in.k[i&m])
		}
	})
	b.Run("op=CmpBig/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.s[i&m].CmpBig(in.y.b[i&m])
		}
	})
	b.Run("op=CmpBig/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.h[i&m].CmpBig(in.y.b[i&m])
		}
	})
	b.Run("op=CmpBig/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.b[i&m].Cmp(in.y.b[i&m])
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

// BenchmarkUint256Bits compares shifts, logic and bit counts with holiman and math/big on random
// lengths. math/big has no shift within 256 bits: its Lsh runs on counts that keep x in them
func BenchmarkUint256Bits(b *testing.B) {
	in := inputsU256()
	b.Run("op=Lsh/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.x.s[i&m].Lsh(in.n[i&m])
		}
	})
	b.Run("op=Lsh/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			hz.Lsh(&in.x.h[i&m], in.nu[i&m])
		}
	})
	b.Run("op=Rsh/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.x.s[i&m].Rsh(in.n[i&m])
		}
	})
	b.Run("op=Rsh/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			hz.Rsh(&in.x.h[i&m], in.nu[i&m])
		}
	})
	b.Run("op=Rsh/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Rsh(in.x.b[i&m], in.nu[i&m])
		}
	})
	b.Run("op=And/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.x.s[i&m].And(in.y.s[i&m])
		}
	})
	b.Run("op=And/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			hz.And(&in.x.h[i&m], &in.y.h[i&m])
		}
	})
	b.Run("op=And/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.And(in.x.b[i&m], in.y.b[i&m])
		}
	})
	b.Run("op=Or/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.x.s[i&m].Or(in.y.s[i&m])
		}
	})
	b.Run("op=Or/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			hz.Or(&in.x.h[i&m], &in.y.h[i&m])
		}
	})
	b.Run("op=Or/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Or(in.x.b[i&m], in.y.b[i&m])
		}
	})
	b.Run("op=Xor/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.x.s[i&m].Xor(in.y.s[i&m])
		}
	})
	b.Run("op=Xor/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			hz.Xor(&in.x.h[i&m], &in.y.h[i&m])
		}
	})
	b.Run("op=Xor/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Xor(in.x.b[i&m], in.y.b[i&m])
		}
	})
	b.Run("op=Not/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.x.s[i&m].Not()
		}
	})
	b.Run("op=Not/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			hz.Not(&in.x.h[i&m])
		}
	})
	b.Run("op=BitLen/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.s[i&m].BitLen()
		}
	})
	b.Run("op=BitLen/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.h[i&m].BitLen()
		}
	})
	b.Run("op=BitLen/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.b[i&m].BitLen()
		}
	})
	b.Run("op=ByteLen/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.s[i&m].ByteLen()
		}
	})
	b.Run("op=ByteLen/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.h[i&m].ByteLen()
		}
	})
	// holiman reads the byte of z in place, so x goes there first
	b.Run("op=Byte/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sByte = in.x.s[i&m].Byte(in.idx[i&m])
		}
	})
	b.Run("op=Byte/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			hz = in.x.h[i&m]
			hz.Byte(&in.hidx[i&m])
		}
	})
	b.Run("op=ExtendSign/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.x.s[i&m].ExtendSign(in.idx[i&m])
		}
	})
	b.Run("op=ExtendSign/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			hz.ExtendSign(&in.x.h[i&m], &in.hidx[i&m])
		}
	})
	b.Run("op=Lsh/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.x.s[i&m].Lsh(in.ln[i&m])
		}
	})
	b.Run("op=Lsh/in=fit/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.Lsh(in.x.b[i&m], uint(in.ln[i&m]))
		}
	})
	b.Run("op=AndNot/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = in.x.s[i&m].AndNot(in.y.s[i&m])
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
			sU256 = in.x.s[i&m].WithBit(in.bi[i&m], in.bv[i&m])
		}
	})
	b.Run("op=WithBit/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.SetBit(in.x.b[i&m], in.bi[i&m], in.bv[i&m])
		}
	})
	b.Run("op=TrailingZeros/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sInt = in.x.s[i&m].TrailingZeros()
		}
	})
	b.Run("op=TrailingZeros/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sUint = in.x.b[i&m].TrailingZeroBits()
		}
	})
}

// BenchmarkUint256Bytes compares the big-endian forms with holiman and math/big on random lengths
func BenchmarkUint256Bytes(b *testing.B) {
	in := inputsU256()
	b.Run("op=Bytes32/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sB32 = in.x.s[i&m].Bytes32()
		}
	})
	b.Run("op=Bytes32/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sB32 = in.x.h[i&m].Bytes32()
		}
	})
	b.Run("op=Bytes20/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sB20 = in.x.s[i&m].Bytes20()
		}
	})
	b.Run("op=Bytes20/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sB20 = in.x.h[i&m].Bytes20()
		}
	})
	b.Run("op=PutBytes32/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			in.x.s[i&m].PutBytes32(sB32[:])
		}
	})
	b.Run("op=PutBytes32/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			in.x.h[i&m].PutUint256(sB32[:])
		}
	})
	b.Run("op=PutBytes32/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			in.x.b[i&m].FillBytes(sB32[:])
		}
	})
	b.Run("op=Bytes/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sBytes = in.x.s[i&m].Bytes()
		}
	})
	b.Run("op=Bytes/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sBytes = in.x.h[i&m].Bytes()
		}
	})
	b.Run("op=Bytes/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sBytes = in.x.b[i&m].Bytes()
		}
	})
	b.Run("op=FromBytes/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256, sBool = fat.Uint256FromBytes(in.be[i&m])
		}
	})
	b.Run("op=FromBytes/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			hz.SetBytes(in.be[i&m])
		}
	})
	b.Run("op=FromBytes/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			bz.SetBytes(in.be[i&m])
		}
	})
}

// BenchmarkUint256Convert compares the conversions with holiman and math/big on random lengths
func BenchmarkUint256Convert(b *testing.B) {
	in := inputsU256()
	b.Run("op=Float64/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sFloat = in.x.s[i&m].Float64()
		}
	})
	b.Run("op=Float64/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sFloat = in.x.h[i&m].Float64()
		}
	})
	b.Run("op=Float64/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sFloat, _ = in.x.b[i&m].Float64()
		}
	})
	b.Run("op=FromBig/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256, sErr = fat.Uint256FromBig(in.x.b[i&m])
		}
	})
	b.Run("op=FromBig/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sBool = hz.SetFromBig(in.x.b[i&m])
		}
	})
	b.Run("op=Big/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			bm = in.x.s[i&m].Big()
		}
	})
	b.Run("op=Big/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			bm = in.x.h[i&m].ToBig()
		}
	})
	b.Run("op=PutBig/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			in.x.s[i&m].PutBig(bz)
		}
	})
	b.Run("op=PutBig/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			in.x.h[i&m].IntoBig(&bz)
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
}

// BenchmarkUint256Text compares text and encodings with holiman and math/big on random lengths:
// math/big writes JSON as a bare number and its own gob, each side reads its own
func BenchmarkUint256Text(b *testing.B) {
	in := inputsU256()
	b.Run("op=String/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sString = in.x.s[i&m].String()
		}
	})
	b.Run("op=String/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sString = in.x.h[i&m].Dec()
		}
	})
	b.Run("op=String/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sString = in.x.b[i&m].String()
		}
	})
	// math/big writes no 0x prefix
	b.Run("op=Hex/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sString = in.x.s[i&m].Hex()
		}
	})
	b.Run("op=Hex/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sString = in.x.h[i&m].Hex()
		}
	})
	b.Run("op=Hex/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sString = in.x.b[i&m].Text(16)
		}
	})
	b.Run("op=PrettyDec/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sString = in.x.s[i&m].PrettyDec(',')
		}
	})
	b.Run("op=PrettyDec/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sString = in.x.h[i&m].PrettyDec(',')
		}
	})
	b.Run("op=FromDec/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256, sErr = fat.Uint256FromDec(in.dec[i&m])
		}
	})
	b.Run("op=FromDec/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sErr = hz.SetFromDecimal(in.dec[i&m])
		}
	})
	b.Run("op=FromDec/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			_, sBool = bz.SetString(in.dec[i&m], 10)
		}
	})
	// math/big takes the digits after 0x
	b.Run("op=FromHex/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256, sErr = fat.Uint256FromHex(in.hex[i&m])
		}
	})
	b.Run("op=FromHex/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sErr = hz.SetFromHex(in.hex[i&m])
		}
	})
	b.Run("op=FromHex/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			_, sBool = bz.SetString(in.hex[i&m][2:], 16)
		}
	})
	b.Run("op=MarshalText/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sBytes, sErr = in.x.s[i&m].MarshalText()
		}
	})
	b.Run("op=MarshalText/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sBytes, sErr = in.x.h[i&m].MarshalText()
		}
	})
	b.Run("op=MarshalText/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sBytes, sErr = in.x.b[i&m].MarshalText()
		}
	})
	b.Run("op=UnmarshalText/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sErr = sU256.UnmarshalText(in.decb[i&m])
		}
	})
	b.Run("op=UnmarshalText/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sErr = hz.UnmarshalText(in.decb[i&m])
		}
	})
	b.Run("op=UnmarshalText/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sErr = bz.UnmarshalText(in.decb[i&m])
		}
	})
	// both write and read a quoted decimal, math/big a bare number
	b.Run("op=MarshalJSON/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sBytes, sErr = in.x.s[i&m].MarshalJSON()
		}
	})
	b.Run("op=MarshalJSON/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sBytes, sErr = in.x.h[i&m].MarshalJSON()
		}
	})
	b.Run("op=UnmarshalJSON/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sErr = sU256.UnmarshalJSON(in.js[i&m])
		}
	})
	b.Run("op=UnmarshalJSON/in=mixed/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			sErr = hz.UnmarshalJSON(in.js[i&m])
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
	b.Run("op=MarshalJSON/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sBytes, sErr = in.x.b[i&m].MarshalJSON()
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
			sErr = sU256.GobDecode(in.gob[i&m])
		}
	})
	b.Run("op=GobDecode/in=mixed/impl=big", func(b *testing.B) {
		for i := range b.N {
			sErr = bz.GobDecode(in.gobb[i&m])
		}
	})
}

// BenchmarkUint256DivToFloat64 compares DivToFloat64 with big.Rat.Float64 of x/y, the rationals
// built before the timer
func BenchmarkUint256DivToFloat64(b *testing.B) {
	for _, set := range DivFloatIn(256, false) {
		x, y := newU256(set.X), newU256(set.Y)
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

// BenchmarkUint256Chain compares a chain of chainN sums, x0 + x1 + ... written
// out in one expression, each sum on the one before, as each API writes it:
// fat on values, holiman in place on a uint256.Int, an array the compiler
// keeps in memory, math/big on a *big.Int. The values are below 2^253, so the
// sum never wraps and math/big gives the same; ns/add is the time of one sum
func BenchmarkUint256Chain(b *testing.B) {
	in := inputsU256()
	b.Run("op=Add/in=chain/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU256 = chainFat((*[chainN + 1]fat.Uint256)(in.cx.s[chainAt(i):]))
		}
		b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*chainN), "ns/add")
	})
	b.Run("op=Add/in=chain/impl=holiman", func(b *testing.B) {
		for i := range b.N {
			chainHoliman(&hz, (*[chainN + 1]uint256.Int)(in.cx.h[chainAt(i):]))
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

// chainFat and chainHoliman add the chainN+1 values of w as one chain
func chainFat(w *[chainN + 1]fat.Uint256) fat.Uint256 {
	return w[0].Addu(w[1]).Addu(w[2]).Addu(w[3]).Addu(w[4]).Addu(w[5])
}

func chainHoliman(z *uint256.Int, w *[chainN + 1]uint256.Int) {
	z.Add(&w[0], &w[1]).Add(z, &w[2]).Add(z, &w[3]).Add(z, &w[4]).Add(z, &w[5])
}
