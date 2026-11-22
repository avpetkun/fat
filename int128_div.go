package fat

import (
	"math"
	"math/bits"

	"github.com/avpetkun/fat/internal/arch"
)

// int128DivPanics and int128DivModPanics are the panics by kernel status.
// One panic from a table keeps Div and DivMod within the inline budget
var (
	int128DivPanics = [3]string{
		1: "fat.Int128.Div: overflow",
		2: "fat.Int128.Div: division by zero",
	}
	int128DivModPanics = [3]string{
		1: "fat.Int128.DivMod: overflow",
		2: "fat.Int128.DivMod: division by zero",
	}
)

// Div returns x / y truncated toward zero.
// It panics for y == 0 and on overflow at MinInt128 / -1
func (x Int128) Div(y Int128) Int128 {
	q, st := x.divo(y)
	if st != 0 {
		panic(int128DivPanics[st])
	}
	return q
}

// Divo returns Divu and whether x / y overflowed, it panics for y == 0
func (x Int128) Divo(y Int128) (q Int128, overflow bool) {
	q, st := x.divo(y)
	if st > 1 {
		panic("fat.Int128.Divo: division by zero")
	}
	return q, st != 0
}

// Divu returns x / y wrapped around as on a Go integer, it panics for y == 0
func (x Int128) Divu(y Int128) Int128 {
	if y.L0|y.L1 == 0 {
		panic("fat.Int128.Divu: division by zero")
	}
	q, _ := x.divo(y)
	return q
}

// Mod returns x % y with the sign of x, it panics for y == 0
func (x Int128) Mod(y Int128) Int128 {
	if y.L0|y.L1 == 0 {
		panic("fat.Int128.Mod: division by zero")
	}
	return x.mod(y)
}

// DivMod returns Div and Mod.
// It panics for y == 0 and on overflow at MinInt128 / -1
func (x Int128) DivMod(y Int128) (q, r Int128) {
	var st uint
	if q, r, st = x.divMod(y); st != 0 {
		panic(int128DivModPanics[st])
	}
	return
}

// divMod is DivMod with a status for its panics: 1 on overflow, 2 for y == 0.
// The quotient wraps on overflow, q and r are not defined for y == 0.
// It is Uint128.divMod on |x| and |y| written out, see the comments there.
// It calls nothing and so needs no stack frame
func (x Int128) divMod(y Int128) (q, r Int128, st uint) {
	if arch.IsAmd64 {
		// a hardware 128-by-64 division per quotient limb there
		if y.L0|y.L1 == 0 {
			return q, r, 2
		}
		// |x| and |y| by (v ^ m) - m with the sign masks
		mx, my := uint64(int64(x.L1)>>63), uint64(int64(y.L1)>>63)
		var u, w Uint128
		var c uint64
		u.L0, c = bits.Sub64(x.L0^mx, mx, 0)
		u.L1, _ = bits.Sub64(x.L1^mx, mx, c)
		w.L0, c = bits.Sub64(y.L0^my, my, 0)
		w.L1, _ = bits.Sub64(y.L1^my, my, c)
		var q0, q1, r0, r1 uint64
		if w.L1 == 0 {
			// the top limb is divided only when not below the divisor
			t := u.L1
			if t >= w.L0 {
				q1 = t / w.L0
				t -= q1 * w.L0
			}
			q0, r0 = bits.Div64(t, u.L0, w.L0)
		} else {
			n := uint(bits.LeadingZeros64(w.L1))
			d := w.L1<<(n&63) | w.L0>>1>>(^n&63)
			k, _ := bits.Div64(u.L1>>1, u.L1<<63|u.L0>>1, d)
			k >>= ^n & 63
			if k != 0 {
				k--
			}
			hi, lo := bits.Mul64(w.L0, k)
			var b uint64
			r0, b = bits.Sub64(u.L0, lo, 0)
			r1, _ = bits.Sub64(u.L1, hi+w.L1*k, b)
			t0, b := bits.Sub64(r0, w.L0, 0)
			t1, b := bits.Sub64(r1, w.L1, b)
			if b == 0 {
				r0 = t0
			}
			if b == 0 {
				r1 = t1
			}
			q0 = k + 1 - b
		}
		// q is negative when the signs differ, r takes the sign of x
		mq := mx ^ my
		q.L0, c = bits.Sub64(q0^mq, mq, 0)
		q.L1, _ = bits.Sub64(q1^mq, mq, c)
		r.L0, c = bits.Sub64(r0^mx, mx, 0)
		r.L1, _ = bits.Sub64(r1^mx, mx, c)
		// |q| = 2^127 fits only as a negative quotient
		st = uint(q1 &^ mq >> 63)
		return
	}
	// |x| and |y| by (v ^ m) - m with the sign masks.
	// Two UnsignedAbs inlined here would leave a branch on the sign of y
	mx, my := uint64(int64(x.L1)>>63), uint64(int64(y.L1)>>63)
	var u, w Uint128
	var c uint64
	u.L0, c = bits.Sub64(x.L0^mx, mx, 0)
	u.L1, _ = bits.Sub64(x.L1^mx, mx, c)
	w.L0, c = bits.Sub64(y.L0^my, my, 0)
	w.L1, _ = bits.Sub64(y.L1^my, my, c)
	var q0, q1, r0, r1 uint64
	um, wm := u.L1, w.L1
	if um > 1<<33-1 {
		um = 1<<33 - 1
	}
	if wm == 0 {
		wm = 1
	}
	if um < wm {
		n, d := u.L1>>1, w.L1>>1+1
		z := u.L1 | w.L1
		if z == 0 {
			n = u.L0
		}
		if z == 0 {
			d = w.L0
		}
		// d == 0 only for y == 0, any quotient does then
		k := n
		if d != 0 {
			k = n / d
		}
		hi, lo := bits.Mul64(k, w.L0)
		hi += k * w.L1
		var b uint64
		r0, b = bits.Sub64(u.L0, lo, 0)
		r1, _ = bits.Sub64(u.L1, hi, b)
		t0, b := bits.Sub64(r0, w.L0, 0)
		t1, b := bits.Sub64(r1, w.L1, b)
		if b == 0 {
			r0 = t0
		}
		if b == 0 {
			r1 = t1
		}
		if b == 0 {
			k++
		}
		q0 = k
	} else if w.L1 == 0 && w.L0 != 0 {
		w := w.L0
		q1 = u.L1 / w
		t := u.L1 - q1*w
		s := uint(bits.LeadingZeros64(w))
		d := w << (s & 63)
		v := (1<<47 - 1) / (d>>40 + 1)
		d0 := d & 1
		e := (v>>1)&-d0 - v*(d>>1+d0)
		hi, _ := bits.Mul64(e, e)
		e += hi >> 22
		hi, _ = bits.Mul64(v<<19, e)
		v = v<<41 + hi
		hi, lo := bits.Mul64(v, d)
		_, c := bits.Add64(lo, d, 0)
		hi, _ = bits.Add64(hi, d, c)
		v -= hi
		q0, r0 = div2by1(t<<(s&63)|u.L0>>1>>(^s&63), u.L0<<(s&63), d, v)
		r0 >>= s & 63
	} else {
		n := uint(bits.LeadingZeros64(w.L1))
		d := w.L1<<(n&63) | w.L0>>1>>(^n&63)
		v := (1<<47 - 1) / (d>>40 + 1)
		d0 := d & 1
		e := (v>>1)&-d0 - v*(d>>1+d0)
		hi, _ := bits.Mul64(e, e)
		e += hi >> 22
		hi, _ = bits.Mul64(v<<19, e)
		v = v<<41 + hi
		hi, lo := bits.Mul64(v, d)
		_, c := bits.Add64(lo, d, 0)
		hi, _ = bits.Add64(hi, d, c)
		v -= hi
		k, _ := div2by1(u.L1>>1, u.L1<<63|u.L0>>1, d, v)
		k >>= ^n & 63
		if k != 0 {
			k--
		}
		hi, lo = bits.Mul64(w.L0, k)
		t := u.Subu(Uint128{lo, hi + w.L1*k})
		if !t.LessThan(w) {
			k++
			t = t.Subu(w)
		}
		q0, r0, r1 = k, t.L0, t.L1
	}
	// q is negative when the signs differ, r takes the sign of x
	mq := mx ^ my
	q.L0, c = bits.Sub64(q0^mq, mq, 0)
	q.L1, _ = bits.Sub64(q1^mq, mq, c)
	r.L0, c = bits.Sub64(r0^mx, mx, 0)
	r.L1, _ = bits.Sub64(r1^mx, mx, c)
	// |q| = 2^127 fits only as a negative quotient
	st = uint(q1 &^ mq >> 63)
	if y.L0|y.L1 == 0 {
		st = 2
	}
	return
}

