# Fixed-size Arithmetic Types

128-bit and 256-bit integers for Go: `Int128`, `Uint128`, `Int256` and `Uint256`.

- All four types are values: assignment copies them, `==` compares them and they work as map keys.
- Methods take and return values, and the arithmetic does not allocate.
- Overflow is never silent unless asked for: every operation that can overflow has a checked form.
- The full set of a numeric type: arithmetic, division, modular arithmetic, powers, roots, GCD, bits, text, JSON, binary and SQL.
- Calls chain into one expression: `x.Mul(y).Add(z).Div64(100)`, with no result variable to prepare.
- Fits the EVM: the wrapping forms with `AddMod`, `MulMod`, `ExpMod` and `ExtendSign` for its arithmetic, 256-bit words as 32 big-endian bytes, `Bytes20` for addresses and `0x` hex.

```sh
go get github.com/avpetkun/fat
```

Needs Go 1.24 or later.

## Usage

```go
import "github.com/avpetkun/fat"

// Sell 1.5 ETH for USDC and take a 0.3% fee
func main() {
    amount := fat.MustUint256FromDec("1500000000000000000") // 1.5 ETH in wei
    price := fat.NewUint256(2_490_637_625)                  // USDC for one ETH
    oneETH := fat.NewUint256(1_000_000_000_000_000_000)

    usdc := amount.MulDiv(price, oneETH) // 3735956437
    fee := usdc.Mul64(3).Div64(1000)     // 11207869

    fmt.Println(usdc.Sub(fee).PrettyDec('_')) // 3_724_748_568
}
```

### Overflow

An operation comes in up to three forms, told apart by the suffix:

| form      | on overflow                         | as                 |
| --------- | ----------------------------------- | ------------------ |
| x.Add(y)  | panics                              | checked arithmetic |
| x.Addo(y) | returns the wrapped result and true | bits.Add64         |
| x.Addu(y) | wraps around                        | a Go integer       |

The same goes for `Sub`, `Mul`, `Exp`, `MulDiv`, for `Div` on the signed types (`MinInt128 / -1`) and for the conversions between the types: `Uint64`, `Uint64o`, `Uint64u`. `Mule` returns the full product of twice the width. A division by zero panics in every form, as in Go.

### Narrow operands

A `64` in the name takes a `uint64` on the unsigned types and an `int64` on the signed ones: `Add64`, `Mul64`, `DivMod64`, `Cmp64`. The 256-bit types take the 128-bit type of their sign by a `128` the same way: `Add128`, `Mul128`, `DivMod128`.

### Products that do not fit

`MulDiv`, `MulDivCeil` and `MulDivMod` divide the exact product, so only the quotient has to fit:

```go
amount := fat.MustUint128FromDec("250000000000000000000000000001")
price := fat.MustUint128FromDec("1234567890123456789")
scale := fat.MustUint128FromDec("1e18")

amount.MulDiv(price, scale)     // 308641972530864197250000000001
amount.MulDivCeil(price, scale) // 308641972530864197250000000002
```

`AddMod`, `SubMod`, `MulMod` and `ExpMod` reduce the exact result the same way, and `ModInverse` and `GCD` complete the set. The time of an operation depends on its values, so the package does not suit secret data.

### Signed types

`Div` truncates toward zero and `Mod` takes the sign of `x`, as `/` and `%` of Go; `Rsh` rounds toward negative infinity, as `>>` does. `Abs` and `Neg` panic for the minimum, and `UnsignedAbs` returns the unsigned type, which holds it.

### Text and JSON

```go
x, err := fat.Uint256FromDec("5e18") // decimal, m e n is m * 10^n
y, err := fat.Int128FromHex("-0xff") // 0x is optional
z := fat.MustUint128FromDec("42")    // panics on a bad string

x.Dec()          // "5000000000000000000", as String
x.Hex()          // "0x4563918244f40000"
x.PrettyDec('_') // "5_000_000_000_000_000_000"

buf = x.AppendDec(buf) // no allocation

fmt.Printf("%d %x %#X", x, x, x)
// >> 5000000000000000000 4563918244f40000 0X4563918244F40000
```

The parsers take a `string` or a `[]byte` and return `ErrParse`, or `ErrNegative` for a negative value given to an unsigned type.

JSON holds a value as a quoted decimal string, since a parser on `float64` loses the digits of a number past 2^53. It reads a string, decimal or hex after `0x`, or a plain number.

### Binary, math/big and SQL

- `MarshalBinary` writes 16 or 32 bytes big-endian, two's complement on the signed types. `Bytes16`, `Bytes20` and `Bytes32` return arrays, `PutBytes32` writes to a buffer and `FromBytes` reads a slice of any length that fits.
- `Big`, `PutBig`, `FromBig` and `CmpBig` work with a `*big.Int`.
- `Value` and `Scan` work with `database/sql`: the value is decimal text, for `NUMERIC`, `DECIMAL` and text columns. `NullUint128`, `NullInt128`, `NullUint256` and `NullInt256` take `NULL`, as `sql.NullInt64` does.

## Performance

