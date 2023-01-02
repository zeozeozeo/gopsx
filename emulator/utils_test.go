package emulator

import (
	"testing"
)

func TestCountLeadingZeroesU32(t *testing.T) {
	assert := func(v bool) {
		if !v {
			t.Error("assert failed")
		}
	}
	for i, x := uint32(0), uint32(0); i < 33; i++ {
		assert(countLeadingZeroesU32(x) == 32-i)
		x = (x << 1) + 1
	}
}

func TestCountLeadingZeroesU16(t *testing.T) {
	assert := func(v bool) {
		if !v {
			t.Error("assert failed")
		}
	}
	for i, x := uint16(0), uint16(0); i < 17; i++ {
		assert(countLeadingZeroesU16(x) == 16-i)
		x = (x << 1) + 1
	}
}