// divo is the quotient of divMod with its status
func (x Int128) divo(y Int128) (q Int128, st uint) {
	if arch.IsAmd64 {
		// a hardware 128-by-64 division per quotient limb there
		if y.L0|y.L1 == 0 {
			return q, 2
		}
		mx, my := uint64(int64(x.L1)>>63), uint64(int64(y.L1)>>63)
		var u, w Uint128
		var c uint64
		u.L0, c = bits.Sub64(x.L0^mx, mx, 0)
		u.L1, _ = bits.Sub64(x.L1^mx, mx, c)
		w.L0, c = bits.Sub64(y.L0^my, my, 0)
		w.L1, _ = bits.Sub64(y.L1^my, my, c)
		var q0, q1 uint64
		if w.L1 == 0 {
			t := u.L1
			if t >= w.L0 {
				q1 = t / w.L0
				t -= q1 * w.L0
			}
			q0, _ = bits.Div64(t, u.L0, w.L0)
		} else {
			n := uint(bits.LeadingZeros64(w.L1))
			d := w.L1<<(n&63) | w.L0>>1>>(^n&63)
			k, _ := bits.Div64(u.L1>>1, u.L1<<63|u.L0>>1, d)
			k >>= ^n & 63
			if k != 0 {
				k--
			}
			hi, lo := bits.Mul64(w.L0, k)
			t0, b := bits.Sub64(u.L0, lo, 0)
			t1, _ := bits.Sub64(u.L1, hi+w.L1*k, b)
			_, b = bits.Sub64(t0, w.L0, 0)
			_, b = bits.Sub64(t1, w.L1, b)
			q0 = k + 1 - b
		}
		mq := mx ^ my
		q.L0, c = bits.Sub64(q0^mq, mq, 0)
		q.L1, _ = bits.Sub64(q1^mq, mq, c)
		st = uint(q1 &^ mq >> 63)
		return
	}
	mx, my := uint64(int64(x.L1)>>63), uint64(int64(y.L1)>>63)
	var u, w Uint128
	var c uint64
	u.L0, c = bits.Sub64(x.L0^mx, mx, 0)
	u.L1, _ = bits.Sub64(x.L1^mx, mx, c)
	w.L0, c = bits.Sub64(y.L0^my, my, 0)
	w.L1, _ = bits.Sub64(y.L1^my, my, c)
	var q0, q1 uint64
	um, wm := u.L1, w.L1
	if um > 1<<33-1 {
		um = 1<<33 - 1
	}
	if wm == 0 {
		wm = 1
	}
	if um < wm {
		n, d := u.L1>>1, w.L1>>1+1
		z := u.L1 | w.L1
		if z == 0 {
			n = u.L0
		}
		if z == 0 {
			d = w.L0
		}
		k := n
		if d != 0 {
			k = n / d
		}
		hi, lo := bits.Mul64(k, w.L0)
		hi += k * w.L1
		t0, b := bits.Sub64(u.L0, lo, 0)
		t1, _ := bits.Sub64(u.L1, hi, b)
		_, b = bits.Sub64(t0, w.L0, 0)
		_, b = bits.Sub64(t1, w.L1, b)
		q0 = k + 1 - b
	} else if w.L1 == 0 && w.L0 != 0 {
		w := w.L0
		q1 = u.L1 / w
		t := u.L1 - q1*w
		s := uint(bits.LeadingZeros64(w))
		d := w << (s & 63)
		v := (1<<47 - 1) / (d>>40 + 1)
		d0 := d & 1
		e := (v>>1)&-d0 - v*(d>>1+d0)
		hi, _ := bits.Mul64(e, e)
		e += hi >> 22
		hi, _ = bits.Mul64(v<<19, e)
		v = v<<41 + hi
		hi, lo := bits.Mul64(v, d)
		_, c := bits.Add64(lo, d, 0)
		hi, _ = bits.Add64(hi, d, c)
		v -= hi
		q0, _ = div2by1(t<<(s&63)|u.L0>>1>>(^s&63), u.L0<<(s&63), d, v)
	} else {
		n := uint(bits.LeadingZeros64(w.L1))
		d := w.L1<<(n&63) | w.L0>>1>>(^n&63)
		v := (1<<47 - 1) / (d>>40 + 1)
		d0 := d & 1
		e := (v>>1)&-d0 - v*(d>>1+d0)
		hi, _ := bits.Mul64(e, e)
		e += hi >> 22
		hi, _ = bits.Mul64(v<<19, e)
		v = v<<41 + hi
		hi, lo := bits.Mul64(v, d)
		_, c := bits.Add64(lo, d, 0)
		hi, _ = bits.Add64(hi, d, c)
		v -= hi
		k, _ := div2by1(u.L1>>1, u.L1<<63|u.L0>>1, d, v)
		k >>= ^n & 63
		if k != 0 {
			k--
		}
		hi, lo = bits.Mul64(w.L0, k)
		t := u.Subu(Uint128{lo, hi + w.L1*k})
		if !t.LessThan(w) {
			k++
		}
		q0 = k
	}
	mq := mx ^ my
	q.L0, c = bits.Sub64(q0^mq, mq, 0)
	q.L1, _ = bits.Sub64(q1^mq, mq, c)
	st = uint(q1 &^ mq >> 63)
	if y.L0|y.L1 == 0 {
		st = 2
	}
	return
}

