package emulator

import (
	"fmt"
)

// Global interconnect. It stores all of the peripherals
type Interconnect struct {
	Bios      *BIOS        // Basic input/output memory
	Ram       *RAM         // RAM
	Dma       *DMA         // Direct Memory Access
	Gpu       *GPU         // Graphics Processing Unit
	CacheCtrl CacheControl // Cache Control register
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
func NewInterconnect(bios *BIOS, ram *RAM, gpu *GPU) *Interconnect {
	inter := &Interconnect{
		Bios: bios,
		Ram:  ram,
		Dma:  NewDMA(),
		Gpu:  gpu,
	}
	return inter
}

// Load value at `addr`
func (inter *Interconnect) Load(addr uint32, size AccessSize, th *TimeHandler) interface{} {
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
	if ok, offset := DMA_RANGE.ContainsAndOffset(absAddr); ok {
		return accessSizeU32(size, inter.DmaReg(offset))
	}
	if ok, offset := GPU_RANGE.ContainsAndOffset(absAddr); ok {
		return inter.Gpu.Load(offset, th)
	}
	if ok, _ := TIMERS_RANGE.ContainsAndOffset(absAddr); ok {
		// fmt.Printf("inter: unhandled read from timers register %d\n", offset)
		// TODO
		return accessSizeU32(size, 0)
	}
	if SPU_RANGE.Contains(absAddr) {
		// ignore this for now (TODO)
		// fmt.Printf("inter: unhandled read from SPU register 0x%x\n", absAddr)
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
func (inter *Interconnect) Store(addr uint32, size AccessSize, val interface{}, th *TimeHandler) {
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
	if ok, offset := DMA_RANGE.ContainsAndOffset(absAddr); ok {
		inter.SetDmaReg(offset, accessSizeToU32(size, val))
		return
	}
	if ok, offset := GPU_RANGE.ContainsAndOffset(absAddr); ok {
		// fmt.Printf("inter: GPU write 0x%x <- 0x%x\n", offset, val)
		valU32 := accessSizeToU32(size, val)
		inter.Gpu.Store(offset, valU32, th)
		return
	}
	if ok, offset := TIMERS_RANGE.ContainsAndOffset(absAddr); ok {
		fmt.Printf("unhandled write to timer register %d <- 0x%x\n", offset, val)
		return
	}
	if SPU_RANGE.Contains(absAddr) {
		// ignore this for now (TODO)
		// fmt.Printf("inter: unhandled write to SPU register at 0x%x\n", addr)
		return
	}
	if CACHE_CONTROL.Contains(absAddr) {
		valU32 := accessSizeToU32(size, val)
		inter.CacheCtrl = CacheControl(valU32)
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

	panicFmt(
		"inter: unhandled write into address 0x%x (abs: 0x%x) <- 0x%x (%d bytes)",
		addr, absAddr, accessSizeToU32(size, val), size,
	)
}

// Shortcut for inter.Load(addr, ACCESS_WORD).(uint32)
func (inter *Interconnect) Load32(addr uint32, th *TimeHandler) uint32 {
	return inter.Load(addr, ACCESS_WORD, th).(uint32)
}

// Shortcut for inter.Load(addr, ACCESS_HALFWORD).(uint16)
func (inter *Interconnect) Load16(addr uint32, th *TimeHandler) uint16 {
	return inter.Load(addr, ACCESS_HALFWORD, th).(uint16)
}

// Shortcut for inter.Load(addr, ACCESS_BYTE).(byte)
func (inter *Interconnect) Load8(addr uint32, th *TimeHandler) byte {
	return inter.Load(addr, ACCESS_BYTE, th).(byte)
}

// Shortcut for inter.Store(addr, ACCESS_WORD, val)
func (inter *Interconnect) Store32(addr, val uint32, th *TimeHandler) {
	inter.Store(addr, ACCESS_WORD, val, th)
}

// Shortcut for inter.Store(addr, ACCESS_HALFWORD, val)
func (inter *Interconnect) Store16(addr uint32, val uint16, th *TimeHandler) {
	inter.Store(addr, ACCESS_HALFWORD, val, th)
}

// Shortcut for inter.Store(addr, ACCESS_BYTE, val)
func (inter *Interconnect) Store8(addr uint32, val byte, th *TimeHandler) {
	inter.Store(addr, ACCESS_BYTE, val, th)
}

func MaskRegion(addr uint32) uint32 {
	return addr & REGION_MASK[addr>>29]
}

// DMA register read
func (inter *Interconnect) DmaReg(offset uint32) uint32 {
	// the DMA uses 32 bit registers
	align := offset & 3
	offset = uint32(int64(offset) & ^3)

	major := (offset & 0x70) >> 4
	minor := offset & 0xf
	var res uint32

	switch {
	case major <= 6: // per-channel registers
		channel := inter.Dma.Channels[PortFromIndex(major)]
		switch minor {
		case 0:
			res = channel.Base
		case 4:
			res = channel.BlockControl()
		case 8:
			res = channel.Control()
		default:
			panicFmt("inter: unhandled DMA read at 0x%x", offset)
		}
	case major == 7: // common DMA registers
		switch minor {
		case 0:
			res = inter.Dma.Control
		case 4:
			res = inter.Dma.Interrupt()
		default:
			panicFmt("inter: unhandled DMA read at 0x%x", offset)
		}
	default:
		panicFmt("inter: unhandled DMA read at 0x%x", offset)
	}

	// byte and halfword reads fetch only a portion of the register
	return res >> (align * 8)
}

func (inter *Interconnect) SetDmaReg(offset, val uint32) {
	// byte and halfword writes are threated like word writes with the *entire*
	// Word value shifted by the alignment
	align := offset & 3
	val = val << (align * 8)
	offset = uint32(int64(offset) & ^3)

	major := (offset & 0x70) >> 4
	minor := offset & 0xf
	var isActive bool
	var port Port

	switch {
	case major <= 6: // per-channel registers
		port = PortFromIndex(major)
		channel := inter.Dma.Channels[port]

		switch minor {
		case 0:
			channel.SetBase(val)
		case 4:
			channel.SetBlockControl(val)
		case 8:
			channel.SetControl(val)
		default:
			panicFmt("inter: unhandled DMA write 0x%x <- 0x%x", offset, val)
		}

		isActive = channel.Active()
	case major == 7: // common DMA registers
		switch minor {
		case 0:
			inter.Dma.SetControl(val)
		case 4:
			inter.Dma.SetInterrupt(val)
		default:
			panicFmt("inter: unhandled DMA write 0x%x <- 0x%x", offset, val)
		}
		isActive = false
	default:
		panicFmt("inter: unhandled DMA write 0x%x <- 0x%x", offset, val)
	}

	if isActive {
		inter.DoDma(port)
	}
}

// Execute a DMA transfer for a port
func (inter *Interconnect) DoDma(port Port) {
	// DMA transfer has been started, for now just process
	// everything in one pass (no chopping or priority handling)

	channel := inter.Dma.Channels[port]
	switch channel.Sync {
	case SYNC_LINKED_LIST:
		inter.DoDmaLinkedList(port)
	default:
		inter.DoDmaBlock(port)
	}
}

// Emulates DMA transfer for Manual and Request synchronization modes
func (inter *Interconnect) DoDmaBlock(port Port) {
	channel := inter.Dma.Channels[port]

	var addrStep uint32 = 4
	var isReverse bool

	switch channel.Step {
	case STEP_INCREMENT:
		isReverse = false // +=
	case STEP_DECREMENT:
		isReverse = true // -=
	}

	addr := channel.Base

	// transfer size in words
	valid, remsz := channel.TransferSize()
	if !valid {
		// shouldn't happen since we shouldn't reach this if we're in linked list mode
		panic("inter: couldn't figure out DMA block transfer size (linked mode)")
	}

	for remsz > 0 {
		// if the address is bogus, Mednafen masks it like this,
		// maybe the RAM address wraps and the two LSBs are ignored,
		// seems reasonable enough
		curAddr := addr & 0x1ffffc

		switch channel.Direction {
		case DIRECTION_FROM_RAM:
			srcWord := inter.Ram.Load32(curAddr)
			switch port {
			case PORT_GPU:
				inter.Gpu.GP0(srcWord)
			default:
				panicFmt("inter: unhandled DMA destination port %d", port)
			}
		case DIRECTION_TO_RAM:
			var srcWord uint32
			switch port {
			case PORT_OTC: // clear ordering table
				switch remsz {
				case 1:
					// last entry contains the end of table marker
					srcWord = 0xffffff
				default:
					// pointer to the previous entry
					srcWord = (addr - 4) & 0x1fffff
				}
			default:
				panicFmt("inter: unhandled DMA source port %d", port)
			}

			inter.Ram.Store32(curAddr, srcWord)
		}

		if isReverse {
			addr -= addrStep
		} else {
			addr += addrStep
		}
		remsz--
	}

	channel.Done()
}

// Emulate DMA transfer for linked list synchronization mode
func (inter *Interconnect) DoDmaLinkedList(port Port) {
	channel := inter.Dma.Channels[port]
	addr := channel.Base & 0x1ffffc

	if channel.Direction == DIRECTION_TO_RAM {
		panic("inter: invalid DMA direction for linked list mode")
	}

	// i don't know if the DMA even supports linked list mode for anything
	// besides the GPU
	if port != PORT_GPU {
		panicFmt("inter: attempted DMA linked list on port %d (expected %d)", port, PORT_GPU)
	}

	for {
		// in linked list mode, each entry starts with a "header" word.
		// The high byte contains the number of words in the "packet"
		// (not counting the header word)
		header := inter.Ram.Load32(addr)
		remsz := header >> 24

		for remsz > 0 {
			addr = (addr + 4) & 0x1ffffc
			command := inter.Ram.Load32(addr)

			// send command to the GPU
			inter.Gpu.GP0(command)

			remsz--
		}

		// the end of table marker is ususally 0xffffff, but mednafen
		// only checks for the MSB so maybe that's what the hardware does?
		// Since this bit is not part of any valid address it makes some sense.
		// TODO: test this
		if header&0x800000 != 0 {
			break
		}

		addr = header & 0x1ffffc
	}

	channel.Done()
}

// Synchronizes all peripherals
func (inter *Interconnect) Sync(th *TimeHandler) {
	if th.NeedsSync(PERIPHERAL_GPU) {
		inter.Gpu.Sync(th)
	}
}

// Load instruction at `pc`
func (inter *Interconnect) LoadInstruction(pc uint32) uint32 {
	absAddr := MaskRegion(pc)

	// TODO: currently only loads instructions from RAM and the BIOS

	if ok, offset := RAM_RANGE.ContainsAndOffset(absAddr); ok {
		return inter.Ram.Load32(offset)
	}
	if ok, offset := BIOS_RANGE.ContainsAndOffset(absAddr); ok {
		return inter.Bios.Load32(offset)
	}

	panicFmt("inter: unhandled instruction load at address 0x%x", pc)
	return 0
}
