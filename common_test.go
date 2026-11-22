package fat

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"github.com/avpetkun/fat/internal/require"
	. "github.com/avpetkun/fat/internal/testutil"
)

// TestAppendDecimalUint64 checks appendDecimal64 against strconv
func TestAppendDecimalUint64(t *testing.T) {
	t.Parallel()
	cases := []struct {
		V uint64
		S string
	}{
		{0, "0"},
		{99, "99"},
		{1234567, "1234567"},
		{12345678901234567, "12345678901234567"},
		{math.MaxUint64, "18446744073709551615"},
	}
	for _, c := range cases {
		require.Equal(t, c.S, string(appendDecimal64(nil, c.V)))
	}

	// both sides of every digit count
	for _, v := range edgeUint64s() {
		require.Equal(t, strconv.FormatUint(v, 10), string(appendDecimal64(nil, v)))
	}

	// every value up to 6 digits
	buf := make([]byte, 0, 20)
	for v := range uint64(1000000) {
		buf = appendDecimal64(buf[:0], v)
		require.Same(t, strconv.FormatUint(v, 10), string(buf))
	}

	// random values of every bit length
	r := rand.New(rand.NewSource(1))
	for range 200000 {
		v := r.Uint64() >> r.Intn(64)
		buf = appendDecimal64(buf[:0], v)
		require.Same(t, strconv.FormatUint(v, 10), string(buf))
	}
}

// TestAppendDecimalUint64Dst checks the reuse of dst and the allocations
func TestAppendDecimalUint64Dst(t *testing.T) {
	// keeps existing content
	require.Equal(t, "id=42", string(appendDecimal64([]byte("id="), 42)))

	// not enough, exact and extra room
	for _, v := range edgeUint64s() {
		want := strconv.FormatUint(v, 10)
		for _, spare := range []int{0, 1, len(want) - 1, len(want), len(want) + 1, 64} {
			require.Append(t, want, spare, func(dst []byte) []byte {
				return appendDecimal64(dst, v)
			})
		}
	}

	// no allocs when dst has room
	buf := make([]byte, 0, 20)
	require.Equal(t, 0.0, testing.AllocsPerRun(100, func() {
		buf = appendDecimal64(buf[:0], math.MaxUint64)
	}))

	// one when dst is full
	full := make([]byte, 100)
	require.Equal(t, 1.0, testing.AllocsPerRun(100, func() {
		_ = appendDecimal64(full, math.MaxUint64)
	}))
}

// TestParseDecimalUint64 checks parseDecimalUint64 on valid input
func TestParseDecimalUint64(t *testing.T) {
	t.Parallel()
	cases := []struct {
		S string
		V uint64
	}{
		{"0", 0},
		{"42", 42},
		{"18446744073709551615", math.MaxUint64},
		// leading zeros
		{"007", 7},
		{"000000000000000000000000042", 42},
		{"0000018446744073709551615", math.MaxUint64},
	}
	for _, c := range cases {
		v, ok := parseDecimalUint64([]byte(c.S))
		require.True(t, ok)
		require.Equal(t, c.V, v)
	}

	// both sides of every digit count
	for _, want := range edgeUint64s() {
		s := strconv.FormatUint(want, 10)

		// reads only s, even with junk around it
		buf := []byte("////////" + s + "::::::::")
		v, ok := parseDecimalUint64(buf[8 : 8+len(s)])
		require.True(t, ok)
		require.Equal(t, want, v)

		// input is left as is
		require.Equal(t, "////////"+s+"::::::::", string(buf))
	}
}

// TestParseDecimalUint64Errors checks that bad input fails with zero
func TestParseDecimalUint64Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		S   string
		Err error
	}{
		{"", ErrBad},

		{"-1", ErrBad},
		{"+1", ErrBad},
		{"12a", ErrBad},
		{"١٢٣", ErrBad}, // arabic-indic digits

		{"18446744073709551616", ErrBad},   // max + 1
		{"30000000000000000000", ErrBad},   // wraps past itself
		{"184467440737095516150", ErrBad},  // extra digit
		{"0018446744073709551616", ErrBad}, // zeros don't hide it

		// bad symbol beats overflow
		{"1844674407370955161x", ErrBad},
		{"18446744073709551616x", ErrBad},
		{"x18446744073709551616", ErrBad},
		{"99999999999999999999999999x", ErrBad},
	}
	for _, c := range cases {
		v, ok := parseDecimalUint64([]byte(c.S))
		require.False(On(t, c.S), ok)
		require.Equal(On(t, c.S), uint64(0), v)
	}
}