// mod is the remainder of divMod, for y != 0 only.
// y == 0 panics on amd64 with a runtime error, elsewhere the result is wrong
func (x Int128) mod(y Int128) (r Int128) {
	if arch.IsAmd64 {
		// a hardware 128-by-64 division per quotient limb there
		mx, my := uint64(int64(x.L1)>>63), uint64(int64(y.L1)>>63)
		var u, w Uint128
		var c uint64
		u.L0, c = bits.Sub64(x.L0^mx, mx, 0)
		u.L1, _ = bits.Sub64(x.L1^mx, mx, c)
		w.L0, c = bits.Sub64(y.L0^my, my, 0)
		w.L1, _ = bits.Sub64(y.L1^my, my, c)
		var r0, r1 uint64
		if w.L1 == 0 {
			t := u.L1
			if t >= w.L0 {
				t %= w.L0
			}
			_, r0 = bits.Div64(t, u.L0, w.L0)
		} else {
			n := uint(bits.LeadingZeros64(w.L1))
			d := w.L1<<(n&63) | w.L0>>1>>(^n&63)
			k, _ := bits.Div64(u.L1>>1, u.L1<<63|u.L0>>1, d)
			k >>= ^n & 63
			if k != 0 {
				k--
			}
			hi, lo := bits.Mul64(w.L0, k)
			var b uint64
			r0, b = bits.Sub64(u.L0, lo, 0)
			r1, _ = bits.Sub64(u.L1, hi+w.L1*k, b)
			t0, b := bits.Sub64(r0, w.L0, 0)
			t1, b := bits.Sub64(r1, w.L1, b)
			if b == 0 {
				r0 = t0
			}
			if b == 0 {
				r1 = t1
			}
		}
		r.L0, c = bits.Sub64(r0^mx, mx, 0)
		r.L1, _ = bits.Sub64(r1^mx, mx, c)
		return
	}
	mx, my := uint64(int64(x.L1)>>63), uint64(int64(y.L1)>>63)
	var u, w Uint128
	var c uint64
	u.L0, c = bits.Sub64(x.L0^mx, mx, 0)
	u.L1, _ = bits.Sub64(x.L1^mx, mx, c)
	w.L0, c = bits.Sub64(y.L0^my, my, 0)
	w.L1, _ = bits.Sub64(y.L1^my, my, c)
	var r0, r1 uint64
	um, wm := u.L1, w.L1
	if um > 1<<33-1 {
		um = 1<<33 - 1
	}
	if wm == 0 {
		wm = 1
	}
	if um < wm {
		n, d := u.L1>>1, w.L1>>1+1
		z := u.L1 | w.L1
		if z == 0 {
			n = u.L0
		}
		if z == 0 {
			d = w.L0
		}
		k := n
		if d != 0 {
			k = n / d
		}
		hi, lo := bits.Mul64(k, w.L0)
		hi += k * w.L1
		var b uint64
		r0, b = bits.Sub64(u.L0, lo, 0)
		r1, _ = bits.Sub64(u.L1, hi, b)
		t0, b := bits.Sub64(r0, w.L0, 0)
		t1, b := bits.Sub64(r1, w.L1, b)
		if b == 0 {
			r0 = t0
		}
		if b == 0 {
			r1 = t1
		}
	} else if w.L1 == 0 && w.L0 != 0 {
		w := w.L0
		t := u.L1 % w
		s := uint(bits.LeadingZeros64(w))
		d := w << (s & 63)
		v := (1<<47 - 1) / (d>>40 + 1)
		d0 := d & 1
		e := (v>>1)&-d0 - v*(d>>1+d0)
		hi, _ := bits.Mul64(e, e)
		e += hi >> 22
		hi, _ = bits.Mul64(v<<19, e)
		v = v<<41 + hi
		hi, lo := bits.Mul64(v, d)
		_, c := bits.Add64(lo, d, 0)
		hi, _ = bits.Add64(hi, d, c)
		v -= hi
		_, r0 = div2by1(t<<(s&63)|u.L0>>1>>(^s&63), u.L0<<(s&63), d, v)
		r0 >>= s & 63
	} else {
		n := uint(bits.LeadingZeros64(w.L1))
		d := w.L1<<(n&63) | w.L0>>1>>(^n&63)
		v := (1<<47 - 1) / (d>>40 + 1)
		d0 := d & 1
		e := (v>>1)&-d0 - v*(d>>1+d0)
		hi, _ := bits.Mul64(e, e)
		e += hi >> 22
		hi, _ = bits.Mul64(v<<19, e)
		v = v<<41 + hi
		hi, lo := bits.Mul64(v, d)
		_, c := bits.Add64(lo, d, 0)
		hi, _ = bits.Add64(hi, d, c)
		v -= hi
		k, _ := div2by1(u.L1>>1, u.L1<<63|u.L0>>1, d, v)
		k >>= ^n & 63
		if k != 0 {
			k--
		}
		hi, lo = bits.Mul64(w.L0, k)
		t := u.Subu(Uint128{lo, hi + w.L1*k})
		if !t.LessThan(w) {
			t = t.Subu(w)
		}
		r0, r1 = t.L0, t.L1
	}
	r.L0, c = bits.Sub64(r0^mx, mx, 0)
	r.L1, _ = bits.Sub64(r1^mx, mx, c)
	return
}

