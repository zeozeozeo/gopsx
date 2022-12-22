package emulator

// Represents the value of the status register
type StatusRegister uint32

// Returns true if the cache is isolated
func (sr StatusRegister) CacheIsolated() bool {
	return uint32(sr)&0x10000 != 0
}

// Returns the address for the exception handler depending on the BEV bit
func (sr StatusRegister) ExceptionHandler() uint32 {
	if uint32(sr)&(1<<22) != 0 {
		return 0xbfc00180
	}
	return 0x80000080
}

// Shift bits [5:0] of the SR two places to the left.
// those bits are three pairs of Interrupt Enable/User Mode
// bits behaving like a stack of 3 entries deep. Entering an
// exception pushes a pair of zeroes by left shifting the stack
// which disables interrupts and puts the CPU in kernel mode.
// The original third entry is discarded (it's up to the kernel
// to handle more than two recursive exception levels)
func (sr *StatusRegister) EnterException() {
	mode := *sr & 0x3f
	*sr = StatusRegister(uint32(int32(*sr) & ^0x3f))
	*sr |= (mode << 2) & 0x3f
}

// Discard the current state of the status register
func (sr *StatusRegister) ReturnFromException() {
	mode := *sr & 0x3f
	*sr = StatusRegister(uint32(int32(*sr) & ^0x3f))
	*sr |= mode >> 2
}
