package emulator

import "fmt"

// Used by the CD-ROM controller
type IrqCode uint8

const (
	IRQ_CODE_SECTOR_READY IrqCode = 1 // CD sector is ready
	IRQ_CODE_DONE         IrqCode = 1 // Command successful (2nd response)
	IRQ_CODE_OK           IrqCode = 3 // Command successful (1st response)
	IRQ_CODE_ERROR        IrqCode = 5 // Invalid command, etc.
)

type CommandControllerState int

const (
	// Controller is idle
	CMD_STATE_IDLE CommandControllerState = iota
	// Controller is making a command or waiting for a return value
	CMD_STATE_RXPENDING CommandControllerState = iota
	// Transaction is done, but we're still waiting for an interrupt
	CMD_STATE_IRQ_PENDING CommandControllerState = iota
)

type CommandState struct {
	State              CommandControllerState
	RxPendingDelay     uint32  // For CMD_STATE_RXPENDING
	RxPendingIrqDelay  uint32  // For CMD_STATE_RXPENDING
	RxPendingIrqCode   IrqCode // For CMD_STATE_RXPENDING
	RxPendingFifo      *FIFO   // For CMD_STATE_RXPENDING (response)
	IrqPendingIrqDelay uint32  // For CMD_STATE_IRQ_PENDING
	IrqPendingIrqCode  IrqCode // For CMD_STATE_IRQ_PENDING
}

func NewCommandState() *CommandState {
	return &CommandState{
		State: CMD_STATE_IDLE,
	}
}

func (cmd *CommandState) IsIdle() bool {
	return cmd.State == CMD_STATE_IDLE
}

type CDRomReadState int

const (
	READ_STATE_IDLE    CDRomReadState = iota
	READ_STATE_READING CDRomReadState = iota
)

// CD-ROM data read state
type ReadState struct {
	State CDRomReadState
	Delay uint32 // For READ_STATE_READING
}

func NewReadState() *ReadState {
	return &ReadState{
		State: READ_STATE_IDLE,
	}
}

func (rstate *ReadState) IsIdle() bool {
	return rstate.State == READ_STATE_IDLE
}

type CmdHandlerFunc func()

// CD-ROM controller
type CdRom struct {
	Index         uint8          // Some registers can change depending on the index
	Params        *FIFO          // FIFO storing the command arguments
	Response      *FIFO          // FIFO storing command responses
	IrqMask       uint8          // 5 bit interrupt mask
	IrqFlags      uint8          // 5 bit interrupt flags
	CmdState      *CommandState  // Command state
	ReadState     *ReadState     // Read state
	OnAcknowledge CmdHandlerFunc // Command handler
	Disc          *Disc          // Currently loaded disc (can be nil)
	SeekTarget    Msf            // Next seek command target
	Position      Msf            // Read position
	DoubleSpeed   bool           // If true, 150 sectors per second, else 75 sectors
	RxSector      *XaSector      // RX buffer sector
	RxActive      bool           // Whether the data RX buffer is active
	RxIndex       uint16         // Index of the next RX sector byte
}

// Disc can be nil
func NewCdRom(disc *Disc) *CdRom {
	cdrom := &CdRom{
		Params:     NewFIFO(),
		Response:   NewFIFO(),
		CmdState:   NewCommandState(),
		ReadState:  NewReadState(),
		Disc:       disc,
		SeekTarget: NewMsf(),
		Position:   NewMsf(),
		RxSector:   NewXaSector(),
	}
	cdrom.OnAcknowledge = cdrom.AckIdle
	return cdrom
}

func (cdrom *CdRom) Status() uint8 {
	r := cdrom.Index

	// https://problemkaputt.de/psx-spx.htm#cdromcontrollerioports
	// TODO: XA-ADPCM fifo empty
	r |= 0 << 2
	r |= uint8(oneIfTrue(cdrom.Params.IsEmpty())) << 3
	r |= uint8(oneIfTrue(cdrom.Params.IsFull())) << 4
	r |= uint8(oneIfTrue(cdrom.Response.IsEmpty())) << 5
	// TODO: Data fifo empty
	r |= 0 << 6
	// Command/parameter transmission busy
	if cdrom.CmdState.State == CMD_STATE_RXPENDING {
		r |= 1 << 7
	}

	return r
}

func (cdrom *CdRom) Irq() bool {
	return cdrom.IrqFlags&cdrom.IrqMask != 0
}

