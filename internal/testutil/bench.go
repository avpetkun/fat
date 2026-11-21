package testutil

import (
	"math/big"
	"math/rand"
	"sync"
)

// compare runs with:
// go test -run '^$' -bench . -count 6 ./... | benchstat -

// BenchMask indexes the 2^18 inputs, too many for a branch predictor to learn
const BenchMask = 1<<18 - 1

// Sinks take a benchmark result on each iteration so the work is kept
var (
	SinkBool   bool
	SinkInt    int
	SinkUint   uint
	SinkUint64 uint64
	SinkBig    *big.Int
)

// BenchInputs holds 2^18 limb values per type, each test views them as its type
type BenchInputs struct {
	// random bit length per value, signed ones with random sign
	U256, V256 [][4]uint64
	I256, J256 [][4]uint64
	U128, V128 [][2]uint64
	I128, J128 [][2]uint64
	U64        []uint64
	I64        []int64

	// pairs whose sum, difference and product fit, x has more bits than y
	FU256, FV256 [][4]uint64
	FI256, FJ256 [][4]uint64
	FU128, FV128 [][2]uint64
	FI128, FJ128 [][2]uint64

	// decimal strings of U256, I256, U128 and I128
	U256s, I256s, U128s, I128s []string
}

// BenchIn builds the BenchInputs once, from fixed seeds
var BenchIn = sync.OnceValue(func() *BenchInputs {
	r := rand.New(rand.NewSource(1))
	in := &BenchInputs{}
	low := func(l [4]uint64) [2]uint64 { return [2]uint64{l[0], l[1]} }
	type u256 struct{ L0, L1, L2, L3 uint64 }
	type u128 struct{ L0, L1 uint64 }
	for range BenchMask + 1 {
		in.U256 = append(in.U256, RandBits(r, 1+r.Intn(256)))
		in.V256 = append(in.V256, RandBits(r, 1+r.Intn(256)))
		in.I256 = append(in.I256, RandSigned256(r, 1+r.Intn(255)))
		in.J256 = append(in.J256, RandSigned256(r, 1+r.Intn(255)))
		in.U128 = append(in.U128, low(RandBits(r, 1+r.Intn(128))))
		in.V128 = append(in.V128, low(RandBits(r, 1+r.Intn(128))))
		in.I128 = append(in.I128, low(RandSigned256(r, 1+r.Intn(127))))
		in.J128 = append(in.J128, low(RandSigned256(r, 1+r.Intn(127))))
		in.U64 = append(in.U64, RandBits(r, 1+r.Intn(64))[0])
		in.I64 = append(in.I64, int64(RandSigned256(r, 1+r.Intn(63))[0]))

		in.U256s = append(in.U256s, RefU256(Limbs256[u256](in.U256[len(in.U256)-1])).String())
		in.I256s = append(in.I256s, RefI256(Limbs256[u256](in.I256[len(in.I256)-1])).String())
		in.U128s = append(in.U128s, RefU128(Limbs128[u128](in.U128[len(in.U128)-1])).String())
		in.I128s = append(in.I128s, RefI128(Limbs128[u128](in.I128[len(in.I128)-1])).String())
	}
	// a source of their own keeps the sets above as they were
	r = rand.New(rand.NewSource(2))
	// bit lengths for w bits: x of 2..w-1, y below x and within w-x
	fit := func(w int) (int, int) {
		lx := 2 + r.Intn(w-2)
		return lx, 1 + r.Intn(min(lx-1, w-lx))
	}
	// x >= y for Sub, RandBits leaves the top bit random at 64*k bits
	ordered := func(x, y [4]uint64) ([4]uint64, [4]uint64) {
		for i := 3; i >= 0; i-- {
			if x[i] != y[i] {
				if x[i] < y[i] {
					return y, x
				}
				break
			}
		}
		return x, y
	}
	for range BenchMask + 1 {
		lx, ly := fit(256)
		x, y := ordered(RandBits(r, lx), RandBits(r, ly))
		in.FU256, in.FV256 = append(in.FU256, x), append(in.FV256, y)
		lx, ly = fit(255)
		in.FI256, in.FJ256 = append(in.FI256, RandSigned256(r, lx)), append(in.FJ256, RandSigned256(r, ly))
		lx, ly = fit(128)
		x, y = ordered(RandBits(r, lx), RandBits(r, ly))
		in.FU128, in.FV128 = append(in.FU128, low(x)), append(in.FV128, low(y))
		lx, ly = fit(127)
		in.FI128, in.FJ128 = append(in.FI128, low(RandSigned256(r, lx))), append(in.FJ128, low(RandSigned256(r, ly)))
	}
	return in
})

// Slice128 views 128-bit limbs as T
func Slice128[T Shape128](l [][2]uint64) []T {
	out := make([]T, len(l))
	for i, x := range l {
		out[i] = Limbs128[T](x)
	}
	return out
}

// Slice256 views 256-bit limbs as T
func Slice256[T Shape256](l [][4]uint64) []T {
	out := make([]T, len(l))
	for i, x := range l {
		out[i] = Limbs256[T](x)
	}
	return out
}

// BoolInt returns 1 for true and 0 for false
func BoolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
