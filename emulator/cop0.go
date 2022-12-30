package emulator

// Coprocessor 0: System Control
type Cop0 struct {
	SR    uint32 // Register 12: status register
	Cause uint32 // Register 13: cause register
	Epc   uint32 // Register 14: exception PC
}

// Creates a new Cop0 instance
func NewCop0() *Cop0 {
	return &Cop0{}
}

func (cop *Cop0) SetSR(sr uint32) {
	cop.SR = sr
}

func (cop *Cop0) SetCause(val uint32) {
	// triggers an interrupt
	cop.Cause = uint32(int64(cop.Cause) & ^0x300)
	cop.Cause |= val & 0x300
}

// Returns value of the cause register
func (cop *Cop0) GetCause(irqState *IrqState) uint32 {
	// bit 10 is the current external interrupt
	return cop.Cause | (oneIfTrue(irqState.Active()) << 10)
}

// Returns true if the cache is isolated
func (cop *Cop0) CacheIsolated() bool {
	return cop.SR&0x10000 != 0
}

// Returns the address of the exception handler
func (cop *Cop0) EnterException(cause Exception, pc uint32, inDelaySlot bool) uint32 {
	// Shift bits [5:0] of the SR two places to the left.
	// those bits are three pairs of Interrupt Enable/User Mode
	// bits behaving like a stack of 3 entries deep. Entering an
	// exception pushes a pair of zeroes by left shifting the stack
	// which disables interrupts and puts the CPU in kernel mode.
	// The original third entry is discarded (it's up to the kernel
	// to handle more than two recursive exception levels)
	mode := cop.SR & 0x3f
	cop.SR = uint32(int64(cop.SR) & ^0x3f)
	cop.SR |= (mode << 2) & 0x3f

	// update `CAUSE` register with exception code
	cop.Cause = uint32(int64(cop.Cause) & ^0x7c)
	cop.Cause |= uint32(cause) << 2

	if inDelaySlot {
		cop.Epc = pc - 4
		cop.Cause = 1 << 31
	} else {
		cop.Epc = pc
		cop.Cause = uint32(int64(cop.Cause) & ^(1 << 31))
	}

	// return exception handler
	if cop.SR&(1<<22) != 0 {
		return 0xbfc00180
	}
	return 0x80000080
}

// Discard the current state of the status register
func (cop *Cop0) ReturnFromException() {
	mode := cop.SR & 0x3f
	cop.SR = uint32(int64(cop.SR) & ^0xf)
	cop.SR |= mode >> 2
}

func (cop *Cop0) IrqEnabled() bool {
	return cop.SR&1 != 0
}

func (cop *Cop0) IrqActive(irqState *IrqState) bool {
	cause := cop.GetCause(irqState)

	// bits [8:9] contain two software interrupts
	pending := (cause & cop.SR) & 0x700

	return cop.IrqEnabled() && pending != 0
}
