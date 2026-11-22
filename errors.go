package fat

import (
	"fmt"
	"strconv"
	"strings"
)

// the errors are returned by value
var (
	_ error = ErrParse{}
	_ error = ErrNegative{}
	_ error = ErrRange("")
	_ error = ErrBinary{}
	_ error = ErrScan{}
)

// ErrParse is text a func or method cannot parse, Error tells why
type ErrParse struct{ fn, text string }

// Error names the func, the text and the reason.
// The parsers only report failure, the reason is found again here
func (e ErrParse) Error() string {
	// one buffer for the whole message, no string for any part
	var buf [128]byte
	b := append(append(buf[:0], "fat."...), e.fn...)
	if strings.HasSuffix(e.fn, ".UnmarshalJSON") {
		// JSON as it came: its quotes tell a string from a bare value
		if e.text == "" {
			return string(append(b, ": empty JSON"...))
		}
		b = append(append(b, ": cannot parse "...), e.text...)
	} else {
		b = strconv.AppendQuote(append(b, ": cannot parse "...), e.text)
	}
	return string(appendErrReason(append(b, ": "...), e.fn, e.text))
}

// ErrNegative is a negative value given to an unsigned type
type ErrNegative struct{ fn, text string }

// Error names the func and the text of the value if any
func (e ErrNegative) Error() string {
	if e.text == "" {
		return "fat." + e.fn + ": negative value"
	}
	var buf [64]byte
	b := append(append(append(buf[:0], "fat."...), e.fn...), ": negative value "...)
	return string(strconv.AppendQuote(b, e.text))
}

// ErrRange is a value past the range of the type, it holds the func name
type ErrRange string

// Error names the func
func (e ErrRange) Error() string {
	return "fat." + string(e) + ": out of range"
}

// ErrBinary is a byte length a method cannot take
type ErrBinary struct {
	fn      string
	n, want int
}

// Error names the method, the length given and the one it takes
func (e ErrBinary) Error() string {
	var buf [64]byte
	b := append(append(append(buf[:0], "fat."...), e.fn...), ": cannot parse "...)
	b = append(appendDecimal64(b, uint64(e.n)), " bytes, want "...)
	return string(appendDecimal64(b, uint64(e.want)))
}

// ErrScan is a Scan source the types do not take: NULL or another type
type ErrScan struct{ typ, text string }

// Error names the method and the source
func (e ErrScan) Error() string { return "fat." + e.typ + ": " + e.text }

// newScanError names what Scan does not take: NULL or the type of src
func newScanError(typ string, src any) error {
	if src == nil {
		return ErrScan{typ, "cannot scan NULL"}
	}
	return ErrScan{typ, fmt.Sprintf("cannot scan %T", src)}
}

// appendErrReason appends why s fails the parser of fn.
// The form comes from the name of fn: FromDec, FromHex or UnmarshalJSON,
// or 0x picks the base; text of digits only fails by its size
func appendErrReason(dst []byte, fn, s string) []byte {
	// a JSON string is read without quotes and escapes,
	// offsets count in its text
	json := strings.HasSuffix(fn, ".UnmarshalJSON")
	quoted := json && len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"'
	if quoted {
		s = s[1 : len(s)-1]
		// the text stays in bytes on the stack, no string is made of it
		var buf [80]byte
		if d, ok := jsonUnescape(buf[:0], s); ok {
			return appendTextReason(dst, fn, d, json, quoted)
		}
	}
	return appendTextReason(dst, fn, s, json, quoted)
}

// appendTextReason is appendErrReason for the text out of its JSON string
func appendTextReason[S string | []byte](dst []byte, fn string, s S, json, quoted bool) []byte {
	dec, hex := strings.HasSuffix(fn, "FromDec"), strings.HasSuffix(fn, "FromHex")
	if len(s) == 0 {
		return append(dst, "empty"...)
	}
	i := 0
	if s[0] == '+' || s[0] == '-' {
		i = 1
	}
	if i == len(s) {
		return append(dst, "no digits after the sign"...)
	}
	prefix := len(s)-i >= 2 && s[i] == '0' && s[i+1]|0x20 == 'x'
	switch {
	case dec && prefix:
		return append(dst, "0x in decimal text"...)
	case json && prefix && !quoted:
		return append(dst, "0x outside quotes"...)
	}
	if prefix {
		if i += 2; i == len(s) {
			return append(dst, "no digits after 0x"...)
		}
	}
	// FromHex takes hex with no prefix too
	if prefix || hex {
		for j := i; j < len(s); j++ {
			if c := s[j] | 0x20; c-'0' > 9 && c-'a' > 5 {
				return appendErrBadChar(dst, "hex digit", s, j)
			}
		}
		return appendErrRange(dst, fn, s)
	}
	// FromDec and Scan take m e n too
	sci := dec || strings.HasSuffix(fn, ".Scan")
	for j := i; j < len(s); j++ {
		if s[j]-'0' > 9 {
			if sci && j > i && s[j]|0x20 == 'e' {
				return appendSciReason(dst, fn, s, j)
			}
			return appendErrBadChar(dst, "digit", s, j)
		}
	}
	return appendErrRange(dst, fn, s)
}

// appendSciReason is appendTextReason for m e n with digits of m before
// its e at s[j]: n is empty or has a bad char, else m * 10^n does not fit
func appendSciReason[S string | []byte](dst []byte, fn string, s S, j int) []byte {
	k := j + 1
	if k < len(s) && s[k] == '+' {
		k++
	}
	if k == len(s) {
		return append(dst, "no digits after the exponent"...)
	}
	for ; k < len(s); k++ {
		if s[k]-'0' > 9 {
			return appendErrBadChar(dst, "digit", s, k)
		}
	}
	return appendErrRange(dst, fn, s)
}

// appendErrRange tells digits that do not fit, or a minus on an unsigned zero.
// s must not be empty
func appendErrRange[S string | []byte](dst []byte, fn string, s S) []byte {
	if s[0] == '-' && strings.Contains(fn, "Uint") {
		return append(dst, "minus sign"...)
	}
	return append(dst, "out of range"...)
}

// appendErrBadChar names the char at s[j] and its offset, for j < len(s) only
func appendErrBadChar[S string | []byte](dst []byte, what string, s S, j int) []byte {
	r, _ := decodeRune(s[j:])
	dst = strconv.AppendQuoteRune(append(append(append(dst, "invalid "...), what...), ' '), r)
	return appendDecimal64(append(dst, " at offset "...), uint64(j))
}
