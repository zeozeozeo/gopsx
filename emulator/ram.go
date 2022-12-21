package emulator

const (
	RAM_ALLOC_SIZE = 2 * 1024 * 1024 // Main PlayStation RAM: 2MB
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

// Loads a value at `offset`
func (ram *RAM) Load(offset uint32, size AccessSize) interface{} {
	var v uint32 = 0
	sizeI := uint32(size)

	for i := uint32(0); i < sizeI; i++ {
		v |= uint32(ram.Data[offset+i]) << (i * 8)
	}
	return accessSizeU32(size, v)
}

// Stores `val` into `offset`
func (ram *RAM) Store(offset uint32, size AccessSize, val interface{}) {
	valU32 := accessSizeToU32(size, val)
	sizeI := uint32(size)

	for i := uint32(0); i < sizeI; i++ {
		ram.Data[offset+i] = byte(valU32 >> (i * 8))
	}
}

// Load a 32 bit little endian word at `offset`
func (ram *RAM) Load32(offset uint32) uint32 {
	b0 := uint32(ram.Data[offset+0])
	b1 := uint32(ram.Data[offset+1])
	b2 := uint32(ram.Data[offset+2])
	b3 := uint32(ram.Data[offset+3])
	return b0 | (b1 << 8) | (b2 << 16) | (b3 << 24)
}

// Load a 16 bit little endian value at `offset`
func (ram *RAM) Load16(offset uint32) uint16 {
	b0 := uint16(ram.Data[offset+0])
	b1 := uint16(ram.Data[offset+1])
	return b0 | (b1 << 8)
}

// Fetches the byte at `offset`
func (ram *RAM) Load8(offset uint32) byte {
	return ram.Data[offset]
}

// Store a 32 bit little endian word `val` into `offset`
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
