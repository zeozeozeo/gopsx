package emulator

// Global interconnect. It stores all of the peripherals
type Interconnect struct {
	Bios *BIOS // Basic input/output memory
}

// Creates a new interconnect instance
func NewInterconnect(bios *BIOS) *Interconnect {
	inter := &Interconnect{
		Bios: bios,
	}
	return inter
}

// Returns a 32bit little endian value at `addr`. Panics
// if the address does not exist
func (inter *Interconnect) Load32(addr uint32) uint32 {
	if BIOS_RANGE.Contains(addr) {
		return inter.Bios.Load32(BIOS_RANGE.Offset(addr))
	}

	// couldn't load address, panic
	panicFmt("interconnect: unhandled load32 at address 0x%x", addr)
	return 0 // shouldn't reach here, but the linter still wants me to put this
}
