package fat

import (
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
