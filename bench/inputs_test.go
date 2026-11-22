package bench

import (
	"math/big"
	"math/rand"
	"sync"
	"testing"

	"github.com/avpetkun/fat"
	"github.com/holiman/uint256"
	"lukechampine.com/uint128"

	"github.com/avpetkun/fat/internal/require"
	. "github.com/avpetkun/fat/internal/testutil"
)

// m indexes the 2^18 inputs of every set
const m = BenchMask

// The sinks take a result on each iteration so the work is kept.
// holiman and math/big write into theirs, the receivers of their calls
var (
	sU128, sU128b fat.Uint128
	lz, lm        uint128.Uint128
	sU64          uint64
	sI64          int64
	sU256, sU256b fat.Uint256
	sI256, sI256b fat.Int256
	sI128, sI128b fat.Int128
	hz, hm        uint256.Int
	bz, bm        = new(big.Int), new(big.Int)
	sBool         bool
	sInt          int
	sUint         uint
	sByte         byte
	sFloat        float64
	sString       string
	sBytes        []byte
	sB20          [20]byte
	sB32          [32]byte
	sErr          error
)

// agreeN is how many inputs of each set the TestAgree tests check
const agreeN = 1 << 14

// chainN is how many dependent sums one chain of the Chain benchmarks takes,
// on chainN+1 values from a window at a multiple of 8
const chainN = 5

// chainAt is the start of the window of chain i
func chainAt(i int) int { return (i & (m >> 3)) << 3 }

// chainBig adds the chainN+1 values of w as one chain on z, as math/big writes it
func chainBig(z *big.Int, w []*big.Int) {
	z.Add(w[0], w[1]).Add(z, w[2]).Add(z, w[3]).Add(z, w[4]).Add(z, w[5])
}

// agree runs f for the first agreeN inputs and requires every text it gives to match the first.
// The cases share the receivers of holiman, so they run one by one
func agree(t *testing.T, name string, f func(i int) []string) {
	t.Run(name, func(t *testing.T) {
		for i := range agreeN {
			r := f(i)
			for _, v := range r[1:] {
				require.Equal(On(t, i), r[0], v)
			}
		}
	})
}

// sDec, hDec and bDec print a result of each side as one text
func sDec(v fat.Uint256) string  { return v.String() }
func hDec(v *uint256.Int) string { return v.Dec() }
func bDec(v *big.Int) string     { return v.String() }

// u256 is one set of 2^18 unsigned values in the three forms
type u256 struct {
	s []fat.Uint256
	h []uint256.Int
	b []*big.Int
}

func newU256(l [][4]uint64) u256 {
	v := u256{s: Slice256[fat.Uint256](l), h: make([]uint256.Int, len(l)), b: make([]*big.Int, len(l))}
	for i, x := range l {
		v.h[i] = uint256.Int(x)
		v.b[i] = RefU256(v.s[i])
	}
	return v
}

// i256 is one set of 2^18 signed values, holiman holds them in two's complement
type i256 struct {
	s []fat.Int256
	h []uint256.Int
	b []*big.Int
}

func newI256(l [][4]uint64) i256 {
	v := i256{s: Slice256[fat.Int256](l), h: make([]uint256.Int, len(l)), b: make([]*big.Int, len(l))}
	for i, x := range l {
		v.h[i] = uint256.Int(x)
		v.b[i] = RefI256(v.s[i])
	}
	return v
}

// iDec and hsDec print a signed result, the one of holiman as two's complement
func iDec(v fat.Int256) string { return v.String() }
func hsDec(v *uint256.Int) string {
	return fat.Int256{L0: v[0], L1: v[1], L2: v[2], L3: v[3]}.String()
}

// u128 is one set of 2^18 unsigned 128-bit values in the three forms
type u128 struct {
	s []fat.Uint128
	l []uint128.Uint128
	b []*big.Int
}

func newU128(l [][2]uint64) u128 {
	v := u128{s: Slice128[fat.Uint128](l), l: make([]uint128.Uint128, len(l)), b: make([]*big.Int, len(l))}
	for i, x := range l {
		v.l[i] = uint128.New(x[0], x[1])
		v.b[i] = RefU128(v.s[i])
	}
	return v
}

// lDec prints a lukechampine value
func lDec(v uint128.Uint128) string { return v.String() }

