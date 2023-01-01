package emulator

import "testing"

func TestDivision(t *testing.T) {
	assert := func(res, want uint32) {
		if res != want {
			t.Errorf("expected 0x%x, got 0x%x", res, want)
		}
	}

	assert(GTEDivide(0, 1), 0)
	assert(GTEDivide(0, 1234), 0)
	assert(GTEDivide(1, 1), 0x10000)
	assert(GTEDivide(2, 2), 0x10000)
	assert(GTEDivide(0xffff, 0xffff), 0xffff)
	assert(GTEDivide(0xffff, 0xfffe), 0x10000)
	assert(GTEDivide(1, 2), 0x8000)
	assert(GTEDivide(1, 3), 0x5555)
	assert(GTEDivide(5, 6), 0xd555)
	assert(GTEDivide(1, 4), 0x4000)
	assert(GTEDivide(10, 40), 0x4000)
	assert(GTEDivide(0xf00, 0xbeef), 0x141d)
	assert(GTEDivide(9876, 8765), 0x12072)
	assert(GTEDivide(200, 10000), 0x51f)
	assert(GTEDivide(0xffff, 0x8000), 0x1fffe)
	assert(GTEDivide(0xe5d7, 0x72ec), 0x1ffff)
}
