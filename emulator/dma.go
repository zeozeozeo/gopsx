package emulator

// Represents the 7 DMA ports
type Port uint32

const (
	PORT_MDEC_IN  Port = 0 // Macroblock decoder input
	PORT_MDEC_OUT Port = 1 // Macroblock decoder output
	PORT_GPU      Port = 2 // Graphics Processing Unit
	PORT_CDROM    Port = 3 // CD-ROM drive
	PORT_SPU      Port = 4 // Sound Processing Unit
	PORT_PIO      Port = 5 // Extension port
	PORT_OTC      Port = 6 // Used to clear the ordering table
)

func PortFromIndex(index uint32) Port {
	switch index {
	case 0:
		return PORT_MDEC_IN
	case 1:
		return PORT_MDEC_OUT
	case 2:
		return PORT_GPU
	case 3:
		return PORT_CDROM
	case 4:
		return PORT_SPU
	case 5:
		return PORT_PIO
	case 6:
		return PORT_OTC
	default:
		panicFmt("dma: invalid port %d", index)
		return 0
	}
}

// Direct Memory Access
type DMA struct {
	Control         uint32 // DMA control register
	IrqEn           bool   // Master IRQ enable
	ChannelIrqEn    uint8  // IRQ enable for individual channels
	ChannelIrqFlags uint8  // IRQ flags for individual channels
	// When set the interrupt is active unconditionally, even
	// if `IrqEn` is false
	ForceIrq bool
	// Bits [0:5] of the interrupt registers are RW but I don't
	// know what they're supposed to do so they're just sent back
	// untouched on reads
	IrqDummy uint8
	Channels [7]*Channel // The 7 channel instances
}

// Return a new reset DMA instance
func NewDMA() *DMA {
	dma := &DMA{
		Control: 0x07654321, // reset value from the Nocash PSX spec
	}

	// allocate channels
	for i := 0; i < len(dma.Channels); i++ {
		dma.Channels[i] = NewChannel()
	}

	return dma
}

// Set the control value
func (dma *DMA) SetControl(val uint32) {
	dma.Control = val
}

// Return the status of the DMA interrupt
func (dma *DMA) Irq() bool {
	channelIrq := dma.ChannelIrqFlags & dma.ChannelIrqEn
	return dma.ForceIrq || (dma.IrqEn && channelIrq != 0)
}

// Return the value of the interrupt register
func (dma *DMA) Interrupt() uint32 {
	var forceIrqVal uint32
	if dma.ForceIrq {
		forceIrqVal = 1
	}
	var irqEnVal uint32
	if dma.IrqEn {
		irqEnVal = 1
	}
	var irqVal uint32
	if dma.Irq() {
		irqVal = 1
	}

	var r uint32 = 0
	r |= uint32(dma.IrqDummy)
	r |= forceIrqVal << 15
	r |= uint32(dma.ChannelIrqEn) << 16
	r |= irqEnVal << 23
	r |= uint32(dma.ChannelIrqFlags) << 24
	r |= irqVal << 31
	return r
}

// Set the value of the interrupt register
func (dma *DMA) SetInterrupt(val uint32) {
	// unknown what bits [5:0] do
	dma.IrqDummy = uint8(val & 0x3f)
	dma.ForceIrq = (val>>15)&1 != 0
	dma.ChannelIrqEn = uint8((val >> 16) & 0x7f)
	dma.IrqEn = (val>>23)&1 != 0

	// writing 1 to a flag resets it
	ack := uint8((val >> 24) & 0x3f)
	dma.ChannelIrqFlags &= ^ack
}
