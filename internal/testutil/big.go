package testutil

import (
	"math/big"
	"slices"
)

// Big1 is 1 as a big.Int, read only
var Big1 = big.NewInt(1)

// LimbsBig builds a big.Int from little-endian limbs, not by the code tested
func LimbsBig(limbs ...uint64) *big.Int {
	b := new(big.Int)
	for _, limb := range slices.Backward(limbs) {
		b.Lsh(b, 64).Or(b, new(big.Int).SetUint64(limb))
	}
	return b
}
