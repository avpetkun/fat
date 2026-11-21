package testutil

import (
	"fmt"
	"testing"
)

// On adds args, the operands under test, to require failure messages
func On(t testing.TB, args ...any) testing.TB { return labeled{t, args} }

// labeled is a testing.TB whose Name ends with its args
type labeled struct {
	testing.TB
	args []any
}

// Name returns the test name followed by the args
func (l labeled) Name() string { return l.TB.Name() + " with " + fmt.Sprint(l.args...) }
