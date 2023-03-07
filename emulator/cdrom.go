package emulator

import "fmt"

// CD-ROM controller
type CdRom struct {
	Index              uint8      // Some registers can change depending on the index
	HostParams         *FIFO      // FIFO storing the command arguments
	HostResponse       *FIFO      // FIFO storing command responses
	Command            *uint8     // Pending command number, can be nil
	IrqFlags           uint8      // 5 bit interrupt flags, low 3 bits are a sub-CPU interrupt
	IrqMask            uint8      // 5 bit interrupt mask
	RxBuffer           [2352]byte // RX data buffer
	Sector             *XaSector  // Disc image sector
	RxActive           bool       // True when want to read sector data
	SubCpu             *SubCpu    // The controllers' sub-CPU
	RxIndex            uint16     // Index of the next RX sector byte
	RxLen              uint16     // RX sector last byte index
	ReadState          *ReadState // CD read state
	ReadPending        bool       // True if a sector read needs to be notified
	Disc               *Disc      // Currently loaded disc, can be nil
	SeekTargetPending  bool       // True if a seek is waiting to be executed
	SeekTarget         *Msf       // Next seek command target
	Position           *Msf       // Current read position
	DoubleSpeed        bool       // If true, 150 sectors per second, else 75 sectorss
	XaAdpcmToSpu       bool       // If true, ADPCM samples are sent to the SPU
	ReadWholeSector    bool       // Reads 0x924 bytes of the sector if true, 0x800 if false
	SectorSizeOverride bool       // If true, overrides the regular sector size
	CddaMode           bool       // Whether the CD-DA mode is enabled
	Autopause          bool       // Whether to pause at the end of the track
	ReportInterrupts   bool       // Whether to generate interrupts for each CD-DA sector
	FilterEnabled      bool       // Whether the ADPCM filter is enabled
	FilterFile         uint8      // Which file numbers should be processed (filter)
	FilterChannel      uint8      // Which channel numbers should be processed (filter)
	Mixer              *Mixer     // CD-DA audio mixer (connected to the SPU)
	Rand               *CdRomRng  // Pseudo-random CD timings RNG
}

// Returns a new CdRom instance
func NewCdRom(disc *Disc) *CdRom {
	return &CdRom{
		HostParams:      NewFIFO(),
		HostResponse:    NewFIFO(),
		Sector:          NewXaSector(),
		Disc:            disc,
		SubCpu:          NewSubCpu(),
		ReadState:       NewReadState(),
		SeekTarget:      NewMsf(),
		Position:        NewMsf(),
		ReadWholeSector: true,
		Mixer:           NewMixer(),
		Rand:            NewCdRomRng(),
	}
}

