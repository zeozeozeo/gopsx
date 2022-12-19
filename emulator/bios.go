package emulator

import (
	"fmt"
	"io"
)

const BIOS_SIZE uint32 = 512 * 1024 // BIOS images are always 512KB in length

// This stores the raw BIOS data
type BIOS struct {
	Data []byte // Raw BIOS data
}

// Loads a BIOS from a reader. Note that the BIOS must be 512 * 1024
// bytes in size
func LoadBIOS(r io.Reader) (*BIOS, error) {
	data := make([]byte, BIOS_SIZE)
	n, err := r.Read(data)
	if err != nil {
		return nil, err
	}
	if n != int(BIOS_SIZE) {
		return nil, fmt.Errorf("invalid BIOS size (expected %d, got %d (bytes))", BIOS_SIZE, n)
	}
	// success
	return &BIOS{Data: data}, nil
}

// Returns a 32 bit little endian value at `offset`. Note that `offset` is
// not the absolute address used by the CPU, instead it is an offset in the
// BIOS memory range
func (bios *BIOS) Load32(offset uint32) uint32 {
	b0 := uint32(bios.Data[offset+0])
	b1 := uint32(bios.Data[offset+1])
	b2 := uint32(bios.Data[offset+2])
	b3 := uint32(bios.Data[offset+3])
	return b0 | (b1 << 8) | (b2 << 16) | (b3 << 24)
}

// Fetch byte at `offset`
func (bios *BIOS) Load8(offset uint32) byte {
	return bios.Data[offset]
}