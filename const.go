package fat

import "math"

// Functions in place of vars: the literal inlines into immediates, a var is
// loaded from memory at every use

// limits of the 128-bit types

// MaxUint128 returns 340282366920938463463374607431768211455
func MaxUint128() Uint128 { return Uint128{math.MaxUint64, math.MaxUint64} }

// MaxInt128 returns 170141183460469231731687303715884105727
func MaxInt128() Int128 { return Int128{math.MaxUint64, math.MaxInt64} }

// MinInt128 returns -170141183460469231731687303715884105728
func MinInt128() Int128 { return Int128{0, 1 << 63} }

// limits of the 256-bit types

// MaxUint256 returns 115792089237316195423570985008687907853269984665640564039457584007913129639935
func MaxUint256() Uint256 {
	return Uint256{math.MaxUint64, math.MaxUint64, math.MaxUint64, math.MaxUint64}
}

// MaxInt256 returns 57896044618658097711785492504343953926634992332820282019728792003956564819967
func MaxInt256() Int256 {
	return Int256{math.MaxUint64, math.MaxUint64, math.MaxUint64, math.MaxInt64}
}

// MinInt256 returns -57896044618658097711785492504343953926634992332820282019728792003956564819968
func MinInt256() Int256 { return Int256{0, 0, 0, 1 << 63} }
