package require

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

// True stops the test when value is false
func True(t testing.TB, value bool) {
	if !value {
		t.Helper()
		failNow(t, "Should be true")
	}
}

// False stops the test when value is true
func False(t testing.TB, value bool) {
	if value {
		t.Helper()
		failNow(t, "Should be false")
	}
}

// Error stops the test when err is nil
func Error(t testing.TB, err error) {
	if err == nil {
		t.Helper()
		failNow(t, "An error is expected but got nil")
	}
}

// NoError stops the test when err is not nil
func NoError(t testing.TB, err error) {
	if err != nil {
		t.Helper()
		failNow(t, "Received unexpected error:", fmt.Sprint(err))
	}
}

// Equal stops the test unless expected and actual are deeply equal.
// Values of different types are never equal
func Equal(t testing.TB, expected, actual any) {
	switch {
	case reflect.DeepEqual(expected, actual):
	case reflect.TypeOf(expected) != reflect.TypeOf(actual):
		t.Helper()
		failNow(t,
			"Not equal types:",
			"expected: "+fmt.Sprintf("%T", expected),
			"actual  : "+fmt.Sprintf("%T", actual),
		)
	default:
		t.Helper()
		failNow(t,
			"Not equal:",
			"expected: "+fmt.Sprint(expected),
			"actual  : "+fmt.Sprint(actual),
		)
	}
}

// Same is Equal for a comparable T, it boxes the values only on failure
func Same[T comparable](t testing.TB, expected, actual T) {
	if expected != actual {
		t.Helper()
		Equal(t, expected, actual)
	}
}

// Big stops the test unless want and got are the same number
func Big(t testing.TB, want, got *big.Int) {
	if want.Cmp(got) != 0 {
		t.Helper()
		failNow(t,
			"Not equal:",
			"expected: "+want.String(),
			"actual  : "+got.String(),
		)
	}
}

// Append checks that fn appends want to a dst with spare free bytes.
// It reuses dst when want fits, and no other byte of dst changes
func Append(t testing.TB, want string, spare int, fn func(dst []byte) []byte) {
	t.Helper()

	const prefix = "ab"
	// printable filler keeps failure output readable
	buf := bytes.Repeat([]byte{'#'}, len(prefix)+spare)
	copy(buf, prefix)
	orig := string(buf)

	got := fn(buf[:len(prefix):len(buf)])
	Equal(t, prefix+want, string(got))

	if spare >= len(want) {
		// result reuses dst, bytes past it are untouched
		True(t, &got[0] == &buf[0])
		Equal(t, orig[len(got):], string(buf[len(got):]))
	} else {
		// result is a new slice, dst is left as is
		True(t, &got[0] != &buf[0])
		Equal(t, orig, string(buf))
	}
}

// failNow stops the test with the lines of errors and the test name
func failNow(t testing.TB, errors ...string) {
	t.Helper()

	var msg strings.Builder
	for i, err := range errors {
		if i == 0 {
			fmt.Fprintf(&msg, "\nError: %s", err)
		} else {
			fmt.Fprintf(&msg, "\n       %s", err)
		}
	}
	fmt.Fprintf(&msg, "\nTest:  %s", t.Name())

	t.Fatal(msg.String())
}
