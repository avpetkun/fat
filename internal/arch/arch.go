// Package arch picks the variant of the code written per arch.
// The variant is a constant, so a test of it costs nothing
// and the other variant is not compiled in.
// Every variant is plain Go and correct on any arch.
// Each arch builds its own by default, generic where it has none,
// and a build tag picks one: -tags arch_amd64 or -tags arch_generic
package arch

// Kind names the variants of the code written per arch
type Kind int

const (
	// Generic is the code of every arch without its own, tuned on Apple arm64
	Generic Kind = iota
	// Amd64 is the code for amd64, where bits.Div64 is one hardware division
	Amd64
)

// IsGeneric and IsAmd64 report the variant in use
const (
	IsGeneric = Current == Generic
	IsAmd64   = Current == Amd64
)