// i128 is one set of 2^18 signed 128-bit values in the two forms:
// neither holiman nor lukechampine has a signed 128-bit type
type i128 struct {
	s []fat.Int128
	b []*big.Int
}

func newI128(l [][2]uint64) i128 {
	v := i128{s: Slice128[fat.Int128](l), b: make([]*big.Int, len(l))}
	for i := range l {
		v.b[i] = RefI128(v.s[i])
	}
	return v
}

// inputsU256 are the Uint256 sets, built once from fixed seeds
var inputsU256 = sync.OnceValue(func() (in struct {
	// random lengths and a modulus z, pairs whose sum, difference and product fit
	x, y, z, fx, fy u256
	// below 2^63 and 2^64, full width
	small, small2, full, full2 u256
	// bases and exponents with x^y in 255 bits, exponents of random length
	ex, ey, ye u256
	// x shifted so x * k and x * w fits, x for Sqrt
	x64, x128, sq u256
	// x shifted down 3 bits, so the chainN+1 values of a chain never wrap
	cx u256
	// a uint64 operand of random length, never zero, and one Sub64 takes from x with no borrow
	k, ks   []uint64
	kb, ksb []*big.Int
	// a Uint128 operand, not zero, and one Sub128 takes from x with no borrow
	w, ws   []fat.Uint128
	wb, wsb []*big.Int
	// shift counts in 0..255 and ones that keep x within 256 bits, bit indexes and values
	n, ln, bi []int
	nu, bv    []uint
	// byte indexes in 0..31, as holiman takes them too
	idx  []int
	hidx []uint256.Int
	// the text of x: decimal, 0x hex with no leading zero, JSON and gob of each side
	dec, hex                 []string
	decb, js, jsb, gob, gobb [][]byte
	// x big-endian with no zero byte in front
	be [][]byte
},
) {
	b := BenchIn()
	in.x, in.y = newU256(b.U256), newU256(b.V256)
	in.fx, in.fy = newU256(b.FU256), newU256(b.FV256)
	r := rand.New(rand.NewSource(11))
	var z, small, small2, full, full2, x64, x128, sq, ye, cx [][4]uint64
	for i, v := range in.x.s {
		z = append(z, RandBits(r, 1+r.Intn(256)))
		small = append(small, RandBits(r, 1+r.Intn(63)))
		small2 = append(small2, RandBits(r, 1+r.Intn(64)))
		full = append(full, RandBits(r, 256))
		full2 = append(full2, RandBits(r, 256))
		v = v.Rsh(64)
		x64 = append(x64, [4]uint64{v.L0, v.L1, v.L2, v.L3})
		v = in.x.s[i].Rsh(128)
		x128 = append(x128, [4]uint64{v.L0, v.L1, v.L2, v.L3})
		v = in.x.s[i]
		sq = append(sq, [4]uint64{v.L0, v.L1, v.L2, v.L3})
		v = in.y.s[i]
		ye = append(ye, [4]uint64{v.L0, v.L1, v.L2, v.L3})
		v = in.x.s[i].Rsh(3)
		cx = append(cx, [4]uint64{v.L0, v.L1, v.L2, v.L3})
		k := b.U64[i]
		in.k, in.kb = append(in.k, k), append(in.kb, new(big.Int).SetUint64(k))
		if in.x.s[i].IsUint64() && in.x.s[i].Uint64() < k {
			k = in.x.s[i].Uint64()
		}
		in.ks, in.ksb = append(in.ks, k), append(in.ksb, new(big.Int).SetUint64(k))
		w := Limbs128[fat.Uint128](b.U128[i])
		if w.IsZero() {
			w = fat.NewUint128(1)
		}
		in.w, in.wb = append(in.w, w), append(in.wb, RefU128(w))
		if in.x.s[i].LessThan(w.Uint256()) {
			w = in.x.s[i].Uint128()
		}
		in.ws, in.wsb = append(in.ws, w), append(in.wsb, RefU128(w))
		sh := r.Intn(256)
		in.n, in.nu = append(in.n, sh), append(in.nu, uint(sh))
		in.ln = append(in.ln, r.Intn(256-in.x.s[i].BitLen()+1))
		in.bi, in.bv = append(in.bi, r.Intn(256)), append(in.bv, uint(r.Intn(2)))
		in.idx = append(in.idx, r.Intn(32))
		in.hidx = append(in.hidx, *uint256.NewInt(uint64(in.idx[i])))
		in.dec = append(in.dec, in.x.s[i].String())
		in.hex = append(in.hex, in.x.s[i].Hex())
		in.decb = append(in.decb, []byte(in.x.s[i].String()))
		js, _ := in.x.s[i].MarshalJSON()
		jsb, _ := in.x.b[i].MarshalJSON()
		gob, _ := in.x.s[i].GobEncode()
		gobb, _ := in.x.b[i].GobEncode()
		in.js, in.jsb, in.gob, in.gobb = append(in.js, js), append(in.jsb, jsb), append(in.gob, gob), append(in.gobb, gobb)
		in.be = append(in.be, in.x.s[i].Bytes())
	}
	in.z, in.small, in.x64, in.sq, in.ye, in.cx = newU256(z), newU256(small), newU256(x64), newU256(sq), newU256(ye), newU256(cx)
	in.small2, in.full = newU256(small2), newU256(full)
	in.full2 = newU256(full2)
	in.x128 = newU256(x128)
	l, e := ExpIn(256, false)
	ey := make([][4]uint64, len(e))
	for i, v := range e {
		ey[i][0] = v
	}
	in.ex, in.ey = newU256(l), newU256(ey)
	return
})