// TestParseDecimalUint64BadByte checks a non-digit byte at every position
func TestParseDecimalUint64BadByte(t *testing.T) {
	t.Parallel()
	// zero padded max, valid at any suffix length
	const digits = "0000018446744073709551615"
	for n := 1; n <= len(digits); n++ {
		s := []byte(digits[len(digits)-n:])
		_, ok := parseDecimalUint64(s)
		require.True(t, ok)

		// every non-digit byte at every position
		for i := range s {
			keep := s[i]
			for c := range 256 {
				if '0' <= c && c <= '9' {
					continue
				}
				s[i] = byte(c)
				v, ok := parseDecimalUint64(s)
				t := On(t, string(s))
				require.False(t, ok)
				require.Same(t, uint64(0), v)
			}
			s[i] = keep
		}
	}
}

// TestParseDecimalUint64Strconv checks random input against strconv
func TestParseDecimalUint64Strconv(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(1))
	for range 300000 {
		s := randNumber(r)
		got, ok := parseDecimalUint64(s)
		want, wantErr := refParseUint64(string(s))
		t := On(t, string(s))
		require.Same(t, want, got)
		require.Same(t, wantErr == nil, ok)
	}
}

// refParseUint64 is strconv.ParseUint of digits only, ErrBad on failure
func refParseUint64(s string) (uint64, error) {
	if s == "" {
		return 0, ErrBad
	}
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return 0, ErrBad
		}
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, ErrBad
	}
	return v, nil
}

// edgeUint64s returns values around every power of ten and of two
func edgeUint64s() []uint64 {
	vs := []uint64{0, math.MaxUint64}
	for p := uint64(1); ; p *= 10 {
		vs = append(vs, p-1, p, p+1)
		if p > math.MaxUint64/10 {
			break
		}
	}
	for shift := range 64 {
		p := uint64(1) << shift
		vs = append(vs, p-1, p, p+1)
	}
	return vs
}

// randNumber returns number-like input, mostly digits, some signs and junk
func randNumber(r *rand.Rand) []byte {
	const junk = "+- ._/:xa\x00\xb0\xff"
	n := r.Intn(26)
	s := make([]byte, 0, n+20)
	switch r.Intn(8) {
	case 0:
		s = append(s, '-')
	case 1:
		s = append(s, '+')
	}
	// start near a limit
	switch r.Intn(6) {
	case 0:
		s = append(s, "1844674407370955161"...)
	case 1:
		s = append(s, "922337203685477580"...)
	case 2:
		s = append(s, "0000"...)
	}
	for len(s) < n {
		if r.Intn(50) == 0 {
			s = append(s, junk[r.Intn(len(junk))])
		} else {
			s = append(s, byte('0'+r.Intn(10)))
		}
	}
	return s
}

// compare with strconv:
// go test -run '^$' -bench Uint64 -count 6 | benchstat -col /fn -

// BenchmarkAppendDecimalUint64 runs appendDecimal64 and strconv.AppendUint
func BenchmarkAppendDecimalUint64(b *testing.B) {
	set := benchUint64Set()
	b.Run("fn=fat", func(b *testing.B) {
		buf := make([]byte, 0, 32)
		b.ReportAllocs()
		for i := range b.N {
			buf = appendDecimal64(buf[:0], set.nums[i&BenchMask])
		}
	})
	b.Run("fn=strconv", func(b *testing.B) {
		buf := make([]byte, 0, 32)
		b.ReportAllocs()
		for i := range b.N {
			buf = strconv.AppendUint(buf[:0], set.nums[i&BenchMask], 10)
		}
	})
}

