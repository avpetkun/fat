//go:build (arch_generic && !arch_amd64) || (!amd64 && !arch_amd64 && !arch_generic)

package arch

// Current is the variant in use
const Current = Generic