func (cdrom *CdRom) TriggerIrq(irq IrqCode, irqState *IrqState) {
	if cdrom.IrqFlags != 0 {
		panic("cdrom: nested interrupt") // TODO
	}

	prevIrq := cdrom.Irq()
	cdrom.IrqFlags = uint8(irq)

	if !prevIrq && cdrom.Irq() {
		irqState.SetHigh(INTERRUPT_CDROM)
	}
}

func (cdrom *CdRom) SetIndex(index uint8) {
	cdrom.Index = index & 3
}

func (cdrom *CdRom) AcknowledgeIrq(val uint8) {
	cdrom.IrqFlags &= ^val

	if cdrom.IrqFlags == 0 {
		if !cdrom.CmdState.IsIdle() {
			panic("cdrom: IRQ acknowledge while controller is busy")
		}

		onAck := cdrom.OnAcknowledge
		cdrom.OnAcknowledge = cdrom.AckIdle
		onAck()
	}
}

func (cdrom *CdRom) SetIrqMask(val uint8) {
	cdrom.IrqMask = val & 0x1f
}

func (cdrom *CdRom) CommandGetStat() {
	if !cdrom.Params.IsEmpty() {
		// TODO
		panic("cdrom: invalid parameters for GetStat")
	}

	response := NewFIFO()
	response.Push(cdrom.DriveStatus())

	var rxDelay uint32
	if cdrom.Disc != nil {
		rxDelay = 24000 // average delay with game disc
	} else {
		rxDelay = 17000 // average delay with tray open
	}

	state := cdrom.CmdState
	state.State = CMD_STATE_RXPENDING
	state.RxPendingDelay = rxDelay
	state.RxPendingIrqDelay = rxDelay + 5401
	state.RxPendingIrqCode = IRQ_CODE_OK
	state.RxPendingFifo = response
}

func (cdrom *CdRom) DriveStatus() uint8 {
	if cdrom.Disc != nil {
		var r uint8
		isReading := !cdrom.ReadState.IsIdle()

		r |= 1 << 1 // motor on
		r |= uint8(oneIfTrue(isReading)) << 5
		return r
	}
	// no disc, pretend that tray is open
	return 0x10
}

func (cdrom *CdRom) CommandTest() {
	if cdrom.Params.Length() != 1 {
		panicFmt(
			"cdrom: invalid number of parameters for Test (expected 1, got %d)",
			cdrom.Params.Length(),
		)
	}

	cmd := cdrom.Params.Pop()
	switch cmd {
	case 0x20:
		cdrom.TestVersion()
	default:
		panicFmt("cdrom: unhandled Test command 0x%x", cmd)
	}
}

func (cdrom *CdRom) TestVersion() {
	// values taken from Mednafen
	response := NewFIFOFromBytes([]byte{
		0x97,
		0x01,
		0x10,
		0xc2,
	})

	var rxDelay uint32
	if cdrom.Disc != nil {
		rxDelay = 21000
	} else {
		rxDelay = 29000
	}

	state := cdrom.CmdState
	state.State = CMD_STATE_RXPENDING
	state.RxPendingDelay = rxDelay
	state.RxPendingIrqDelay = rxDelay + 9711
	state.RxPendingIrqCode = IRQ_CODE_OK
	state.RxPendingFifo = response
}

func (cdrom *CdRom) PushParam(param uint8) {
	if cdrom.Params.IsFull() {
		panic("cdrom: attempted to push param to full FIFO")
	}
	cdrom.Params.Push(param)
}

func (cdrom *CdRom) Command(cmd uint8, irqState *IrqState, th *TimeHandler) {
	if !cdrom.CmdState.IsIdle() {
		panic("cdrom: recieved command while controller is busy")
	}

	cdrom.Response.Clear()
	fmt.Printf("cdrom: cdrom command 0x%x\n", cmd)

	var handler CmdHandlerFunc
	switch cmd {
	case 0x01:
		handler = cdrom.CommandGetStat
	case 0x02:
		handler = cdrom.CommandSetLoc
	case 0x06:
		handler = cdrom.CommandReadN
	case 0x09:
		handler = cdrom.CommandPause
	case 0x0e:
		handler = cdrom.CommandSetMode
	case 0x15:
		handler = cdrom.CommandSeekL
	case 0x1a:
		handler = cdrom.CommandGetId
	case 0x1e:
		handler = cdrom.CommandReadToc
	case 0x19:
		handler = cdrom.CommandTest
	default:
		panicFmt("cdrom: unhandled command 0x%x", cmd)
	}

	if cdrom.IrqFlags == 0 {
		// we already acknowledged the previous command
		handler()

		if cdrom.CmdState.State == CMD_STATE_RXPENDING {
			// schedule an interrupt
			th.SetNextSyncDelta(PERIPHERAL_CDROM, uint64(cdrom.CmdState.RxPendingIrqDelay))
		}
	} else {
		// execute this command after the current one is acknowledged
		cdrom.OnAcknowledge = handler
	}

	if cdrom.ReadState.State == READ_STATE_READING {
		th.SetNextSyncDeltaIfCloser(PERIPHERAL_CDROM, uint64(cdrom.ReadState.Delay))
	}

	cdrom.Params.Clear()
}

