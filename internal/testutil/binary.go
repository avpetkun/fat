package testutil

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/avpetkun/fat/internal/require"
)

// RefBE is x in size bytes big-endian, two's complement when negative
func RefBE(x *big.Int, size int) []byte {
	// Mod is Euclidean, a negative x becomes x + 2^(8 size)
	return new(big.Int).Mod(x, new(big.Int).Lsh(Big1, uint(8*size))).FillBytes(make([]byte, size))
}

// Be is b left-padded to size bytes with fill
func Be(size int, fill byte, b ...byte) []byte {
	return append(bytes.Repeat([]byte{fill}, size-len(b)), b...)
}

// BinaryOf is AppendBinary to an empty slice
func BinaryOf(x interface{ AppendBinary([]byte) ([]byte, error) }) []byte {
	b, _ := x.AppendBinary(nil)
	return b
}

// RunBinaryLengths checks UnmarshalBinary and GobDecode on every length.
// Only the type size is taken, its value is compared with math/big
func RunBinaryLengths[T Num[T], PT NumPtr[T]](t *testing.T, k Kind[T]) {
	size := k.Bits / 8
	src := bytes.Repeat([]byte{0x81}, size+1)
	for n := range size + 2 {
		want := new(big.Int).SetBytes(src[:n])
		if k.Signed {
			// 0x81 on top is negative
			want.Sub(want, new(big.Int).Lsh(Big1, uint(8*n)))
		}
		var x, y T
		for name, dec := range map[string]func([]byte) error{"UnmarshalBinary": PT(&x).UnmarshalBinary, "GobDecode": PT(&y).GobDecode} {
			t := On(t, name, n)
			err := dec(src[:n])
			if n != size {
				require.Error(t, err)
				continue
			}
			require.NoError(t, err)
		}
		if n == size {
			require.Big(On(t, n), want, k.Ref(x))
			require.Big(On(t, n), want, k.Ref(y))
		}
	}
}
