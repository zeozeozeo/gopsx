package emulator

import "testing"

func TestCrc32Table(t *testing.T) {
	assert := func(v bool) {
		if !v {
			t.Error("assert failed")
		}
	}

	for i := uint32(0); i < 0x100; i++ {
		r := i
		for j := 0; j < 8; j++ {
			var x uint32 = 0
			if r&1 != 0 {
				x = 0xd8018001
			}
			r = (r >> 1) ^ x
		}

		assert(CRC32_TABLE[i] == r)
	}
}

func TestCrc32(t *testing.T) {
	assert := func(v bool) {
		if !v {
			t.Error("assert failed")
		}
	}

	assert(Crc32(nil) == 0x00000000)
	assert(Crc32([]byte{0}) == 0x00000000)
	assert(Crc32([]byte("test string")) == 0x15c8ac07)
}
