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

type AccessSize uint32

// Types of accesses supported by the PlayStation archeticture

var (
	ACCESS_BYTE     AccessSize = 1 // 8 bit
	ACCESS_HALFWORD AccessSize = 2 // 16 bit
	ACCESS_WORD     AccessSize = 4 // 32 bit
)

func accessSizeU32(size AccessSize, val uint32) interface{} {
	switch size {
	case ACCESS_BYTE:
		return byte(val)
	case ACCESS_HALFWORD:
		return uint16(val)
	default: // handles ACCESS_WORD and invalid cases
		return val
	}
}

func accessSizeToU32(size AccessSize, val interface{}) uint32 {
	switch size {
	case ACCESS_BYTE:
		return uint32(val.(byte))
	case ACCESS_HALFWORD:
		return uint32(val.(uint16))
	default: // handles ACCESS_WORD and invalid cases
		return val.(uint32)
	}
}