// BenchmarkParseDecimalUint64 runs parseDecimalUint64 and strconv.ParseUint
func BenchmarkParseDecimalUint64(b *testing.B) {
	set := benchUint64Set()
	b.Run("fn=fat", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := range b.N {
			SinkUint64, SinkBool = parseDecimalUint64(set.bytes[i&BenchMask])
		}
	})
	b.Run("fn=strconv", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := range b.N {
			SinkUint64, _ = strconv.ParseUint(set.strs[i&BenchMask], 10, 64)
		}
	})
}

// benchSet holds benchmark inputs as values, strings and bytes
type benchSet[T any] struct {
	nums  []T
	strs  []string
	bytes [][]byte
}

// benchUint64Set has random values of mixed digit counts
func benchUint64Set() benchSet[uint64] {
	r := rand.New(rand.NewSource(1))
	var set benchSet[uint64]
	for range BenchMask + 1 {
		v := randUint64(r, 1+r.Intn(20), math.MaxUint64)
		s := strconv.FormatUint(v, 10)
		set.nums = append(set.nums, v)
		set.strs = append(set.strs, s)
		set.bytes = append(set.bytes, []byte(s))
	}
	return set
}

// randUint64 returns a random value of n decimal digits, at most max
func randUint64(r *rand.Rand, n int, max uint64) uint64 {
	lo := uint64(1)
	for range n - 1 {
		lo *= 10
	}
	hi := max
	if lo <= max/10 {
		hi = lo*10 - 1
	}
	if n == 1 {
		lo = 0
	}
	return lo + r.Uint64()%(hi-lo+1)
}

// checkTextBytes checks that a parser and its Must form read bytes as strings
func checkTextBytes[T comparable](t *testing.T, name string, fs func(string) (T, error), fb func([]byte) (T, error), ms func(string) T, mb func([]byte) T) {
	for _, s := range []string{"", "0", "+7", "-5", "-0", "ff", "0xFf", "-0x1f", "x", "1_0", strings.Repeat("9", 38), strings.Repeat("9", 70), strings.Repeat("f", 64), strings.Repeat("9", 100)} {
		t := On(t, name, s)
		x, err := fs(s)
		y, errb := fb([]byte(s))
		require.Equal(t, x, y)
		require.Equal(t, fmt.Sprint(err), fmt.Sprint(errb))
		if err == nil {
			require.Equal(t, x, ms(s))
			require.Equal(t, x, mb([]byte(s)))
			continue
		}
		require.Equal(t, fmt.Sprint(PanicValue(func() { ms(s) })), fmt.Sprint(PanicValue(func() { mb([]byte(s)) })))
	}
}

// TestTextBytes checks the parsers on bytes and that they never allocate
func TestTextBytes(t *testing.T) {
	checkTextBytes(t, "Uint128FromDec", Uint128FromDec, Uint128FromDec, MustUint128FromDec, MustUint128FromDec)
	checkTextBytes(t, "Uint128FromHex", Uint128FromHex, Uint128FromHex, MustUint128FromHex, MustUint128FromHex)
	checkTextBytes(t, "Int128FromDec", Int128FromDec, Int128FromDec, MustInt128FromDec, MustInt128FromDec)
	checkTextBytes(t, "Int128FromHex", Int128FromHex, Int128FromHex, MustInt128FromHex, MustInt128FromHex)
	checkTextBytes(t, "Uint256FromDec", Uint256FromDec, Uint256FromDec, MustUint256FromDec, MustUint256FromDec)
	checkTextBytes(t, "Uint256FromHex", Uint256FromHex, Uint256FromHex, MustUint256FromHex, MustUint256FromHex)
	checkTextBytes(t, "Int256FromDec", Int256FromDec, Int256FromDec, MustInt256FromDec, MustInt256FromDec)
	checkTextBytes(t, "Int256FromHex", Int256FromHex, Int256FromHex, MustInt256FromHex, MustInt256FromHex)

	// neither the bytes nor the string are copied, whatever the length
	b, s := []byte(strings.Repeat("7", 70)), strings.Repeat("7", 70)
	var u1 Uint128
	var i1 Int128
	var u2 Uint256
	var i2 Int256
	var err error
	require.Equal(t, 0.0, testing.AllocsPerRun(100, func() {
		u1, err = Uint128FromDec(b[:38])
		i1, err = Int128FromHex(b[:30])
		u2, err = Uint256FromDec(b)
		i2, err = Int256FromDec(b)
		u2, err = Uint256FromHex(s[:64])
		i2 = MustInt256FromDec(b)
	}))
	require.NoError(t, err)
	require.False(t, u1.IsZero() || i1.IsZero() || u2.IsZero() || i2.IsZero())
}

