package testutil

import (
	"bytes"
	"math"
	"math/big"
	"math/rand"
	"strings"
	"testing"

	"github.com/avpetkun/fat/internal/require"
)

// RunBytes16 checks Bytes16 and PutBytes16 of the 128-bit types with big.Int,
// the word of a signed one in two's complement
func RunBytes16[T interface {
	Num[T]
	Bytes16() [16]byte
	PutBytes16(dst []byte)
}](t *testing.T, k Kind[T]) {
	t.Parallel()
	k.Values(func(x T) {
		t := On(t, k.Ref(x))
		word := RefBE(k.Ref(x), 16)
		b := x.Bytes16()
		require.Equal(t, string(word), string(b[:]))
		// PutBytes16 fills the start of dst and leaves the rest
		for _, spare := range []int{0, 1, 9} {
			dst := bytes.Repeat([]byte{'#'}, 16+spare)
			x.PutBytes16(dst)
			require.Equal(On(t, spare), string(word)+strings.Repeat("#", spare), string(dst))
		}
	})
	// one byte short
	require.Equal(t, "fat."+k.Name+".PutBytes16: short buffer", PanicValue(func() { k.Zero.PutBytes16(make([]byte, 15)) }))
}

// RunBytes checks the byte API with big.Int: ByteLen, Byte, ExtendSign,
// Bytes32, Bytes20, the Put and Append forms, Bytes and FromBytes.
// Bytes and FromBytes of a signed type use its two's complement word.
// FromBytes also reads a negative word with 0xFF bytes in front of it
func RunBytes[T Num[T]](t *testing.T, k Kind[T]) {
	t.Parallel()
	w := k.Bits / 8
	k.Values(func(x T) {
		xb := k.Ref(x)
		t := On(t, xb)
		word := RefBE(xb, 32)
		b32 := x.Bytes32()
		require.Equal(t, string(word), string(b32[:]))
		b20 := x.Bytes20()
		require.Equal(t, string(word[12:]), string(b20[:]))
		// the Put forms fill the start of dst and leave the rest
		for _, spare := range []int{0, 1, 9} {
			dst := bytes.Repeat([]byte{'#'}, 32+spare)
			x.PutBytes32(dst)
			require.Equal(On(t, spare), string(word)+strings.Repeat("#", spare), string(dst))
			dst = bytes.Repeat([]byte{'#'}, 20+spare)
			x.PutBytes20(dst)
			require.Equal(On(t, spare), string(word[12:])+strings.Repeat("#", spare), string(dst))
		}
		be := word[32-w:]
		for _, i := range []int{0, 1, 7, 8, w/2 - 1, w / 2, w - 2, w - 1, w, w + 1, 32, 1 << 20, math.MaxInt} {
			want := byte(0)
			if i < w {
				want = be[i]
			}
			require.Same(On(t, i), want, x.Byte(i))
		}
		// the word without the zero bytes in front
		want := bytes.TrimLeft(be, "\x00")
		require.Same(t, len(want), x.ByteLen())
		require.Equal(t, string(want), string(x.Bytes()))
		for _, spare := range []int{0, len(want), 40} {
			require.Append(On(t, spare), string(want), spare, x.AppendBytes)
		}
		// x reads back with zero bytes in front, a negative x with 0xFF ones
		ins := [][]byte{want, be, append(make([]byte, 9), be...)}
		if xb.Sign() < 0 {
			ins = append(ins, word, append(bytes.Repeat([]byte{0xFF}, 9), word...))
		}
		for _, b := range ins {
			v, ok := k.FromBytes(b)
			require.True(On(t, b), ok)
			require.Same(On(t, b), x, v)
		}
		for b := range w + 2 {
			require.Equal(On(t, b), extendRef(k, xb, b), x.ExtendSign(b))
		}
		require.Same(t, x, x.ExtendSign(math.MaxInt))
	})
	// every length up to past the width, filled at random
	r := rand.New(rand.NewSource(8))
	fills := []byte{0x00, 0xFF, 0x7F, 0x80, 0x01}
	for n := range w + 12 {
		for range 300 {
			b := make([]byte, n)
			r.Read(b)
			// a run of one byte in front, where the sign and the cut lie
			if n > 0 && r.Intn(2) == 0 {
				f := fills[r.Intn(len(fills))]
				for i := range r.Intn(n + 1) {
					b[i] = f
				}
			}
			checkFromBytes(t, k, b)
		}
	}
}

// extendRef is the big.Int reference of ExtendSign: the low b+1 bytes signed
func extendRef[T Num[T]](k Kind[T], x *big.Int, b int) T {
	if b >= k.Bits/8-1 {
		return k.FromBig(x)
	}
	v := new(big.Int).SetBytes(RefBE(x, b+1))
	if v.Bit(8*b+7) == 1 {
		v.Sub(v, new(big.Int).Lsh(Big1, uint(8*b+8)))
	}
	return k.FromBig(k.Wrap(v))
}

// checkFromBytes checks FromBytes and MustFromBytes on b with big.Int
func checkFromBytes[T Num[T]](t *testing.T, k Kind[T], b []byte) {
	// the word as unsigned, then the type's value of it
	u := new(big.Int).SetBytes(b)
	want := k.Wrap(u)
	fits := u.BitLen() <= k.Bits
	if k.Signed && len(b) > 0 && b[0] >= 0x80 {
		// b in two's complement: a negative word with its sign bytes in front
		s := BigSub(u, new(big.Int).Lsh(Big1, uint(8*len(b))))
		fits = fits || k.Fits(s)
	}
	t2 := On(t, b)
	v, ok := k.FromBytes(b)
	require.Same(t2, fits, ok)
	if !ok {
		require.Same(t2, k.Zero, v)
		require.Equal(t2, "fat.Must"+k.Name+"FromBytes: out of range", PanicValue(func() { k.MustFromBytes(b) }))
		return
	}
	require.Big(t2, want, k.Ref(v))
	require.Same(t2, v, k.MustFromBytes(b))
}