// Read table of contents
func (cdrom *CdRom) CommandReadToc() {
	cdrom.OnAcknowledge = cdrom.AckReadToc

	state := cdrom.CmdState
	state.State = CMD_STATE_RXPENDING
	state.RxPendingDelay = 45000
	state.RxPendingIrqDelay = 45000 + 5401
	state.RxPendingIrqCode = IRQ_CODE_OK
	state.RxPendingFifo = NewFIFOFromBytes([]byte{cdrom.DriveStatus()})
}

func (cdrom *CdRom) AckReadToc() {
	var rxDelay uint32
	if cdrom.Disc != nil {
		rxDelay = 16000000 // ~0.5 seconds
	} else {
		rxDelay = 11000
	}

	state := cdrom.CmdState
	state.State = CMD_STATE_RXPENDING
	state.RxPendingDelay = rxDelay
	state.RxPendingIrqDelay = rxDelay + 1859
	state.RxPendingIrqCode = IRQ_CODE_DONE
	state.RxPendingFifo = NewFIFOFromBytes([]byte{cdrom.DriveStatus()})
}

// Read the CD-ROM's identification string
func (cdrom *CdRom) CommandGetId() {
	state := cdrom.CmdState
	if cdrom.Disc != nil {
		// respond with the status byte and then the disc identificator
		cdrom.OnAcknowledge = cdrom.AckGetId

		// send status byte
		state.State = CMD_STATE_RXPENDING
		state.RxPendingDelay = 26000
		state.RxPendingIrqDelay = 26000 + 5401
		state.RxPendingIrqCode = IRQ_CODE_OK
		state.RxPendingFifo = NewFIFOFromBytes([]byte{cdrom.DriveStatus()})
	} else {
		// pretend that the disc tray is open
		state.State = CMD_STATE_RXPENDING
		state.RxPendingDelay = 20000
		state.RxPendingIrqDelay = 20000 + 6776
		state.RxPendingIrqCode = IRQ_CODE_ERROR
		state.RxPendingFifo = NewFIFOFromBytes([]byte{0x11, 0x80})
	}
}

func (cdrom *CdRom) AckGetId() {
	disc := cdrom.GetDiscOrPanic()

	var regionByte byte
	switch disc.Region {
	case REGION_JAPAN:
		regionByte = 'I'
	case REGION_NORTH_AMERICA:
		regionByte = 'A'
	case REGION_EUROPE:
		regionByte = 'E'
	}

	response := NewFIFOFromBytes([]byte{
		cdrom.DriveStatus(),       // status
		0x00,                      // licensed, not audio
		0x20,                      // disc type
		0x00,                      // session info exists
		'S', 'C', 'E', regionByte, // region string
	})

	state := cdrom.CmdState
	state.State = CMD_STATE_RXPENDING
	state.RxPendingDelay = 7336
	state.RxPendingIrqDelay = 7336 + 12376
	state.RxPendingIrqCode = IRQ_CODE_DONE
	state.RxPendingFifo = response
}

// Execute seek command
func (cdrom *CdRom) CommandSeekL() {
	// make sure we're not on track 1's pregap
	if cdrom.SeekTarget.ToU32() < MsfFromBcd(0x00, 0x02, 0x00).ToU32() {
		panicFmt("cdrom: seek to track 1's pregap %s", cdrom.SeekTarget)
	}

	cdrom.Position = cdrom.SeekTarget
	cdrom.OnAcknowledge = cdrom.AckSeekL

	state := cdrom.CmdState
	state.State = CMD_STATE_RXPENDING
	state.RxPendingDelay = 35000
	state.RxPendingIrqDelay = 35000 + 5401
	state.RxPendingIrqCode = IRQ_CODE_OK
	state.RxPendingFifo = NewFIFOFromBytes([]byte{cdrom.DriveStatus()})
}

