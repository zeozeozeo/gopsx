package emulator

// State of the interrupt register
type IrqState struct {
	Status uint16 // Interrupt status
	Mask   uint16 // Interrupt mask
}

// Represents an interrupt state
type Interrupt uint16

const (
	INTERRUPT_VBLANK Interrupt = 0 // GPU is in vertical blanking
	INTERRUPT_DMA    Interrupt = 3 // DMA transfer complete
)

// Returns a new interrupt instance
func NewIrqState() *IrqState {
	return &IrqState{}
}

// Returns true if any interrupt is active
func (state *IrqState) Active() bool {
	return (state.Status & state.Mask) != 0
}

func (state *IrqState) Acknowledge(ack uint16) {
	state.Status &= ack
}

func (state *IrqState) SetMask(mask uint16) {
	state.Mask = mask
}

func (state *IrqState) SetHigh(interrupt Interrupt) {
	state.Status |= 1 << interrupt
}
