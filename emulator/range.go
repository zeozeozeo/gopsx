package emulator

var (
	BIOS_RANGE = NewRange(0xbfc00000, BIOS_SIZE) // The range of the BIOS in the system memory
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
