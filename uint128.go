package fat

// Uint128 is an unsigned 128-bit integer, L0 is its low limb and L1 the high
type Uint128 struct{ L0, L1 uint64 }

// --- constructors ---------------------------------------

// NewUint128 returns v as a Uint128
func NewUint128(v uint64) Uint128 {
	return Uint128{v, 0}
}

// --- getters --------------------------------------------

// IsZero reports whether x == 0
func (x Uint128) IsZero() bool {
	return x.L0|x.L1 == 0
}

// IsUint64 reports whether x fits in a uint64
func (x Uint128) IsUint64() bool {
	return x.L1 == 0
}

// Uint64 returns x as a uint64, it panics when x does not fit
func (x Uint128) Uint64() uint64 {
	if !x.IsUint64() {
		panic("fat.Uint128.Uint64: out of range")
	}
	return x.L0
}

// Uint64o returns Uint64u and whether x does not fit
func (x Uint128) Uint64o() (r uint64, overflow bool) {
	return x.Uint64u(), !x.IsUint64()
}

// Uint64u returns the low 64 bits of x as a Go conversion does
func (x Uint128) Uint64u() uint64 {
	return x.L0
}

// Uint256 returns x widened to 256 bits
func (x Uint128) Uint256() Uint256 {
	return Uint256{x.L0, x.L1, 0, 0}
}

// IsInt64 reports whether x fits in an int64
func (x Uint128) IsInt64() bool {
	return x.L1|x.L0>>63 == 0
}

// Int64 returns x as an int64, it panics when x does not fit
func (x Uint128) Int64() int64 {
	if !x.IsInt64() {
		panic("fat.Uint128.Int64: out of range")
	}
	return int64(x.L0)
}

// Int64o returns Int64u and whether x does not fit
func (x Uint128) Int64o() (r int64, overflow bool) {
	return x.Int64u(), !x.IsInt64()
}

// Int64u returns the low 64 bits of x as a Go conversion does
func (x Uint128) Int64u() int64 {
	return int64(x.L0)
}

// IsInt128 reports whether x fits in an Int128
func (x Uint128) IsInt128() bool {
	return x.L1>>63 == 0
}

// Int128 returns x as an Int128, it panics when x does not fit
func (x Uint128) Int128() Int128 {
	if !x.IsInt128() {
		panic("fat.Uint128.Int128: out of range")
	}
	return Int128(x)
}

// Int128o returns Int128u and whether x does not fit
func (x Uint128) Int128o() (r Int128, overflow bool) {
	return x.Int128u(), !x.IsInt128()
}

// Int128u returns the bits of x as a Go conversion does
func (x Uint128) Int128u() Int128 {
	return Int128(x)
}

// Int256 returns x widened to 256 bits
func (x Uint128) Int256() Int256 {
	return Int256{x.L0, x.L1, 0, 0}
}
