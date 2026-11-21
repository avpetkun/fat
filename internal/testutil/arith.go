package testutil

import (
	"math"
	"math/big"
	"testing"

	"github.com/avpetkun/fat/internal/require"
)

// Arith is a binary operation checked against math/big
type Arith[T any] struct {
	Name  string
	Width int                    // y is narrowed to this many bits
	Fn    func(x, y T) (T, bool) // bool is the reported overflow
	Ref   func(x, y *big.Int) *big.Int
	Over  bool   // whether Fn reports overflow
	Panic string // what Fn panics with past the range, empty when it wraps
}

// RunArith checks every op on the pairs of k against math/big, a subtest each.
// The op wraps, panics or reports overflow as its fields say
func RunArith[T Num[T]](t *testing.T, k Kind[T], ops []Arith[T]) {
	for _, op := range ops {
		t.Run(op.Name, func(t *testing.T) {
			t.Parallel()
			k.Pairs(func(x, y T) {
				y = k.Narrow(y, op.Width)
				xb, yb := k.Ref(x), k.Ref(y)
				want := op.Ref(xb, yb)
				t := On(t, xb, yb)
				if op.Panic != "" && !k.Fits(want) {
					require.Equal(On(t, "exact", want), op.Panic, PanicValue(func() { op.Fn(x, y) }))
					return
				}
				got, over := op.Fn(x, y)
				require.Big(t, k.Wrap(want), k.Ref(got))
				if op.Over {
					require.Same(On(t, "exact", want), !k.Fits(want), over)
				}
			})
		})
	}
}

// RunOps64 checks the methods with a 64-bit operand against math/big.
// The operand is the low limb of y, signed for a signed type
func RunOps64[T Num[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	o := k.Ops64
	mask := new(big.Int).SetUint64(math.MaxUint64)
	type op struct {
		name string
		ref  func(x, y *big.Int) *big.Int
		fn   func(x T, v uint64) T
		over func(x T, v uint64) (T, bool)
	}
	ops := []op{
		{"Add64", BigAdd, o.Add, o.Addo},
		{"Sub64", BigSub, o.Sub, o.Subo},
	}
	k.Pairs(func(x, y T) {
		xb := k.Ref(x)
		v := new(big.Int).And(k.Ref(y), mask).Uint64()
		vb := new(big.Int).SetUint64(v)
		if k.Signed {
			vb.SetInt64(int64(v))
		}
		t := On(t, xb, vb)

		require.Equal(t, xb.Cmp(vb), o.Cmp(x, v))
		require.Big(t, vb, k.Ref(o.New(v)))

		// the u forms wrap where the plain ones panic
		require.Big(t, k.Wrap(BigAdd(xb, vb)), k.Ref(o.Addu(x, v)))
		require.Big(t, k.Wrap(BigSub(xb, vb)), k.Ref(o.Subu(x, v)))
		for _, op := range ops {
			want := op.ref(xb, vb)
			wrapped := k.Wrap(want)
			// the plain form panics where the o form reports overflow,
			// an unsigned one names a lost sign
			if k.Fits(want) {
				require.Big(t, wrapped, k.Ref(op.fn(x, v)))
			} else {
				msg := "overflow"
				if !k.Signed && want.Sign() < 0 {
					msg = "negative value"
				}
				require.Equal(On(t, op.name), "fat."+k.Name+"."+op.name+": "+msg, PanicValue(func() { op.fn(x, v) }))
			}
			got, over := op.over(x, v)
			require.Big(t, wrapped, k.Ref(got))
			require.Same(On(t, "exact", want), !k.Fits(want), over)
		}
	})
}
