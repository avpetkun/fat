package testutil

import (
	"testing"
)

// RunBinaryLengths checks UnmarshalBinary and GobDecode on every length.
// Only the type size is taken, its value is compared with math/big
func RunBinaryLengths[T Num[T], PT NumPtr[T]](t *testing.T, k Kind[T]) {
}
