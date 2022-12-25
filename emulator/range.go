package emulator

var (
	// The range of the BIOS in the system memory
	BIOS_RANGE = NewRange(0x1fc00000, BIOS_SIZE)
	// Memory latency and expansion mapping (also known as SYSCONTROL)
	MEMCONTROL_RANGE = NewRange(0x1f801000, 36)
	// Register that has something to do with RAM configuration, configured by the BIOS
	RAMSIZE_RANGE = NewRange(0x1f801060, 4)
	// Cache control register, full address since it's in KSEG2
	CACHE_CONTROL_RANGE = NewRange(0xfffe0130, 4)
	// Main RAM: 2MB mirrored four times over the first 8MB
	RAM_RANGE = NewRange(0x00000000, 8*1024*1024)
	// SPU (Sound Processing Unit)
	SPU_RANGE = NewRange(0x1f801c00, 640)
	// Expansion region 1
	EXPANSION_1_RANGE = NewRange(0x1f000000, 512*1024)
	// Expansion region 2
	EXPANSION_2_RANGE = NewRange(0x1f802000, 66)
	// Interrupt Control registers (status and mask)
	IRQ_CONTROL_RANGE = NewRange(0x1f801070, 8)
	// Timer registers
	TIMERS_RANGE = NewRange(0x1f801100, 0x30)
	// Direct Memory Access registers
	DMA_RANGE = NewRange(0x1f801080, 0x80)
	// GPU
	GPU_RANGE   = NewRange(0x1f801810, 8)
	// The CD-ROM controller
	CDROM_RANGE = NewRange(0x1f801800, 0x4)
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

func (r *Range) ContainsAndOffset(addr uint32) (bool, uint32) {
	ok := addr >= r.Start && addr < r.Start+r.Length
	if ok {
		return true, addr - r.Start
	}
	return false, 0
}
