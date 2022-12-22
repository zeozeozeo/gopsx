package emulator

type CacheControl uint32

// Returns whether the instruction cache is enabled
func (cache CacheControl) ICacheEnabled() bool {
	return uint32(cache)&0x800 != 0
}

func (cache CacheControl) TagTestMode() bool {
	return uint32(cache)&4 != 0
}

type ICacheLine struct {
	// Tag: high 22 bits of the address associated with the cache line
	// Valid bits: 3 bit index of the first word in this line
	TagValid uint32
	Line     [4]Instruction // 4 words per line
}

func NewCacheLine() *ICacheLine {
	return &ICacheLine{
		TagValid: 0x0,
		Line:     [4]Instruction{0x00bad0d, 0x00bad0d, 0x00bad0d, 0x00bad0d}, // BREAK opcode
	}
}

// Returns the tag of the cache line
func (cline *ICacheLine) Tag() uint32 {
	return cline.TagValid & 0xfffff000
}

// Returns the first valid word of the cache line
func (cline *ICacheLine) ValidIndex() uint32 {
	return (cline.TagValid >> 2) & 0x7 // [4:2]
}

// Sets the cache line's tag and vali bits, `pc` is the first valid PC
// in the cacheline
func (cline *ICacheLine) SetTagValid(pc uint32) {
	cline.TagValid = pc & 0xfffff00c
}

// Invalidates the entire cache line
func (cline *ICacheLine) Invalidate() {
	cline.TagValid |= 0x10 // set bit 4 to be outside of the valid cache line range
}

// Returns the instruction at `index`
func (cline *ICacheLine) Get(index uint32) Instruction {
	return cline.Line[index]
}

// Sets the instruction at `index` to `instruction`
func (cline *ICacheLine) Set(index uint32, instruction Instruction) {
	cline.Line[index] = instruction
}