// inputsI256 are the Int256 sets, built once from fixed seeds
var inputsI256 = sync.OnceValue(func() (in struct {
	// random lengths and signs, never MinInt256 and a modulus z, pairs whose sum, difference and product fit
	x, y, z, fx, fy i256
	// below 2^63 in magnitude
	small i256
	// bases and exponents with x^y in 255 bits, exponents of random length as |y|
	ex, ey, ye i256
	// x shifted so x * k and x * w fits, x as |x| for Sqrt
	x64, x128, sq i256
	// a int64 operand, not zero, and one Sub64 takes from x with no borrow
	k, ks   []int64
	kb, ksb []*big.Int
	// a Int128 operand, not zero, and one Sub128 takes from x with no borrow
	w, ws   []fat.Int128
	wb, wsb []*big.Int
	// shift counts in 0..255 and ones that keep x within 255 bits, bit indexes and values
	n, ln, bi []int
	nu, bv    []uint
	// the text of x: decimal, 0x hex with no leading zero, JSON and gob of each side
	dec, hex                 []string
	decb, js, jsb, gob, gobb [][]byte
},
) {
	b := BenchIn()
	in.x, in.y = newI256(b.I256), newI256(b.J256)
	in.fx, in.fy = newI256(b.FI256), newI256(b.FJ256)
	r := rand.New(rand.NewSource(12))
	var z, small, x64, x128, sq, ye [][4]uint64
	for i, v := range in.x.s {
		z = append(z, RandSigned256(r, 1+r.Intn(255)))
		small = append(small, RandSigned256(r, 1+r.Intn(63)))
		v = v.Rsh(65)
		x64 = append(x64, [4]uint64{v.L0, v.L1, v.L2, v.L3})
		v = in.x.s[i].Rsh(129)
		x128 = append(x128, [4]uint64{v.L0, v.L1, v.L2, v.L3})
		v = in.x.s[i].Abs()
		sq = append(sq, [4]uint64{v.L0, v.L1, v.L2, v.L3})
		v = in.y.s[i].Abs()
		ye = append(ye, [4]uint64{v.L0, v.L1, v.L2, v.L3})
		k := b.I64[i]
		if k == 0 {
			k = 1
		}
		in.k, in.kb = append(in.k, k), append(in.kb, big.NewInt(k))
		in.ks, in.ksb = append(in.ks, k), append(in.ksb, big.NewInt(k))
		w := Limbs128[fat.Int128](b.I128[i])
		if w.IsZero() {
			w = fat.NewInt128(1)
		}
		in.w, in.wb = append(in.w, w), append(in.wb, RefI128(w))
		in.ws, in.wsb = append(in.ws, w), append(in.wsb, RefI128(w))
		sh := r.Intn(256)
		in.n, in.nu = append(in.n, sh), append(in.nu, uint(sh))
		in.ln = append(in.ln, r.Intn(255-in.x.s[i].BitLen()+1))
		in.bi, in.bv = append(in.bi, r.Intn(255)), append(in.bv, uint(r.Intn(2)))
		in.dec = append(in.dec, in.x.s[i].String())
		in.hex = append(in.hex, in.x.s[i].Hex())
		in.decb = append(in.decb, []byte(in.x.s[i].String()))
		js, _ := in.x.s[i].MarshalJSON()
		jsb, _ := in.x.b[i].MarshalJSON()
		gob, _ := in.x.s[i].GobEncode()
		gobb, _ := in.x.b[i].GobEncode()
		in.js, in.jsb, in.gob, in.gobb = append(in.js, js), append(in.jsb, jsb), append(in.gob, gob), append(in.gobb, gobb)
	}
	in.z, in.small, in.x64, in.sq, in.ye = newI256(z), newI256(small), newI256(x64), newI256(sq), newI256(ye)
	in.x128 = newI256(x128)
	l, e := ExpIn(256, true)
	ey := make([][4]uint64, len(e))
	for i, v := range e {
		ey[i][0] = v
	}
	in.ex, in.ey = newI256(l), newI256(ey)
	return
})

