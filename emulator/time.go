package emulator

import "math"

// Keeps track of the emulation time
type TimeHandler struct {
	// Keeps track of the current execution time. It is measured in
	// the CPU clock at 33.8685MHz (~29.525960700946ns)
	Cycles     uint64
	NextSync   uint64 // Next time a peripheral needs to be synchronized
	TimeSheets [6]*TimeSheet
}

// Represents a TimeSheet index
type Peripheral uint32

const (
	PERIPHERAL_GPU        Peripheral = iota // Graphics Processing Unit
	PERIPHERAL_TIMER0     Peripheral = iota // Timer 0
	PERIPHERAL_TIMER1     Peripheral = iota // Timer 1
	PERIPHERAL_TIMER2     Peripheral = iota // Timer 2
	PERIPHERAL_PADMEMCARD Peripheral = iota // Gamepad and memory card controller
	PERIPHERAL_CDROM      Peripheral = iota // CD-ROM controller
)

// Returns a new instance of TimeHandler
func NewTimeHandler() *TimeHandler {
	th := &TimeHandler{
		NextSync: math.MaxUint64,
	}
	for i := 0; i < len(th.TimeSheets); i++ {
		th.TimeSheets[i] = NewTimeSheet()
	}
	return th
}

// Advance the current time by `cycles`
func (th *TimeHandler) Tick(cycles uint64) {
	th.Cycles += cycles
}

// Synchronizes a peripheral
func (th *TimeHandler) Sync(from Peripheral) uint64 {
	return th.TimeSheets[from].Sync(th.Cycles)
}

func (th *TimeHandler) SetNextSyncDelta(from Peripheral, delta uint64) {
	at := th.Cycles + delta
	th.TimeSheets[from].NextSync = at

	if at < th.NextSync {
		th.NextSync = at
	}
}

func (th *TimeHandler) MaybeSetNextSync(from Peripheral, at uint64) {
	sheet := th.TimeSheets[from]

	if sheet.NextSync > at {
		sheet.NextSync = at
	}
}

func (th *TimeHandler) MaybeSetNextSyncDelta(from Peripheral, delta uint64) {
	at := th.Cycles + delta
	th.MaybeSetNextSync(from, at)
}

// Called when there's no event scheduled
func (th *TimeHandler) RemoveNextSync(from Peripheral) {
	th.TimeSheets[from].NextSync = math.MaxUint64
}

// Returns true if a peripheral needs to be synchronized
func (th *TimeHandler) ShouldSync() bool {
	return th.NextSync <= th.Cycles
}

func (th *TimeHandler) UpdatePendingSync() {
	// find minimum next sync value
	var min uint64 = math.MaxUint64
	for _, sheet := range th.TimeSheets {
		if sheet.NextSync < min {
			min = sheet.NextSync
		}
	}

	th.NextSync = min
}

// Returns true if the peripheral reached the time of the next forced
// synchronization
func (th *TimeHandler) NeedsSync(from Peripheral) bool {
	return th.TimeSheets[from].NeedsSync(th.Cycles)
}

// Keeps track of synchronization of different peripherals
type TimeSheet struct {
	LastSync uint64 // Time of the last synchronization
	NextSync uint64 // Date of the next synchronization
}

// Returns a new TimeSheet instance
func NewTimeSheet() *TimeSheet {
	return &TimeSheet{}
}

// Set the time sheet to the current time and return the time
// since the last synchronization
func (sheet *TimeSheet) Sync(cycles uint64) uint64 {
	delta := cycles - sheet.LastSync
	sheet.LastSync = cycles
	return delta
}

// Returns true if the peripheral reached `NextSync`
func (sheet *TimeSheet) NeedsSync(cycles uint64) bool {
	return sheet.NextSync <= cycles
}

type FracCycles uint64

// The amount of fixed point fractional bits
const FRAC_CYCLES_FRAC_BITS uint64 = 16

func FracCyclesFromFixed(fixed uint64) FracCycles {
	return FracCycles(fixed)
}

func FracCyclesFromCycles(cycles uint64) FracCycles {
	return FracCycles(cycles << FRAC_CYCLES_FRAC_BITS)
}

func FracCyclesFromF32(val float32) FracCycles {
	precision := float32(1 << FRAC_CYCLES_FRAC_BITS)
	return FracCycles(uint64(val * precision))
}

func (fc FracCycles) GetFixed() uint64 {
	return uint64(fc)
}

func (fc FracCycles) Add(val FracCycles) FracCycles {
	return FracCycles(fc.GetFixed() + val.GetFixed())
}

func (fc FracCycles) Multiply(val FracCycles) FracCycles {
	v := fc.GetFixed() * val.GetFixed()
	// the shift amount is doubled after multiplication
	return FracCycles(v >> FRAC_CYCLES_FRAC_BITS)
}

func (fc FracCycles) Divide(denominator FracCycles) FracCycles {
	numerator := fc.GetFixed() << FRAC_CYCLES_FRAC_BITS
	return FracCycles(numerator / denominator.GetFixed())
}

func (fc FracCycles) Ceil() uint64 {
	shift := FRAC_CYCLES_FRAC_BITS
	var align uint64 = (1 << shift) - 1
	return (uint64(fc) + align) >> shift
}
