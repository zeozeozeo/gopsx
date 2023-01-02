package emulator

// 1kb scratchpad (fast RAM)
const SCRATCH_PAD_SIZE = 1024

type ScratchPad struct {
	Data [SCRATCH_PAD_SIZE]byte
}

// Returns a new ScratchPad instance initialized with garbage values
func NewScratchPad() *ScratchPad {
	sp := &ScratchPad{}
	// initialize scratchpad with garbage values
	for i := 0; i < len(sp.Data); i++ {
		sp.Data[i] = 0xab
	}
	return sp
}

// Loads a value at `offset`
func (sp *ScratchPad) Load(offset uint32, size AccessSize) interface{} {
	var v uint32 = 0
	sizeI := uint32(size)

	for i := uint32(0); i < sizeI; i++ {
		v |= uint32(sp.Data[offset+i]) << (i * 8)
	}
	return accessSizeU32(size, v)
}

// Stores `val` into `offset`
func (sp *ScratchPad) Store(offset uint32, size AccessSize, val interface{}) {
	valU32 := accessSizeToU32(size, val)
	sizeI := uint32(size)

	for i := uint32(0); i < sizeI; i++ {
		sp.Data[offset+i] = byte(valU32 >> (i * 8))
	}
}

// Load a 32 bit little endian word at `offset`
func (sp *ScratchPad) Load32(offset uint32) uint32 {
	return sp.Load(offset, ACCESS_WORD).(uint32)
}

// Load a 16 bit little endian value at `offset`
func (sp *ScratchPad) Load16(offset uint32) uint16 {
	return sp.Load(offset, ACCESS_HALFWORD).(uint16)
}

// Fetches the byte at `offset`
func (sp *ScratchPad) Load8(offset uint32) byte {
	return sp.Load(offset, ACCESS_BYTE).(byte)
}

// Store a 32 bit little endian word `val` into `offset`
func (sp *ScratchPad) Store32(offset, val uint32) {
	sp.Store(offset, ACCESS_WORD, val)
}

// Stores a 16 bit little endian value into `offset`
func (sp *ScratchPad) Store16(offset uint32, val uint16) {
	sp.Store(offset, ACCESS_HALFWORD, val)
}

// Sets the byte at `offset`
func (sp *ScratchPad) Store8(offset uint32, val byte) {
	sp.Store(offset, ACCESS_BYTE, val)
}
