package emulator

import "fmt"

// Global interconnect. It stores all of the peripherals
type Interconnect struct {
	Bios *BIOS // Basic input/output memory
	Ram  *RAM  // RAM
}

// Mask array used to strip the region bits of a CPU address. The mask
// is selected using 3 MSBs of the address so each entry matches 512MB
// of the address space. KSEG2 doesn't share anything with the other
// regions, so it is not used
var REGION_MASK = [8]uint32{
	// KUSEG: 2048MB
	0xffffffff, 0xffffffff, 0xffffffff, 0xffffffff,
	// KSEG0: 512MB
	0x7fffffff,
	// KSEG1: 512MB
	0x1fffffff,
	// KSEG2: 1024MB
	0xffffffff, 0xffffffff,
}

// Creates a new interconnect instance
func NewInterconnect(bios *BIOS, ram *RAM) *Interconnect {
	inter := &Interconnect{
		Bios: bios,
		Ram:  ram,
	}
	return inter
}

// Returns a 32bit little endian value at `addr`. Panics
// if the address does not exist
func (inter *Interconnect) Load32(addr uint32) uint32 {
	addr = MaskRegion(addr)
	if addr%4 != 0 {
		panicFmt("interconnect: unaligned Load32 address 0x%x", addr)
	}

	// handle ranges
	if BIOS_RANGE.Contains(addr) {
		return inter.Bios.Load32(BIOS_RANGE.Offset(addr))
	}
	if RAM_RANGE.Contains(addr) {
		return inter.Ram.Load32(RAM_RANGE.Offset(addr))
	}

	// couldn't load address, panic
	panicFmt("interconnect: unhandled Load32 at address 0x%x", addr)
	return 0 // shouldn't reach here, but the linter still wants me to put this
}

func (inter *Interconnect) Store32(addr, val uint32) {
	addr = MaskRegion(addr)
	if addr%4 != 0 {
		panicFmt("interconnect: unaligned Store32 of 0x%x into address 0x%x", val, addr)
	}

	// handle MEMCONTROL
	if MEM_CONTROL.Contains(addr) {
		switch MEM_CONTROL.Offset(addr) {
		case 0: // expansion 1 base address
			if val != 0x1f000000 {
				panicFmt("interconnect: bad expansion 1 base address 0x%x", addr)
			}
		case 4: // expansion 2 base address
			if val != 0x1f802000 {
				panicFmt("interconnect: bad expansion 2 base address 0x%x", addr)
			}
		default:
			// FIXME: add proper logging for this
			fmt.Println("interconnect: unhandled write to MEMCONTROL register")
		}
		return
	}

	// handle RAMSIZE
	// the exact purpose of this register is partially unknown, but
	// it seems to be configuring the memory controller, so hopefully
	// it's safe to just ignore it
	if RAM_SIZE.Contains(addr) {
		// FIXME: add proper logging for this
		fmt.Println("interconnect: ignoring write to RAMSIZE register")
		return
	}

	// handle CACHECONTROL (FIXME: stub)
	if CACHE_CONTROL.Contains(addr) {
		fmt.Println("interconnect: unhandled write to CACHECONTROL register")
		return
	}

	// RAM
	if RAM_RANGE.Contains(addr) {
		inter.Ram.Store32(RAM_RANGE.Offset(addr), val)
		return
	}

	panicFmt("interconnect: unhandled Store32 of 0x%x into address 0x%x", val, addr)
}

func (inter *Interconnect) Store16(addr uint32, val uint16) {
	addr = MaskRegion(addr)
	if addr%2 != 0 {
		panicFmt("interconnect: unaligned Store16 into address 0x%x", addr)
	}

	panicFmt("interconnect: unhandled Store16 into address 0x%x", addr)
}

func MaskRegion(addr uint32) uint32 {
	return addr & REGION_MASK[addr>>29]
}
