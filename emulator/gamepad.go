package emulator

import "fmt"

type SerialTarget int

const (
	TARGET_PADMEMCARD1 SerialTarget = 0
	TARGET_PADMEMCARD2 SerialTarget = 1
)

func SerialTargetFromControl(val uint16) SerialTarget {
	if val&0x2000 != 0 {
		return TARGET_PADMEMCARD2
	}
	return TARGET_PADMEMCARD1
}

// Gamepad and memory card
type PadMemCard struct {
	BaudDiv    uint16       // Serial clock divider
	Mode       uint8        // Serial config
	TxEn       bool         // Whether transmission is enabled
	Select     bool         // Whether the target peripheral select signal is set
	Target     SerialTarget // Specifies the memory card port
	Unknown    uint8        // Control register bits 3 and 5
	RxEn       bool         // Not sure what this does
	Dsr        bool         // Data Set Ready signal
	DsrIt      bool         // Whether an interrupt should be generated on a DSR pulse
	Interrupt  bool         // Interrupt level
	Response   uint8        // Response byte
	RxNotEmpty bool         // Whether the RX FIFO is not empty
	Pad1       *Gamepad     // Slot 1
	Pad2       *Gamepad     // Slot 2
	Bus        *Bus         // Bus state
}

func NewPadMemCard() *PadMemCard {
	return &PadMemCard{
		Target:   TARGET_PADMEMCARD1,
		Response: 0xff,
		Pad1:     NewGamepad(GAMEPAD_TYPE_DIGITAL),
		Pad2:     NewGamepad(GAMEPAD_TYPE_DISCONNECTED),
		Bus:      NewBus(BUS_STATE_IDLE),
	}
}

// Returns value of the status register
func (card *PadMemCard) Status() uint32 {
	var r uint32

	// TX ready bits
	r |= 5
	r |= oneIfTrue(card.RxNotEmpty) << 1
	// RX parity error (will always be 0)
	r |= 0 << 3
	r |= oneIfTrue(card.Dsr) << 7
	r |= oneIfTrue(card.Interrupt) << 9
	// TODO: add baud rate counter in [31:11]
	r |= 0 << 11

	return r
}

// Sets card.Mode
func (card *PadMemCard) SetMode(mode uint8) {
	card.Mode = mode
}

// Returns value of the control register
func (card *PadMemCard) Control() uint16 {
	var r uint16

	r |= uint16(card.Unknown)
	r |= uint16(oneIfTrue(card.TxEn)) << 0
	r |= uint16(oneIfTrue(card.Select)) << 1
	r |= uint16(oneIfTrue(card.RxEn)) << 2
	// TODO: add other interrupts
	r |= uint16(oneIfTrue(card.DsrIt)) << 12
	r |= uint16(card.Target) << 13

	return r
}

func (card *PadMemCard) SetControl(val uint16, irqState *IrqState) {
	if val&0x40 != 0 {
		// soft reset
		card.SoftReset()
	} else {
		if val&0x10 != 0 {
			// interrupt acknowledge
			card.Acknowledge(irqState)
		}

		prevSelect := card.Select

		card.Unknown = uint8(val) & 0x28
		card.TxEn = val&1 != 0
		card.Select = (val>>1)&1 != 0
		card.RxEn = (val>>2)&1 != 0
		card.DsrIt = (val>>12)&1 != 0
		card.Target = SerialTargetFromControl(val)

		if card.RxEn {
			panic("gamepad: RxEn is not implemented")
		}
		if card.DsrIt && !card.Interrupt && card.Dsr {
			panic("gamepad: DsrIt while DSR is active")
		}
		if val&0xf00 != 0 {
			panicFmt("gamepad: unsupported interrupt 0x%x", val)
		}
		if !prevSelect && card.Select {
			card.Pad1.Select()
		}
	}
}

func (card *PadMemCard) Acknowledge(irqState *IrqState) {
	card.Interrupt = false

	if card.Dsr && card.DsrIt {
		fmt.Println("gamepad: acknowledge when DSR is active")
		card.Interrupt = true
		irqState.SetHigh(INTERRUPT_PADMEMCARD)
	}
}

func (card *PadMemCard) SoftReset() {
	card.BaudDiv = 0
	card.Mode = 0
	card.Select = false
	card.Target = TARGET_PADMEMCARD1
	card.Unknown = 0
	card.Interrupt = false
	card.RxNotEmpty = false
	card.Bus.State = BUS_STATE_IDLE
	card.Dsr = false
}

