package fat

// Uint256 is an unsigned 256-bit integer of 64-bit limbs, L0 is the lowest
type Uint256 struct{ L0, L1, L2, L3 uint64 }

// --- constructors ---------------------------------------

// NewUint256 returns v as a Uint256
func NewUint256(v uint64) Uint256 {
	return Uint256{v, 0, 0, 0}
}

// --- getters --------------------------------------------

// IsZero reports whether x == 0
func (x Uint256) IsZero() bool {
	return x.L0|x.L1|x.L2|x.L3 == 0
}

// IsUint64 reports whether x fits in a uint64
func (x Uint256) IsUint64() bool {
	return x.L1|x.L2|x.L3 == 0
}

// Uint64 returns x as a uint64, it panics when x does not fit
func (x Uint256) Uint64() uint64 {
	if !x.IsUint64() {
		panic("fat.Uint256.Uint64: out of range")
	}
	return x.L0
}

// Uint64o returns Uint64u and whether x does not fit
func (x Uint256) Uint64o() (r uint64, overflow bool) {
	return x.Uint64u(), !x.IsUint64()
}

// Uint64u returns the low 64 bits of x as a Go conversion does
func (x Uint256) Uint64u() uint64 {
	return x.L0
}

// IsUint128 reports whether x fits in a Uint128
func (x Uint256) IsUint128() bool {
	return x.L2|x.L3 == 0
}

// Uint128 returns x as a Uint128, it panics when x does not fit
func (x Uint256) Uint128() Uint128 {
	if !x.IsUint128() {
		panic("fat.Uint256.Uint128: out of range")
	}
	return Uint128{x.L0, x.L1}
}

// Uint128o returns Uint128u and whether x does not fit
func (x Uint256) Uint128o() (r Uint128, overflow bool) {
	return x.Uint128u(), !x.IsUint128()
}

// Uint128u returns the low 128 bits of x as a Go conversion does
func (x Uint256) Uint128u() Uint128 {
	return Uint128{x.L0, x.L1}
}

// IsInt64 reports whether x fits in an int64
func (x Uint256) IsInt64() bool {
	return x.L1|x.L2|x.L3|x.L0>>63 == 0
}

// Int64 returns x as an int64, it panics when x does not fit
func (x Uint256) Int64() int64 {
	if !x.IsInt64() {
		panic("fat.Uint256.Int64: out of range")
	}
	return int64(x.L0)
}

// Int64o returns Int64u and whether x does not fit
func (x Uint256) Int64o() (r int64, overflow bool) {
	return x.Int64u(), !x.IsInt64()
}

// Int64u returns the low 64 bits of x as a Go conversion does
func (x Uint256) Int64u() int64 {
	return int64(x.L0)
}

// IsInt128 reports whether x fits in an Int128
func (x Uint256) IsInt128() bool {
	return x.L2|x.L3|x.L1>>63 == 0
}

// Int128 returns x as an Int128, it panics when x does not fit
func (x Uint256) Int128() Int128 {
	if !x.IsInt128() {
		panic("fat.Uint256.Int128: out of range")
	}
	return Int128{x.L0, x.L1}
}

// Int128o returns Int128u and whether x does not fit
func (x Uint256) Int128o() (r Int128, overflow bool) {
	return x.Int128u(), !x.IsInt128()
}

// Int128u returns the low 128 bits of x as a Go conversion does
func (x Uint256) Int128u() Int128 {
	return Int128{x.L0, x.L1}
}

// IsInt256 reports whether x fits in an Int256
func (x Uint256) IsInt256() bool {
	return x.L3>>63 == 0
}

// Int256 returns x as an Int256, it panics when x does not fit
func (x Uint256) Int256() Int256 {
	if !x.IsInt256() {
		panic("fat.Uint256.Int256: out of range")
	}
	return Int256(x)
}

// Int256o returns Int256u and whether x does not fit
func (x Uint256) Int256o() (r Int256, overflow bool) {
	return x.Int256u(), !x.IsInt256()
}

// Int256u returns the bits of x as a Go conversion does
func (x Uint256) Int256u() Int256 {
	return Int256(x)
}
