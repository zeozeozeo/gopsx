package emulator

import (
	"errors"
	"fmt"
)

var errOverflow = errors.New("integer overflow")

// Formatted panic()
func panicFmt(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}

/*
func todo() {
	panic("TODO")
}
*/

// Adds two signed integers and checks for overflow
func add32Overflow(a, b int32) (int32, error) {
	c := a + b
	if (c > a) == (b > 0) {
		return c, nil
	}
	return c, errOverflow
}

// Subtracts two signed integers and checks for overflow
func sub32Overflow(a, b int32) (int32, error) {
	c := a - b
	if (c < a) == (b > 0) {
		return c, nil
	}
	return c, errOverflow
}
