//go:build (arch_amd64 && !arch_generic) || (amd64 && !arch_amd64 && !arch_generic)

package arch

// Current is the variant in use
const Current = Amd64
