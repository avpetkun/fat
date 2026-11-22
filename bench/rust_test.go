package bench

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/avpetkun/fat"

	"github.com/avpetkun/fat/internal/require"
)

// The comparison with Rust's u128 and i128 lives in rust/, run by
// rust/run.sh: TestDumpRust writes the inputs and fat's results for the
// Rust side, the BenchmarkRust* below time fat on the same cases.
// They skip unless RUST_DATA names the directory of that dump

// rustCase is one case of the Rust comparison, f gives fat's result
type rustCase struct {
	name string
	x, y []fat.Uint128
	f    func(x, y fat.Uint128) fat.Uint128
}

// rustDir returns the dump directory and skips without one
func rustDir(tb testing.TB) string {
	dir := os.Getenv("RUST_DATA")
	if dir == "" {
		tb.Skip("RUST_DATA is not set, run rust/run.sh")
	}
	return dir
}

// rustCases lists the cases in the names both sides print
func rustCases() []rustCase {
	u, s := inputsU128(), inputsI128()
	us := func(v []fat.Int128) []fat.Uint128 {
		out := make([]fat.Uint128, len(v))
		for i, x := range v {
			out[i] = fat.Uint128(x)
		}
		return out
	}
	ui := func(f func(x, y fat.Int128) fat.Int128) func(x, y fat.Uint128) fat.Uint128 {
		return func(x, y fat.Uint128) fat.Uint128 { return fat.Uint128(f(fat.Int128(x), fat.Int128(y))) }
	}
	const ub, ib = "RustUint128/", "RustInt128/"
	sx, sy, sfx, sfy, ss := us(s.x.s), us(s.y.s), us(s.fx.s), us(s.fy.s), us(s.small.s)
	return []rustCase{
		{ub + "op=Add/in=fit", u.fx.s, u.fy.s, fat.Uint128.Add},
		{ub + "op=Sub/in=fit", u.fx.s, u.fy.s, fat.Uint128.Sub},
		{ub + "op=Mul/in=fit", u.fx.s, u.fy.s, fat.Uint128.Mul},
		{ub + "op=Addu/in=mixed", u.x.s, u.y.s, fat.Uint128.Addu},
		{ub + "op=Subu/in=mixed", u.x.s, u.y.s, fat.Uint128.Subu},
		{ub + "op=Mulu/in=mixed", u.x.s, u.y.s, fat.Uint128.Mulu},
		{ub + "op=Div/in=mixed", u.x.s, u.y.s, fat.Uint128.Div},
		{ub + "op=Div/in=div64", u.x.s, u.small.s, fat.Uint128.Div},
		{ub + "op=Div/in=full", u.full.s, u.y.s, fat.Uint128.Div},
		{ub + "op=Mod/in=mixed", u.x.s, u.y.s, fat.Uint128.Mod},
		{ub + "op=Mod/in=div64", u.x.s, u.small.s, fat.Uint128.Mod},
		{ub + "op=Mod/in=full", u.full.s, u.y.s, fat.Uint128.Mod},
		{ib + "op=Add/in=fit", sfx, sfy, ui(fat.Int128.Add)},
		{ib + "op=Sub/in=fit", sfx, sfy, ui(fat.Int128.Sub)},
		{ib + "op=Mul/in=fit", sfx, sfy, ui(fat.Int128.Mul)},
		{ib + "op=Addu/in=mixed", sx, sy, ui(fat.Int128.Addu)},
		{ib + "op=Subu/in=mixed", sx, sy, ui(fat.Int128.Subu)},
		{ib + "op=Mulu/in=mixed", sx, sy, ui(fat.Int128.Mulu)},
		{ib + "op=Div/in=mixed", sx, sy, ui(fat.Int128.Div)},
		{ib + "op=Div/in=div64", sx, ss, ui(fat.Int128.Div)},
		{ib + "op=Mod/in=mixed", sx, sy, ui(fat.Int128.Mod)},
		{ib + "op=Mod/in=div64", sx, ss, ui(fat.Int128.Mod)},
	}
}