func (cdrom *CdRom) Load(offset uint32,
	size AccessSize,
	th *TimeHandler,
	irqState *IrqState,
) uint32 {
	cdrom.Sync(th, irqState)

	if size != ACCESS_BYTE {
		panicFmt("cdrom: tried to load %d bytes (expected 1)", size)
	}

	index := cdrom.Index

	switch offset {
	case 0:
		return uint32(cdrom.HostStatus())
	case 1: // RESULT register
		if cdrom.HostResponse.IsEmpty() {
			fmt.Println("cdrom: RESULT register read with empty response FIFO")
		}
		fmt.Println("RESULT read")
		return uint32(cdrom.HostResponse.Pop())
	case 3:
		switch index {
		case 0:
			return uint32(cdrom.IrqMask | 0xe0)
		case 1:
			return uint32(cdrom.IrqFlags | 0xe0)
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
		panicFmt("cdrom: tried to store %d bytes (expected 1)", size)
	}
	index := cdrom.Index

	switch offset {
	case 0: // ADDRESS register
		cdrom.Index = val & 3
	case 1:
		switch index {
		case 0:
			cdrom.SetCommand(val, th)
		case 3: // ATV2 register
			cdrom.Mixer.CdRightToSpuRight = val
		default:
			panic("cdrom: not implemented")
		}
	case 2:
		switch index {
		case 0:
			cdrom.SetParameter(val)
		case 1:
			cdrom.SetHostInterruptMask(val)
		case 2: // ATV0 register
			cdrom.Mixer.CdLeftToSpuLeft = val
		case 3: // ATV3 register
			cdrom.Mixer.CdRightToSpuLeft = val
		default:
			panic("cdrom: not implemented")
		}
	case 3:
		switch index {
		case 0:
			cdrom.SetHostChipControl(val)
		case 1:
			cdrom.HostClipClearControl(val, th)
		case 2: // ATV1 register
			cdrom.Mixer.CdLeftToSpuRight = val
		case 3:
			fmt.Printf("cdrom: mixer apply 0x%x\n", val)
		default:
			panic("cdrom: not implemented")
		}
	default:
		panic("cdrom: not implemented")
	}
}

func (cdrom *CdRom) Sync(th *TimeHandler, irqState *IrqState) {
	delta := th.Sync(PERIPHERAL_CDROM)
	remainingCycles := uint32(delta)
	subcpu := cdrom.SubCpu

	for remainingCycles > 0 {
		var elapsed uint32
		if subcpu.IsInCommand() {
			if subcpu.Timer > remainingCycles {
				subcpu.Timer -= remainingCycles
				elapsed = remainingCycles
			} else {
				stepRemaining := subcpu.Timer
				cdrom.NextSubCpuStep(irqState)
				elapsed = stepRemaining
			}
		} else {
			// no command pending
			elapsed = remainingCycles
		}

		// commands can cause async responses, process async events
		if subcpu.AsyncResponse.IsReady() {
			delay := subcpu.AsyncResponse.Delay

			if delay > elapsed {
				subcpu.AsyncResponse.Delay -= elapsed
			} else {
				// process async response
				subcpu.AsyncResponse.Delay = 0
				cdrom.MaybeProcessAsyncResponse(th)
			}
		}

		// handle sector reads
		if cdrom.ReadState.IsReading() {
			delay := cdrom.ReadState.Delay

			if delay > elapsed {
				cdrom.ReadState.Delay -= elapsed
			} else {
				leftover := elapsed - delay

				// read sector
				cdrom.ReadSector()
				cdrom.MaybeNotifyRead(th)

				// set next sector read delay
				cdrom.ReadState.Delay = cdrom.CyclesPerSector() - leftover
			}
		}

		remainingCycles -= elapsed
	}

	cdrom.PredictNextSync(th)
}

func (cdrom *CdRom) PredictNextSync(th *TimeHandler) {
	th.RemoveNextSync(PERIPHERAL_CDROM)

	if cdrom.SubCpu.IsInCommand() {
		// sync at the next sub-CPU step
		delta := uint64(cdrom.SubCpu.Timer)
		th.SetNextSyncDelta(PERIPHERAL_CDROM, delta)
	} else if cdrom.IrqFlags == 0 {
		// sync at the next async response

		if cdrom.SubCpu.AsyncResponse.IsReady() {
			delta := uint64(cdrom.SubCpu.AsyncResponse.Delay)
			th.SetNextSyncDelta(PERIPHERAL_CDROM, delta)
		}
	}

	if cdrom.ReadState.IsReading() {
		th.MaybeSetNextSyncDelta(PERIPHERAL_CDROM, uint64(cdrom.ReadState.Delay))
	}
}

// Read a word from the RX buffer
func (cdrom *CdRom) DmaReadWord() uint32 {
	b0 := uint32(cdrom.GetByte())
	b1 := uint32(cdrom.GetByte())
	b2 := uint32(cdrom.GetByte())
	b3 := uint32(cdrom.GetByte())
	return b0 | (b1 << 8) | (b2 << 16) | (b3 << 24)
}

// HSTS register read
func (cdrom *CdRom) HostStatus() uint8 {
	r := cdrom.Index

	// TODO: ADPCM busy
	r |= 0 << 2
	r |= uint8(oneIfTrue(cdrom.HostParams.IsEmpty())) << 3    // PRMEMPT
	r |= uint8(oneIfTrue(!cdrom.HostParams.IsFull())) << 4    // PRMWRDY
	r |= uint8(oneIfTrue(!cdrom.HostResponse.IsEmpty())) << 5 // RSLRRDY
	r |= uint8(oneIfTrue(cdrom.RxIndex < cdrom.RxLen)) << 6   // DRQSTS
	r |= uint8(oneIfTrue(cdrom.SubCpu.IsBusy())) << 7         // BUSYSTS
	return r
}

// COMMAND register write
func (cdrom *CdRom) SetCommand(val uint8, th *TimeHandler) {
	if cdrom.Command != nil {
		panic("cdrom: nested command")
	}

	v := val
	cdrom.Command = &v
	cdrom.MaybeStartCommand(th)
}

// PARAMETER register write
func (cdrom *CdRom) SetParameter(val uint8) {
	if cdrom.Command != nil {
		panic("cdrom: attempted to push parameter while in command")
	}
	if cdrom.HostParams.IsFull() {
		// FIXME: this should wrap around the parameter FIFO
		panic("cdrom: parameter FIFO overflow")
	}

	cdrom.HostParams.Push(val)
}

// HINTMSK register write
func (cdrom *CdRom) SetHostInterruptMask(val uint8) {
	if val&0x18 != 0 {
		fmt.Printf("cdrom: unhandled HINTMSK mask 0x%x\n", val)
	}

	cdrom.IrqMask = val & 0x1f
}

// HCLRCTL register write
func (cdrom *CdRom) HostClipClearControl(val uint8, th *TimeHandler) {
	cdrom.IrqAck(val&0x1f, th)

	if val&0x40 != 0 {
		cdrom.HostParams.Clear()
	}
	if val&0xa0 != 0 {
		panicFmt("cdrom: unhandled HCLRCTL: 0x%x", val)
	}
}

// HCHPCTL register write
func (cdrom *CdRom) SetHostChipControl(val uint8) {
	prevActive := cdrom.RxActive
	cdrom.RxActive = val&0x80 != 0

	if cdrom.RxActive {
		if !prevActive {
			cdrom.RxIndex = 0
		}
	} else {
		// TODO: check if this is correct
		idx := cdrom.RxIndex
		adjust := (idx & 4) << 1
		cdrom.RxIndex = uint16(int32(idx) & ^7) + adjust
	}

	if val&0x7f != 0 {
		panicFmt("cdrom: unhandled HCHPCTL 0x%x", val)
	}
}

func (cdrom *CdRom) IrqAck(v uint8, th *TimeHandler) {
	cdrom.IrqFlags &= ^v

	cdrom.MaybeStartCommand(th)
	cdrom.MaybeProcessAsyncResponse(th)
	cdrom.MaybeNotifyRead(th)
}

func (cdrom *CdRom) MaybeStartCommand(th *TimeHandler) {
	subcpu := cdrom.SubCpu
	if cdrom.Command != nil && cdrom.IrqFlags == 0 && !subcpu.IsInCommand() {
		// emulate the random pending command delay
		delay := TIMING_COMMAND_PENDING +
			(cdrom.Rand.Next() % TIMING_COMMAND_PENDING_VARIATION)

		subcpu.StartCommand(delay)
		cdrom.PredictNextSync(th)
	}
}

func (cdrom *CdRom) MaybeProcessAsyncResponse(th *TimeHandler) {
	subcpu := cdrom.SubCpu
	if subcpu.AsyncResponse.IsReady() && cdrom.IrqFlags == 0 && !subcpu.IsInCommand() {
		// run response sequcne
		handler := subcpu.AsyncResponse.Handler
		subcpu.AsyncResponse.Reset()
		subcpu.Response.Clear()

		subcpu.IrqCode = IRQ_CODE_DONE
		rxDelay := handler()

		subcpu.Sequence = SUBCPU_ASYNCRXPUSH
		subcpu.Timer = rxDelay

		cdrom.PredictNextSync(th)
	}
}

func (cdrom *CdRom) MaybeNotifyRead(th *TimeHandler) {
	subcpu := cdrom.SubCpu
	if cdrom.ReadPending && cdrom.IrqFlags == 0 && !subcpu.IsInCommand() {
		subcpu.Response.Clear()
		subcpu.IrqCode = IRQ_CODE_SECTOR_READY

		cdrom.PushStatus()
		subcpu.Sequence = SUBCPU_ASYNCRXPUSH
		subcpu.Timer = TIMING_READ_RX_PUSH

		cdrom.ReadPending = false
		cdrom.PredictNextSync(th)
	}
}

// Processes the next sub-CPU step
func (cdrom *CdRom) NextSubCpuStep(irqState *IrqState) {
	subcpu := cdrom.SubCpu

	switch subcpu.Sequence {
	case SUBCPU_IDLE:
		panic("cdrom: idle sub-CPU sequence")
	case SUBCPU_COMMANDPENDING, SUBCPU_PARAMPUSH:
		cdrom.HandleSubCpuCommandTransfer(subcpu)
	case SUBCPU_EXECUTION:
		cdrom.HandleSubCpuCommandExecution(subcpu)
	case SUBCPU_RXFLUSH, SUBCPU_RXPUSH:
		cdrom.HandleSubCpuRx(subcpu)
	case SUBCPU_BUSYDELAY:
		cdrom.HandleSubCpuBusyDelay(subcpu)
	case SUBCPU_IRQDELAY:
		cdrom.HandleSubCpuIrqDelay(subcpu, irqState)
	case SUBCPU_ASYNCRXPUSH:
		cdrom.HandleSubCpuAsyncRxPush(subcpu)
	}
}

// SUBCPU_ASYNCRXPUSH
func (cdrom *CdRom) HandleSubCpuAsyncRxPush(subcpu *SubCpu) {
	b := subcpu.Response.Pop()
	cdrom.HostResponse.Push(b)
	fmt.Println("push")

	if subcpu.Response.IsEmpty() {
		subcpu.Timer = TIMING_IRQ_DELAY
		subcpu.Sequence = SUBCPU_IRQDELAY
	} else {
		subcpu.Timer = TIMING_RXPUSH
		subcpu.Sequence = SUBCPU_ASYNCRXPUSH
	}
}

// SUBCPU_IRQDELAY
func (cdrom *CdRom) HandleSubCpuIrqDelay(subcpu *SubCpu, irqState *IrqState) {
	cdrom.Command = nil
	cdrom.TriggerIrq(subcpu.IrqCode, irqState)
	subcpu.Sequence = SUBCPU_IDLE
}

// SUBCPU_BUSYDELAY
func (cdrom *CdRom) HandleSubCpuBusyDelay(subcpu *SubCpu) {
	cdrom.SubCpu.Timer = TIMING_IRQ_DELAY
	cdrom.SubCpu.Sequence = SUBCPU_IRQDELAY
}

// SUBCPU_RXFLUSH, SUBCPU_RXPUSH
func (cdrom *CdRom) HandleSubCpuRx(subcpu *SubCpu) {
	b := subcpu.Response.Pop()
	cdrom.HostResponse.Push(b)
	fmt.Println("push")

	if subcpu.Response.IsEmpty() {
		subcpu.Timer = TIMING_BUSY_DELAY
		subcpu.Sequence = SUBCPU_BUSYDELAY
	} else {
		subcpu.Timer = TIMING_RXPUSH
		subcpu.Sequence = SUBCPU_RXPUSH
	}
}

// SUBCPU_EXECUTION
func (cdrom *CdRom) HandleSubCpuCommandExecution(subcpu *SubCpu) {
	cdrom.HostResponse.Clear()
	subcpu.Timer = TIMING_RXFLUSH
	subcpu.Sequence = SUBCPU_RXFLUSH
}

// SUBCPU_COMMANDPENDING, SUBCPU_PARAMPUSH
func (cdrom *CdRom) HandleSubCpuCommandTransfer(subcpu *SubCpu) {
	if cdrom.HostParams.IsEmpty() {
		// all params are recieved, run the command
		cdrom.ExecuteCommand()

		subcpu.Timer = TIMING_EXECUTION
		subcpu.Sequence = SUBCPU_EXECUTION
	} else {
		// send next parameter
		param := cdrom.HostParams.Pop()
		subcpu.Params.Push(param)

		subcpu.Timer = TIMING_PARAM_PUSH
		subcpu.Sequence = SUBCPU_PARAMPUSH
	}
}

// Triggers an interrupt
func (cdrom *CdRom) TriggerIrq(irq IrqCode, irqState *IrqState) {
	if cdrom.IrqFlags != 0 {
		panic("cdrom: nested interrupt")
	}

	cdrom.IrqFlags = uint8(irq)

	if cdrom.Irq() {
		// rising edge interrupt
		irqState.SetHigh(INTERRUPT_CDROM)
	}
}

// Returns true if the controller is in an interrupt
func (cdrom *CdRom) Irq() bool {
	return cdrom.IrqFlags&cdrom.IrqMask != 0
}

// Read a byte from the RX buffer
func (cdrom *CdRom) GetByte() byte {
	b := cdrom.RxBuffer[cdrom.RxIndex]

	if cdrom.RxActive {
		cdrom.RxIndex++

		if cdrom.RxIndex >= cdrom.RxLen {
			// end of transfer, set RxActive to false
			cdrom.RxActive = false
		}
	} else {
		panic("cdrom: ReadByte() while RxActive is false")
	}

	return b
}

// Reads the current sector
func (cdrom *CdRom) ReadSector() {
	if cdrom.ReadPending {
		panic("cdrom: attempted to read sector while another read is pending")
	}

	position := cdrom.Position
	disc := cdrom.Disc
	if disc == nil {
		panic("cdrom: attempted to read sector without a disc")
	}

	sector, err := disc.ReadSector(position)
	if err != nil {
		panicFmt("cdrom: couldn't read sector: %s", err)
	}

	var data []byte
	if cdrom.ReadWholeSector {
		data = sector.DataNoSyncPattern() // skip sync pattern
	} else {
		// only read data after the XA subheader
		data, err = sector.Mode2XaPayload()
		if err != nil {
			panicFmt("cdrom: couldn't get mode 2 payload: %s", err)
		}
		if len(data) > 2048 {
			// mode 2 form 2 sector, should only be read with ReadWholeSector?
			fmt.Println("cdrom: partial mode 2 form 2 sector read")
			data = data[0:2048]
		}
	}

	// copy data into the RX buffer
	copy(cdrom.RxBuffer[:], data)

	// go to the next position
	next, err := cdrom.Position.Next()
	if err != nil {
		panicFmt("cdrom: msf: %s", err)
	}
	cdrom.Position = next
	cdrom.ReadPending = true
}

// Runs the command in `cdrom.Command`
func (cdrom *CdRom) ExecuteCommand() {
	if cdrom.Command == nil {
		panic("cdrom: tried to execute command while `cdrom.Command` is nil")
	}

	var minParam, maxParam uint8
	var handler func()
	cmd := *cdrom.Command

	switch cmd {
	case 0x01:
		minParam, maxParam, handler = 0, 0, cdrom.CommandGetStat
	case 0x02:
		minParam, maxParam, handler = 3, 3, cdrom.CommandSetLoc
	case 0x06:
		minParam, maxParam, handler = 0, 0, cdrom.CommandRead
	case 0x09:
		minParam, maxParam, handler = 0, 0, cdrom.CommandPause
	case 0x0a:
		minParam, maxParam, handler = 0, 0, cdrom.CommandInit
	case 0x0b:
		minParam, maxParam, handler = 0, 0, cdrom.CommandMute
	case 0x0c:
		minParam, maxParam, handler = 0, 0, cdrom.CommandDemute
	case 0x0d:
		minParam, maxParam, handler = 2, 2, cdrom.CommandSetFilter
	case 0x0e:
		minParam, maxParam, handler = 1, 1, cdrom.CommandSetMode
	case 0x0f:
		minParam, maxParam, handler = 0, 0, cdrom.CommandGetParam
	case 0x11:
		minParam, maxParam, handler = 0, 0, cdrom.CommandGetLocP
	case 0x15:
		minParam, maxParam, handler = 0, 0, cdrom.CommandSeekL
	case 0x19:
		minParam, maxParam, handler = 1, 1, cdrom.CommandTest
	case 0x1a:
		minParam, maxParam, handler = 0, 0, cdrom.CommandGetId
	case 0x1b:
		minParam, maxParam, handler = 0, 0, cdrom.CommandRead
	case 0x1e:
		minParam, maxParam, handler = 0, 0, cdrom.CommandReadToc
	default:
		panicFmt("cdrom: unhandled command 0x%x", cmd)
	}

	paramsLen := cdrom.SubCpu.Params.Length()
	if paramsLen < minParam || paramsLen > maxParam {
		panicFmt(
			"cdrom: unexpected amount of params for 0x%x (expected %d-%d params, got %d)",
			cmd, minParam, maxParam, paramsLen,
		)
	}

	handler()
}

// Get status byte
func (cdrom *CdRom) CommandGetStat() {
	cdrom.PushStatus()
}

func (cdrom *CdRom) CommandSetLoc() {
	m := cdrom.SubCpu.Params.Pop()
	s := cdrom.SubCpu.Params.Pop()
	f := cdrom.SubCpu.Params.Pop()

	cdrom.SeekTarget = MsfFromBcd(m, s, f)
	cdrom.SeekTargetPending = true
	cdrom.PushStatus()
}

// Start read sequence
func (cdrom *CdRom) CommandRead() {
	if cdrom.ReadState.IsReading() {
		fmt.Println("cdrom: read while already reading")
	}
	if cdrom.SeekTargetPending {
		cdrom.DoSeek()
	}

	readDelay := cdrom.CyclesPerSector()
	cdrom.ReadState.MakeReading(readDelay)
	cdrom.PushStatus()
}

// Stop reading sectors
func (cdrom *CdRom) CommandPause() {
	var asyncDelay uint32
	if cdrom.ReadState.IsIdle() {
		fmt.Println("cdrom: pause when not reading")
		asyncDelay = 9000
	} else {
		asyncDelay = 1000000
	}

	cdrom.ReadState.MakeIdle() // TODO: is this right?
	cdrom.SubCpu.ScheduleAsyncResponse(cdrom.AsyncPause, asyncDelay)
	cdrom.PushStatus()
}

func (cdrom *CdRom) AsyncPause() uint32 {
	cdrom.PushStatus()
	return TIMING_PAUSE_RX_PUSH
}

// Initialize the CD-ROM controller
func (cdrom *CdRom) CommandInit() {
	cdrom.ReadState.MakeIdle()
	cdrom.ReadPending = false

	cdrom.SubCpu.ScheduleAsyncResponse(cdrom.AsyncInit, TIMING_INIT)
	cdrom.PushStatus()
}

// CommandInit response
func (cdrom *CdRom) AsyncInit() uint32 {
	cdrom.Position = NewMsf()
	cdrom.SeekTarget = NewMsf()
	cdrom.ReadState.MakeIdle()
	cdrom.DoubleSpeed = false
	cdrom.XaAdpcmToSpu = false
	cdrom.ReadWholeSector = false
	cdrom.SectorSizeOverride = false
	cdrom.FilterEnabled = false
	cdrom.ReportInterrupts = false
	cdrom.Autopause = false
	cdrom.CddaMode = false

	cdrom.PushStatus()
	return TIMING_INIT_RX_PUSH
}

// Mute audio playback
func (cdrom *CdRom) CommandMute() {
	cdrom.PushStatus()
}

// Demute audio playback
func (cdrom *CdRom) CommandDemute() {
	cdrom.PushStatus()
}

// Set ADPCM filters
func (cdrom *CdRom) CommandSetFilter() {
	cdrom.FilterFile = cdrom.SubCpu.Params.Pop()
	cdrom.FilterChannel = cdrom.SubCpu.Params.Pop()
	cdrom.PushStatus()
}

// Set CD-ROM controller parameters
func (cdrom *CdRom) CommandSetMode() {
	mode := cdrom.SubCpu.Params.Pop()

	cdrom.CddaMode = (mode>>0)&1 != 0
	cdrom.Autopause = (mode>>1)&1 != 0
	cdrom.ReportInterrupts = (mode>>2)&1 != 0
	cdrom.FilterEnabled = (mode>>3)&1 != 0
	cdrom.SectorSizeOverride = (mode>>4)&1 != 0
	cdrom.ReadWholeSector = (mode>>5)&1 != 0
	cdrom.XaAdpcmToSpu = (mode>>6)&1 != 0
	cdrom.DoubleSpeed = (mode>>7)&1 != 0

	if cdrom.CddaMode ||
		cdrom.Autopause ||
		cdrom.ReportInterrupts ||
		cdrom.SectorSizeOverride {
		panicFmt("cdrom: unhandled mode 0x%x", mode)
	}

	cdrom.PushStatus()
}

// Responds with CD-ROM controller parameters
func (cdrom *CdRom) CommandGetParam() {
	var mode uint8

	mode |= uint8(oneIfTrue(cdrom.CddaMode)) << 0
	mode |= uint8(oneIfTrue(cdrom.Autopause)) << 1
	mode |= uint8(oneIfTrue(cdrom.ReportInterrupts)) << 2
	mode |= uint8(oneIfTrue(cdrom.FilterEnabled)) << 3
	mode |= uint8(oneIfTrue(cdrom.SectorSizeOverride)) << 4
	mode |= uint8(oneIfTrue(cdrom.ReadWholeSector)) << 5
	mode |= uint8(oneIfTrue(cdrom.XaAdpcmToSpu)) << 6
	mode |= uint8(oneIfTrue(cdrom.DoubleSpeed)) << 7

	cdrom.SubCpu.Response.PushSlice([]byte{
		cdrom.DriveStatus(),
		0, // always 0
		cdrom.FilterFile,
		cdrom.FilterChannel,
	})
}

// Get current drive head position
func (cdrom *CdRom) CommandGetLocP() {
	if cdrom.Position.ToU32() < MsfFromBcd(0x00, 0x02, 0x00).ToU32() {
		panic("cdrom: GetLocP in track 1's pregap")
	}
	panic("cdrom: GetLocP is not implemented") // TODO
}

// Seek command, the target position is set by the previous SetLoc command
func (cdrom *CdRom) CommandSeekL() {
	// initial := cdrom.Position.ToU32()
	// target := cdrom.SeekTarget.ToU32()

	cdrom.DoSeek()
	cdrom.PushStatus()

	cdrom.SubCpu.ScheduleAsyncResponse(cdrom.AsyncSeekL, 1000000)
	/*
		cdrom.SubCpu.ScheduleAsyncResponse(
			cdrom.AsyncSeekL,
			cdrom.CalcSeekTime(initial, target, true, false),
		)
	*/
}

// SeekL async response
func (cdrom *CdRom) AsyncSeekL() uint32 {
	cdrom.PushStatus()
	return TIMING_SEEKL_RX_PUSH
}

// Execute a pending seek command
func (cdrom *CdRom) DoSeek() {
	// don't seek to track 1's pregap
	if cdrom.SeekTarget.ToU32() < MsfFromBcd(0x00, 0x02, 0x00).ToU32() {
		panicFmt("cdrom: attempted to seek to track 1's pregap (%s)", cdrom.SeekTarget)
	}

	cdrom.Position = cdrom.SeekTarget
	cdrom.SeekTargetPending = false
}

// Test command, has a lot of subcommands
func (cdrom *CdRom) CommandTest() {
	paramsLen := cdrom.SubCpu.Params.Length()
	if paramsLen != 1 {
		panicFmt(
			"cdrom: invalid amount of parameters for CommandTest (expected 1, got %d)",
			paramsLen,
		)
	}

	// TODO: implement other subcommands
	param := cdrom.SubCpu.Params.Pop()
	switch param {
	case 0x20:
		cdrom.TestVersion()
	default:
		panicFmt("cdrom: unhandled test subcommand 0x%x", param)
	}
}

// Read table of contents
func (cdrom *CdRom) CommandReadToc() {
	cdrom.PushStatus()
	// TODO: should this stop ReadN/ReadS?
	cdrom.SubCpu.ScheduleAsyncResponse(cdrom.AsyncReadToc, TIMING_READTOC_ASYNC)
}

// Read table of contents
func (cdrom *CdRom) AsyncReadToc() uint32 {
	cdrom.PushStatus()
	return TIMING_READTOC_RX_PUSH
}

// Responds with the CD-ROM identification string
func (cdrom *CdRom) CommandGetId() {
	if cdrom.Disc != nil {
		cdrom.PushStatus()
		cdrom.SubCpu.ScheduleAsyncResponse(cdrom.AsyncGetId, TIMING_GET_ID_ASYNC)
	} else {
		// no disc, pretend that the CD tray is open
		cdrom.SubCpu.Response.Push(0x11)
		cdrom.SubCpu.Response.Push(0x80)
		cdrom.SubCpu.SetIrqCode(IRQ_CODE_ERROR)
	}
}

// Asynchronous GetId response
func (cdrom *CdRom) AsyncGetId() uint32 {
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

	cdrom.SubCpu.Response.PushSlice([]byte{
		0x00,                      // licensed, not audio
		0x20,                      // disc type
		0x00,                      // session info exists
		'S', 'C', 'E', regionByte, // region string
	})
	return TIMING_GET_ID_RX_PUSH
}

// Responds with the CD-ROM version number
func (cdrom *CdRom) TestVersion() {
	cdrom.SubCpu.Response.Push(0x97) // year
	cdrom.SubCpu.Response.Push(0x01) // month
	cdrom.SubCpu.Response.Push(0x10) // day
	cdrom.SubCpu.Response.Push(0xc2) // version
}

// Returns a disc, panics if it is nil
func (cdrom *CdRom) GetDiscOrPanic() *Disc {
	if cdrom.Disc == nil {
		panic("cdrom: no disc")
	}
	return cdrom.Disc
}

// Returns the first status byte of many commands
func (cdrom *CdRom) DriveStatus() byte {
	if cdrom.Disc != nil {
		// disc inserted
		var r byte

		isReading := cdrom.ReadState.IsReading()
		r |= 1 << 1 // motor on
		r |= byte(oneIfTrue(isReading)) << 5
		return r
	}

	// no disc, pretend that the CD tray is open
	return 0x10
}

// Pushes the first status byte of many commands
func (cdrom *CdRom) PushStatus() {
	cdrom.SubCpu.Response.Push(cdrom.DriveStatus())
}

func (cdrom *CdRom) CyclesPerSector() uint32 {
	return (CPU_FREQ_HZ / 75) >> oneIfTrue(cdrom.DoubleSpeed)
}
