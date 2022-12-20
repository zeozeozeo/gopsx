package emulator

import (
	"fmt"
)

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

// Load value at `addr`
func (inter *Interconnect) Load(addr uint32, size AccessSize) interface{} {
	absAddr := MaskRegion(addr)

	if ok, offset := RAM_RANGE.ContainsAndOffset(absAddr); ok {
		return inter.Ram.Load(offset, size)
	}
	if ok, offset := BIOS_RANGE.ContainsAndOffset(absAddr); ok {
		return inter.Bios.Load(offset, size)
	}
	if ok, offset := IRQ_CONTROL.ContainsAndOffset(absAddr); ok {
		fmt.Printf("inter: IRQ control read %d\n", offset)
		return accessSizeU32(size, 0)
	}
	if DMA_RANGE.Contains(absAddr) {
		fmt.Printf("interconnect: ignoring DMA read at 0x%x\n", addr)
		return accessSizeU32(size, 0)
	}
	if ok, offset := GPU_RANGE.ContainsAndOffset(absAddr); ok {
		fmt.Printf("inter: GPU read offset 0x%x\n", offset)
		switch offset {
		// GPUSTAT: set bit 28 to signal that the GPU is ready to recieve
		// DMA blocks
		case 4:
			return accessSizeU32(size, 0x10000000)
		}
		return accessSizeU32(size, 0)
	}
	if ok, offset := TIMERS_RANGE.ContainsAndOffset(absAddr); ok {
		fmt.Printf("inter: unhandled read from timers register %d\n", offset)
		return accessSizeU32(size, 0)
	}
	if SPU_RANGE.Contains(absAddr) {
		fmt.Printf("inter: unhandled read from SPU register 0x%x\n", absAddr)
		return accessSizeU32(size, 0)
	}
	if EXPANSION_1.Contains(absAddr) {
		fmt.Printf("inter: ignoring read from expansion 1 0x%x\n", absAddr)
		return accessSizeU32(size, 0)
	}

	panicFmt("inter: unhandled load at address 0x%x", addr)
	return accessSizeU32(size, 0)
}

// Write value into `addr`
func (inter *Interconnect) Store(addr uint32, size AccessSize, val interface{}) {
	absAddr := MaskRegion(addr)

	if ok, offset := RAM_RANGE.ContainsAndOffset(absAddr); ok {
		inter.Ram.Store(offset, size, val)
		return
	}
	if ok, offset := MEM_CONTROL.ContainsAndOffset(absAddr); ok {
		valU32 := accessSizeToU32(size, val)
		switch offset {
		case 0: // expansion 1 base address
			if valU32 != 0x1f000000 {
				panicFmt("inter: bad expansion 1 base address 0x%x", addr)
			}
		case 4: // expansion 2 base address
			if valU32 != 0x1f802000 {
				panicFmt("inter: bad expansion 2 base address 0x%x", addr)
			}
		default:
			fmt.Printf("inter: unhandled write to MEM_CONTROL register 0x%x\n", addr)
		}
		return
	}
	if ok, offset := IRQ_CONTROL.ContainsAndOffset(addr); ok {
		fmt.Printf("inter: ignoring IRQCONTROL: 0x%x <- 0x%x\n", offset, val)
		return
	}
	if DMA_RANGE.Contains(absAddr) {
		fmt.Printf("inter: ignoring DMA write 0x%x <- 0x%x\n", addr, val)
		return
	}
	if ok, offset := GPU_RANGE.ContainsAndOffset(absAddr); ok {
		fmt.Printf("inter: GPU write 0x%x <- 0x%x\n", offset, val)
		return
	}
	if ok, offset := TIMERS_RANGE.ContainsAndOffset(absAddr); ok {
		fmt.Printf("unhandled write to timer register %d <- 0x%x", offset, val)
		return
	}
	if SPU_RANGE.Contains(absAddr) {
		fmt.Printf("inter: unhandled write to SPU register at 0x%x\n", addr)
		return
	}
	if CACHE_CONTROL.Contains(absAddr) {
		fmt.Printf("inter: unhandled write to CACHE_CONTROL at 0x%x\n", addr)
		return
	}
	if RAM_SIZE.Contains(absAddr) {
		// ignore writes to this address
		return
	}
	if ok, offset := EXPANSION_2.ContainsAndOffset(addr); ok {
		fmt.Printf("inter: unhandled write to EXPANSION 2 register %d\n", offset)
		return
	}

	panicFmt("inter: unhandled write into address 0x%x <- 0x%x", addr, accessSizeToU32(size, val))
}

// Shortcut for inter.Load(addr, ACCESS_WORD).(uint32)
func (inter *Interconnect) Load32(addr uint32) uint32 {
	return inter.Load(addr, ACCESS_WORD).(uint32)
}

// Shortcut for inter.Load(addr, ACCESS_HALFWORD).(uint16)
func (inter *Interconnect) Load16(addr uint32) uint16 {
	return inter.Load(addr, ACCESS_HALFWORD).(uint16)
}

// Shortcut for inter.Load(addr, ACCESS_BYTE).(byte)
func (inter *Interconnect) Load8(addr uint32) byte {
	return inter.Load(addr, ACCESS_BYTE).(byte)
}

// Shortcut for inter.Store(addr, ACCESS_WORD, val)
func (inter *Interconnect) Store32(addr, val uint32) {
	inter.Store(addr, ACCESS_WORD, val)
}

// Shortcut for inter.Store(addr, ACCESS_HALFWORD, val)
func (inter *Interconnect) Store16(addr uint32, val uint16) {
	inter.Store(addr, ACCESS_HALFWORD, val)
}

// Shortcut for inter.Store(addr, ACCESS_BYTE, val)
func (inter *Interconnect) Store8(addr uint32, val byte) {
	inter.Store(addr, ACCESS_BYTE, val)
}

func MaskRegion(addr uint32) uint32 {
	return addr & REGION_MASK[addr>>29]
}