// TestDumpRust writes per case x, y and fat's first agreeN results as
// little-endian 128-bit words, plus the GOMAXPROCS suffix of the names
func TestDumpRust(t *testing.T) {
	dir := rustDir(t)
	put := func(b []byte, v []fat.Uint128) []byte {
		for _, x := range v {
			b = binary.LittleEndian.AppendUint64(b, x.L0)
			b = binary.LittleEndian.AppendUint64(b, x.L1)
		}
		return b
	}
	for _, c := range rustCases() {
		r := make([]fat.Uint128, agreeN)
		for i := range r {
			r[i] = c.f(c.x[i], c.y[i])
		}
		name := strings.ReplaceAll(c.name, "/", "_") + ".bin"
		b := put(put(put(nil, c.x), c.y), r)
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), b, 0o644))
	}
	procs := strconv.Itoa(runtime.GOMAXPROCS(0))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "procs"), []byte(procs), 0o644))
}

// BenchmarkRustUint128 is fat's side of the Rust comparison for Uint128:
// the checked forms panic as checked_* with unwrap does,
// the u forms wrap as wrapping_* does
func BenchmarkRustUint128(b *testing.B) {
	rustDir(b)
	in := inputsU128()
	b.Run("op=Add/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.fx.s[i&m].Add(in.fy.s[i&m])
		}
	})
	b.Run("op=Sub/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.fx.s[i&m].Sub(in.fy.s[i&m])
		}
	})
	b.Run("op=Mul/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.fx.s[i&m].Mul(in.fy.s[i&m])
		}
	})
	b.Run("op=Addu/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].Addu(in.y.s[i&m])
		}
	})
	b.Run("op=Subu/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].Subu(in.y.s[i&m])
		}
	})
	b.Run("op=Mulu/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sU128 = in.x.s[i&m].Mulu(in.y.s[i&m])
		}
	})
	for _, set := range []struct {
		name string
		x, y u128
	}{{"mixed", in.x, in.y}, {"div64", in.x, in.small}, {"full", in.full, in.y}} {
		x, y := set.x.s, set.y.s
		b.Run("op=Div/in="+set.name+"/impl=fat", func(b *testing.B) {
			for i := range b.N {
				sU128 = x[i&m].Div(y[i&m])
			}
		})
		b.Run("op=Mod/in="+set.name+"/impl=fat", func(b *testing.B) {
			for i := range b.N {
				sU128 = x[i&m].Mod(y[i&m])
			}
		})
	}
}

// BenchmarkRustInt128 is fat's side of the Rust comparison for Int128
func BenchmarkRustInt128(b *testing.B) {
	rustDir(b)
	in := inputsI128()
	b.Run("op=Add/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI128 = in.fx.s[i&m].Add(in.fy.s[i&m])
		}
	})
	b.Run("op=Sub/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI128 = in.fx.s[i&m].Sub(in.fy.s[i&m])
		}
	})
	b.Run("op=Mul/in=fit/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI128 = in.fx.s[i&m].Mul(in.fy.s[i&m])
		}
	})
	b.Run("op=Addu/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI128 = in.x.s[i&m].Addu(in.y.s[i&m])
		}
	})
	b.Run("op=Subu/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI128 = in.x.s[i&m].Subu(in.y.s[i&m])
		}
	})
	b.Run("op=Mulu/in=mixed/impl=fat", func(b *testing.B) {
		for i := range b.N {
			sI128 = in.x.s[i&m].Mulu(in.y.s[i&m])
		}
	})
	for _, set := range []struct {
		name string
		x, y i128
	}{{"mixed", in.x, in.y}, {"div64", in.x, in.small}} {
		x, y := set.x.s, set.y.s
		b.Run("op=Div/in="+set.name+"/impl=fat", func(b *testing.B) {
			for i := range b.N {
				sI128 = x[i&m].Div(y[i&m])
			}
		})
		b.Run("op=Mod/in="+set.name+"/impl=fat", func(b *testing.B) {
			for i := range b.N {
				sI128 = x[i&m].Mod(y[i&m])
			}
		})
	}
}