func (cdrom *CdRom) AckSeekL() {
	// FIXME: we should measure how much time it would take
	//        for the drive to physically move the head

	state := cdrom.CmdState
	state.State = CMD_STATE_RXPENDING
	state.RxPendingDelay = 1000000
	state.RxPendingIrqDelay = 1000000 + 1859
	state.RxPendingIrqCode = IRQ_CODE_DONE
	state.RxPendingFifo = NewFIFOFromBytes([]byte{cdrom.DriveStatus()})
}

func (cdrom *CdRom) CommandSetMode() {
	paramsLen := cdrom.Params.Length()
	if paramsLen != 1 {
		// FIXME: should trigger IRQ code 5 and respond with 0x13, 0x20
		panicFmt(
			"cdrom: invalid number of parameters for SetMode (expected 1, got %d)",
			paramsLen,
		)
	}

	mode := cdrom.Params.Pop()
	cdrom.DoubleSpeed = (mode & 0x80) != 0

	if mode&0x7f != 0 {
		panicFmt("cdrom: unhandled mode 0x%x", mode)
	}

	state := cdrom.CmdState
	state.State = CMD_STATE_RXPENDING
	state.RxPendingDelay = 22000
	state.RxPendingIrqDelay = 22000 + 5391
	state.RxPendingIrqCode = IRQ_CODE_OK
	state.RxPendingFifo = NewFIFOFromBytes([]byte{cdrom.DriveStatus()})
}

func (cdrom *CdRom) CommandPause() {
	if cdrom.ReadState.IsIdle() {
		panic("cdrom: call to Pause when not reading")
	}

	cdrom.OnAcknowledge = cdrom.AckPause

	state := cdrom.CmdState
	state.State = CMD_STATE_RXPENDING
	state.RxPendingDelay = 25000
	state.RxPendingIrqDelay = 25000 + 5393
	state.RxPendingIrqCode = IRQ_CODE_OK
	state.RxPendingFifo = NewFIFOFromBytes([]byte{cdrom.DriveStatus()})
}

func (cdrom *CdRom) AckPause() {
	cdrom.ReadState.State = READ_STATE_IDLE

	state := cdrom.CmdState
	state.State = CMD_STATE_RXPENDING
	state.RxPendingDelay = 2000000
	state.RxPendingIrqDelay = 2000000 + 1858
	state.RxPendingIrqCode = IRQ_CODE_DONE
	state.RxPendingFifo = NewFIFOFromBytes([]byte{cdrom.DriveStatus()})
}

// Start reading
func (cdrom *CdRom) CommandReadN() {
	if !cdrom.ReadState.IsIdle() {
		panic("cdrom: ReadN call while reading")
	}

	readDelay := cdrom.CyclesPerSector()
	cdrom.ReadState.State = READ_STATE_READING
	cdrom.ReadState.Delay = readDelay

	state := cdrom.CmdState
	state.State = CMD_STATE_RXPENDING
	state.RxPendingDelay = 28000
	state.RxPendingIrqDelay = 28000 + 5401
	state.RxPendingIrqCode = IRQ_CODE_OK
	state.RxPendingFifo = NewFIFOFromBytes([]byte{cdrom.DriveStatus()})
}

// Save where the next seek should be, but don't seek yet
func (cdrom *CdRom) CommandSetLoc() {
	paramsLen := cdrom.Params.Length()
	if paramsLen != 3 {
		// FIXME: should trigger IRQ 5 and respond with 0x13, 0x20
		panicFmt(
			"cdrom: unexpected amount of parameters for SetLoc (got %d, expected 3)",
			paramsLen,
		)
	}

	m := cdrom.Params.Pop()
	s := cdrom.Params.Pop()
	f := cdrom.Params.Pop()
	cdrom.SeekTarget = MsfFromBcd(m, s, f)

	state := cdrom.CmdState
	state.State = CMD_STATE_RXPENDING
	if cdrom.Disc != nil {
		state.RxPendingDelay = 35000
		state.RxPendingIrqDelay = 35000 + 5399
		state.RxPendingIrqCode = IRQ_CODE_OK
		state.RxPendingFifo = NewFIFOFromBytes([]byte{cdrom.DriveStatus()})
	} else {
		state.RxPendingDelay = 25000
		state.RxPendingIrqDelay = 25000 + 6763
		state.RxPendingIrqCode = IRQ_CODE_ERROR
		state.RxPendingFifo = NewFIFOFromBytes([]byte{0x11, 0x80})
	}
}