// TestJSONUnescape checks jsonUnescape on bytes and on a string against
// the decoder of encoding/json, bad escapes and bad UTF-8 included
func TestJSONUnescape(t *testing.T) {
	t.Parallel()
	pieces := []string{`\`, `u`, `\u`, `0031`, `d83d`, `de00`, `D83D`, `dC00`, `00e9`, `0`, `"`, "\x01", "\xff", "é",
		`n`, `t`, `b`, `f`, `r`, `/`, `1`, `x`, `g`, `\\`, `\"`, `\/`, `\u0031`, `\ud83d\ude00`, `\ud83d`}
	r := rand.New(rand.NewSource(1))
	for range 300000 {
		var sb strings.Builder
		for range r.Intn(7) {
			sb.WriteString(pieces[r.Intn(len(pieces))])
		}
		d := sb.String()
		t := On(t, d)
		got, ok := jsonUnescape([]byte("pre"), d)
		gotb, okb := jsonUnescape([]byte("pre"), []byte(d))
		require.Equal(t, string(got), string(gotb))
		require.Equal(t, ok, okb)
		// false for no escapes and for what is not valid JSON
		var want string
		if err := json.Unmarshal([]byte(`"`+d+`"`), &want); err != nil || !strings.Contains(d, `\`) {
			require.False(t, ok)
			continue
		}
		require.True(t, ok)
		require.Equal(t, "pre"+want, string(got))
	}
}

// TestMontgomery checks montMul, montSqr and the forms on 64 and 128 bits
// with big.Int: a * b / R % m for odd m of every length, near R too,
// and a, b at random and next to m, where the last subtraction of m comes
// and goes and the sum carries past R
func TestMontgomery(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(3))
	one := big.NewInt(1)
	for _, w := range []int{64, 128, 256} {
		R := new(big.Int).Lsh(one, uint(w))
		for range 3000 {
			n := 2 + r.Intn(w-1)
			m := new(big.Int).Rand(r, new(big.Int).Lsh(one, uint(n)))
			m.SetBit(m, n-1, 1).SetBit(m, 0, 1)
			if r.Intn(4) == 0 {
				// R - 1 - 2j
				m.Sub(R, big.NewInt(1+2*r.Int63n(1000)))
			}
			rinv := new(big.Int).ModInverse(R, m)
			pick := func() *big.Int {
				switch r.Intn(5) {
				case 0:
					return new(big.Int).Sub(m, one)
				case 1:
					return new(big.Int).Sub(m, big.NewInt(2))
				case 2:
					return big.NewInt(int64(r.Intn(2)))
				}
				return new(big.Int).Rand(r, m)
			}
			a, b := pick(), pick()
			want := func(a, b *big.Int) *big.Int {
				v := new(big.Int).Mul(a, b)
				return v.Mod(v.Mul(v, rinv), m)
			}
			t := On(t, w, m, a, b)
			switch w {
			case 64:
				mp := montInv(m.Uint64())
				require.Big(t, want(a, b), new(big.Int).SetUint64(montMul64(a.Uint64(), b.Uint64(), m.Uint64(), mp)))
			case 128:
				x, y, mm := MustUint128FromBig(a), MustUint128FromBig(b), MustUint128FromBig(m)
				mp := montInv(mm.L0)
				require.Big(t, want(a, b), montMul128(x, y, mm, mp).Big())
				require.Big(t, want(a, a), montSqr128(x, mm, mp).Big())
			default:
				x, y, mm := MustUint256FromBig(a), MustUint256FromBig(b), MustUint256FromBig(m)
				mp := montInv(mm.L0)
				require.Big(t, want(a, b), montMul(x, y, mm, mp).Big())
				require.Big(t, want(a, a), montSqr(x, mm, mp).Big())
			}
		}
	}
}
