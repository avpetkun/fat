package testutil

import (
	"database/sql/driver"
	"math/big"
	"testing"
)

// Num is the methods of the four types the shared tests use
type Num[T any] interface {
	comparable
	IsZero() bool
	BitLen() int
	Bit(i int) uint
	WithBit(i int, bit uint) T
	Lsh(n int) T
	Rsh(n int) T
	Not() T
	Cmp(y T) int
	LessThan(y T) bool
	LessOrEqual(y T) bool
	GreaterThan(y T) bool
	GreaterOrEqual(y T) bool
	Max(y T) T
	Min(y T) T
	CmpBig(y *big.Int) int
	String() string
	Hex() string
	Dec() string
	AppendDec(dst []byte) []byte
	AppendHex(dst []byte) []byte
	PrettyDec(sep byte) string
	AppendText(b []byte) ([]byte, error)
	MarshalText() ([]byte, error)
	Value() (driver.Value, error)
	Exp(y T) T
	Expo(y T) (T, bool)
	Expu(y T) T
	ExpMod(y, m T) T
	Sqrt() T
	Log10() int
	ByteLen() int
	Byte(i int) byte
	ExtendSign(b int) T
	Bytes32() [32]byte
	Bytes20() [20]byte
	PutBytes32(dst []byte)
	PutBytes20(dst []byte)
	Bytes() []byte
	AppendBytes(dst []byte) []byte
	Float64() float64
	DivToFloat64(y T) float64
	DivToFloat64x(y T) (float64, bool)
}

// NumPtr is the pointer side of Num, the unmarshalers and Scan
type NumPtr[T any] interface {
	*T
	UnmarshalText(text []byte) error
	UnmarshalJSON(text []byte) error
	UnmarshalBinary(b []byte) error
	GobDecode(b []byte) error
	Scan(src any) error
}

// SignNum is Num with Sign
type SignNum[T any] interface {
	Num[T]
	Sign() int
}

// SignedNum adds the methods only Int128 and Int256 have
type SignedNum[T any] interface {
	SignNum[T]
	IsNegative() bool
	IsPositive() bool
	Neg() T
	Abs() T
}

// Kind describes one of the four types for the shared tests
type Kind[T Num[T]] struct {
	Name    string
	Bits    int
	Signed  bool
	Ref     func(T) *big.Int
	Wrap    func(*big.Int) *big.Int
	Fits    func(*big.Int) bool
	FromBig func(*big.Int) T
	// FromBytes reads big-endian bytes, two's complement for a signed type
	FromBytes     func([]byte) (T, bool)
	MustFromBytes func([]byte) T
	// FromFloat64 truncates toward zero
	FromFloat64     func(float64) (T, bool)
	MustFromFloat64 func(float64) T
	// Small makes a small non-negative v
	Small       func(v uint64) T
	FromDec     func(s string) (T, error)
	FromHex     func(s string) (T, error)
	MustFromDec func(s string) T
	MustFromHex func(s string) T
	Dirty       T // every limb 7, unmarshalers must overwrite it all
	Zero        T
	Max, Min    T
	Values      func(func(T))
	Pairs       func(func(x, y T))
	// Narrow cuts y to width bits and extends it back, by zeros or by its sign
	Narrow func(y T, width int) T
	Shifts []int
	BitIdx []int
	// CmpMore checks the comparisons only this type has, nil for none
	CmpMore func(t testing.TB, x, y T, xb *big.Int)
	// Ops64 are the methods with a 64-bit operand
	Ops64 Ops64[T]
}

// Digits is the number of decimal digits of Max
func (k Kind[T]) Digits() int { return len(k.Ref(k.Max).String()) }

// Over is Max + 1 as a big.Int, the least positive value out of range
func (k Kind[T]) Over() *big.Int {
	return BigAdd(k.Ref(k.Max), Big1)
}

// Ops64 are the methods with a 64-bit operand, passed as its uint64 bits
type Ops64[T any] struct {
	New              func(v uint64) T
	Cmp              func(x T, v uint64) int
	Add, Sub, Mul    func(x T, v uint64) T
	Addo, Subo, Mulo func(x T, v uint64) (T, bool)
	Addu, Subu, Mulu func(x T, v uint64) T
}

// Signed64 adapts a method with an int64 operand to the uint64 one of Ops64
func Signed64[X, R any](f func(X, int64) R) func(X, uint64) R {
	return func(x X, v uint64) R { return f(x, int64(v)) }
}

// Signed64Pair is Signed64 for a method with two results
func Signed64Pair[X, R1, R2 any](f func(X, int64) (R1, R2)) func(X, uint64) (R1, R2) {
	return func(x X, v uint64) (R1, R2) { return f(x, int64(v)) }
}
