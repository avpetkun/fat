package testutil

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/avpetkun/fat/internal/require"
)

// RunErrorText checks that a parse error names the func, input and reason
func RunErrorText[T Num[T], PT NumPtr[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	typ := "fat." + k.Name
	for _, c := range []struct{ s, why string }{
		{"x", "invalid digit 'x' at offset 0"},
		{"0b2", "invalid digit 'b' at offset 1"},
		{"--1", "invalid digit '-' at offset 1"},
		{" 1", "invalid digit ' ' at offset 0"},
		{"0x", "no digits after 0x"},
		{"0x1g", "invalid hex digit 'g' at offset 3"},
	} {
		text, js, quoted := PT(new(T)).UnmarshalText([]byte(c.s)), PT(new(T)).UnmarshalJSON([]byte(c.s)), PT(new(T)).UnmarshalJSON([]byte(`"`+c.s+`"`))
		// an unsigned type tells a leading minus apart
		if !k.Signed && c.s[0] == '-' {
			require.Equal(On(t, c.s), typ+`.UnmarshalText: negative value "`+c.s+`"`, fmt.Sprint(text))
			require.Equal(On(t, c.s), typ+`.UnmarshalJSON: negative value "`+c.s+`"`, fmt.Sprint(js))
			require.Equal(On(t, c.s), typ+`.UnmarshalJSON: negative value "`+c.s+`"`, fmt.Sprint(quoted))
			continue
		}
		require.Equal(On(t, c.s), typ+`.UnmarshalText: cannot parse "`+c.s+`": `+c.why, fmt.Sprint(text))
		// JSON as it came, a bare 0x is not a JSON number
		why := c.why
		if strings.HasPrefix(c.s, "0x") {
			why = "0x outside quotes"
		}
		require.Equal(On(t, c.s), typ+`.UnmarshalJSON: cannot parse `+c.s+`: `+why, fmt.Sprint(js))
		require.Equal(On(t, c.s), typ+`.UnmarshalJSON: cannot parse "`+c.s+`": `+c.why, fmt.Sprint(quoted))
	}
	// the least hex past the width: 2^bits, or 2^(bits-1) for a signed type
	over := "0x1" + strings.Repeat("0", k.Bits/4)
	if k.Signed {
		over = "0x8" + strings.Repeat("0", k.Bits/4-1)
	}
	// FromDec and FromHex return zero and the reason on failure.
	// An empty reason is a success, a leading minus fails on an unsigned type
	for _, c := range []struct{ s, dec, hex string }{
		{"0xg", "0x in decimal text", "invalid hex digit 'g' at offset 2"},
		{"0b1", "invalid digit 'b' at offset 1", ""},
		{"ff", "invalid digit 'f' at offset 0", ""},
		{"g", "invalid digit 'g' at offset 0", "invalid hex digit 'g' at offset 0"},
		{"+-1", "invalid digit '-' at offset 1", "invalid hex digit '-' at offset 1"},
		{"1x", "invalid digit 'x' at offset 1", "invalid hex digit 'x' at offset 1"},
		{"0x", "0x in decimal text", "no digits after 0x"},
		{"+", "no digits after the sign", "no digits after the sign"},
		{"", "empty", "empty"},
		{"-x", "invalid digit 'x' at offset 1", "invalid hex digit 'x' at offset 1"},
		{"-5", "", ""},
		{"-0x5", "0x in decimal text", ""},
		{"12x4", "invalid digit 'x' at offset 2", "invalid hex digit 'x' at offset 2"},
		{"9e99", "out of range", ""},
		{"١٢٣", "invalid digit '١' at offset 0", "invalid hex digit '١' at offset 0"},
		{"\x00", `invalid digit '\x00' at offset 0`, `invalid hex digit '\x00' at offset 0`},
		{"-0xfg", "0x in decimal text", "invalid hex digit 'g' at offset 4"},
		{over, "0x in decimal text", "out of range"},
		{k.Over().String(), "out of range", "out of range"},
	} {
		for name, f := range map[string]func(string) (T, error){"FromDec": k.FromDec, "FromHex": k.FromHex} {
			t := On(t, c.s, name)
			why := c.dec
			if name == "FromHex" {
				why = c.hex
			}
			want := "fat." + k.Name + name + ": cannot parse " + strconv.Quote(c.s) + ": " + why
			if !k.Signed && c.s != "" && c.s[0] == '-' {
				want = "fat." + k.Name + name + ": negative value " + strconv.Quote(c.s)
			} else if why == "" {
				x, err := f(c.s)
				require.NoError(t, err)
				b, _ := RefDec(c.s)
				if name == "FromHex" {
					b, _ = RefHex(c.s)
				}
				require.Equal(t, k.FromBig(b), x)
				continue
			}
			x, err := f(c.s)
			require.Equal(t, want, fmt.Sprint(err))
			require.True(t, x.IsZero())
		}
	}
	// JSON as it came, its quotes tell a string from a bare value
	for _, c := range []struct{ in, why string }{
		{`""`, "empty"},
		{`"\u0031x"`, "invalid digit 'x' at offset 1"},
		{`"\u0030x\u0067"`, "invalid hex digit 'g' at offset 2"},
		{`"\x"`, `invalid digit '\\' at offset 0`},
		{`"`, `invalid digit '"' at offset 0`},
		{`true`, "invalid digit 't' at offset 0"},
		{`1.5`, "invalid digit '.' at offset 1"},
		{`"1.5"`, "invalid digit '.' at offset 1"},
		{`"0x"`, "no digits after 0x"},
		{`"0xg"`, "invalid hex digit 'g' at offset 2"},
		{`"0x+1"`, "invalid hex digit '+' at offset 2"},
		{`"ff"`, "invalid digit 'f' at offset 0"},
		{`0x1`, "0x outside quotes"},
		{`"` + over + `"`, "out of range"},
	} {
		require.Equal(On(t, c.in), typ+".UnmarshalJSON: cannot parse "+c.in+": "+c.why, fmt.Sprint(PT(new(T)).UnmarshalJSON([]byte(c.in))))
	}
	require.Equal(t, typ+".UnmarshalJSON: empty JSON", fmt.Sprint(PT(new(T)).UnmarshalJSON(nil)))
	if k.Signed {
		return
	}
	require.Equal(t, typ+".UnmarshalJSON: cannot parse -0: minus sign", fmt.Sprint(PT(new(T)).UnmarshalJSON([]byte("-0"))))
	// any other leading minus is a negative value, whatever follows it
	for _, s := range []string{"-5", "-0x1f", "-x", "-" + strings.Repeat("9", 100)} {
		want := ": negative value " + strconv.Quote(s)
		require.Equal(On(t, s), typ+".UnmarshalText"+want, fmt.Sprint(PT(new(T)).UnmarshalText([]byte(s))))
		require.Equal(On(t, s), typ+".UnmarshalJSON"+want, fmt.Sprint(PT(new(T)).UnmarshalJSON([]byte(s))))
		require.Equal(On(t, s), typ+".UnmarshalJSON"+want, fmt.Sprint(PT(new(T)).UnmarshalJSON([]byte(`"`+s+`"`))))
	}
}

// RunPanicText checks that every panic names the func or the method
func RunPanicText[T Num[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	typ := "fat." + k.Name
	// the parse panics carry the error itself
	must := "fat.Must" + k.Name
	require.Equal(t, must+`FromDec: cannot parse "x": invalid digit 'x' at offset 0`, fmt.Sprint(PanicValue(func() { k.MustFromDec("x") })))
	require.Equal(t, must+`FromDec: cannot parse "0x5": 0x in decimal text`, fmt.Sprint(PanicValue(func() { k.MustFromDec("0x5") })))
	require.Equal(t, must+`FromHex: cannot parse "x": invalid hex digit 'x' at offset 0`, fmt.Sprint(PanicValue(func() { k.MustFromHex("x") })))
	require.Equal(t, must+`FromHex: cannot parse "0x": no digits after 0x`, fmt.Sprint(PanicValue(func() { k.MustFromHex("0x") })))
	if !k.Signed {
		require.Equal(t, must+`FromDec: negative value "-5"`, fmt.Sprint(PanicValue(func() { k.MustFromDec("-5") })))
		require.Equal(t, must+`FromHex: negative value "-0x5"`, fmt.Sprint(PanicValue(func() { k.MustFromHex("-0x5") })))
	}
	big256 := new(big.Int).Lsh(Big1, 256)
	require.Equal(t, "fat.Must"+k.Name+"FromBig: out of range", PanicValue(func() { k.FromBig(big256) }))
	if !k.Signed {
		// a negative big.Int is told apart from a wide one, wide or not itself
		for _, b := range []*big.Int{big.NewInt(-1), new(big.Int).Neg(big256)} {
			require.Equal(On(t, b), "fat.Must"+k.Name+"FromBig: negative value", PanicValue(func() { k.FromBig(b) }))
		}
	}
	require.Equal(t, typ+".Bit: negative index", PanicValue(func() { k.Zero.Bit(-1) }))
	require.Equal(t, typ+".WithBit: negative index", PanicValue(func() { k.Zero.WithBit(-1, 0) }))
	require.Equal(t, typ+".WithBit: bit not 0 or 1", PanicValue(func() { k.Zero.WithBit(0, 2) }))
	// a negative shift panics as the << operator does
	for _, n := range []int{-1, math.MinInt} {
		require.Equal(On(t, n), typ+".Lsh: negative shift", PanicValue(func() { k.Zero.Lsh(n) }))
		require.Equal(On(t, n), typ+".Rsh: negative shift", PanicValue(func() { k.Zero.Rsh(n) }))
	}
}
