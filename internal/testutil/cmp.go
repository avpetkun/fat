package testutil

import (
	"math/big"
	"testing"

	"github.com/avpetkun/fat/internal/require"
)

// RunCmp checks Cmp, LessThan and the rest, Max, Min and CmpBig against math/big.
// CmpBig also gets big.Int values just and far outside the range
func RunCmp[T Num[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	k.Pairs(func(x, y T) {
		xb, yb := k.Ref(x), k.Ref(y)
		t := On(t, xb, yb)

		c := xb.Cmp(yb)
		require.Equal(t, c, x.Cmp(y))
		require.Equal(t, c < 0, x.LessThan(y))
		require.Equal(t, c <= 0, x.LessOrEqual(y))
		require.Equal(t, c > 0, x.GreaterThan(y))
		require.Equal(t, c >= 0, x.GreaterOrEqual(y))
		hi, lo := x, y
		if c < 0 {
			hi, lo = y, x
		}
		require.Equal(t, hi, x.Max(y))
		require.Equal(t, lo, x.Min(y))
		require.Equal(t, c, x.CmpBig(yb))
		nb := new(big.Int).Neg(yb)
		require.Equal(t, xb.Cmp(nb), x.CmpBig(nb))
		if k.CmpMore != nil {
			k.CmpMore(t, x, y, xb)
		}
	})

	// big.Int just outside the range, and far outside it
	far := new(big.Int).Lsh(Big1, uint(2*k.Bits))
	k.Values(func(x T) {
		require.Equal(t, -1, x.CmpBig(k.Over()))
		require.Equal(t, 1, x.CmpBig(BigSub(k.Ref(k.Min), Big1)))
		require.Equal(t, -1, x.CmpBig(far))
		require.Equal(t, 1, x.CmpBig(new(big.Int).Neg(far)))
		if !k.Signed {
			require.Equal(t, 1, x.CmpBig(new(big.Int).Neg(k.Over())))
		}
	})
}

// RunSign checks Sign and IsZero against math/big
func RunSign[T SignNum[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	k.Values(func(x T) {
		xb := k.Ref(x)
		t := On(t, xb)

		s := xb.Sign()
		require.Equal(t, s, x.Sign())
		require.Equal(t, s == 0, x.IsZero())
	})
}

// RunSigned checks IsNegative, IsPositive, Neg, Abs and UnsignedAbs against math/big,
// uabs gives UnsignedAbs of x as a big.Int. Neg and Abs of the minimum must panic with overflow
func RunSigned[T SignedNum[T]](t *testing.T, k Kind[T], uabs func(T) *big.Int) {
	t.Parallel()
	k.Values(func(x T) {
		xb := k.Ref(x)
		t := On(t, xb)

		require.Equal(t, xb.Sign() < 0, x.IsNegative())
		require.Equal(t, xb.Sign() > 0, x.IsPositive())
		// only -minimum and |minimum| do not fit
		if neg := new(big.Int).Neg(xb); k.Fits(neg) {
			require.Big(t, neg, k.Ref(x.Neg()))
		} else {
			require.Equal(t, "fat."+k.Name+".Neg: overflow", PanicValue(func() { x.Neg() }))
		}
		abs := new(big.Int).Abs(xb)
		require.Big(t, abs, uabs(x))
		if k.Fits(abs) {
			require.Big(t, abs, k.Ref(x.Abs()))
		} else {
			require.Equal(t, "fat."+k.Name+".Abs: overflow", PanicValue(func() { x.Abs() }))
		}
	})
	require.Equal(t, "fat."+k.Name+".Neg: overflow", PanicValue(func() { k.Min.Neg() }))
	require.Equal(t, "fat."+k.Name+".Abs: overflow", PanicValue(func() { k.Min.Abs() }))
}