func (card *PadMemCard) SendCommand(cmd uint8, th *TimeHandler) {
	if !card.TxEn {
		panic("gamepad: SendCommand while TxEn is false")
	}
	if card.Bus.IsBusy() {
		fmt.Printf("gamepad: command 0x%x while bus is busy!\n", cmd)
	}

	// no response by default
	var response uint8 = 0xff
	var dsr bool = false

	if card.Select {
		switch card.Target {
		case TARGET_PADMEMCARD1:
			response, dsr = card.Pad1.SendCommand(cmd)
		case TARGET_PADMEMCARD2:
			response, dsr = card.Pad2.SendCommand(cmd)
		}
	}

	// TODO: handle `Mode`
	txDuration := 8 * uint64(card.BaudDiv)
	card.Bus.State = BUS_STATE_TRANSFER
	card.Bus.DsrResponse = response
	card.Bus.Dsr = dsr
	card.Bus.TxDuration = txDuration

	th.SetNextSyncDelta(PERIPHERAL_PADMEMCARD, txDuration)
}

func (card *PadMemCard) Sync(th *TimeHandler, irqState *IrqState) {
	delta := th.Sync(PERIPHERAL_GPU)

	switch card.Bus.State {
	case BUS_STATE_IDLE:
		th.RemoveNextSync(PERIPHERAL_PADMEMCARD)
	case BUS_STATE_TRANSFER:
		card.HandleTransfer(th, irqState, delta)
	case BUS_STATE_DSR:
		card.HandleBusDsr(th, delta)
	}
}

func (card *PadMemCard) HandleBusDsr(th *TimeHandler, delta uint64) {
	delay := card.Bus.RemainingCycles
	if delta < delay {
		delay -= delta
		card.Bus.RemainingCycles = delay
	} else {
		// DSR pulse is over
		card.Dsr = false
		card.Bus.State = BUS_STATE_IDLE
	}
	th.RemoveNextSync(PERIPHERAL_PADMEMCARD)
}

func (card *PadMemCard) HandleTransfer(th *TimeHandler, irqState *IrqState, delta uint64) {
	resp := card.Bus.DsrResponse
	dsr := card.Bus.Dsr
	dur := card.Bus.TxDuration

	if delta < dur {
		// continue transfer
		dur -= delta
		card.Bus.TxDuration = dur

		if card.DsrIt {
			th.SetNextSyncDelta(PERIPHERAL_PADMEMCARD, dur)
		} else {
			th.RemoveNextSync(PERIPHERAL_PADMEMCARD)
		}
	} else {
		// end of transfer
		if card.RxNotEmpty {
			panic("gamepad: RX while FIFO is not empty")
		}

		card.Response = resp
		card.RxNotEmpty = true
		card.Dsr = dsr

		if card.Dsr {
			if card.DsrIt {
				if !card.Interrupt {
					irqState.SetHigh(INTERRUPT_PADMEMCARD)
				}
				card.Interrupt = true
			}

			dsrDuration := 10
			card.Bus.RemainingCycles = uint64(dsrDuration)
		} else {
			card.Bus.State = BUS_STATE_IDLE
		}
		th.RemoveNextSync(PERIPHERAL_PADMEMCARD)
	}
}

func (card *PadMemCard) Store(
	offset uint32,
	val interface{},
	size AccessSize,
	th *TimeHandler,
	irqState *IrqState,
) {
	card.Sync(th, irqState)

	switch offset {
	case 0:
		if size != ACCESS_BYTE {
			panicFmt("gamepad: unhandled store size %d (expected %d)", size, ACCESS_BYTE)
		}
		card.SendCommand(accessSizeToU8(size, val), th)
	case 8:
		card.SetMode(accessSizeToU8(size, val))
	case 10: // control
		if size == ACCESS_BYTE {
			panic("gamepad: byte gamepad control access")
		}
		card.SetControl(accessSizeToU16(size, val), irqState)
	case 14:
		card.BaudDiv = accessSizeToU16(size, val)
	default:
		panicFmt(
			"gamepad: unhandled write to gamepad register %d <- 0x%x",
			offset, accessSizeToU16(size, val),
		)
	}
}

func (card *PadMemCard) Load(
	th *TimeHandler,
	irqState *IrqState,
	offset uint32,
	size AccessSize,
) interface{} {
	card.Sync(th, irqState)

	switch offset {
	case 0:
		card.RxNotEmpty = false
		card.Response = 0xff
		return card.Response
	case 4:
		return accessSizeU32(size, card.Status())
	case 10:
		return accessSizeU16(size, card.Control())
	default:
		panicFmt("gamepad: unhandled read from register %d", offset)
	}
	return 0
}
