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
