package emulator

const (
	RAM_ALLOC_SIZE = 2 * 1024 * 1024
)

type RAM struct {
	Data []byte // RAM buffer
}

// Creates a new RAM instance (allocates `RAM_ALLOC_SIZE` bytes)
func NewRAM() *RAM {
	// FIXME: default RAM contents should be garbage
	return &RAM{
		Data: make([]byte, RAM_ALLOC_SIZE),
	}
}

// Load the 32 bit little endian word at `offset`
func (ram *RAM) Load32(offset uint32) uint32 {
	b0 := uint32(ram.Data[offset+0])
	b1 := uint32(ram.Data[offset+1])
	b2 := uint32(ram.Data[offset+2])
	b3 := uint32(ram.Data[offset+3])
	return b0 | (b1 << 8) | (b2 << 16) | (b3 << 24)
}

// Fetches the byte at `offset`
func (ram *RAM) Load8(offset uint32) byte {
	return ram.Data[offset]
}

// Store the 32 bit little endian word `val` into `offset`
func (ram *RAM) Store32(offset, val uint32) {
	ram.Data[offset+0] = byte(val)
	ram.Data[offset+1] = byte(val >> 8)
	ram.Data[offset+2] = byte(val >> 16)
	ram.Data[offset+3] = byte(val >> 24)
}

// Stores a 16 bit little endian value into `offset`
func (ram *RAM) Store16(offset uint32, val uint16) {
	ram.Data[offset+0] = byte(val)
	ram.Data[offset+1] = byte(val >> 8)
}

// Sets the byte at `offset`
func (ram *RAM) Store8(offset uint32, val byte) {
	ram.Data[offset] = val
}