// inputsU128 are the Uint128 sets, built once from fixed seeds
var inputsU128 = sync.OnceValue(func() (in struct {
	// random lengths and a modulus z, pairs whose sum, difference and product fit
	x, y, z, fx, fy u128
	// below 2^63 and 2^64, full width
	small, small2, full u128
	// bases and exponents with x^y in 127 bits, exponents of random length
	ex, ey, ye u128
	// x shifted so x * k fits, x for Sqrt
	x64, sq u128
	// x shifted down 3 bits, so the chainN+1 values of a chain never wrap
	cx u128
	// a uint64 operand of random length, never zero, and one Sub64 takes from x with no borrow
	k, ks   []uint64
	kb, ksb []*big.Int
	// shift counts in 0..127 and ones that keep x within 128 bits, bit indexes and values
	n, ln, bi []int
	nu, bv    []uint
	// the text of x: decimal, 0x hex with no leading zero, JSON and gob of each side
	dec, hex                 []string
	decb, js, jsb, gob, gobb [][]byte
	// x big-endian with no zero byte in front, and in 16 bytes
	be, be16 [][]byte
},
) {
	b := BenchIn()
	in.x, in.y = newU128(b.U128), newU128(b.V128)
	in.fx, in.fy = newU128(b.FU128), newU128(b.FV128)
	r := rand.New(rand.NewSource(13))
	low := func(l [4]uint64) [2]uint64 { return [2]uint64{l[0], l[1]} }
	var z, small, small2, full, x64, sq, ye, cx [][2]uint64
	for i, v := range in.x.s {
		z = append(z, low(RandBits(r, 1+r.Intn(128))))
		small = append(small, low(RandBits(r, 1+r.Intn(63))))
		small2 = append(small2, low(RandBits(r, 1+r.Intn(64))))
		full = append(full, low(RandBits(r, 128)))
		v = v.Rsh(64)
		x64 = append(x64, [2]uint64{v.L0, v.L1})
		v = in.x.s[i]
		sq = append(sq, [2]uint64{v.L0, v.L1})
		v = in.y.s[i]
		ye = append(ye, [2]uint64{v.L0, v.L1})
		v = in.x.s[i].Rsh(3)
		cx = append(cx, [2]uint64{v.L0, v.L1})
		k := b.U64[i]
		in.k, in.kb = append(in.k, k), append(in.kb, new(big.Int).SetUint64(k))
		if in.x.s[i].IsUint64() && in.x.s[i].Uint64() < k {
			k = in.x.s[i].Uint64()
		}
		in.ks, in.ksb = append(in.ks, k), append(in.ksb, new(big.Int).SetUint64(k))
		sh := r.Intn(128)
		in.n, in.nu = append(in.n, sh), append(in.nu, uint(sh))
		in.ln = append(in.ln, r.Intn(128-in.x.s[i].BitLen()+1))
		in.bi, in.bv = append(in.bi, r.Intn(128)), append(in.bv, uint(r.Intn(2)))
		in.dec = append(in.dec, in.x.s[i].String())
		in.hex = append(in.hex, in.x.s[i].Hex())
		in.decb = append(in.decb, []byte(in.x.s[i].String()))
		js, _ := in.x.s[i].MarshalJSON()
		jsb, _ := in.x.b[i].MarshalJSON()
		gob, _ := in.x.s[i].GobEncode()
		gobb, _ := in.x.b[i].GobEncode()
		in.js, in.jsb, in.gob, in.gobb = append(in.js, js), append(in.jsb, jsb), append(in.gob, gob), append(in.gobb, gobb)
		in.be = append(in.be, in.x.s[i].Bytes())
		bin, _ := in.x.s[i].MarshalBinary()
		in.be16 = append(in.be16, bin)
	}
	in.z, in.small, in.x64, in.sq, in.ye, in.cx = newU128(z), newU128(small), newU128(x64), newU128(sq), newU128(ye), newU128(cx)
	in.small2, in.full = newU128(small2), newU128(full)
	l, e := ExpIn(128, false)
	ey := make([][2]uint64, len(e))
	for i, v := range e {
		ey[i][0] = v
	}
	in.ex, in.ey = newU128(Low128(l)), newU128(ey)
	return
})