[bench](bench) is a module of its own that compares fat with [holiman/uint256](https://github.com/holiman/uint256), [lukechampine.com/uint128](https://github.com/lukechampine/uint128) and `math/big` on the same 2^18 values per case, after a test that every side returns the same results.

The results below are from an Apple M4 Pro with Go 1.27.4.

`Uint256` and `Int256` against `math/big` and holiman/uint256:

| operation         | fat, ns | math/big, ns | big / fat | holiman, ns | holiman / fat |
| ----------------- | ------- | ------------ | --------- | ----------- | ------------- |
| Add               | 0.851   | 11.7         | 13.7x     | 1.08        | 1.27x         |
| Mul               | 1.88    | 15.6         | 8.29x     | 2.06        | 1.10x         |
| Div               | 10      | 55           | 5.48x     | 15.7        | 1.57x         |
| Div (64-bit)      | 14.1    | 93.1         | 6.59x     | 28.8        | 2.04x         |
| Mod               | 10.1    | 48.7         | 4.83x     | 15.8        | 1.57x         |
| AddMod            | 12.9    | 82.7         | 6.39x     | 23.8        | 1.84x         |
| MulMod            | 24.9    | 130          | 5.21x     | 51.5        | 2.07x         |
| MulDiv            | 25      | 129          | 5.18x     | 46.1        | 1.84x         |
| Exp               | 13.4    | 43.8         | 3.28x     | 20.9        | 1.56x         |
| Sqrt              | 10.5    | 522          | 50.0x     | 149         | 14.3x         |
| Cmp               | 0.976   | 8.82         | 9.03x     | 4.69        | 4.81x         |
| Rsh               | 1.64    | 12.6         | 7.68x     | 4.78        | 2.91x         |
| And               | 0.864   | 11.2         | 12.9x     | 1.08        | 1.25x         |
| BitLen            | 0.861   | 1.27         | 1.47x     | 4.12        | 4.78x         |
| Float64           | 2.77    | 44.6         | 16.1x     | 9.2         | 3.32x         |
| String            | 33.7    | 130          | 3.85x     | 92.6        | 2.75x         |
| Hex               | 22.5    | 56.6         | 2.52x     | 27.7        | 1.23x         |
| FromDec           | 13.3    | 222          | 16.7x     | 57          | 4.28x         |
| UnmarshalJSON     | 13.7    | 226          | 16.5x     | 69.8        | 5.09x         |
| Abs (Int256)      | 0.802   | 5.68         | 7.08x     | 3.63        | 4.52x         |
| Div (Int256)      | 11.7    | 54.4         | 4.67x     | 22.4        | 1.92x         |
| Mod (Int256)      | 11.7    | 49.7         | 4.25x     | 23          | 1.97x         |
| LessThan (Int256) | 0.953   | 10.4         | 10.9x     | 4.26        | 4.47x         |
| Rsh (Int256)      | 1.65    | 19.6         | 11.9x     | 7.64        | 4.63x         |

`Uint128` against `math/big` and lukechampine/uint128:

| operation     | fat, ns | math/big, ns | big / fat | lukechampine, ns | lukechampine / fat |
| ------------- | ------- | ------------ | --------- | ---------------- | ------------------ |
| Add           | 0.619   | 9.4          | 15.2x     | 0.631            | 1.02x              |
| Mul           | 1.04    | 11.6         | 11.1x     | 4.08             | 3.91x              |
| Div           | 4.32    | 35.5         | 8.20x     | 11.4             | 2.64x              |
| Mod           | 4.56    | 32.4         | 7.09x     | 11.4             | 2.50x              |
| DivMod        | 4.77    | 27           | 5.66x     | 11.4             | 2.39x              |
| Div64         | 2.94    | 27.7         | 9.40x     | 6.8              | 2.31x              |
| Cmp           | 0.694   | 11.5         | 16.6x     | 4.66             | 6.72x              |
| Cmp64         | 0.673   | 12.1         | 18.0x     | 4.46             | 6.63x              |
| Rsh           | 0.945   | 11.9         | 12.6x     | 2.98             | 3.15x              |
| And           | 0.585   | 8.68         | 14.9x     | 0.588            | 1.01x              |
| BitLen        | 0.564   | 1.25         | 2.21x     | 2.91             | 5.15x              |
| String        | 21.6    | 90.8         | 4.21x     | 42.6             | 1.97x              |
| FromDec       | 9.56    | 124          | 13.0x     | 326              | 34.1x              |
| UnmarshalText | 10.4    | 126          | 12.1x     | 326              | 31.4x              |

To run full benchmark:

```sh
go test -C bench -run '^$' -bench . -benchtime 200ms -count 6 | benchstat -col /impl -
```

A case runs over 2^18 values that do not repeat and vary in length.

## Design

- **Structs of `uint64` limbs, no arrays.** The compiler keeps the limbs in registers and inlines.
- **No branch where the data decides.** `Min`, `Max`, absolute values and the rounding pick by conditional moves or masks.
- **Short values stay fast.** A 128-bit value is often near 2^64, and the operations do not pay for the full width.
- **Tuned per architecture.** The code is optimized for amd64, Graviton and Apple cores, with separate code paths where they need different ones.
- **Algorithms before micro-tuning.** `GCD` is Pornin's optimized binary GCD, and `ModInverse` runs the same rounds under Montgomery's reduction.
- **Integers only.** No `float64` estimate decides a carry or an overflow, and no large lookup table competes for the cache.
- **Checked with `math/big`.** The tests compare every operation with `math/big` and cover 100% of the code.

## Tests

Each arch builds its own variant by default, and a build tag picks one outright on any machine:

```sh
go test ./...                       # the variant of this arch
go test -tags arch_generic .        # the generic code
go test -tags arch_amd64 .          # the amd64 code, runs on any arch
```

One run covers only its own variant, so 100% coverage takes both runs merged:

```sh
rm -rf cover && mkdir cover
go test -tags arch_generic -cover . -args -test.gocoverdir=$PWD/cover
go test -tags arch_amd64 -cover . -args -test.gocoverdir=$PWD/cover
go tool covdata percent -i=cover
```
