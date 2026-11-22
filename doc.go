// Package fat has fixed-size integers of 128 and 256 bits: Int128, Uint128,
// Int256 and Uint256
//
// All four types are values: assignment copies them, == compares them and
// they work as map keys, and the zero value is 0.
// Methods take and return values, leave their receiver as it was and chain
// into one expression, and the arithmetic does not allocate
//
//	a := fat.MustUint256FromDec("1000000000000000000000000")
//	b := fat.NewUint256(3)
//	q, r := a.Mul64(7).Add(b).DivMod(b)
//
// # Overflow
//
// An operation comes in up to three forms, told apart by the suffix:
//
//	x.Add(y)  // panics on overflow
//	x.Addo(y) // returns the wrapped result and whether it overflowed
//	x.Addu(y) // wraps around, as a Go integer does
//
// The same goes for Sub, Mul, Exp, MulDiv, for Div on the signed types and
// for the conversions between the types, such as Uint64, Uint64o and Uint64u.
// Mule returns the full product of twice the width.
// A division by zero panics in every form, as in Go
//
// # Narrow operands
//
// A 64 in the name takes a uint64 on the unsigned types and an int64 on the
// signed ones: Add64, Mul64, DivMod64, Cmp64.
// The 256-bit types take the 128-bit type of their sign by a 128 the same
// way: Add128, Mul128, DivMod128
//
// # Products that do not fit
//
// MulDiv, MulDivCeil and MulDivMod divide the exact product, so only the
// quotient has to fit.
// AddMod, SubMod, MulMod and ExpMod reduce the exact sum, difference, product
// or power the same way, and ModInverse and GCD complete the set.
// The time of an operation depends on its values, so the package does not
// suit secret data
//
// # Signed types
//
// Div truncates toward zero and Mod takes the sign of x, as / and % of Go.
// Rsh rounds toward negative infinity, as >> does.
// Abs and Neg panic for the minimum, and UnsignedAbs and GCD return the
// unsigned type, which holds its absolute value
//
// # Text and JSON
//
// FromDec and FromHex parse a string or a byte slice and return ErrParse,
// or ErrNegative for a negative value given to an unsigned type.
// Their Must forms panic.
// Dec, Hex, PrettyDec and String format a value, AppendDec and AppendHex
// write to a buffer, and Format serves the verbs of fmt as big.Int does.
// JSON holds a value as a quoted decimal string, since a parser on float64
// loses the digits of a number past 2^53.
// It reads a string, decimal or hex after 0x, or a plain number
//
// # Binary, math/big and SQL
//
// MarshalBinary writes 16 or 32 bytes big-endian, two's complement on the
// signed types, and UnmarshalBinary takes that length only.
// Bytes16, Bytes20 and Bytes32 return arrays, PutBytes32 writes to a buffer
// and FromBytes reads a slice of any length that fits.
// Big, PutBig, FromBig and CmpBig work with a big.Int.
// Value writes the decimal text, for NUMERIC, DECIMAL and text columns, and
// Scan reads an integer or text.
// NULL is an error for Scan: NullInt128, NullUint128, NullInt256 and
// NullUint256 take it, as sql.NullInt64 does
//
// # EVM
//
// The wrapping forms with AddMod, MulMod, ExpMod and ExtendSign follow the
// arithmetic of the EVM, but a division by zero panics where the EVM
// returns 0.
// A 256-bit word is the 32 big-endian bytes of Bytes32, an address the 20
// of Bytes20, and Hex writes the 0x form
package fat
