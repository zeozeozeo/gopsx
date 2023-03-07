package emulator

import (
	"fmt"
	"math"
)

// IRQ code used by the CD-ROM controller
type IrqCode uint8

const (
	IRQ_CODE_SECTOR_READY IrqCode = 1 // CD sector is ready
	IRQ_CODE_DONE         IrqCode = 2 // Command successful (2nd response)
	IRQ_CODE_OK           IrqCode = 3 // Command successful (1st response)
	IRQ_CODE_ERROR        IrqCode = 5 // Invalid command, etc.
)

type CdRomReadState int

const (
	READ_STATE_IDLE    CdRomReadState = iota
	READ_STATE_READING CdRomReadState = iota
)

// CD-ROM data read state
type ReadState struct {
	State CdRomReadState
	Delay uint32 // For READ_STATE_READING
}

func NewReadState() *ReadState {
	return &ReadState{
		State: READ_STATE_IDLE,
	}
}

func (rstate *ReadState) MakeIdle() {
	rstate.State = READ_STATE_IDLE
	rstate.Delay = 0
}

func (rstate *ReadState) MakeReading(delay uint32) {
	rstate.State = READ_STATE_READING
	rstate.Delay = delay
}

func (rstate *ReadState) IsIdle() bool {
	return rstate.State == READ_STATE_IDLE
}

func (rstate *ReadState) IsReading() bool {
	return rstate.State == READ_STATE_READING
}

func (cdrom *CdRom) CalcSeekTime(initial, target uint32, motorOn, paused bool) uint32 {
	var ret int64

	if !motorOn {
		initial = 0
		ret += 33868800
	}

	diff := absInt64(int64(initial) - int64(target))
	ret += maxInt64(diff*33868800*1000/(72*60*75)/1000, 20000)

	if diff >= 2250 {
		ret += 33868800 * 300 / 1000
	} else if paused {
		if cdrom.DoubleSpeed {
			ret += 1237952 * 2
		} else {
			ret += 1237952
		}
	} else if diff >= 3 && diff < 12 {
		if cdrom.DoubleSpeed {
			ret += 33868800 / (75 * 2) * 4
		} else {
			ret += 33868800 / 75 * 4
		}
	}

	ret += int64(cdrom.Rand.Next() % 25000)
	if ret > math.MaxUint32 {
		ret = math.MaxUint32
	}

	fmt.Printf("cdrom: CalcSeekTime(): %d\n", ret)
	return uint32(ret)
}
