package emulator

// State of the interrupt register
type IrqState struct {
	Status uint16 // Interrupt status
	Mask   uint16 // Interrupt mask
}

// Represents an interrupt state
type Interrupt uint16

const (
	INTERRUPT_VBLANK     Interrupt = 0 // GPU is in vertical blanking
	INTERRUPT_CDROM      Interrupt = 2 // CD-ROM controller
	INTERRUPT_DMA        Interrupt = 3 // DMA transfer complete
	INTERRUPT_TIMER0     Interrupt = 4 // Timer 0 interrupt
	INTERRUPT_TIMER1     Interrupt = 5 // Timer 0 interrupt
	INTERRUPT_TIMER2     Interrupt = 6 // Timer 0 interrupt
	INTERRUPT_PADMEMCARD Interrupt = 7 // Gamepad and memory card controllers
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
