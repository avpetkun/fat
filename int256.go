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
