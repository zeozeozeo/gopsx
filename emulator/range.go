package emulator

var (
	// The range of the BIOS in the system memory
	BIOS_RANGE = NewRange(0x1fc00000, BIOS_SIZE)
	// Memory latency and expansion mapping (also known as SYSCONTROL)
	MEM_CONTROL = NewRange(0x1f801000, 36)
	// Register that has something to do with RAM configuration, configured by the BIOS
	RAM_SIZE = NewRange(0x1f801060, 4)
	// Cache control register, full address since it's in KSEG2
	CACHE_CONTROL = NewRange(0xfffe0130, 4)
	// RAM
	RAM_RANGE = NewRange(0x00000000, RAM_ALLOC_SIZE)
)

type Range struct {
	Start  uint32 // Start address
	Length uint32 // Length of the mapping
}

func NewRange(start uint32, length uint32) Range {
	return Range{Start: start, Length: length}
}

// Returns whether `addr` is located inside this range
func (r *Range) Contains(addr uint32) bool {
	return addr >= r.Start && addr < r.Start+r.Length
}

// Returns the offset between `addr` and the `Start` of the range.
// Does not check if the range contains the address, so if `addr`
// is smaller than `Start`, there will be an overflow
func (r *Range) Offset(addr uint32) uint32 {
	return addr - r.Start
}
