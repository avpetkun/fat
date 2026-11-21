package fat

// Int128 is a signed 128-bit integer in two's complement.
// L0 holds the low 64 bits and L1 the high ones
type Int128 struct{ L0, L1 uint64 }

// --- constructors ---------------------------------------

// NewInt128 returns v as an Int128
func NewInt128(v int64) Int128 {
	ext := uint64(v >> 63)
	return Int128{uint64(v), ext}
}

// --- getters --------------------------------------------

// IsZero reports whether x == 0
func (x Int128) IsZero() bool {
	return x.L0|x.L1 == 0
}

// IsNegative reports whether x < 0
func (x Int128) IsNegative() bool {
	return x.L1>>63 == 1
}

// IsInt64 reports whether x fits in an int64
func (x Int128) IsInt64() bool {
	return x.L1 == uint64(int64(x.L0)>>63)
}

// Int64 returns x as an int64, it panics when x does not fit
func (x Int128) Int64() int64 {
	if !x.IsInt64() {
		panic("fat.Int128.Int64: out of range")
	}
	return int64(x.L0)
}

// Int64o returns Int64u and whether x does not fit
func (x Int128) Int64o() (r int64, overflow bool) {
	return x.Int64u(), !x.IsInt64()
}

// Int64u returns the low 64 bits of x as a Go conversion does
func (x Int128) Int64u() int64 {
	return int64(x.L0)
}

// Int256 returns x widened to 256 bits
func (x Int128) Int256() Int256 {
	m := uint64(int64(x.L1) >> 63)
	return Int256{x.L0, x.L1, m, m}
}

// IsUint64 reports whether x fits in a uint64
func (x Int128) IsUint64() bool {
	return x.L1 == 0
}

// Uint64 returns x as a uint64, it panics when x does not fit
func (x Int128) Uint64() uint64 {
	if !x.IsUint64() {
		if x.IsNegative() {
			panic("fat.Int128.Uint64: negative value")
		}
		panic("fat.Int128.Uint64: out of range")
	}
	return x.L0
}

// Uint64o returns Uint64u and whether x does not fit
func (x Int128) Uint64o() (r uint64, overflow bool) {
	return x.Uint64u(), !x.IsUint64()
}

// Uint64u returns the low 64 bits of x as a Go conversion does
func (x Int128) Uint64u() uint64 {
	return x.L0
}

// IsUint128 reports whether x fits in a Uint128
func (x Int128) IsUint128() bool {
	return int64(x.L1) >= 0
}

// Uint128 returns x as a Uint128, it panics when x does not fit
func (x Int128) Uint128() Uint128 {
	if !x.IsUint128() {
		panic("fat.Int128.Uint128: negative value")
	}
	return Uint128(x)
}

// Uint128o returns Uint128u and whether x does not fit
func (x Int128) Uint128o() (r Uint128, overflow bool) {
	return x.Uint128u(), !x.IsUint128()
}

// Uint128u returns the bits of x as a Go conversion does
func (x Int128) Uint128u() Uint128 {
	return Uint128(x)
}

// IsUint256 reports whether x fits in a Uint256
func (x Int128) IsUint256() bool {
	return int64(x.L1) >= 0
}

// Uint256 returns x as a Uint256, it panics when x does not fit
func (x Int128) Uint256() Uint256 {
	if !x.IsUint256() {
		panic("fat.Int128.Uint256: negative value")
	}
	return Uint256{x.L0, x.L1, 0, 0}
}

// Uint256o returns Uint256u and whether x does not fit
func (x Int128) Uint256o() (r Uint256, overflow bool) {
	return x.Uint256u(), !x.IsUint256()
}

// Uint256u returns x sign-extended to 256 bits as a Go conversion does
func (x Int128) Uint256u() Uint256 {
	m := uint64(int64(x.L1) >> 63)
	return Uint256{x.L0, x.L1, m, m}
}