// Div64 returns x / y truncated toward zero.
// It panics for y == 0 and on overflow at MinInt128 / -1
func (x Int128) Div64(y int64) Int128 {
	if y == 0 {
		panic("fat.Int128.Div64: division by zero")
	}
	return x.div64(y, "fat.Int128.Div64: overflow")
}

// Div64o returns Div64u and whether x / y overflowed, it panics for y == 0
func (x Int128) Div64o(y int64) (q Int128, overflow bool) {
	if y == 0 {
		panic("fat.Int128.Div64o: division by zero")
	}
	return x.div64o(y)
}

// Div64u returns x / y wrapped around as on a Go integer, it panics for y == 0
func (x Int128) Div64u(y int64) Int128 {
	if y == 0 {
		panic("fat.Int128.Div64u: division by zero")
	}
	q, _ := x.div64o(y)
	return q
}

// Mod64 returns x % y with the sign of x, it panics for y == 0
func (x Int128) Mod64(y int64) int64 {
	if y == 0 {
		panic("fat.Int128.Mod64: division by zero")
	}
	return x.mod64(y)
}

// DivMod64 returns Div64 and Mod64.
// It panics for y == 0 and on overflow at MinInt128 / -1
func (x Int128) DivMod64(y int64) (Int128, int64) {
	if y == 0 {
		panic("fat.Int128.DivMod64: division by zero")
	}
	return x.divMod64(y, "fat.Int128.DivMod64: overflow")
}

// divMod64 is DivMod64 for y != 0 only, y == 0 panics with a runtime error.
// It panics with overflow at MinInt128 / -1
func (x Int128) divMod64(y int64, overflow string) (q Int128, rem int64) {
	if arch.IsAmd64 {
		// a hardware 128-by-64 division per quotient limb there
		mx, my := uint64(int64(x.L1)>>63), uint64(y>>63)
		u := x.UnsignedAbs()
		w := uint64(y) ^ my - my
		var q1 uint64
		t := u.L1
		if t >= w {
			q1 = t / w
			t -= q1 * w
		}
		q0, ra := bits.Div64(t, u.L0, w)
		// the quotient is negative when the signs differ: (v ^ m) - m
		mq := mx ^ my
		var c uint64
		q.L0, c = bits.Add64(q0^mq, mq&1, 0)
		q.L1, _ = bits.Add64(q1^mq, 0, c)
		// |q| = 2^127 fits only as a negative quotient
		if int64(q1&^mq) < 0 {
			panic(overflow)
		}
		// the remainder takes the sign of x
		return q, int64(ra ^ mx - mx)
	}
	// Uint128.divMod64 on |x| and |y| written out, faster than the call.
	// The reciprocal from a 24-bit seed, its last step beside the product:
	// both faster on Graviton in our benchmarks
	my := uint64(y >> 63)
	u := x.UnsignedAbs()
	w := uint64(y) ^ my - my
	s := uint(bits.LeadingZeros64(w))
	d := w << (s & 63)
	v := (1<<47 - 1) / (d>>40 + 1)
	q1 := u.L1 / w
	ra := u.L1 - q1*w
	d0 := d & 1
	e := (v>>1)&-d0 - v*(d>>1+d0)
	hi, _ := bits.Mul64(e, e)
	e += hi >> 22
	hi, _ = bits.Mul64(v<<19, e)
	v = v<<41 + hi
	hi, lo := bits.Mul64(v, d)
	_, c := bits.Add64(lo, d, 0)
	t, _ := bits.Add64(hi, d, c)
	uh, ul := ra<<(s&63)|u.L0>>1>>(^s&63), u.L0<<(s&63)
	qh, ql := bits.Mul64(v, uh)
	ql, c = bits.Add64(ql, ul, 0)
	qh, _ = bits.Add64(qh, uh, c)
	ql, c = bits.Add64(ql, uh&t, 0)
	qh, _ = bits.Add64(qh, 0, c)
	r := ul - d - qh*d
	q0 := qh + 1
	if r > ql {
		q0 = qh
		r += d
	}
	if r >= d {
		q0++
	}
	// the quotient is negative when the signs differ: (v ^ m) - m
	mq := uint64(int64(x.L1)>>63) ^ my
	q.L0, c = bits.Add64(q0^mq, mq&1, 0)
	q.L1, _ = bits.Add64(q1^mq, 0, c)
	// |q| = 2^127 fits only as a negative quotient
	if int64(q1&^mq) < 0 {
		panic(overflow)
	}
	// x = q*y + r in the low limb
	return q, int64(x.L0 - q.L0*uint64(y))
}