func (cdrom *CdRom) Load(th *TimeHandler, irqState *IrqState, size AccessSize, offset uint32) uint8 {
	cdrom.Sync(th, irqState)

	if size != ACCESS_BYTE {
		panicFmt("cdrom: tried to load %d bytes (expected %d)", size, ACCESS_BYTE)
	}

	index := cdrom.Index

	switch offset {
	case 0:
		return cdrom.Status()
	case 1:
		if cdrom.Response.IsEmpty() {
			fmt.Println("cdrom: response FIFO is empty!")
		}
		return cdrom.Response.Pop()
	case 3:
		switch index {
		case 0:
			return cdrom.IrqMask | 0xe0
		case 1:
			return cdrom.IrqFlags | 0xe0
		default:
			panic("cdrom: not implemented")
		}
	default:
		panic("cdrom: not implemented")
	}
}

func (cdrom *CdRom) Store(
	offset uint32,
	size AccessSize,
	val uint8,
	th *TimeHandler,
	irqState *IrqState,
) {
	cdrom.Sync(th, irqState)

	if size != ACCESS_BYTE {
		panicFmt("cdrom: tried to store %d bytes (expected %d)", size, ACCESS_BYTE)
	}

	index := cdrom.Index

	switch offset {
	case 0:
		cdrom.SetIndex(val)
	case 1:
		switch index {
		case 0:
			cdrom.Command(val, irqState, th)
		default:
			panic("cdrom: not implemented")
		}
	case 2:
		switch index {
		case 0:
			cdrom.PushParam(val)
		case 1:
			cdrom.SetIrqMask(val)
		default:
			panic("cdrom: not implemented")
		}
	case 3:
		switch index {
		case 0:
			cdrom.Config(val)
		case 1:
			cdrom.AcknowledgeIrq(val & 0x1f)
			if val&0x40 != 0 {
				cdrom.Params.Clear()
			}
			if val&0xa0 != 0 {
				panic("cdrom: not implemented")
			}
		default:
			panic("cdrom: not implemented")
		}
	default:
		panic("cdrom: not implemented")
	}
}

// SYNC ////////////////////////////////////////////

func (cdrom *CdRom) Sync(th *TimeHandler, irqState *IrqState) {
	delta := th.Sync(PERIPHERAL_CDROM)

	// handle command
	switch cdrom.CmdState.State {
	case CMD_STATE_IDLE:
		cdrom.HandleIdleState(th)
	case CMD_STATE_RXPENDING:
		cdrom.HandleRxPendingState(th, irqState, delta)
	case CMD_STATE_IRQ_PENDING:
		cdrom.HandleIrqPendingState(th, irqState, delta)
	}

	// check if we have a read pending
	if cdrom.ReadState.State == READ_STATE_READING {
		delay := cdrom.ReadState.Delay
		var nextSync uint32

		if uint64(delay) > delta {
			nextSync = delay - uint32(delta)
		} else {
			cdrom.SectorRead(irqState)
			// prepare for next sector
			nextSync = cdrom.CyclesPerSector()
		}
		cdrom.ReadState.State = READ_STATE_READING
		cdrom.ReadState.Delay = nextSync

		th.SetNextSyncDeltaIfCloser(PERIPHERAL_CDROM, uint64(nextSync))
	}
}

// Amount of CPU cycles required to read a single
func (cdrom *CdRom) CyclesPerSector() uint32 {
	cycles := CPU_FREQ_HZ / 75
	return cycles >> oneIfTrue(cdrom.DoubleSpeed)
}

func (cdrom *CdRom) HandleIrqPendingState(th *TimeHandler, irqState *IrqState, delta uint64) {
	state := cdrom.CmdState
	irqDelay := state.IrqPendingIrqDelay
	irqCode := state.IrqPendingIrqCode

	if uint64(irqDelay) > delta {
		// didn't reach the interrupt yet
		newIrqDelay := irqDelay - uint32(delta)

		th.SetNextSyncDelta(PERIPHERAL_CDROM, uint64(irqDelay))
		state.State = CMD_STATE_IRQ_PENDING
		state.IrqPendingIrqDelay = newIrqDelay
		state.IrqPendingIrqCode = irqCode
	} else {
		// reached interrupt
		cdrom.TriggerIrq(irqCode, irqState)
		th.RemoveNextSync(PERIPHERAL_CDROM)
		state.State = CMD_STATE_IDLE
	}
}

