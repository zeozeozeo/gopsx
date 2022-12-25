package emulator

import (
	"errors"
	"fmt"
)

var errOverflow = errors.New("integer overflow")

// Names of registers
var RegisterNames = []string{
	"r0", "at", "v0", "v1", "a0", "a1", "a2", "a3", // 00
	"t0", "t1", "t2", "t3", "t4", "t5", "t6", "t7", // 08
	"s0", "s1", "s2", "s3", "s4", "s5", "s6", "s7", // 10
	"t8", "t9", "k0", "k1", "gp", "sp", "fp", "ra", // 18
}

// Returns the name of the register index
func GetRegisterName(index uint32) string {
	return RegisterNames[index]
}

// Returns the register index by it's name (in RegisterNames).
// Returns 0 if the register name does not exist
func GetRegisterIndexByName(name string) uint32 {
	for idx, n := range RegisterNames {
		if n == name {
			return uint32(idx)
		}
	}
	return 0
}

// Formatted panic()
func panicFmt(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}

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

func accessSizeToU16(size AccessSize, val interface{}) uint16 {
	switch size {
	case ACCESS_BYTE:
		return uint16(val.(byte))
	case ACCESS_HALFWORD:
		return val.(uint16)
	default: // handles ACCESS_WORD and invalid cases
		return uint16(val.(uint32))
	}
}

func accessSizeToU8(size AccessSize, val interface{}) uint8 {
	switch size {
	case ACCESS_BYTE:
		return val.(byte)
	case ACCESS_HALFWORD:
		return uint8(val.(uint16))
	default: // handles ACCESS_WORD and invalid cases
		return uint8(val.(uint32))
	}
}

func oneIfTrue(val bool) uint32 {
	if val {
		return 1
	}
	return 0
}
