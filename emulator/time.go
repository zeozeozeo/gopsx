package emulator

// Keeps track of the emulation time
type TimeHandler struct {
	// Keeps track of the current execution time. It is measured in
	// the CPU clock at 33.8685MHz (~29.525960700946ns)
	Cycles     uint64
	TimeSheets []*TimeSheet
}

// Represents a TimeSheet index
type Peripheral uint32

const (
	PERIPHERAL_GPU Peripheral = 0 // Graphics Processing Unit
)

// Returns a new instance of TimeHandler
func NewTimeHandler() *TimeHandler {
	th := &TimeHandler{
		TimeSheets: []*TimeSheet{NewTimeSheet()},
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
	th.TimeSheets[from].NextSync = th.Cycles + delta
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