func (cdrom *CdRom) HandleRxPendingState(th *TimeHandler, irqState *IrqState, delta uint64) {
	state := cdrom.CmdState
	rxDelay := state.RxPendingDelay
	irqDelay := state.RxPendingIrqDelay
	irqCode := state.RxPendingIrqCode
	response := state.RxPendingFifo

	if uint64(rxDelay) > delta {
		// update counters
		rxDelay -= uint32(delta)
		irqDelay -= uint32(delta)

		th.SetNextSyncDelta(PERIPHERAL_CDROM, uint64(rxDelay))
		state.State = CMD_STATE_RXPENDING
		state.RxPendingDelay = rxDelay
		state.RxPendingIrqDelay = irqDelay
		state.RxPendingIrqCode = irqCode
		state.RxPendingFifo = response
	} else {
		// end of transfer
		cdrom.Response = response

		if uint64(irqDelay) > delta {
			// schedule an interrupt
			newIrqDelay := irqDelay - uint32(delta)
			th.SetNextSyncDelta(PERIPHERAL_CDROM, uint64(newIrqDelay))

			state.State = CMD_STATE_IRQ_PENDING
			state.IrqPendingIrqDelay = newIrqDelay
			state.IrqPendingIrqCode = irqCode
		} else {
			// irq reached
			cdrom.TriggerIrq(irqCode, irqState)
			th.RemoveNextSync(PERIPHERAL_CDROM)
			state.State = CMD_STATE_IDLE
		}
	}
}

func (cdrom *CdRom) HandleIdleState(th *TimeHandler) {
	th.RemoveNextSync(PERIPHERAL_CDROM)
	cdrom.CmdState.State = CMD_STATE_IDLE
}

// Read a byte from the RX buffer
func (cdrom *CdRom) GetByte() byte {
	if cdrom.RxIndex >= 0x800 {
		panic("cdrom: RX read reached 0x800")
	}

	v := cdrom.RxSector.DataByte(cdrom.RxIndex)

	if cdrom.RxActive {
		cdrom.RxIndex++
	} else {
		panic("cdrom: ReadByte() while not active")
	}

	return v
}

func (cdrom *CdRom) DmaReadWord() uint32 {
	b0 := uint32(cdrom.GetByte())
	b1 := uint32(cdrom.GetByte())
	b2 := uint32(cdrom.GetByte())
	b3 := uint32(cdrom.GetByte())
	return b0 | (b1 << 8) | (b2 << 16) | (b3 << 24)
}

func (cdrom *CdRom) GetDiscOrPanic() *Disc {
	if cdrom.Disc == nil {
		panic("cdrom: disc is nil")
	}
	return cdrom.Disc
}

// Called when a sector has been read
func (cdrom *CdRom) SectorRead(irqState *IrqState) {
	position := cdrom.Position
	disc := cdrom.GetDiscOrPanic()
	fmt.Printf("cdrom: read sector %s\n", position)

	sector, err := disc.ReadDataSector(position)
	if err != nil {
		panicFmt("cdrom: failed to read sector %s", position)
	}
	cdrom.RxSector = sector

	if cdrom.IrqFlags == 0 {
		cdrom.Response = NewFIFOFromBytes([]byte{cdrom.DriveStatus()})
		cdrom.TriggerIrq(IRQ_CODE_SECTOR_READY, irqState)
	}

	cdrom.Position = cdrom.Position.Next()
}

func (cdrom *CdRom) Config(conf uint8) {
	prevActive := cdrom.RxActive
	cdrom.RxActive = conf&0x80 != 0

	if cdrom.RxActive {
		if !prevActive {
			// go to the beginning of the RX FIFO
			cdrom.RxIndex = 0
		}
	} else {
		// align to next multiple of 8 bytes
		i := cdrom.RxIndex
		adjust := (i & 4) << 1
		cdrom.RxIndex = uint16(int32(i) & ^7) + adjust
	}

	if conf&0x7f != 0 {
		panicFmt("cdrom: unhandled config 0x%x", conf)
	}
}

func (cdrom *CdRom) AckIdle() {
	cdrom.CmdState.State = CMD_STATE_IDLE
}