// div64 is divMod64 without the remainder and its multiply ending the chain.
// For y != 0 only, y == 0 panics with a runtime error
func (x Int128) div64(y int64, overflow string) (q Int128) {
	if arch.IsAmd64 {
		// a hardware 128-by-64 division per quotient limb there
		my := uint64(y >> 63)
		u := x.UnsignedAbs()
		w := uint64(y) ^ my - my
		var q1 uint64
		t := u.L1
		if t >= w {
			q1 = t / w
			t -= q1 * w
		}
		q0, _ := bits.Div64(t, u.L0, w)
		mq := uint64(int64(x.L1)>>63) ^ my
		var c uint64
		q.L0, c = bits.Add64(q0^mq, mq&1, 0)
		q.L1, _ = bits.Add64(q1^mq, 0, c)
		if int64(q1&^mq) < 0 {
			panic(overflow)
		}
		return q
	}
	// Uint128.div64 on |x| and |y| written out, faster on Graviton
	my := uint64(y >> 63)
	u := x.UnsignedAbs()
	w := uint64(y) ^ my - my
	s := uint(bits.LeadingZeros64(w))
	d := w << (s & 63)
	v := (1<<47 - 1) / (d>>40 + 1)
	q1 := u.L1 / w
	ra := u.L1 - q1*w
	d0 := d & 1
	e := (v>>1)&-d0 - v*(d>>1+d0)
	hi, _ := bits.Mul64(e, e)
	e += hi >> 22
	hi, _ = bits.Mul64(v<<19, e)
	v = v<<41 + hi
	hi, lo := bits.Mul64(v, d)
	_, c := bits.Add64(lo, d, 0)
	t, _ := bits.Add64(hi, d, c)
	uh, ul := ra<<(s&63)|u.L0>>1>>(^s&63), u.L0<<(s&63)
	qh, ql := bits.Mul64(v, uh)
	ql, c = bits.Add64(ql, ul, 0)
	qh, _ = bits.Add64(qh, uh, c)
	ql, c = bits.Add64(ql, uh&t, 0)
	qh, _ = bits.Add64(qh, 0, c)
	r := ul - d - qh*d
	q0 := qh + 1
	if r > ql {
		q0 = qh
		r += d
	}
	if r >= d {
		q0++
	}
	mq := uint64(int64(x.L1)>>63) ^ my
	q.L0, c = bits.Add64(q0^mq, mq&1, 0)
	q.L1, _ = bits.Add64(q1^mq, 0, c)
	if int64(q1&^mq) < 0 {
		panic(overflow)
	}
	return q
}

// div64o is div64 with MinInt128 / -1 wrapped and reported.
// For y != 0 only: y == 0 is a runtime panic on amd64, a wrong result elsewhere
func (x Int128) div64o(y int64) (q Int128, over bool) {
	if arch.IsAmd64 {
		// a hardware 128-by-64 division per quotient limb there
		my := uint64(y >> 63)
		u := x.UnsignedAbs()
		w := uint64(y) ^ my - my
		var q1 uint64
		t := u.L1
		if t >= w {
			q1 = t / w
			t -= q1 * w
		}
		q0, _ := bits.Div64(t, u.L0, w)
		mq := uint64(int64(x.L1)>>63) ^ my
		var c uint64
		q.L0, c = bits.Add64(q0^mq, mq&1, 0)
		q.L1, _ = bits.Add64(q1^mq, 0, c)
		return q, int64(q1&^mq) < 0
	}
	// as Uint128.div64, faster on Graviton in our benchmarks.
	// w is never 0, its test only saves the stack frame of a divide panic
	my := uint64(y >> 63)
	u := x.UnsignedAbs()
	w := uint64(y) ^ my - my
	s := uint(bits.LeadingZeros64(w))
	d := w << (s & 63)
	v := (1<<47 - 1) / (d>>40 + 1)
	q1 := u.L1
	if w != 0 {
		q1 = u.L1 / w
	}
	ra := u.L1 - q1*w
	d0 := d & 1
	e := (v>>1)&-d0 - v*(d>>1+d0)
	hi, _ := bits.Mul64(e, e)
	e += hi >> 22
	hi, _ = bits.Mul64(v<<19, e)
	v = v<<41 + hi
	hi, lo := bits.Mul64(v, d)
	_, c := bits.Add64(lo, d, 0)
	t, _ := bits.Add64(hi, d, c)
	uh, ul := ra<<(s&63)|u.L0>>1>>(^s&63), u.L0<<(s&63)
	qh, ql := bits.Mul64(v, uh)
	ql, c = bits.Add64(ql, ul, 0)
	qh, _ = bits.Add64(qh, uh, c)
	ql, c = bits.Add64(ql, uh&t, 0)
	qh, _ = bits.Add64(qh, 0, c)
	r := ul - d - qh*d
	q0 := qh + 1
	if r > ql {
		q0 = qh
		r += d
	}
	if r >= d {
		q0++
	}
	mq := uint64(int64(x.L1)>>63) ^ my
	q.L0, c = bits.Add64(q0^mq, mq&1, 0)
	q.L1, _ = bits.Add64(q1^mq, 0, c)
	return q, int64(q1&^mq) < 0
}

