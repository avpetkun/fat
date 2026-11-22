package testutil

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/avpetkun/fat/internal/require"
)

// RunDecimal checks String, Dec, PrettyDec, AppendDec and MarshalText with
// big.Int, and that FromDec and UnmarshalText read them back
func RunDecimal[T Num[T], PT NumPtr[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	require.Equal(t, k.Ref(k.Min).String(), k.Min.String())

	k.Values(func(x T) {
		s := k.Ref(x).String()
		t := On(t, s)

		require.Equal(t, s, x.String())
		require.Equal(t, s, fmt.Sprint(x))
		require.Equal(t, s, x.Dec())
		require.Equal(t, PrettyDec(s, ','), x.PrettyDec(','))
		require.Equal(t, PrettyDec(s, '_'), x.PrettyDec('_'))
		for _, spare := range []int{0, len(s) - 1, len(s), 100} {
			require.Append(t, s, spare, func(dst []byte) []byte {
				b, _ := x.AppendText(dst)
				return b
			})
			require.Append(t, s, spare, x.AppendDec)
		}
		b, err := x.MarshalText()
		require.NoError(t, err)
		require.Equal(t, s, string(b))

		// parsing into a dirty value
		y := k.Dirty
		require.NoError(t, PT(&y).UnmarshalText([]byte(s)))
		require.Equal(t, x, y)
		y, err = k.FromDec(s)
		require.NoError(t, err)
		require.Equal(t, x, y)

		// a plus and leading zeros after the sign, even past the max length
		sign, digits := "+", s
		if s[0] == '-' {
			sign, digits = "-", s[1:]
		}
		ins := []string{sign + digits, sign + strings.Repeat("0", 100) + digits}
		if !k.Signed {
			ins = append(ins, "0"+s, strings.Repeat("0", 100)+s, "+00"+s)
		}
		for _, in := range ins {
			v, err := k.FromDec(in)
			require.NoError(On(t, in), err)
			require.Equal(On(t, in), x, v)
		}
		y = k.Zero
		require.NoError(t, PT(&y).UnmarshalText([]byte(sign+digits)))
		require.Equal(t, x, y)
	})

	// every bit length and every power of ten boundary
	for b := 1; b <= k.Bits; b++ {
		x := k.Max.Rsh(k.Bits - b)
		require.Equal(t, k.Ref(x).String(), x.String())
		require.Equal(t, PrettyDec(k.Ref(x).String(), ','), x.PrettyDec(','))
	}
	for p := big.NewInt(1); k.Fits(p); p = BigMul(p, big.NewInt(10)) {
		for _, d := range []int64{-1, 0, 1} {
			for _, v := range []*big.Int{BigAdd(p, big.NewInt(d)), new(big.Int).Neg(BigAdd(p, big.NewInt(d)))} {
				if k.Fits(v) {
					require.Equal(t, v.String(), k.FromBig(v).String())
					require.Equal(t, PrettyDec(v.String(), ' '), k.FromBig(v).PrettyDec(' '))
				}
			}
		}
	}
}

// RunSci checks m e n in FromDec and MustFromDec against big.Int:
// m * 10^n, a zero m for any n, the range and the errors, and that
// UnmarshalText keeps to plain decimal
func RunSci[T Num[T], PT NumPtr[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	max := k.Ref(k.Max).String()
	ms := []string{"0", "1", "7", "+12", "000123", "-0", "-3", "-00045", max, max[:len(max)-1], "-" + max}
	ns := []string{"0", "1", "2", "18", "19", "38", "39", "76", "77", "78", "999", "1000", "99999999999999999999999"}
	ten := big.NewInt(10)
	for _, m := range ms {
		mb, _ := new(big.Int).SetString(m, 10)
		for _, n := range ns {
			for _, e := range []string{"e", "E", "e+", "E+00"} {
				s := m + e + n
				t := On(t, s)
				nb, _ := new(big.Int).SetString(n, 10)
				var want *big.Int
				if mb.Sign() == 0 {
					want = new(big.Int)
				} else if nb.Cmp(big.NewInt(1000)) <= 0 {
					want = new(big.Int).Mul(mb, new(big.Int).Exp(ten, nb, nil))
				}
				why := ""
				switch {
				case !k.Signed && mb.Sign() < 0:
					why = "negative"
				case !k.Signed && m[0] == '-':
					why = "minus sign"
				case want == nil || !k.Fits(want):
					why = "out of range"
				}
				x, err := k.FromDec(s)
				if why == "" {
					require.NoError(t, err)
					require.Equal(t, k.FromBig(want), x)
					require.Equal(t, x, k.MustFromDec(s))
					continue
				}
				q := strconv.Quote(s)
				if why == "negative" {
					require.Equal(t, "fat."+k.Name+"FromDec: negative value "+q, fmt.Sprint(err))
				} else {
					require.Equal(t, "fat."+k.Name+"FromDec: cannot parse "+q+": "+why, fmt.Sprint(err))
				}
				require.True(t, x.IsZero())
				must := strings.Replace(fmt.Sprint(err), "fat."+k.Name, "fat.Must"+k.Name, 1)
				require.Equal(t, must, fmt.Sprint(PanicValue(func() { k.MustFromDec(s) })))
			}
		}
	}
	// the errors of a bad form
	for _, c := range []struct{ s, why string }{
		{"1e", "no digits after the exponent"},
		{"1E+", "no digits after the exponent"},
		{"1e-5", "invalid digit '-' at offset 2"},
		{"1e5e5", "invalid digit 'e' at offset 3"},
		{"1e 5", "invalid digit ' ' at offset 2"},
		{"1.5e3", "invalid digit '.' at offset 1"},
		{"e5", "invalid digit 'e' at offset 0"},
		{"+e5", "invalid digit 'e' at offset 1"},
		{"1ee5", "invalid digit 'e' at offset 2"},
	} {
		t := On(t, c.s)
		q := strconv.Quote(c.s)
		_, err := k.FromDec(c.s)
		require.Equal(t, "fat."+k.Name+"FromDec: cannot parse "+q+": "+c.why, fmt.Sprint(err))
	}
	_, err := k.FromDec("0x1e5")
	require.Equal(t, "fat."+k.Name+"FromDec: cannot parse \"0x1e5\": 0x in decimal text", fmt.Sprint(err))
}

// RunHex checks Hex and AppendHex with big.Int at every digit count.
// FromHex, FromDec and UnmarshalText are checked with big.Int on any prefix
func RunHex[T Num[T], PT NumPtr[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	pad := strings.Repeat("0", k.Bits/4+8)
	k.Values(func(x T) {
		xb := k.Ref(x)
		s := xb.Text(16)
		t := On(t, s)

		// 0x or 0X, either case, a sign, zero padding even past the max length
		sign, digits := "", s
		if s[0] == '-' {
			sign, digits = "-", s[1:]
		}
		plus := sign + "+"[len(sign):]
		up := strings.ToUpper(digits)
		ins := []string{sign + "0x" + digits, plus + "0X" + up, sign + "0x" + pad + digits, plus + "0x" + pad + up}
		if !k.Signed {
			ins = append(ins, fmt.Sprintf("0x%0*x", k.Bits/4, xb))
		}
		for _, in := range ins {
			v, err := k.FromHex(in)
			require.NoError(On(t, in), err)
			require.Equal(On(t, in), x, v)
			// UnmarshalText reads them as FromHex
			v = k.Dirty
			require.NoError(On(t, in), PT(&v).UnmarshalText([]byte(in)))
			require.Equal(On(t, in), x, v)
		}

		// minimal lowercase digits, as big.Int.Text
		h := sign + "0x" + digits
		require.Equal(t, h, x.Hex())
		for _, spare := range []int{0, 7, len(h) - 1, len(h), 100} {
			require.Append(t, h, spare, x.AppendHex)
		}
	})

	// every digit count
	for i := range k.Bits {
		for _, x := range []T{k.Small(1).Lsh(i), k.Max.Rsh(i), k.Min.Rsh(i)} {
			require.Equal(On(t, i), HexPrefixed(k.Ref(x)), x.Hex())
		}
	}
	require.Equal(t, "0x0", k.Zero.Hex())
	require.Equal(t, "0", k.Zero.PrettyDec(','))

	hexParse(t, k)
	bigBases[T, PT](t, k)
	edge := k.Max
	if k.Signed {
		edge = k.Min
	}
	require.Equal(t, edge, k.MustFromHex(HexPrefixed(k.Ref(edge))))
	require.Equal(t, edge, k.MustFromDec(k.Ref(edge).String()))
}

// RunParseErrors checks the error and the zero result of FromDec on malformed
// and out of range text
func RunParseErrors[T Num[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	// digits is the decimal length of max, over is max + 1
	digits, over, signed := k.Digits(), k.Over(), k.Signed
	s := over.String()
	cases := []struct {
		s   string
		err error
	}{
		{"", ErrBad},
		{" 1", ErrBad},
		{"1 ", ErrBad},
		{"1_0", ErrBad},
		{"12a", ErrBad},
		{"0x", ErrBad},
		{"0b1", ErrBad},
		{"0x1g", ErrBad},
		{"١٢٣", ErrBad}, // arabic-indic digits
		{"1" + strings.Repeat("0", 25) + "x", ErrBad},
		{strings.Repeat("9", digits-1) + "x", ErrBad},
		{s, ErrBad},
		// same length as max, overflows in the multiply
		{string(s[0]+1) + strings.Repeat("0", len(s)-1), ErrBad},
		{strings.Repeat("9", digits), ErrBad},
		{strings.Repeat("9", digits+1), ErrBad},
		{strings.Repeat("9", 100), ErrBad},
	}
	if signed {
		cases = append(cases, []struct {
			s   string
			err error
		}{
			{"-", ErrBad},
			{"+", ErrBad},
			{"--1", ErrBad},
			{"+-1", ErrBad},
			{"-+1", ErrBad},
			{"1-", ErrBad},
			{"−1", ErrBad}, // unicode minus
			{"-" + new(big.Int).Add(over, Big1).String(), ErrBad},                // min - 1
			{new(big.Int).Sub(new(big.Int).Lsh(over, 1), Big1).String(), ErrBad}, // unsigned max
			{"-1" + s[1:] + "0", ErrBad},
		}...)
	} else {
		cases = append(cases, []struct {
			s   string
			err error
		}{
			// a plus as in big.Int, a minus fails with ErrNegative
			{"+", ErrBad},
			{"+0", nil},
			{"++1", ErrBad},
			{"+-1", ErrBad},
			{"-+1", ErrBad},
			{"+x", ErrBad},
			{"-1", ErrBad},
			{"-0", ErrBad},
		}...)
	}
	// a bad byte at every position of every length that fits
	for n := 1; n <= digits; n++ {
		for p := range n {
			b := []byte(strings.Repeat("1", n))
			b[p] = 'x'
			cases = append(cases, struct {
				s   string
				err error
			}{string(b), ErrBad})
		}
	}
	for _, c := range cases {
		s := c.s
		v, err := k.FromDec(s)
		if err != nil {
			// the quoted text with a reason, or ErrNegative's for a minus
			if !k.Signed && NegativeText(s) {
				require.Equal(On(t, s), "fat."+k.Name+"FromDec: negative value "+strconv.Quote(s), err.Error())
			} else {
				want := "fat." + k.Name + "FromDec: cannot parse " + strconv.Quote(s) + ": "
				require.True(On(t, s, err), strings.HasPrefix(err.Error(), want) && len(err.Error()) > len(want))
			}
		}
		require.Equal(On(t, s), c.err != nil, err != nil)
		require.True(On(t, s), v.IsZero())
	}
}

// hexPrefixed reports whether 0x or 0X follows an optional sign in s
func hexPrefixed(s string) bool {
	if s != "" && (s[0] == '+' || s[0] == '-') {
		s = s[1:]
	}
	return len(s) >= 2 && s[0] == '0' && s[1]|0x20 == 'x'
}

// RefDec is the big.Int reference of FromDec: RefParse with no 0x prefix
func RefDec(s string) (*big.Int, bool) {
	if hexPrefixed(s) {
		return nil, false
	}
	return RefParse(s)
}

// RefHex is the big.Int reference of FromHex, 0x or 0X optional after a sign
func RefHex(s string) (*big.Int, bool) {
	sign, d := "", s
	if d != "" && (d[0] == '+' || d[0] == '-') {
		sign, d = d[:1], d[1:]
	}
	if len(d) > 2 && d[0] == '0' && d[1]|0x20 == 'x' {
		d = d[2:]
	}
	// SetString takes a sign of its own
	if d == "" || d[0] == '+' || d[0] == '-' {
		return nil, false
	}
	return new(big.Int).SetString(sign+d, 16)
}

// NegativeText reports whether an unsigned parser fails on s with ErrNegative:
// a minus with a nonzero value after it
func NegativeText(s string) bool {
	if len(s) < 2 || s[0] != '-' {
		return false
	}
	s = s[1:]
	if len(s) > 2 && s[0] == '0' && s[1]|0x20 == 'x' {
		s = s[2:]
	}
	return strings.Trim(s, "0") != ""
}

// RefParse is the big.Int reference of UnmarshalText: hex after 0x, or decimal
func RefParse(s string) (*big.Int, bool) {
	sign, d := "", s
	if d != "" && (d[0] == '+' || d[0] == '-') {
		sign, d = d[:1], d[1:]
	}
	base := 10
	if len(d) > 2 && d[0] == '0' && d[1]|0x20 == 'x' {
		base, d = 16, d[2:]
	}
	// SetString takes a sign of its own
	if d == "" || d[0] == '+' || d[0] == '-' {
		return nil, false
	}
	return new(big.Int).SetString(sign+d, base)
}

// hexParse checks FromHex with RefHex on every byte at every position,
// on prefixes, signs and padding and at the edges of the range
func hexParse[T Num[T]](t *testing.T, k Kind[T]) {
	digits, signed, fits := k.Bits/4, k.Signed, k.Fits
	check := func(s string, wantErr error) {
		t := On(t, s)
		v, err := k.FromHex(s)
		require.Equal(t, wantErr != nil, err != nil)
		if err != nil {
			require.True(t, v.IsZero())
			return
		}
		want, _ := RefHex(s)
		require.Big(t, want, k.Ref(v))
	}

	// every byte at every position of every length, with and without 0x
	for n := 1; n <= digits; n++ {
		for p := range n {
			for c := range 256 {
				b := []byte(strings.Repeat("1", n))
				b[p] = byte(c)
				for _, s := range []string{"0x" + string(b), string(b)} {
					want, ok := RefHex(s)
					if ok && fits(want) {
						check(s, nil)
					} else {
						check(s, ErrBad)
					}
				}
			}
		}
	}

	z, f, g := strings.Repeat("0", digits), strings.Repeat("f", digits), strings.Repeat("g", digits)
	cases := []struct {
		s   string
		err error
	}{
		{"", ErrBad},
		// no prefix is hex too
		{"1", nil},
		{"0", nil},
		{"+1", nil},
		{"fF", nil},
		{z, nil},
		{"g", ErrBad},
		{"1g", ErrBad},
		{"1" + z, ErrBad},
		{strings.Repeat("0", 100) + "1", nil},
		{"0x", ErrBad},
		{"0X", ErrBad},
		{"0X1", nil},
		{"0xF", nil},
		{"0x" + z, nil},
		{"x1", ErrBad},
		{"00x1", ErrBad},
		{"0x0x1", ErrBad},
		{" 0x1", ErrBad},
		{"0x 1", ErrBad},
		{"0x1_0", ErrBad},
		{"0x_1", ErrBad},
		// b is a hex digit
		{"0b1", nil},
		{"0o7", ErrBad},
		{"0xg", ErrBad},
		{"0x" + f[1:] + "g", ErrBad},
		{"0x1" + z, ErrBad},
		{"0x" + f + "f", ErrBad},
		{"0x0" + f + "f", ErrBad},
		// past the max length the range error wins, as with decimals
		{"0x" + g + "g", ErrBad},
		// zeros past the max length
		{"0x" + strings.Repeat("0", 100) + "1", nil},
		{"0x" + strings.Repeat("0", 100), nil},
		// no sign after the prefix
		{"0x+1", ErrBad},
		{"0x-1", ErrBad},
		{"+0x", ErrBad},
		{"+0x+1", ErrBad},
		{"++0x1", ErrBad},
		{"+0x1", nil},
	}
	if signed {
		cases = append(cases, []struct {
			s   string
			err error
		}{
			{"-0x", ErrBad},
			{"--0x1", ErrBad},
			{"+-0x1", ErrBad},
			{"-+0x1", ErrBad},
			{"-0x+1", ErrBad},
			{"-0xx1", ErrBad},
			{"0x8" + z[1:], ErrBad},
			{"-0x8" + z[2:] + "1", ErrBad},
			{"-0x8" + z[1:], nil},
			{"-0X8" + z[1:], nil},
			{"+0x7" + f[1:], nil},
			{"-0x0", nil},
			{"-0x1", nil},
			{"-1", nil},
			{"-8" + z[1:], nil},
			{"8" + z[1:], ErrBad},
			{"--1", ErrBad},
		}...)
	} else {
		cases = append(cases, []struct {
			s   string
			err error
		}{
			// a minus fails with ErrNegative
			{"-0x1", ErrBad},
			{"-0x0", ErrBad},
			{"-1", ErrBad},
			{"-0", ErrBad},
			{f, nil},
			{"+0x" + f, nil},
			{"0x" + f, nil},
		}...)
	}
	for _, c := range cases {
		check(c.s, c.err)
	}
}

// bigBases checks FromDec, FromHex and UnmarshalText with their big.Int
// references on the values of k with every base prefix, sign and underscore
func bigBases[T Num[T], PT NumPtr[T]](t *testing.T, k Kind[T]) {
	width := k.Bits
	var vals []*big.Int
	k.Values(func(x T) { vals = append(vals, k.Ref(x)) })
	same := func(t testing.TB, s string, want *big.Int, ok bool, v T, err error) {
		// unsigned types take a plus only
		if !ok || !k.Signed && s != "" && s[0] == '-' || !k.Fits(want) {
			require.Error(t, err)
			require.True(t, v.IsZero())
			return
		}
		require.NoError(t, err)
		require.Big(t, want, k.Ref(v))
	}
	check := func(s string) {
		v, err := k.FromDec(s)
		want, ok := RefDec(s)
		same(On(t, "FromDec", s), s, want, ok, v, err)
		v, err = k.FromHex(s)
		want, ok = RefHex(s)
		same(On(t, "FromHex", s), s, want, ok, v, err)
		var u T
		err = PT(&u).UnmarshalText([]byte(s))
		want, ok = RefParse(s)
		same(On(t, "UnmarshalText", s), s, want, ok, u, err)
	}
	prefixes := map[int][]string{2: {"0b", "0B"}, 8: {"0o", "0O", "0"}, 10: {""}, 16: {"0x", "0X"}}
	// the edge values come first, then every 16th random one
	for i, x := range vals {
		if i >= 100 && i%16 != 0 {
			continue
		}
		abs := new(big.Int).Abs(x)
		for base, pres := range prefixes {
			d := abs.Text(base)
			ds := []string{d}
			if base == 16 {
				ds = append(ds, strings.ToUpper(d))
			}
			for _, sign := range []string{"", "+", "-"} {
				for _, d := range ds {
					// decimal and 0x only, a leading 0 is a decimal zero
					for _, s := range []string{sign + d, sign + "00" + d} {
						check(s)
					}
					for _, pre := range pres {
						for _, s := range []string{sign + pre + d, sign + pre + "00" + d} {
							check(s)
						}
						// underscores in valid and invalid places
						for _, s := range []string{sign + pre + "_" + d, sign + pre + d + "_"} {
							check(s)
						}
						if len(d) > 1 {
							for _, s := range []string{sign + pre + d[:1] + "_" + d[1:], sign + pre + d[:1] + "__" + d[1:], sign + pre + d[:len(d)-1] + "_" + d[len(d)-1:]} {
								check(s)
							}
						}
					}
				}
			}
		}
	}
	for _, s := range []string{
		"", "0", "00", "08", "017", "0_7", "0__7", "0_x1", "00x1", "_", "__", "_1", "1_", "1__0",
		"0b", "0o", "0x", "0B", "0x_", "0b_", "0b2", "0o8", "0xg", "0b1_", "0x_1_f",
		"0x-1", "0b+1", "-0b1", "+0o7", "--0x1", "+-0b1", "-_1", "+0x_f", "-0",
		"+", "-", "++1", "-+1", "++0x1", "-+0b1", "+_1", "+0", "+08",
		"0b1" + strings.Repeat("0", width), "0b" + strings.Repeat("1", width), "0b" + strings.Repeat("1", width+1),
		"0o" + strings.Repeat("7", width/3+2), "0o1" + strings.Repeat("0", width/3), "0o" + strings.Repeat("0", 200) + "7", "0b" + strings.Repeat("0", 300) + "1",
		"0x" + strings.Repeat("f", width/4+1), "1_" + strings.Repeat("0", 100),
	} {
		check(s)
	}
}

// RunFmt checks fmt verbs and flags with big.Int on the values of k.
// An unknown verb names the type
func RunFmt[T Num[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	name := "fat." + k.Name
	var vals []*big.Int
	k.Values(func(x T) { vals = append(vals, k.Ref(x)) })
	format := func(layout string, v *big.Int) string { return fmt.Sprintf(layout, k.FromBig(v)) }
	layouts := []string{
		"%b", "%o", "%O", "%d", "%s", "%v", "%x", "%X",
		"%#b", "%#o", "%#O", "%#x", "%#X", "%#d", "%+d", "% d", "%+ d", "% x", "%+X",
		"%08d", "%-8d|", "%40x", "%-+#40X|", "%0+50b", "%0-30o|", "%.30d", "%.0d", "%.0x", "%#.0x", "%8.3d", "%08.3d", "%+.0d", "%5.0d|",
	}
	var sub []*big.Int
	for i, v := range vals {
		if i < 100 || i%64 == 0 {
			sub = append(sub, v)
		}
	}
	for _, v := range sub {
		for _, l := range layouts {
			require.Equal(On(t, l, v), fmt.Sprintf(l, v), format(l, v))
		}
		// an unknown verb names the type
		for _, l := range []string{"%q", "%c", "%e"} {
			require.Equal(On(t, l, v), strings.Replace(fmt.Sprintf(l, v), "big.Int", name, 1), format(l, v))
		}
	}

	// a pointer formats too
	x := k.FromBig(k.Wrap(big.NewInt(-42)))
	require.Equal(t, fmt.Sprint(k.Ref(x)), fmt.Sprint(&x))
}

// RunJSON checks the quoted decimal JSON of values and map keys with big.Int.
// Bare numbers are read too, quoted text as UnmarshalText reads it
func RunJSON[T Num[T], PT NumPtr[T]](t *testing.T, k Kind[T]) {
	width, signed, ref, from := k.Bits, k.Signed, k.Ref, k.FromBig
	var vals []T
	i := 0
	k.Values(func(x T) {
		if i++; i < 100 || i%256 == 0 {
			vals = append(vals, x)
		}
	})
	type doc struct{ A T }
	type bdoc struct{ A *big.Int }
	five := from(big.NewInt(5))
	for _, v := range vals {
		t := On(t, ref(v))
		js, err := json.Marshal(doc{v})
		require.NoError(t, err)
		require.Equal(t, `{"A":"`+ref(v).String()+`"}`, string(js))
		var out doc
		require.NoError(t, json.Unmarshal(js, &out))
		require.Equal(t, v, out.A)
		bj, _ := json.Marshal(bdoc{ref(v)})
		out = doc{}
		require.NoError(t, json.Unmarshal(bj, &out))
		require.Equal(t, v, out.A)
		js, err = json.Marshal(map[T]int{v: 1})
		require.NoError(t, err)
		require.Equal(t, `{"`+ref(v).String()+`":1}`, string(js))
		var m map[T]int
		require.NoError(t, json.Unmarshal(js, &m))
		require.Equal(t, map[T]int{v: 1}, m)
		// quoted hex as Hex writes it, a minus before 0x
		out = doc{five}
		require.NoError(t, json.Unmarshal([]byte(`{"A":"`+HexPrefixed(ref(v))+`"}`), &out))
		require.Equal(t, v, out.A)
		// the magnitude with no sign, |MinInt| overflows
		a := new(big.Int).Abs(ref(v))
		h := a.Text(16)
		out = doc{five}
		err = json.Unmarshal([]byte(`{"A":"0x`+h+`"}`), &out)
		if signed && a.BitLen() >= width {
			require.Error(t, err)
			require.Equal(t, five, out.A)
		} else {
			require.NoError(t, err)
			require.Equal(t, from(a), out.A)
		}
		// a bad digit first or last
		for _, bh := range []string{"g" + h[1:], h[:len(h)-1] + "g"} {
			out = doc{five}
			require.Error(On(t, bh), json.Unmarshal([]byte(`{"A":"0x`+bh+`"}`), &out))
			require.Equal(On(t, bh), five, out.A)
		}
	}
	// decimal, or hex in quotes after 0x, with leading zeros and a sign
	good := map[string]int64{`7`: 7, `"7"`: 7, `"+7"`: 7, `0`: 0, `"00"`: 0, `"017"`: 17, `"0x10"`: 16, `"0X1f"`: 31,
		`"0x0"`: 0, `"0x00"`: 0, `"0x0010"`: 16, `"+0x10"`: 16, `"0x` + strings.Repeat("0", 100) + `7"`: 7,
		`"` + strings.Repeat("0", 100) + `7"`: 7,
		// a string with escapes reads as its text
		`"\u0031"`: 1, `"\u002b7"`: 7, `"0x\u0031f"`: 31, `"\u0030X10"`: 16}
	bad := []string{`1.5`, `1e3`, `"x"`, `""`, `true`, `"`, `[1]`, `"1_000"`, `"0b11"`, `"0o17"`, `"ff"`,
		`"0x"`, `"0x+1"`, `"0x-3"`, `"0x_1"`, `"0xg"`, `"0x1` + strings.Repeat("0", 64) + `"`,
		`"\u0067"`, `"0x\u0067"`, `"\u0031\u002e5"`, `"\""`}
	if signed {
		good[`-3`], good[`"-3"`], good[`"-03"`], good[`"-0x3"`], good[`"-0X03"`], good[`"-0x0"`], good[`-0`] = -3, -3, -3, -3, -3, 0, 0
		good[`"\u002d3"`] = -3
	} else {
		bad = append(bad, `-1`, `"-1"`, `-0`, `"-0"`, `"-0x3"`, `"-0x0"`, `"\u002d1"`)
	}
	// null leaves the value as is, as encoding/json does with an int
	out := doc{five}
	require.NoError(t, json.Unmarshal([]byte(`{"A":null}`), &out))
	require.Equal(t, five, out.A)
	require.NoError(t, PT(&out.A).UnmarshalJSON([]byte(`null`)))
	require.Equal(t, five, out.A)
	for in, want := range good {
		t := On(t, in)
		out := doc{from(big.NewInt(5))}
		require.NoError(t, json.Unmarshal([]byte(`{"A":`+in+`}`), &out))
		require.Equal(t, from(big.NewInt(want)), out.A)
	}
	// an error leaves the value as is, in UnmarshalText too
	for _, in := range bad {
		t := On(t, in)
		out := doc{five}
		require.Error(t, json.Unmarshal([]byte(`{"A":`+in+`}`), &out))
		require.Equal(t, five, out.A)
	}
	for _, in := range []string{"", "x", "1.5", "0x", "1_", "-", "-1", strings.Repeat("9", 100)} {
		x := five
		if !(signed && in == "-1") {
			require.Error(On(t, in), PT(&x).UnmarshalText([]byte(in)))
			require.Equal(On(t, in), five, x)
		}
	}
	var x T
	require.Error(t, PT(&x).UnmarshalJSON([]byte(`"`)))
}

// PrettyDec groups the decimal s by 3 digits with sep, after the sign
func PrettyDec(s string, sep byte) string {
	sign := ""
	if s[0] == '-' {
		sign, s = "-", s[1:]
	}
	var b []byte
	for i := range len(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			b = append(b, sep)
		}
		b = append(b, s[i])
	}
	return sign + string(b)
}

// HexPrefixed returns v in hex with 0x after the sign, the output of Hex
func HexPrefixed(v *big.Int) string {
	if v.Sign() < 0 {
		return "-0x" + new(big.Int).Neg(v).Text(16)
	}
	return "0x" + v.Text(16)
}
