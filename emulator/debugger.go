package emulator

import "fmt"

type Debugger struct {
	Breakpoints      []uint32 // All breakpoint addresses
	ReadWatchpoints  []uint32 // All read watchpoints
	WriteWatchpoints []uint32 // All write watchpoints
}

func NewDebugger() *Debugger {
	return &Debugger{}
}

// Adds a breakpoint when the instruction at `addr` is about to be executed
func (debugger *Debugger) AddBreakpoint(addr uint32) {
	// check if that breakpoint already exists
	for _, breakpoint := range debugger.Breakpoints {
		if breakpoint == addr {
			return
		}
	}
	debugger.Breakpoints = append(debugger.Breakpoints, addr)
}

// Deletes a breakpoint at `addr`. Does nothing if it doesn't exist
func (debugger *Debugger) DeleteBreakpoint(addr uint32) {
	for idx, breakpoint := range debugger.Breakpoints {
		if breakpoint == addr {
			// remove this breakpoint
			debugger.Breakpoints = append(debugger.Breakpoints[:idx], debugger.Breakpoints[idx+1:]...)
			return
		}
	}
}

// Adds a memory read watchpoint for `addr`
func (debugger *Debugger) AddReadWatchpoint(addr uint32) {
	for _, watchpoint := range debugger.ReadWatchpoints {
		if watchpoint == addr {
			return
		}
	}
	debugger.ReadWatchpoints = append(debugger.ReadWatchpoints, addr)
}

// Adds a memory write watchpoint for `addr`
func (debugger *Debugger) AddWriteWatchpoint(addr uint32) {
	for _, watchpoint := range debugger.WriteWatchpoints {
		if watchpoint == addr {
			return
		}
	}
	debugger.WriteWatchpoints = append(debugger.WriteWatchpoints, addr)
}

// Deletes a memory read watchpoint at `addr`. Does nothing if it doesn't exist
func (debugger *Debugger) DeleteReadWatchpoint(addr uint32) {
	for idx, breakpoint := range debugger.ReadWatchpoints {
		if breakpoint == addr {
			// remove this breakpoint
			debugger.ReadWatchpoints = append(
				debugger.ReadWatchpoints[:idx],
				debugger.ReadWatchpoints[idx+1:]...,
			)
			return
		}
	}
}

// Deletes a memory write watchpoint at `addr`. Does nothing if it doesn't exist
func (debugger *Debugger) DeleteWriteWatchpoint(addr uint32) {
	for idx, breakpoint := range debugger.WriteWatchpoints {
		if breakpoint == addr {
			// remove this breakpoint
			debugger.WriteWatchpoints = append(
				debugger.WriteWatchpoints[:idx],
				debugger.WriteWatchpoints[idx+1:]...,
			)
			return
		}
	}
}

// Debugger entrypoint
func (debugger *Debugger) changedPc(pc uint32) {
	// check if a breakpoint exists for this address
	for _, breakpoint := range debugger.Breakpoints {
		if breakpoint == pc {
			fmt.Printf("debugger: reached breakpoint 0x%x\n", pc)
			debugger.Debug()
			return
		}
	}
}

// Called by the CPU when it's about to read a value from memory
func (debugger *Debugger) memoryRead(addr uint32) {
	for _, watchpoint := range debugger.ReadWatchpoints {
		if watchpoint == addr {
			fmt.Printf("debugger: triggered read watchpoint 0x%x\n", addr)
			debugger.Debug()
			return
		}
	}
}

// Called by the CPU when it's about to write a value to memory
func (debugger *Debugger) memoryWrite(addr uint32) {
	for _, watchpoint := range debugger.WriteWatchpoints {
		if watchpoint == addr {
			fmt.Printf("debugger: triggered write watchpoint 0x%x\n", addr)
			debugger.Debug()
			return
		}
	}
}

func (debugger *Debugger) Debug() {
	panic("TODO: not implemented")
}