// mod64 is divMod64 without the quotient.
// For y != 0 only: y == 0 is a runtime panic on amd64, a wrong result elsewhere
func (x Int128) mod64(y int64) int64 {
	if arch.IsAmd64 {
		// a hardware 128-by-64 division per quotient limb there
		my := uint64(y >> 63)
		mx := uint64(int64(x.L1) >> 63)
		u := x.UnsignedAbs()
		w := uint64(y) ^ my - my
		t := u.L1
		if t >= w {
			t %= w
		}
		_, ra := bits.Div64(t, u.L0, w)
		return int64(ra ^ mx - mx)
	}
	// as Uint128.div64, faster on Graviton in our benchmarks.
	// w is never 0, its test only saves the stack frame of a divide panic
	my := uint64(y >> 63)
	mx := uint64(int64(x.L1) >> 63)
	u := x.UnsignedAbs()
	w := uint64(y) ^ my - my
	ra := u.L1
	if w != 0 {
		ra = u.L1 % w
	}
	s := uint(bits.LeadingZeros64(w))
	d := w << (s & 63)
	v := (1<<47 - 1) / (d>>40 + 1)
	d0 := d & 1
	e := (v>>1)&-d0 - v*(d>>1+d0)
	hi, _ := bits.Mul64(e, e)
	e += hi >> 22
	hi, _ = bits.Mul64(v<<19, e)
	v = v<<41 + hi
	hi, lo := bits.Mul64(v, d)
	_, c := bits.Add64(lo, d, 0)
	t, _ := bits.Add64(hi, d, c)
	uh, ul := ra<<(s&63)|u.L0>>1>>(^s&63), u.L0<<(s&63)
	qh, ql := bits.Mul64(v, uh)
	ql, c = bits.Add64(ql, ul, 0)
	qh, _ = bits.Add64(qh, uh, c)
	ql, c = bits.Add64(ql, uh&t, 0)
	qh, _ = bits.Add64(qh, 0, c)
	r := ul - d - qh*d
	q0 := qh + 1
	if r > ql {
		q0 = qh
		r += d
	}
	if r >= d {
		q0++
	}
	// x = q*y + r in the low limb, q*y has the sign of x
	return int64(x.L0 - q0*(w^mx-mx))
}

// int128MulDivPanics, int128MulDivModPanics and int128MulDivCeilPanics are the panics by kernel status.
// One panic from a table keeps the wrappers within the inline budget
var (
	int128MulDivPanics = [3]string{
		1: "fat.Int128.MulDiv: overflow",
		2: "fat.Int128.MulDiv: division by zero",
	}
	int128MulDivModPanics = [3]string{
		1: "fat.Int128.MulDivMod: overflow",
		2: "fat.Int128.MulDivMod: division by zero",
	}
	int128MulDivCeilPanics = [3]string{
		1: "fat.Int128.MulDivCeil: overflow",
		2: "fat.Int128.MulDivCeil: division by zero",
	}
)

// MulDiv returns x * y / d truncated toward zero, the product in 256 bits.
// It panics for d == 0 and when the quotient overflows
func (x Int128) MulDiv(y, d Int128) (q Int128) {
	q, _, st := x.mulDivMod(y, d)
	if st != 0 {
		panic(int128MulDivPanics[st])
	}
	return
}

// MulDivo returns MulDivu and whether the quotient overflowed.
// It panics for d == 0
func (x Int128) MulDivo(y, d Int128) (q Int128, overflow bool) {
	q, _, st := x.mulDivMod(y, d)
	if st > 1 {
		panic("fat.Int128.MulDivo: division by zero")
	}
	return q, st != 0
}

// MulDivu returns x * y / d wrapped around, the product in 256 bits.
// It panics for d == 0
func (x Int128) MulDivu(y, d Int128) (q Int128) {
	q, _, st := x.mulDivMod(y, d)
	if st > 1 {
		panic("fat.Int128.MulDivu: division by zero")
	}
	return
}

// MulDivMod returns MulDiv and x * y % d with the sign of x * y.
// It panics for d == 0 and when the quotient overflows
func (x Int128) MulDivMod(y, d Int128) (q, r Int128) {
	q, r, st := x.mulDivMod(y, d)
	if st != 0 {
		panic(int128MulDivModPanics[st])
	}
	return
}

// MulDivCeil returns x * y / d rounded toward +Inf, the product in 256 bits.
// It panics for d == 0 and when the quotient overflows
func (x Int128) MulDivCeil(y, d Int128) Int128 {
	q, st := x.mulDivCeil(y, d)
	if st != 0 {
		panic(int128MulDivCeilPanics[st])
	}
	return q
}

// mulDivMod is MulDivMod with the low bits of q and a status:
// 1 when q overflowed, 2 for d == 0
func (x Int128) mulDivMod(y, d Int128) (q, r Int128, st uint) {
	qa, ra, st := x.UnsignedAbs().mulDivMod(y.UnsignedAbs(), d.UnsignedAbs())
	// q takes the sign of x * y / d and r that of x * y, by (v ^ m) - m
	mr := uint64(int64(x.L1^y.L1) >> 63)
	mq := mr ^ uint64(int64(d.L1)>>63)
	var b uint64
	q.L0, b = bits.Sub64(qa.L0^mq, mq, 0)
	q.L1, _ = bits.Sub64(qa.L1^mq, mq, b)
	r.L0, b = bits.Sub64(ra.L0^mr, mr, 0)
	r.L1, _ = bits.Sub64(ra.L1^mr, mr, b)
	// a nonzero q that fits keeps its sign, and 2^127 fits only as a negative.
	// The unsigned status carries over: a zero d leaves q = 0 and f = 0
	_, f := bits.Sub64(0, qa.L0|qa.L1, 0)
	f &= (q.L1 ^ mq) >> 63
	return q, r, st | uint(f)
}

