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
	absAddr := MaskRegion(addr)
	if addr%4 != 0 {
		panicFmt("interconnect: unaligned Load32 address 0x%x\n", addr)
	}

	// handle ranges
	if BIOS_RANGE.Contains(absAddr) {
		return inter.Bios.Load32(BIOS_RANGE.Offset(absAddr))
	}
	if RAM_RANGE.Contains(absAddr) {
		return inter.Ram.Load32(RAM_RANGE.Offset(absAddr))
	}
	if IRQ_CONTROL.Contains(absAddr) {
		fmt.Printf(
			"interconnect: ignoring IRQCONTROL read at address 0x%x\n",
			addr,
		)
		return 0
	}
	if DMA_RANGE.Contains(absAddr) {
		fmt.Printf("interconnect: ignoring DMA read at 0x%x\n", addr)
		return 0
	}
	if GPU_RANGE.Contains(absAddr) {
		offset := GPU_RANGE.Offset(absAddr)
		fmt.Printf("interconnect: GPU read offset 0x%x\n", offset)
		switch offset {
		// GPUSTAT: set bit 28 to signal that the GPU is ready to recieve
		// DMA blocks
		case 4:
			return 0x10000000
		}
		return 0
	}

	// couldn't load address, panic
	panicFmt("interconnect: unhandled Load32 at address 0x%x", addr)
	return 0
}

func (inter *Interconnect) Load16(addr uint32) uint16 {
	absAddr := MaskRegion(addr)
	if addr%2 != 0 {
		panicFmt("interconnect: unaligned Load16 address 0x%x", addr)
	}

	if SPU_RANGE.Contains(absAddr) {
		fmt.Printf("interconnect: ignoring read from SPU register at 0x%x\n", addr)
		return 0
	}
	if RAM_RANGE.Contains(absAddr) {
		return inter.Ram.Load16(RAM_RANGE.Offset(absAddr))
	}
	if IRQ_CONTROL.Contains(absAddr) {
		fmt.Printf("interconnect: IRQ control read 0x%x", IRQ_CONTROL.Offset(absAddr))
		return 0
	}

	panicFmt("interconnect: unhandled Load16 at address 0x%x", addr)
	return 0
}

func (inter *Interconnect) Load8(addr uint32) byte {
	absAddr := MaskRegion(addr)

	if RAM_RANGE.Contains(absAddr) {
		return inter.Ram.Load8(RAM_RANGE.Offset(absAddr))
	}
	if BIOS_RANGE.Contains(absAddr) {
		return inter.Bios.Load8(BIOS_RANGE.Offset(absAddr))
	}
	if EXPANSION_1.Contains(absAddr) {
		// no expansion implemented
		fmt.Printf("interconnect: ignoring Load8 at 0x%x, no expansion implemented\n", addr)
		return 0xff
	}

	panicFmt("interconnect: unhandled Load8 at address 0x%x", addr)
	return 0
}

// Store 32 bit `val` into `addr`
func (inter *Interconnect) Store32(addr, val uint32) {
	absAddr := MaskRegion(addr)
	if addr%4 != 0 {
		panicFmt("interconnect: unaligned Store32 of 0x%x into address 0x%x", val, addr)
	}

	// handle MEMCONTROL
	if MEM_CONTROL.Contains(absAddr) {
		switch MEM_CONTROL.Offset(absAddr) {
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
	if RAM_SIZE.Contains(absAddr) {
		fmt.Printf("interconnect: ignoring write to RAMSIZE register 0x%x\n", addr)
		return
	}
	if CACHE_CONTROL.Contains(absAddr) {
		fmt.Printf("interconnect: unhandled write to CACHECONTROL register 0x%x\n", addr)
		return
	}
	if RAM_RANGE.Contains(absAddr) {
		inter.Ram.Store32(RAM_RANGE.Offset(absAddr), val)
		return
	}
	if IRQ_CONTROL.Contains(absAddr) {
		fmt.Printf(
			"interconnect: ignoring IRQCONTROL: 0x%x <- 0x%x\n",
			IRQ_CONTROL.Offset(absAddr), val,
		)
		return
	}
	if DMA_RANGE.Contains(absAddr) {
		fmt.Printf("interconnect: ignoring DMA write 0x%x <- 0x%x\n", addr, val)
		return
	}
	if GPU_RANGE.Contains(absAddr) {
		fmt.Printf("interconnect: GPU write 0x%x <- 0x%x\n", GPU_RANGE.Offset(absAddr), val)
		return
	}
	if TIMERS_RANGE.Contains(absAddr) {
		fmt.Printf("unhandled write to timer register %d <- 0x%x", TIMERS_RANGE.Offset(absAddr), val)
		return
	}

	panicFmt("interconnect: unhandled Store32 into address 0x%x <- 0x%x\n", addr, val)
}

// Store 16 bit `val` into `addr`
func (inter *Interconnect) Store16(addr uint32, val uint16) {
	absAddr := MaskRegion(addr)
	if addr%2 != 0 {
		panicFmt("interconnect: unaligned Store16 into address 0x%x", addr)
	}

	if RAM_RANGE.Contains(absAddr) {
		inter.Ram.Store16(RAM_RANGE.Offset(absAddr), val)
		return
	}
	if SPU_RANGE.Contains(absAddr) {
		fmt.Printf("interconnect: ignoring write to SPU register 0x%d\n", addr)
		return
	}
	if TIMERS_RANGE.Contains(absAddr) {
		fmt.Printf(
			"interconnect: ignoring write to timer register at offset 0x%x\n",
			TIMERS_RANGE.Offset(absAddr),
		)
		return
	}
	if IRQ_CONTROL.Contains(absAddr) {
		fmt.Printf("interconnect: IRQ control write 0x%x <- %d\n", IRQ_CONTROL.Offset(absAddr), val)
		return
	}

	panicFmt("interconnect: unhandled Store16 into address 0x%x <- 0x%x", addr, val)
}

func (inter *Interconnect) Store8(addr uint32, val uint8) {
	absAddr := MaskRegion(addr)

	if RAM_RANGE.Contains(absAddr) {
		inter.Ram.Store8(RAM_RANGE.Offset(absAddr), val)
		return
	}
	if EXPANSION_2.Contains(absAddr) {
		fmt.Printf(
			"interconnect: ignoring write to expansion 2 register 0x%x\n",
			EXPANSION_2.Offset(absAddr),
		)
		return
	}

	panicFmt("interconnect: unhandled store8 into address 0x%x", addr)
}

func MaskRegion(addr uint32) uint32 {
	return addr & REGION_MASK[addr>>29]
}
