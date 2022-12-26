package emulator

type BusState int

const (
	BUS_STATE_IDLE     BusState = iota // Bus is idle
	BUS_STATE_TRANSFER BusState = iota // Bus transaction in progress
	BUS_STATE_DSR      BusState = iota // Bus DSR is set
)

type Bus struct {
	State           BusState // Bus state
	DsrResponse     uint8    // DSR response
	Dsr             bool     // DSR
	TxDuration      uint64   // Transfer duration (cycles)
	RemainingCycles uint64   // Remaining DSR cycles
}

func (bus *Bus) IsBusy() bool {
	return bus.State != BUS_STATE_IDLE
}

func NewBus(state BusState) *Bus {
	return &Bus{State: state}
}
