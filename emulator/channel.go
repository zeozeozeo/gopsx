package emulator

// DMA transfer direction (To/From RAM)
type Direction uint32

const (
	DIRECTION_TO_RAM   Direction = 0
	DIRECTION_FROM_RAM Direction = 1
)

// DMA transfer step
type Step uint32

const (
	STEP_INCREMENT Step = 0
	STEP_DECREMENT Step = 1
)

// DMA transfer synchronization mode
type Sync uint32

const (
	// Transfer starts when the CPU writes to the Trigger bit and transfers
	// everything at once
	SYNC_MANUAL Sync = 0
	// Sync blocks to DMA requests
	SYNC_REQUEST Sync = 1
	// Used to transfer GPU command lists
	SYNC_LINKED_LIST Sync = 2
)

type Channel struct {
	Enable     bool
	Direction  Direction
	Step       Step
	Sync       Sync
	Trigger    bool   // Used to start the DMA transfer when `Sync` is `SYNC_MANUAL`
	Chop       bool   // If true, the DMA "chops" the transfer and lets the CPU run in the gaps
	ChopDmaSz  uint8  // Chopping DMA window size (log2 number of words)
	ChopCpuSz  uint8  // Chopping CPU window size (log2 number of cycles)
	Dummy      uint8  // Unknown 2 RW bits in configuration register
	Base       uint32 // DMA start address
	BlockSize  uint16 // Size of a block in words
	BlockCount uint16 // Block count, only used when `Sync` is `SYNC_REQUEST`
}

// Create a new channel instance
func NewChannel() *Channel {
	return &Channel{
		Direction: DIRECTION_TO_RAM,
		Step:      STEP_INCREMENT,
		Sync:      SYNC_MANUAL,
	}
}

func (ch *Channel) Control() uint32 {
	var r uint32 = 0
	r |= uint32(ch.Direction) << 0
	r |= uint32(ch.Step) << 1
	r |= oneIfTrue(ch.Chop) << 8
	r |= uint32(ch.Sync) << 9
	r |= uint32(ch.ChopDmaSz) << 16
	r |= uint32(ch.ChopCpuSz) << 20
	r |= oneIfTrue(ch.Enable) << 24
	r |= oneIfTrue(ch.Trigger) << 28
	r |= uint32(ch.Dummy) << 29

	return r
}

func (ch *Channel) SetControl(val uint32) {
	if val&1 != 0 {
		ch.Direction = DIRECTION_FROM_RAM
	} else {
		ch.Direction = DIRECTION_TO_RAM
	}

	if (val>>1)&1 != 0 {
		ch.Step = STEP_DECREMENT
	} else {
		ch.Step = STEP_INCREMENT
	}

	ch.Chop = (val>>8)&1 != 0

	syncMode := (val >> 9) & 3
	switch syncMode {
	case 0:
		ch.Sync = SYNC_MANUAL
	case 1:
		ch.Sync = SYNC_REQUEST
	case 2:
		ch.Sync = SYNC_LINKED_LIST
	default:
		panicFmt("channel: unknown DMA sync mode %d", syncMode)
	}

	ch.ChopDmaSz = uint8((val >> 16) & 7)
	ch.ChopCpuSz = uint8((val >> 20) & 7)

	ch.Enable = (val>>24)&1 != 0
	ch.Trigger = (val>>28)&1 != 0
	ch.Dummy = uint8((val >> 29) & 3)
}

// Set the channel base address. Only bits [0:23] are significant, so
// only 16MB are addressable by the DMA
func (ch *Channel) SetBase(val uint32) {
	ch.Base = val & 0xffffff
}

// Return value of the Block Control register
func (ch *Channel) BlockControl() uint32 {
	bs := uint32(ch.BlockSize)
	bc := uint32(ch.BlockCount)
	return (bc << 16) | bs
}

// Set value of the Block Control register
func (ch *Channel) SetBlockControl(val uint32) {
	ch.BlockSize = uint16(val)
	ch.BlockCount = uint16(val >> 16)
}

// Return true if the channel has been started
func (ch *Channel) Active() bool {
	// in manual sync mode the CPU must set the `Trigger` bit to
	// start the transfer
	trigger := true
	if ch.Sync == SYNC_MANUAL {
		trigger = ch.Trigger
	}

	return ch.Enable && trigger
}

// Returns the DMA transfer size in bytes. `valid` is false for linked
// list mode
func (ch *Channel) TransferSize() (valid bool, size uint32) {
	bs := uint32(ch.BlockSize)
	bc := uint32(ch.BlockCount)

	switch ch.Sync {
	// for manual mode, only the block size is used
	case SYNC_MANUAL:
		return true, bs
	// in DMA request mode we must transfer `bc` blocks
	case SYNC_REQUEST:
		return true, bc * bs
	// in linked list mode the size is not known ahead of time;
	// we stop when we encounter the end of list marker (0xffffff)
	case SYNC_LINKED_LIST:
		return false, 0
	}
	return false, 0
}

// Set the channel status to `completed` state
func (ch *Channel) Done() {
	ch.Enable = false
	ch.Trigger = false
	// FIXME: need to set the correct value for the other fields (in particular interrupts)
}
