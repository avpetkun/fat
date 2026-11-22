package testutil

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/avpetkun/fat/internal/require"
)

// RunSQL checks Value and Scan against math/big, and sql.Null over the type
func RunSQL[T Num[T], PT NumPtr[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	typ := "fat." + k.Name
	scan := func(t testing.TB, want T, src any) {
		y := k.Dirty
		require.NoError(t, PT(&y).Scan(src))
		require.Equal(t, want, y)
	}
	k.Values(func(x T) {
		xb := k.Ref(x)
		s := xb.String()
		t := On(t, s)

		v, err := x.Value()
		require.NoError(t, err)
		require.Equal(t, any(s), v)
		v, err = driver.DefaultParameterConverter.ConvertValue(x)
		require.NoError(t, err)
		require.Equal(t, any(s), v)

		// text as UnmarshalText, integers when they fit
		scan(t, x, s)
		scan(t, x, []byte(s))
		scan(t, x, HexPrefixed(xb))
		if xb.IsInt64() {
			scan(t, x, xb.Int64())
		}
		if xb.IsUint64() {
			scan(t, x, xb.Uint64())
		}

		// sql.Null calls Value and Scan of the type
		var n sql.Null[T]
		require.NoError(t, n.Scan(s))
		require.True(t, n.Valid)
		require.Equal(t, x, n.V)
		v, err = n.Value()
		require.NoError(t, err)
		require.Equal(t, any(s), v)
	})

	// every int64 and uint64 edge
	for _, i := range []int64{0, 1, -1, 1 << 62, -1 << 63, 1<<63 - 1} {
		if b := big.NewInt(i); k.Fits(b) {
			scan(On(t, i), k.FromBig(b), i)
		} else {
			require.Equal(On(t, i), typ+".Scan: negative value "+strconv.Quote(strconv.FormatInt(i, 10)), fmt.Sprint(PT(new(T)).Scan(i)))
		}
	}
	for _, u := range []uint64{0, 1, 1 << 63, 1<<64 - 1} {
		scan(On(t, u), k.FromBig(new(big.Int).SetUint64(u)), u)
	}

	// errors name the type and leave x as is
	over := k.Over().String()
	for _, c := range []struct {
		src  any
		want string
	}{
		{nil, typ + ".Scan: cannot scan NULL"},
		{1.0, typ + ".Scan: cannot scan float64"},
		{true, typ + ".Scan: cannot scan bool"},
		{int32(1), typ + ".Scan: cannot scan int32"},
		{"", typ + `.Scan: cannot parse "": empty`},
		{"x", typ + `.Scan: cannot parse "x": invalid digit 'x' at offset 0`},
		{[]byte("1.5"), typ + `.Scan: cannot parse "1.5": invalid digit '.' at offset 1`},
		{over, typ + `.Scan: cannot parse "` + over + `": out of range`},
		{"0x" + strings.Repeat("f", k.Bits/4+1), typ + `.Scan: cannot parse "0x` + strings.Repeat("f", k.Bits/4+1) + `": out of range`},
	} {
		y := k.Small(7)
		require.Equal(On(t, c.src), c.want, fmt.Sprint(PT(&y).Scan(c.src)))
		require.Equal(On(t, c.src), k.Small(7), y)
	}
	for _, c := range []struct {
		src  any
		text string
	}{{"-5", "-5"}, {[]byte("-0x1f"), "-0x1f"}, {int64(-1), "-1"}} {
		src := c.src
		y := k.Small(7)
		err := PT(&y).Scan(src)
		if k.Signed {
			require.NoError(On(t, src), err)
		} else {
			require.Equal(On(t, src), typ+".Scan: negative value "+strconv.Quote(c.text), fmt.Sprint(err))
			require.Equal(On(t, src), k.Small(7), y)
		}
	}

	// sql.Null takes NULL itself and writes it back
	n := sql.Null[T]{V: k.Small(7), Valid: true}
	require.NoError(t, n.Scan(nil))
	require.False(t, n.Valid)
	v, err := n.Value()
	require.NoError(t, err)
	require.Equal(t, nil, v)
}

// RunNull checks Scan, Value and JSON of a Null type over the values of k.
// The func get reads its value and Valid, mk builds one
func RunNull[T Num[T], N interface {
	driver.Valuer
	json.Marshaler
}, PN interface {
	*N
	sql.Scanner
	json.Unmarshaler
}](t *testing.T, k Kind[T], get func(N) (T, bool), mk func(T, bool) N) {
	t.Parallel()
	typ := "fat." + k.Name
	k.Values(func(x T) {
		s := k.Ref(x).String()
		t := On(t, s)

		n := mk(k.Dirty, false)
		require.NoError(t, PN(&n).Scan(s))
		v, ok := get(n)
		require.True(t, ok)
		require.Equal(t, x, v)
		dv, err := n.Value()
		require.NoError(t, err)
		require.Equal(t, any(s), dv)
		dv, err = driver.DefaultParameterConverter.ConvertValue(n)
		require.NoError(t, err)
		require.Equal(t, any(s), dv)

		// JSON as the type writes it, read back Valid
		js, err := json.Marshal(n)
		require.NoError(t, err)
		require.Equal(t, `"`+s+`"`, string(js))
		n = mk(k.Dirty, false)
		require.NoError(t, json.Unmarshal(js, PN(&n)))
		v, ok = get(n)
		require.True(t, ok)
		require.Equal(t, x, v)
	})

	// JSON null is not Valid both ways, in a field too
	js, err := json.Marshal(mk(k.Small(7), false))
	require.NoError(t, err)
	require.Equal(t, "null", string(js))
	type doc struct{ A, B N }
	var d doc
	require.NoError(t, json.Unmarshal([]byte(`{"A":"5","B":null}`), &d))
	require.Equal(t, doc{mk(k.Small(5), true), mk(k.Zero, false)}, d)
	js, err = json.Marshal(d)
	require.NoError(t, err)
	require.Equal(t, `{"A":"5","B":null}`, string(js))
	nj := mk(k.Dirty, true)
	require.NoError(t, json.Unmarshal([]byte("null"), PN(&nj)))
	require.Equal(t, mk(k.Zero, false), nj)
	// a JSON error leaves it as is, Valid or not, as encoding/json does
	for _, valid := range []bool{true, false} {
		nj = mk(k.Small(7), valid)
		require.Equal(On(t, valid), typ+`.UnmarshalJSON: cannot parse "x": invalid digit 'x' at offset 0`, fmt.Sprint(json.Unmarshal([]byte(`"x"`), PN(&nj))))
		require.Equal(On(t, valid), mk(k.Small(7), valid), nj)
	}

	// NULL clears a valid one
	n := mk(k.Dirty, true)
	require.NoError(t, PN(&n).Scan(nil))
	v, ok := get(n)
	require.False(t, ok)
	require.Equal(t, k.Zero, v)
	dv, err := n.Value()
	require.NoError(t, err)
	require.Equal(t, nil, dv)
	// a NULL Value even with a value in it
	dv, err = mk(k.Small(7), false).Value()
	require.NoError(t, err)
	require.Equal(t, nil, dv)

	// an error leaves it not Valid
	n = mk(k.Small(7), true)
	require.Equal(t, typ+`.Scan: cannot parse "x": invalid digit 'x' at offset 0`, fmt.Sprint(PN(&n).Scan("x")))
	_, ok = get(n)
	require.False(t, ok)
	require.Equal(t, typ+".Scan: cannot scan float64", fmt.Sprint(PN(&n).Scan(1.0)))
}
