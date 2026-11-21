package require

import (
	"bytes"
	"errors"
	"math"
	"math/big"
	"slices"
	"testing"
	"time"
)

// requireT is a testing.TB that records a failure and goes on
type requireT struct {
	testing.TB
	failed bool
}

// Name returns a fixed test name
func (t *requireT) Name() string { return "fake" }

// Helper does nothing
func (t *requireT) Helper() {}

// Fatal records the failure
func (t *requireT) Fatal(args ...any) { t.failed = true }

// TestRequire checks that every assertion passes and fails when it should
func TestRequire(t *testing.T) {
	type point struct{ x, y int }

	// success

	True(t, true)
	False(t, false)
	Error(t, errors.New("error"))
	NoError(t, nil)
	Equal(t, "123", "123")
	Equal(t, nil, nil)
	Equal(t, []int{1, 2}, []int{1, 2})
	Equal(t, point{1, 2}, point{1, 2})
	Same(t, 1, 1)
	Same(t, "a", "a")
	Same(t, point{1, 2}, point{1, 2})
	Big(t, big.NewInt(-5), big.NewInt(-5))

	// app returns a correct append of s
	app := func(s string) func([]byte) []byte {
		return func(dst []byte) []byte { return append(dst, s...) }
	}
	Append(t, "", 0, app(""))
	Append(t, "12", 0, app("12"))
	Append(t, "12", 1, app("12"))
	Append(t, "12", 2, app("12"))
	Append(t, "12", 8, app("12"))

	// fails

	r := new(requireT)
	fail := func(assert func()) {
		t.Helper()
		// failed stays set when a later check in assert passes
		r.failed = false
		assert()
		True(t, r.failed)
	}

	fail(func() { True(r, false) })
	fail(func() { False(r, true) })

	fail(func() { Error(r, nil) })
	fail(func() { NoError(r, errors.New("boom")) })

	fail(func() { Equal(r, 1, 2) })
	fail(func() { Equal(r, 1, int64(1)) })
	fail(func() { Equal(r, nil, 1) })
	fail(func() { Equal(r, 1, nil) })
	fail(func() { Equal(r, "a", "a ") })
	fail(func() { Equal(r, []int{1}, []int{2}) })
	fail(func() { Equal(r, []int{1}, []int64{1}) })
	fail(func() { Equal(r, point{1, 2}, point{3, 4}) })
	fail(func() { Equal(r, &point{1, 2}, (*point)(nil)) })
	fail(func() { Equal(r, []byte("a"), []byte("b")) })
	fail(func() { Equal(r, time.Second, time.Minute) })
	fail(func() { Equal(r, math.NaN(), math.NaN()) })

	fail(func() { Same(r, 1, 2) })
	fail(func() { Same(r, "a", "a ") })
	fail(func() { Same(r, point{1, 2}, point{3, 4}) })
	fail(func() { Same(r, math.NaN(), math.NaN()) })

	fail(func() { Big(r, big.NewInt(1), big.NewInt(2)) })
	fail(func() { Big(r, big.NewInt(1), big.NewInt(-1)) })

	// wrong content
	fail(func() { Append(r, "12", 8, app("13")) })
	fail(func() { Append(r, "12", 8, app("123")) })
	fail(func() { Append(r, "12", 0, app("1")) })
	// drops existing content
	fail(func() {
		Append(r, "12", 8, func(dst []byte) []byte {
			return append(dst[:0], "12"...)
		})
	})
	// copies though dst has room
	fail(func() {
		Append(r, "12", 8, func(dst []byte) []byte {
			return append(bytes.Clone(dst), "12"...)
		})
	})
	// writes past the result
	fail(func() {
		Append(r, "12", 8, func(dst []byte) []byte {
			return append(dst, "12!"...)[:len(dst)+2]
		})
	})
	// touches dst though it has to grow
	fail(func() {
		Append(r, "12", 1, func(dst []byte) []byte {
			dst[:cap(dst)][len(dst)] = '1'
			return append(slices.Clip(dst), "12"...)
		})
	})
}
