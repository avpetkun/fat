package fat

// Int256 is a signed 256-bit integer in two's complement, L0 is the low limb
type Int256 struct{ L0, L1, L2, L3 uint64 }

// --- constructors ---------------------------------------

// NewInt256 returns v as an Int256
func NewInt256(v int64) Int256 {
	ext := uint64(v >> 63)
	return Int256{uint64(v), ext, ext, ext}
}

// --- getters --------------------------------------------

// IsZero reports whether x == 0
func (x Int256) IsZero() bool {
	return x.L0|x.L1|x.L2|x.L3 == 0
}

// IsNegative reports whether x < 0
func (x Int256) IsNegative() bool {
	return x.L3>>63 == 1
}

// IsInt64 reports whether x fits in an int64
func (x Int256) IsInt64() bool {
	// every upper limb is the sign extension of L0
	m := uint64(int64(x.L0) >> 63)
	return (x.L1^m)|(x.L2^m)|(x.L3^m) == 0
}

// Int64 returns x as an int64, it panics when x does not fit
func (x Int256) Int64() int64 {
	if !x.IsInt64() {
		panic("fat.Int256.Int64: out of range")
	}
	return int64(x.L0)
}

// Int64o returns Int64u and whether x does not fit
func (x Int256) Int64o() (r int64, overflow bool) {
	return x.Int64u(), !x.IsInt64()
}

// Int64u returns the low 64 bits of x as a Go conversion does
func (x Int256) Int64u() int64 {
	return int64(x.L0)
}

// IsInt128 reports whether x fits in an Int128
func (x Int256) IsInt128() bool {
	m := uint64(int64(x.L1) >> 63)
	return (x.L2^m)|(x.L3^m) == 0
}

// Int128 returns x as an Int128, it panics when x does not fit
func (x Int256) Int128() Int128 {
	if !x.IsInt128() {
		panic("fat.Int256.Int128: out of range")
	}
	return Int128{x.L0, x.L1}
}

// Int128o returns Int128u and whether x does not fit
func (x Int256) Int128o() (r Int128, overflow bool) {
	return x.Int128u(), !x.IsInt128()
}

// Int128u returns the low 128 bits of x as a Go conversion does
func (x Int256) Int128u() Int128 {
	return Int128{x.L0, x.L1}
}

// IsUint64 reports whether x fits in a uint64
func (x Int256) IsUint64() bool {
	return x.L1|x.L2|x.L3 == 0
}

// Uint64 returns x as a uint64, it panics when x does not fit
func (x Int256) Uint64() uint64 {
	if !x.IsUint64() {
		if x.IsNegative() {
			panic("fat.Int256.Uint64: negative value")
		}
		panic("fat.Int256.Uint64: out of range")
	}
	return x.L0
}

// Uint64o returns Uint64u and whether x does not fit
func (x Int256) Uint64o() (r uint64, overflow bool) {
	return x.Uint64u(), !x.IsUint64()
}

// Uint64u returns the low 64 bits of x as a Go conversion does
func (x Int256) Uint64u() uint64 {
	return x.L0
}

// IsUint128 reports whether x fits in a Uint128
func (x Int256) IsUint128() bool {
	return x.L2|x.L3 == 0
}

// Uint128 returns x as a Uint128, it panics when x does not fit
func (x Int256) Uint128() Uint128 {
	if !x.IsUint128() {
		if x.IsNegative() {
			panic("fat.Int256.Uint128: negative value")
		}
		panic("fat.Int256.Uint128: out of range")
	}
	return Uint128{x.L0, x.L1}
}

// Uint128o returns Uint128u and whether x does not fit
func (x Int256) Uint128o() (r Uint128, overflow bool) {
	return x.Uint128u(), !x.IsUint128()
}

// Uint128u returns the low 128 bits of x as a Go conversion does
func (x Int256) Uint128u() Uint128 {
	return Uint128{x.L0, x.L1}
}

// IsUint256 reports whether x fits in a Uint256, that is x >= 0
func (x Int256) IsUint256() bool {
	return int64(x.L3) >= 0
}

// Uint256 returns x as a Uint256, it panics for x < 0
func (x Int256) Uint256() Uint256 {
	if !x.IsUint256() {
		panic("fat.Int256.Uint256: negative value")
	}
	return Uint256(x)
}

// Uint256o returns Uint256u and whether x < 0
func (x Int256) Uint256o() (r Uint256, overflow bool) {
	return x.Uint256u(), !x.IsUint256()
}

// Uint256u returns the bits of x as a Go conversion does
func (x Int256) Uint256u() Uint256 {
	return Uint256(x)
}
