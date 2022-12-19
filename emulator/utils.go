package emulator

import (
	"errors"
	"fmt"
	"math"
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
func add32Overflow(left, right int32) (int32, error) {
	if right > 0 {
		if left > math.MaxInt32-right {
			return 0, errOverflow
		}
	} else {
		if left < math.MinInt32-right {
			return 0, errOverflow
		}
	}
	return left + right, nil
}
