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

func TestAbsInt64(t *testing.T) {
	assert := func(v bool) {
		if !v {
			t.Error("assert failed")
		}
	}

	assert(absInt64(-5) == 5)
	assert(absInt64(18) == 18)
	assert(absInt64(0) == 0)
	assert(absInt64(-999999) == 999999)
}

func TestMaxInt64(t *testing.T) {
	assert := func(v bool) {
		if !v {
			t.Error("assert failed")
		}
	}

	assert(maxInt64(1, 2) == 2)
	assert(maxInt64(-100, 100) == 100)
	assert(maxInt64(888, -5) == 888)
	assert(maxInt64(-11, -22) == -11)
}
