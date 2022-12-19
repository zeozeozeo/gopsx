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
		panicFmt("interconnect: unaligned Load32 address 0x%x\n", addr)
	}

	// handle ranges
	if BIOS_RANGE.Contains(addr) {
		return inter.Bios.Load32(BIOS_RANGE.Offset(addr))
	}
	if RAM_RANGE.Contains(addr) {
		return inter.Ram.Load32(RAM_RANGE.Offset(addr))
	}
	if IRQ_CONTROL.Contains(addr) {
		fmt.Printf(
			"interconnect: ignoring IRQCONTROL read at offset 0x%x\n",
			IRQ_CONTROL.Offset(addr),
		)
		return 0
	}

	// couldn't load address, panic
	panicFmt("interconnect: unhandled Load32 at address 0x%x", addr)
	return 0 // shouldn't reach here, but the linter still wants me to put this
}

func (inter *Interconnect) Load8(addr uint32) byte {
	addr = MaskRegion(addr)

	if RAM_RANGE.Contains(addr) {
		return inter.Ram.Load8(RAM_RANGE.Offset(addr))
	}
	if BIOS_RANGE.Contains(addr) {
		return inter.Bios.Load8(BIOS_RANGE.Offset(addr))
	}
	if EXPANSION_1.Contains(addr) {
		// no expansion implemented
		fmt.Printf("interconnect: ignoring Load8 at 0x%x, no expansion implemented\n", addr)
		return 0xff
	}

	panicFmt("interconnect: unhandled Load8 at address 0x%x", addr)
	return 0
}

// Store 32 bit `val` into `addr`
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
			fmt.Printf("interconnect: unhandled write to MEMCONTROL register 0x%x\n", addr)
		}
		return
	}

	if RAM_SIZE.Contains(addr) {
		fmt.Printf("interconnect: ignoring write to RAMSIZE register 0x%x\n", addr)
		return
	}
	if CACHE_CONTROL.Contains(addr) {
		fmt.Printf("interconnect: unhandled write to CACHECONTROL register 0x%x\n", addr)
		return
	}
	if RAM_RANGE.Contains(addr) {
		inter.Ram.Store32(RAM_RANGE.Offset(addr), val)
		return
	}
	if IRQ_CONTROL.Contains(addr) {
		fmt.Printf(
			"interconnect: ignoring IRQCONTROL: 0x%x <- 0x%x\n",
			IRQ_CONTROL.Offset(addr), val,
		)
		return
	}

	panicFmt("interconnect: unhandled Store32 of 0x%x into address 0x%x", val, addr)
}

// Store 16 bit `val` into `addr`
func (inter *Interconnect) Store16(addr uint32, val uint16) {
	addr = MaskRegion(addr)
	if addr%2 != 0 {
		panicFmt("interconnect: unaligned Store16 into address 0x%x", addr)
	}

	if RAM_RANGE.Contains(addr) {
		inter.Ram.Store16(RAM_RANGE.Offset(addr), val)
		return
	}
	if SPU_RANGE.Contains(addr) {
		fmt.Printf("interconnect: ignoring write to SPU register 0x%d\n", addr)
		return
	}
	if TIMERS_RANGE.Contains(addr) {
		fmt.Printf(
			"interconnect: ignoring write to timer register at offset 0x%x\n",
			TIMERS_RANGE.Offset(addr),
		)
		return
	}

	panicFmt("interconnect: unhandled Store16 into address 0x%x", addr)
}

func (inter *Interconnect) Store8(addr uint32, val uint8) {
	addr = MaskRegion(addr)

	if RAM_RANGE.Contains(addr) {
		inter.Ram.Store8(RAM_RANGE.Offset(addr), val)
		return
	}
	if EXPANSION_2.Contains(addr) {
		fmt.Printf(
			"interconnect: ignoring write to expansion 2 register 0x%x\n",
			EXPANSION_2.Offset(addr),
		)
		return
	}

	panicFmt("interconnect: unhandled store8 into address 0x%x", addr)
}

func MaskRegion(addr uint32) uint32 {
	return addr & REGION_MASK[addr>>29]
}
