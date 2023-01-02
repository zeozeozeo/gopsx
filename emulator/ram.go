package emulator

const (
	RAM_ALLOC_SIZE = 2 * 1024 * 1024 // Main PlayStation RAM: 2MB
)

type RAM struct {
	Data [RAM_ALLOC_SIZE]byte // RAM buffer
}

// Creates a new RAM instance (allocates `RAM_ALLOC_SIZE` bytes and fills
// them with garbage values)
func NewRAM() *RAM {
	ram := &RAM{}
	for i := 0; i < len(ram.Data); i++ {
		ram.Data[i] = 0xcd
	}
	return ram
}

// Loads a value at `offset`
func (ram *RAM) Load(offset uint32, size AccessSize) interface{} {
	var v uint32 = 0
	sizeI := uint32(size)
	offset &= 0x1fffff

	for i := uint32(0); i < sizeI; i++ {
		v |= uint32(ram.Data[offset+i]) << (i * 8)
	}
	return accessSizeU32(size, v)
}

// Stores `val` into `offset`
func (ram *RAM) Store(offset uint32, size AccessSize, val interface{}) {
	valU32 := accessSizeToU32(size, val)
	sizeI := uint32(size)
	offset &= 0x1fffff

	for i := uint32(0); i < sizeI; i++ {
		ram.Data[offset+i] = byte(valU32 >> (i * 8))
	}
}

// Load a 32 bit little endian word at `offset`
func (ram *RAM) Load32(offset uint32) uint32 {
	return ram.Load(offset, ACCESS_WORD).(uint32)
}

// Load a 16 bit little endian value at `offset`
func (ram *RAM) Load16(offset uint32) uint16 {
	return ram.Load(offset, ACCESS_HALFWORD).(uint16)
}

// Fetches the byte at `offset`
func (ram *RAM) Load8(offset uint32) byte {
	return ram.Load(offset, ACCESS_BYTE).(byte)
}

// Store a 32 bit little endian word `val` into `offset`
func (ram *RAM) Store32(offset, val uint32) {
	ram.Store(offset, ACCESS_WORD, val)
}

// Stores a 16 bit little endian value into `offset`
func (ram *RAM) Store16(offset uint32, val uint16) {
	ram.Store(offset, ACCESS_HALFWORD, val)
}

// Sets the byte at `offset`
func (ram *RAM) Store8(offset uint32, val byte) {
	ram.Store(offset, ACCESS_BYTE, val)
}
