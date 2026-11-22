package testutil

import (
	"errors"
	"fmt"
	"testing"

	"github.com/avpetkun/fat/internal/require"
)

// On adds args, the operands under test, to require failure messages
func On(t testing.TB, args ...any) testing.TB { return labeled{t, args} }

// labeled is a testing.TB whose Name ends with its args
type labeled struct {
	testing.TB
	args []any
}

// Name returns the test name followed by the args
func (l labeled) Name() string { return l.TB.Name() + " with " + fmt.Sprint(l.args...) }

// PanicValue returns what f panics with, nil when it returns
func PanicValue(f func()) (v any) {
	defer func() { v = recover() }()
	f()
	return nil
}

// CheckConvert checks that conv returns want when the value fits.
// Otherwise conv must panic naming the method, as a negative value if neg
func CheckConvert[R any](t testing.TB, name string, fits, neg bool, want R, conv func() R) {
	if fits {
		require.Equal(On(t, name), want, conv())
		return
	}
	msg := "fat." + name + ": out of range"
	if neg {
		msg = "fat." + name + ": negative value"
	}
	require.Equal(On(t, name), msg, PanicValue(func() { conv() }))
}

// CheckConvertUO checks that u and o return trunc and o reports !fits
func CheckConvertUO[R any](t testing.TB, name string, fits bool, trunc R, u func() R, o func() (R, bool)) {
	t = On(t, name)
	require.Equal(t, trunc, u())
	r, over := o()
	require.Equal(t, trunc, r)
	require.Equal(t, !fits, over)
}

// ErrBad marks a parse case that must fail, the parsers do not say why
var ErrBad = errors.New("bad")