// mulDivCeil is MulDivCeil with the low bits of q and a status as of mulDivMod
func (x Int128) mulDivCeil(y, d Int128) (q Int128, st uint) {
	qa, ra, st := x.UnsignedAbs().mulDivMod(y.UnsignedAbs(), d.UnsignedAbs())
	// truncation has rounded a negative quotient up already,
	// a positive one with a remainder adds one
	mq := uint64(int64(x.L1^y.L1^d.L1) >> 63)
	nz := ra.L0 | ra.L1
	var c uint64
	qa.L0, c = bits.Add64(qa.L0, (nz|-nz)>>63&^mq, 0)
	qa.L1, _ = bits.Add64(qa.L1, 0, c)
	// the sign by (v ^ m) - m and the overflow test as in mulDivMod.
	// A carry out of qa comes only from an overflowed q
	var b uint64
	q.L0, b = bits.Sub64(qa.L0^mq, mq, 0)
	q.L1, _ = bits.Sub64(qa.L1^mq, mq, b)
	_, f := bits.Sub64(0, qa.L0|qa.L1, 0)
	f &= (q.L1 ^ mq) >> 63
	return q, st | uint(f)
}

// AddMod returns (x + y) % m of the exact sum, with the sign of the sum.
// It panics for m == 0
func (x Int128) AddMod(y, m Int128) Int128 {
	if m.L0|m.L1 == 0 {
		panic("fat.Int128.AddMod: division by zero")
	}
	// the 129-bit sum as its sign and |x + y|.
	// An overflow flips the sign of the wrapped sum
	s0, c := bits.Add64(x.L0, y.L0, 0)
	s1, _ := bits.Add64(x.L1, y.L1, c)
	ov := (x.L1 ^ s1) & (y.L1 ^ s1)
	ms := uint64(int64(s1^ov) >> 63)
	a0, c := bits.Sub64(s0^ms, ms, 0)
	a1, _ := bits.Sub64(s1^ms, ms, c)
	mm := uint64(int64(m.L1) >> 63)
	w0, c := bits.Sub64(m.L0^mm, mm, 0)
	w1, _ := bits.Sub64(m.L1^mm, mm, c)
	// only MinInt128 + MinInt128 = -2^128 is past 128 bits.
	// 2^128 - |m| leaves the same remainder
	if a0|a1|^ov>>63 == 0 {
		a0, c = bits.Sub64(0, w0, 0)
		a1, _ = bits.Sub64(0, w1, c)
	}
	r := Uint128{a0, a1}.mod(Uint128{w0, w1})
	// the sign of the sum by (v ^ m) - m
	r0, c := bits.Sub64(r.L0^ms, ms, 0)
	r1, _ := bits.Sub64(r.L1^ms, ms, c)
	return Int128{r0, r1}
}

// SubMod returns (x - y) % m of the exact difference, with the sign of the
// difference. It panics for m == 0
func (x Int128) SubMod(y, m Int128) Int128 {
	if m.L0|m.L1 == 0 {
		panic("fat.Int128.SubMod: division by zero")
	}
	// the 129-bit difference as its sign and |x - y|, which fits 128 bits.
	// An overflow flips the sign of the wrapped difference
	s0, c := bits.Sub64(x.L0, y.L0, 0)
	s1, _ := bits.Sub64(x.L1, y.L1, c)
	ov := (x.L1 ^ y.L1) & (x.L1 ^ s1)
	ms := uint64(int64(s1^ov) >> 63)
	a0, c := bits.Sub64(s0^ms, ms, 0)
	a1, _ := bits.Sub64(s1^ms, ms, c)
	mm := uint64(int64(m.L1) >> 63)
	w0, c := bits.Sub64(m.L0^mm, mm, 0)
	w1, _ := bits.Sub64(m.L1^mm, mm, c)
	r := Uint128{a0, a1}.mod(Uint128{w0, w1})
	// the sign of the difference by (v ^ m) - m
	r0, c := bits.Sub64(r.L0^ms, ms, 0)
	r1, _ := bits.Sub64(r.L1^ms, ms, c)
	return Int128{r0, r1}
}

// MulMod returns x * y % m with the sign of x * y, the product in 256 bits.
// It panics for m == 0
func (x Int128) MulMod(y, m Int128) Int128 {
	_, ra, st := x.UnsignedAbs().mulDivMod(y.UnsignedAbs(), m.UnsignedAbs())
	if st > 1 {
		panic("fat.Int128.MulMod: division by zero")
	}
	// the sign of x * y by (v ^ m) - m
	mr := uint64(int64(x.L1^y.L1) >> 63)
	r0, b := bits.Sub64(ra.L0^mr, mr, 0)
	r1, _ := bits.Sub64(ra.L1^mr, mr, b)
	return Int128{r0, r1}
}

