package fat

import (
	"strconv"
)

// the errors are returned by value
var (
	_ error = ErrNegative{}
	_ error = ErrRange("")
)

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
