package emulator

import "fmt"

// Used by the CD-ROM controller
type IrqCode uint8

const (
	IRQ_CODE_OK IrqCode = 3
)

// CD-ROM controller
type CdRom struct {
	Index    uint8 // Some registers can change depending on the index
	Params   *FIFO // FIFO storing the command arguments
	Response *FIFO // FIFO storing command responses
	IrqMask  uint8 // 5 bit interrupt mask
	IrqFlags uint8 // 5 bit interrupt flags
}

func NewCdRom() *CdRom {
	return &CdRom{
		Params:   NewFIFO(),
		Response: NewFIFO(),
	}
}

func (cdrom *CdRom) Status() uint8 {
	r := cdrom.Index

	// https://problemkaputt.de/psx-spx.htm#cdromcontrollerioports
	// TODO: XA-ADPCM fifo empty
	r |= 0 << 2
	r |= uint8(oneIfTrue(cdrom.Params.IsEmpty())) << 3
	r |= uint8(oneIfTrue(cdrom.Params.IsFull())) << 4
	r |= uint8(oneIfTrue(cdrom.Response.IsEmpty())) << 5
	// TODO: Data fifo empty
	r |= 0 << 6
	// TODO: Command/parameter transmission busy
	r |= 0 << 7

	return r
}

func (cdrom *CdRom) Irq() bool {
	return cdrom.IrqFlags&cdrom.IrqMask != 0
}

func (cdrom *CdRom) TriggerIrq(irq IrqCode) {
	if cdrom.IrqFlags != 0 {
		panic("cdrom: nested interrupt") // TODO
	}
	cdrom.IrqFlags = uint8(irq)
}

func (cdrom *CdRom) SetIndex(index uint8) {
	cdrom.Index = index & 3
}

func (cdrom *CdRom) AcknowledgeIrq(val uint8) {
	cdrom.IrqFlags &= ^val
}

func (cdrom *CdRom) SetIrqMask(val uint8) {
	cdrom.IrqMask = val & 0x1f
}

func (cdrom *CdRom) CommandGetStat() {
	if !cdrom.Params.IsEmpty() {
		// TODO
		panic("cdrom: invalid parameters for GetStat")
	}

	// FIXME: for now, just pretend that the CD tray is open
	cdrom.Response.Push(0x10)
	cdrom.TriggerIrq(IRQ_CODE_OK)
}

func (cdrom *CdRom) CommandTest() {
	if cdrom.Params.Length() != 1 {
		panicFmt(
			"cdrom: invalid number of parameters for Test (expected 1, got %d)",
			cdrom.Params.Length(),
		)
	}

	cmd := cdrom.Params.Pop()
	switch cmd {
	case 0x20:
		cdrom.TestVersion()
	default:
		panicFmt("cdrom: unhandled Test command 0x%x", cmd)
	}
}

func (cdrom *CdRom) TestVersion() {
	// values taken from Mednafen
	cdrom.Response.Push(0x97) // year
	cdrom.Response.Push(0x01) // month
	cdrom.Response.Push(0x10) // day
	cdrom.Response.Push(0xc2) // version
	cdrom.TriggerIrq(IRQ_CODE_OK)
}

func (cdrom *CdRom) PushParam(param uint8) {
	if cdrom.Params.IsFull() {
		panic("cdrom: attempted to push param to full FIFO")
	}
	cdrom.Params.Push(param)
}

func (cdrom *CdRom) Command(cmd uint8) {
	cdrom.Response.Clear()

	switch cmd {
	case 0x01:
		cdrom.CommandGetStat()
	case 0x19:
		cdrom.CommandTest()
	default:
		panicFmt("cdrom: unhandled command 0x%x", cmd)
	}

	cdrom.Params.Clear()
}

func (cdrom *CdRom) Load(size AccessSize, offset uint32) uint8 {
	if size != ACCESS_BYTE {
		panicFmt("cdrom: tried to load %d bytes (expected %d)", size, ACCESS_BYTE)
	}

	index := cdrom.Index

	switch offset {
	case 0:
		return cdrom.Status()
	case 1:
		if cdrom.Response.IsEmpty() {
			fmt.Println("cdrom: response FIFO is empty!")
		}
		return cdrom.Response.Pop()
	case 3:
		switch index {
		case 1:
			return cdrom.IrqFlags
		default:
			panic("cdrom: not implemented")
		}
	default:
		panic("cdrom: not implemented")
	}
}

func (cdrom *CdRom) Store(offset uint32, size AccessSize, val uint8, irqState *IrqState) {
	if size != ACCESS_BYTE {
		panicFmt("cdrom: tried to store %d bytes (expected %d)", size, ACCESS_BYTE)
	}

	index := cdrom.Index
	prevIrq := cdrom.Irq()

	switch offset {
	case 0:
		cdrom.SetIndex(val)
	case 1:
		switch index {
		case 0:
			cdrom.Command(val)
		default:
			panic("cdrom: not implemented")
		}
	case 2:
		switch index {
		case 0:
			cdrom.PushParam(val)
		case 1:
			cdrom.SetIrqMask(val)
		default:
			panic("cdrom: not implemented")
		}
	case 3:
		switch index {
		case 1:
			cdrom.AcknowledgeIrq(val & 0x1f)
			if val&0x40 != 0 {
				cdrom.Params.Clear()
			}
			if val&0xa0 != 0 {
				panic("cdrom: not implemented")
			}
		default:
			panic("cdrom: not implemented")
		}
	default:
		panic("cdrom: not implemented")
	}

	if !prevIrq && cdrom.Irq() {
		irqState.SetHigh(INTERRUPT_CDROM)
	}
}