// DivToFloat64 returns the float64 nearest to x / y, ties go to the even one.
// It panics for y == 0
func (x Int128) DivToFloat64(y Int128) float64 {
	// Uint128.DivToFloat64 on |x| and |y|, taken by conditional moves.
	// On amd64 (v ^ m) - m is faster, elsewhere a little slower
	var x0, x1, y0, y1, b uint64
	if arch.IsAmd64 {
		mx, my := uint64(int64(x.L1)>>63), uint64(int64(y.L1)>>63)
		x0, b = bits.Sub64(x.L0^mx, mx, 0)
		x1, _ = bits.Sub64(x.L1^mx, mx, b)
		y0, b = bits.Sub64(y.L0^my, my, 0)
		y1, _ = bits.Sub64(y.L1^my, my, b)
	} else {
		x0, b = bits.Sub64(0, x.L0, 0)
		x1, _ = bits.Sub64(0, x.L1, b)
		if int64(x.L1) >= 0 {
			x0, x1 = x.L0, x.L1
		}
		y0, b = bits.Sub64(0, y.L0, 0)
		y1, _ = bits.Sub64(0, y.L1, b)
		if int64(y.L1) >= 0 {
			y0, y1 = y.L0, y.L1
		}
	}
	neg := (x.L1 ^ y.L1) & (1 << 63)
	xh, xm, nx := x1, x0, uint64(0)
	if x1 == 0 {
		xh, xm, nx = x0, 0, 1
	}
	yh, ym, ny := y1, y0, uint64(0)
	if y1 == 0 {
		yh, ym, ny = y0, 0, 1
	}
	if arch.IsAmd64 {
		lx, ly := uint(bits.Len64(xh)), uint(bits.Len64(yh))
		u := xh<<(-lx&63) | xm>>1>>((lx-1)&63)
		d := yh<<(-ly&63) | ym>>1>>((ly-1)&63)
		var h uint64
		if u >= d {
			h = 1
		}
		u >>= h & 1
		e := uint64(lx) - uint64(ly) + (ny-nx)<<6 + h + (1023 - 64 + 1)
		if int64(d) < 0 {
			q, _ := bits.Div64(u, 0, d)
			if (q-1<<10+2)&(1<<11-1) > 3 {
				return float64(int64(q>>1))*math.Float64frombits(e<<52|neg) + 0
			}
		}
	} else {
		sx, sy := uint(bits.LeadingZeros64(xh)), uint(bits.LeadingZeros64(yh))
		tx := ^sx
		u := xh<<(sx&63) | xm>>1>>(tx&63)
		d := yh<<(sy&63) | ym>>1>>(^sy&63)
		var h uint64
		if u >= d {
			h = 1
		}
		u >>= h & 1
		e := uint64(tx) + uint64(sy) + (ny-nx)<<6 + h + (1023 - 64 + 1)
		if int64(d) < 0 {
			v := (1<<47 - 1) / (d>>40 + 1)
			d0 := d & 1
			t := (v>>1)&-d0 - v*(d>>1+d0)
			hi, _ := bits.Mul64(t, t)
			t += hi >> 22
			hi, _ = bits.Mul64(v<<19, t)
			v = v<<41 + hi
			hi, _ = bits.Mul64(v, u)
			q := u + hi
			if (q-1<<10+3)&(1<<11-1) > 4 {
				return float64(q)*math.Float64frombits(e<<52|neg) + 0
			}
		}
	}
	return divFloatExact(Uint256{x0, x1, 0, 0}, Uint256{y0, y1, 0, 0}, neg, "fat.Int128.DivToFloat64: division by zero")
}

// DivToFloat64x returns DivToFloat64 and whether it is exactly x / y.
// It panics for y == 0
func (x Int128) DivToFloat64x(y Int128) (float64, bool) {
	// DivToFloat64 with digits near a multiple of 2^10 left to divFloatNear128.
	// An exact quotient ends at a multiple of 2^11, a zero numerator included.
	// Only y == 0 has no top bit in d
	var x0, x1, y0, y1, b uint64
	if arch.IsAmd64 {
		mx, my := uint64(int64(x.L1)>>63), uint64(int64(y.L1)>>63)
		x0, b = bits.Sub64(x.L0^mx, mx, 0)
		x1, _ = bits.Sub64(x.L1^mx, mx, b)
		y0, b = bits.Sub64(y.L0^my, my, 0)
		y1, _ = bits.Sub64(y.L1^my, my, b)
	} else {
		x0, b = bits.Sub64(0, x.L0, 0)
		x1, _ = bits.Sub64(0, x.L1, b)
		if int64(x.L1) >= 0 {
			x0, x1 = x.L0, x.L1
		}
		y0, b = bits.Sub64(0, y.L0, 0)
		y1, _ = bits.Sub64(0, y.L1, b)
		if int64(y.L1) >= 0 {
			y0, y1 = y.L0, y.L1
		}
	}
	neg := (x.L1 ^ y.L1) & (1 << 63)
	xh, xm, nx := x1, x0, uint64(0)
	if x1 == 0 {
		xh, xm, nx = x0, 0, 1
	}
	yh, ym, ny := y1, y0, uint64(0)
	if y1 == 0 {
		yh, ym, ny = y0, 0, 1
	}
	if arch.IsAmd64 {
		lx, ly := uint(bits.Len64(xh)), uint(bits.Len64(yh))
		u := xh<<(-lx&63) | xm>>1>>((lx-1)&63)
		d := yh<<(-ly&63) | ym>>1>>((ly-1)&63)
		var h uint64
		if u >= d {
			h = 1
		}
		u1 := u
		u >>= h & 1
		e := uint64(lx) - uint64(ly) + (ny-nx)<<6 + h + (1023 - 64 + 1)
		if int64(d) < 0 {
			q, _ := bits.Div64(u, 0, d)
			if (q+2)&(1<<10-1) > 3 {
				return float64(int64(q>>1))*math.Float64frombits(e<<52|neg) + 0, false
			}
			return divFloatNear128(u1, xm<<(-lx&63), d, ym<<(-ly&63), q, (q+2)&^(1<<10-1), h, e-1, neg)
		}
	} else {
		sx, sy := uint(bits.LeadingZeros64(xh)), uint(bits.LeadingZeros64(yh))
		tx := ^sx
		u := xh<<(sx&63) | xm>>1>>(tx&63)
		d := yh<<(sy&63) | ym>>1>>(^sy&63)
		var h uint64
		if u >= d {
			h = 1
		}
		u1 := u
		u >>= h & 1
		e := uint64(tx) + uint64(sy) + (ny-nx)<<6 + h + (1023 - 64 + 1)
		if int64(d) < 0 {
			v := (1<<47 - 1) / (d>>40 + 1)
			d0 := d & 1
			t := (v>>1)&-d0 - v*(d>>1+d0)
			hi, _ := bits.Mul64(t, t)
			t += hi >> 22
			hi, _ = bits.Mul64(v<<19, t)
			v = v<<41 + hi
			hi, _ = bits.Mul64(v, u)
			q := u + hi
			if (q+3)&(1<<10-1) > 4 {
				return float64(q)*math.Float64frombits(e<<52|neg) + 0, false
			}
			return divFloatNear128(u1, xm<<(sx&63), d, ym<<(sy&63), q, (q+3)&^(1<<10-1), h, e, neg)
		}
	}
	panic("fat.Int128.DivToFloat64x: division by zero")
}