// inputsI128 are the Int128 sets, built once from fixed seeds
var inputsI128 = sync.OnceValue(func() (in struct {
	// random lengths and signs, never MinInt128 and a modulus z, pairs whose sum, difference and product fit
	x, y, z, fx, fy i128
	// below 2^63 in magnitude
	small i128
	// bases and exponents with x^y in 127 bits, exponents of random length as |y|
	ex, ey, ye i128
	// x shifted so x * k fits, x as |x| for Sqrt
	x64, sq i128
	// a int64 operand, not zero, and one Sub64 takes from x with no borrow
	k, ks   []int64
	kb, ksb []*big.Int
	// shift counts in 0..127 and ones that keep x within 127 bits, bit indexes and values
	n, ln, bi []int
	nu, bv    []uint
	// the text of x: decimal, 0x hex with no leading zero, JSON and gob of each side
	dec, hex                 []string
	decb, js, jsb, gob, gobb [][]byte
},
) {
	b := BenchIn()
	in.x, in.y = newI128(b.I128), newI128(b.J128)
	in.fx, in.fy = newI128(b.FI128), newI128(b.FJ128)
	r := rand.New(rand.NewSource(14))
	low := func(l [4]uint64) [2]uint64 { return [2]uint64{l[0], l[1]} }
	var z, small, x64, sq, ye [][2]uint64
	for i, v := range in.x.s {
		z = append(z, low(RandSigned256(r, 1+r.Intn(127))))
		small = append(small, low(RandSigned256(r, 1+r.Intn(63))))
		v = v.Rsh(65)
		x64 = append(x64, [2]uint64{v.L0, v.L1})
		v = in.x.s[i].Abs()
		sq = append(sq, [2]uint64{v.L0, v.L1})
		v = in.y.s[i].Abs()
		ye = append(ye, [2]uint64{v.L0, v.L1})
		k := b.I64[i]
		if k == 0 {
			k = 1
		}
		in.k, in.kb = append(in.k, k), append(in.kb, big.NewInt(k))
		in.ks, in.ksb = append(in.ks, k), append(in.ksb, big.NewInt(k))
		sh := r.Intn(128)
		in.n, in.nu = append(in.n, sh), append(in.nu, uint(sh))
		in.ln = append(in.ln, r.Intn(127-in.x.s[i].BitLen()+1))
		in.bi, in.bv = append(in.bi, r.Intn(127)), append(in.bv, uint(r.Intn(2)))
		in.dec = append(in.dec, in.x.s[i].String())
		in.hex = append(in.hex, in.x.s[i].Hex())
		in.decb = append(in.decb, []byte(in.x.s[i].String()))
		js, _ := in.x.s[i].MarshalJSON()
		jsb, _ := in.x.b[i].MarshalJSON()
		gob, _ := in.x.s[i].GobEncode()
		gobb, _ := in.x.b[i].GobEncode()
		in.js, in.jsb, in.gob, in.gobb = append(in.js, js), append(in.jsb, jsb), append(in.gob, gob), append(in.gobb, gobb)
	}
	in.z, in.small, in.x64, in.sq, in.ye = newI128(z), newI128(small), newI128(x64), newI128(sq), newI128(ye)
	l, e := ExpIn(128, true)
	ey := make([][2]uint64, len(e))
	for i, v := range e {
		ey[i][0] = v
	}
	in.ex, in.ey = newI128(Low128(l)), newI128(ey)
	return
})
