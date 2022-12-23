package emulator

import "fmt"

// CPU state
type CPU struct {
	// The program counter register: points to the next instruction
	PC uint32
	// Next value for the PC, used to simulate the branch delay slot
	NextPC uint32
	// Address of the instruction currently being executed. Used for
	// setting EPC in exceptions
	CurrentPC uint32
	// General purpose registers. The first value must always be 0
	Regs [32]uint32
	// 2nd set of registers to emulate the load delay slot correctly. They
	// contain the output of the current instruction
	OutRegs [32]uint32
	// Load initiated by the current instruction. The first value is the register
	// index, the second value is the value
	Load [2]uint32
	// Memory interface
	Inter *Interconnect
	// Set by the current instruction if a branch occured and the next instruction
	// will be in the delay slot
	BranchOccured bool
	// Set if the current instruction executes in the delay slot
	DelaySlot bool

	Cop0 *Cop0 // Coprocessor 0: System Control
	// HI register for division remainder and multiplication high result
	Hi uint32
	// LO register for division quotient and multiplication low result
	Lo uint32
	// Cop0 register 13: Cause Register
	Debugger *Debugger
	// Instruction Cache (256 cache lines)
	ICache [0x100]*ICacheLine
	Th     *TimeHandler // Keeps track of the emulation time
}

// Creates a new CPU state
func NewCPU(inter *Interconnect) *CPU {
	var pc uint32 = 0xbfc00000 // PC reset value at the beginning of the BIOS
	cpu := &CPU{
		PC:     pc,
		NextPC: pc + 4,
		// NextInstruction: Instruction(0x0), // NOP
		Inter:    inter,
		Hi:       0xdeadbeef, // junk
		Lo:       0xdeadbeef, // junk
		Debugger: NewDebugger(),
		Th:       NewTimeHandler(),
		Cop0:     NewCop0(),
	}

	// initialize registers to 0..32 (the values are not initialized on reset,
	// so we can put some garbage in them. note that the first value should
	// always be zero)
	for i := 0; i < len(cpu.Regs); i++ {
		cpu.Regs[i] = uint32(i)
	}

	// initialize cache lines
	for i := 0; i < len(cpu.ICache); i++ {
		cpu.ICache[i] = NewCacheLine()
	}

	return cpu
}

// Runs the instruction at the program counter and increments it
func (cpu *CPU) RunNextInstruction() {
	// synchronize peripherals
	cpu.Inter.Sync(cpu.Th)

	// save the address of the current instruction to save in EPC in case of an exception
	pc := cpu.PC
	cpu.CurrentPC = pc

	// debugger entrypoint
	cpu.Debugger.changedPc(pc)

	// FIXME: there's no need to check if PC is incorectly aligned for each instruction,
	//        instead we could make jump and branch instructions not capable of setting
	//        unaligned PC addresses
	if cpu.CurrentPC%4 != 0 {
		// PC is not correctly aligned
		fmt.Println("cpu: PC is not correctly aligned!")
		cpu.Exception(EXCEPTION_LOAD_ADDRESS_ERROR)
		return
	}

	// fetch instruction at PC
	instruction := cpu.FetchInstruction()

	// increment PC to point to the next instruction (all instructions are 32 bit long)
	cpu.PC = cpu.NextPC
	cpu.NextPC += 4

	// execute the pending load (if any, otherwise it will load $zero, which is a NOP)
	// `cpu.SetReg` only works on `cpu.OutRegs`, so this operation won't be visible by
	// the next instruction
	reg, val := cpu.Load[0], cpu.Load[1]
	cpu.SetReg(reg, val)

	// reset the load to target register 0 for the next instruction
	cpu.Load[0] = 0
	cpu.Load[1] = 0

	// if the last instruction was a branch then we're in the delay slot
	cpu.DelaySlot = cpu.BranchOccured
	cpu.BranchOccured = false

	if cpu.Cop0.IrqActive(cpu.Inter.IrqState) {
		cpu.Exception(EXCEPTION_INTERRUPT)
	} else {
		// no interrupts pending
		cpu.DecodeAndExecute(instruction)
	}

	// copy the output registers as input for the next instruction
	// FIXME: this is copying 128 bytes of registers for each instruction,
	//        there could be a better way to do this
	cpu.Regs = cpu.OutRegs
}

func (cpu *CPU) FetchInstruction() Instruction {
	pc := cpu.CurrentPC
	cc := cpu.Inter.CacheCtrl

	// KSEG1 is never cached
	kseg1 := (pc & 0xe0000000) == 0xa0000000

	if !kseg1 && cc.ICacheEnabled() {
		tag := pc & 0xfffff000           // cache tag: bits [31:12]
		line := cpu.ICache[(pc>>4)&0xff] // cache line: bits [11:4]
		index := (pc >> 2) & 3           // cache line index: bits [3:2]

		// check line tag and validity
		if line.Tag() != tag || line.ValidIndex() > index {
			// cache miss, get the cacheline at the current index
			cpc := pc

			// fetching takes 3 cycles + 1 instruction on average
			cpu.Th.Tick(3)

			for i := index; i < 4; i++ {
				cpu.Th.Tick(1)
				instruction := Instruction(cpu.Inter.LoadInstruction(cpc))
				line.Set(i, instruction)
				cpc += 4
			}

			line.SetTagValid(pc) // set tag and valid bits
		}

		return line.Get(index)
	}

	// cache is disabled, get instruction from memory
	// this takes 4 cycles on average
	cpu.Th.Tick(4)
	return Instruction(cpu.Inter.LoadInstruction(pc))
}

// Returns a 32bit little endian value at `addr`
func (cpu *CPU) Load32(addr uint32) uint32 {
	cpu.Debugger.memoryRead(addr)
	return cpu.Inter.Load32(addr, cpu.Th)
}

// Returns a 16bit little endian value at `addr`
func (cpu *CPU) Load16(addr uint32) uint16 {
	cpu.Debugger.memoryRead(addr)
	return cpu.Inter.Load16(addr, cpu.Th)
}

// Returns the byte at `addr`
func (cpu *CPU) Load8(addr uint32) byte {
	cpu.Debugger.memoryRead(addr)
	return cpu.Inter.Load8(addr, cpu.Th)
}

func (cpu *CPU) Store(addr uint32, size AccessSize, val interface{}) {
	if cpu.Cop0.CacheIsolated() {
		cpu.CacheMaintenance(addr, size, val)
	} else {
		cpu.Debugger.memoryWrite(addr)
		cpu.Inter.Store(addr, size, val, cpu.Th)
	}
}

// Handles writes when the cache is isolated
func (cpu *CPU) CacheMaintenance(addr uint32, size AccessSize, val interface{}) {
	// FIXME: this is not the full cache implementation, just cache invalidation
	//        for now
	cc := cpu.Inter.CacheCtrl
	valU32 := accessSizeToU32(size, val)

	if !cc.ICacheEnabled() {
		panicFmt("cpu: cache maintenance while instruction cache is disabled 0x%x", valU32)
	}
	if size != ACCESS_WORD || valU32 != 0 {
		panicFmt("cpu: unsupported write while cache is isolated 0x%x", valU32)
	}

	// get the cache line for this address
	line := cpu.ICache[(addr>>4)&0xff]

	if cc.TagTestMode() {
		// in tag test mode, the write will invalidate the entire targeted
		// cache line
		line.Invalidate()
	} else {
		// the write ends up directly in the cache
		index := (addr >> 2) & 3
		instruction := Instruction(valU32)
		line.Set(index, instruction)
	}
}

// Store 32 bit value into memory
func (cpu *CPU) Store32(addr, val uint32) {
	cpu.Store(addr, ACCESS_WORD, val)
}

// Store 16 bit value into memory
func (cpu *CPU) Store16(addr uint32, val uint16) {
	cpu.Store(addr, ACCESS_HALFWORD, val)
}

// Store 8 bit value into memory
func (cpu *CPU) Store8(addr uint32, val uint8) {
	cpu.Store(addr, ACCESS_BYTE, val)
}

// Decodes and executes an instruction. Panics if the instruction is unhandled
func (cpu *CPU) DecodeAndExecute(instruction Instruction) {
	// http://problemkaputt.de/psx-spx.htm#cpuopcodeencoding

	// simulate instruction execution time
	cpu.Th.Tick(1)

	switch instruction.Function() {
	case 0b001111: // Load Upper Immediate
		cpu.OpLUI(instruction)
	case 0b001101: // Bitwise Or Immediate
		cpu.OpORI(instruction)
	case 0b101011: // Store Word
		cpu.OpSW(instruction)
	case 0b000000: // execute subfunction
		switch instruction.Subfunction() {
		case 0b000000: // Shift Left Logical
			cpu.OpSLL(instruction)
		case 0b000010: // Shift Right Logical
			cpu.OpSRL(instruction)
		case 0b100101: // Bitwise OR
			cpu.OpOR(instruction)
		case 0b100100: // Bitwise AND
			cpu.OpAND(instruction)
		case 0b101011: // Set on Less Than Unsigned
			cpu.OpSLTU(instruction)
		case 0b100001: // Add Unsigned
			cpu.OpADDU(instruction)
		case 0b001000: // Jump Register
			cpu.OpJR(instruction)
		case 0b100000: // Add and generate an exception on overflow
			cpu.OpADD(instruction)
		case 0b001001: // Jump And Link Register
			cpu.OpJALR(instruction)
		case 0b100011: // Subtract Unsigned
			cpu.OpSUBU(instruction)
		case 0b000011: // Shift Right Arithmetic
			cpu.OpSRA(instruction)
		case 0b011010: // Divide (signed)
			cpu.OpDIV(instruction)
		case 0b010010: // Move From LO
			cpu.OpMFLO(instruction)
		case 0b010000: // Move From HI
			cpu.OpMFHI(instruction)
		case 0b011011: // Divide Unsigned
			cpu.OpDIVU(instruction)
		case 0b101010: // Set on Less Than (signed)
			cpu.OpSLT(instruction)
		case 0b001100: // System Call
			cpu.OpSyscall()
		case 0b010011: // Move To LO
			cpu.OpMTLO(instruction)
		case 0b010001: // Move To HI
			cpu.OpMTHI(instruction)
		case 0b000100: // Shift Left Logical Variable
			cpu.OpSLLV(instruction)
		case 0b100111: // Bitwise Not Or
			cpu.OpNOR(instruction)
		case 0b000111: // Shift Right Arithmetic Variable
			cpu.OpSRAV(instruction)
		case 0b000110: // Shift Right Logical Variable
			cpu.OpSRLV(instruction)
		case 0b011001: // Multiply Unsigned
			cpu.OpMULTU(instruction)
		case 0b100110: // Bitwise eXclusive OR
			cpu.OpXOR(instruction)
		case 0b001101: // Break
			cpu.OpBreak()
		case 0b011000: // Multiply (signed)
			cpu.OpMULT(instruction)
		case 0b100010: // Subtract and check for signed overflow
			cpu.OpSUB(instruction)
		default:
			panicFmt("cpu: unhandled instruction 0x%x", instruction)
		}
	case 0b001001: // Add Immediate Unsigned
		cpu.OpADDIU(instruction)
	case 0b000010: // Jump
		cpu.OpJ(instruction)
	case 0b010000: // Coprocessor 0 opcode
		cpu.OpCOP0(instruction)
	case 0b000101: // Branch if Not Equal
		cpu.OpBNE(instruction)
	case 0b001000: // Add Immediate Unsigned and check for overflow
		cpu.OpADDI(instruction)
	case 0b100011: // Load Word
		cpu.OpLW(instruction)
	case 0b101001: // Store Halfword
		cpu.OpSH(instruction)
	case 0b000011: // Jump And Link
		cpu.OpJAL(instruction)
	case 0b001100: // Bitwise And Immediate
		cpu.OpANDI(instruction)
	case 0b101000: // Store Byte
		cpu.OpSB(instruction)
	case 0b100000: // Load Byte
		cpu.OpLB(instruction)
	case 0b000100: // Branch if Equal
		cpu.OpBEQ(instruction)
	case 0b000111: // Branch if Greater Than Zero
		cpu.OpBGTZ(instruction)
	case 0b000110: // Branch if Less than or Equal to Zero
		cpu.OpBLEZ(instruction)
	case 0b100100: // Load Byte Unsigned
		cpu.OpLBU(instruction)
	case 0b000001: // BGEZ, BLTZ, BGEZAL, BLTZAL
		cpu.OpBXX(instruction)
	case 0b001010: // Set if Less Than Immediate (signed)
		cpu.OpSLTI(instruction)
	case 0b001011: // Set if Less Than Immediate Unsigned
		cpu.OpSLTIU(instruction)
	case 0b100101: // Load Halfword Unsigned
		cpu.OpLHU(instruction)
	case 0b100001: // Load Halfword (signed)
		cpu.OpLH(instruction)
	case 0b001110: // Bitwise eXclusive Or Immediate
		cpu.OpXORI(instruction)
	case 0b010001: // Coprocessor 1 opcode (does not exist on the PlayStation)
		cpu.OpCOP1()
	case 0b010011: // Coprocessor 3 opcode (does not exist on the PlayStation)
		cpu.OpCOP3()
	case 0b010010: // Coprocessor 2 opcode (GTE)
		cpu.OpCOP2(instruction)
	case 0b100010: // Load Word Left
		cpu.OpLWL(instruction)
	case 0b100110: // Load Word Right
		cpu.OpLWR(instruction)
	case 0b101010: // Store Word Left
		cpu.OpSWL(instruction)
	case 0b101110: // Store Word Right
		cpu.OpSWR(instruction)
	case 0b110000: // Load Word in Coprocessor 0 (not supported)
		cpu.OpLWC0()
	case 0b110001: // Load Word in Coprocessor 1 (not supported)
		cpu.OpLWC1()
	case 0b110010: // Load Word in Coprocessor 2
		cpu.OpLWC2(instruction)
	case 0b110011: // Load Word in Coprocessor 3 (not supported)
		cpu.OpLWC3()
	case 0b111000: // Store Word in Coprocessor 0 (not supported)
		cpu.OpSWC0()
	case 0b111001: // Store Word in Coprocessor 1 (not supported)
		cpu.OpSWC1()
	case 0b111010: // Store Word in Coprocessor 2
		cpu.OpSWC2(instruction)
	case 0b111011: // Store Word in Coprocessor 3 (not supported)
		cpu.OpSWC3()
	default:
		cpu.OpIllegal(instruction)
	}
}

// Load Upper Immediate
func (cpu *CPU) OpLUI(instruction Instruction) {
	i := instruction.Imm()
	t := instruction.T()

	// low 16 bits are set to 0
	v := i << 16
	cpu.SetReg(t, v)
}

// Bitwise Or Immediate
func (cpu *CPU) OpORI(instruction Instruction) {
	i := instruction.Imm()
	t := instruction.T()
	s := instruction.S()
	cpu.SetReg(t, cpu.Reg(s)|i)
}

// Bitwise And Immediate
func (cpu *CPU) OpANDI(instruction Instruction) {
	i := instruction.Imm()
	t := instruction.T()
	s := instruction.S()
	cpu.SetReg(t, cpu.Reg(s)&i)
}

// Store Word
func (cpu *CPU) OpSW(instruction Instruction) {
	i := instruction.ImmSE()
	t := instruction.T()
	s := instruction.S()

	addr := cpu.Reg(s) + i
	v := cpu.Reg(t)

	// address must be 32 bit aligned
	if addr%4 == 0 {
		cpu.Store32(addr, v)
	} else {
		cpu.Exception(EXCEPTION_STORE_ADDRESS_ERROR)
	}
}

// Branch to immediate value `offset`
func (cpu *CPU) Branch(offset uint32) {
	// offset immediates are always shifted two places to the right since `PC`
	// addresses have to be aligned on 32 bits at all times
	offset <<= 2
	cpu.NextPC = cpu.PC + offset
	cpu.BranchOccured = true
}

// Branch if Not Equal
func (cpu *CPU) OpBNE(instruction Instruction) {
	i := instruction.ImmSE()
	s := instruction.S()
	t := instruction.T()

	if cpu.Reg(s) != cpu.Reg(t) {
		cpu.Branch(i)
	}
}

// Shift Left Logical
func (cpu *CPU) OpSLL(instruction Instruction) {
	i := instruction.Shift()
	t := instruction.T()
	d := instruction.D()

	v := cpu.Reg(t) << i
	cpu.SetReg(d, v)
}

// Add Immediate Unsigned
func (cpu *CPU) OpADDIU(instruction Instruction) {
	i := instruction.ImmSE()
	t := instruction.T()
	s := instruction.S()

	v := cpu.Reg(s) + i
	cpu.SetReg(t, v)
}

// Jump
func (cpu *CPU) OpJ(instruction Instruction) {
	i := instruction.ImmJump()
	// the instructions must be aligned to a 32 bit boundary, so really
	// J encodes 28 bits of the target address (shifted by 2)
	cpu.NextPC = (cpu.NextPC & 0xf0000000) | (i << 2)
	cpu.BranchOccured = true
}

// Bitwise OR
func (cpu *CPU) OpOR(instruction Instruction) {
	d := instruction.D()
	s := instruction.S()
	t := instruction.T()

	v := cpu.Reg(s) | cpu.Reg(t)
	cpu.SetReg(d, v)
}

// Bitwise AND
func (cpu *CPU) OpAND(instruction Instruction) {
	d := instruction.D()
	s := instruction.S()
	t := instruction.T()

	v := cpu.Reg(s) & cpu.Reg(t)
	cpu.SetReg(d, v)
}

// Coprocessor 0 opcode
func (cpu *CPU) OpCOP0(instruction Instruction) {
	switch instruction.S() {
	case 0b00000: // Move From Coprocessor 0
		cpu.OpMFC0(instruction)
	case 0b00100: // Move To Coprocessor 0
		cpu.OpMTC0(instruction)
	case 0b10000: // Return From Expression
		cpu.OpRFE(instruction)
	default:
		panicFmt("cpu: unhandled cop0 instruction 0x%x", instruction)
	}
}

func (cpu *CPU) OpMTC0(instruction Instruction) {
	cpuR := instruction.T()
	copR := instruction.D()

	v := cpu.Reg(cpuR)

	switch copR {
	case 3, 5, 6, 7, 9, 11: // breakpoints registers
		if v != 0 {
			panicFmt("cpu: unhandled write of 0x%x to cop0r%d", v, copR)
		}
	case 12: // status register
		cpu.Cop0.SetSR(v)
	case 13: // cause register
		if v != 0 {
			panicFmt("cpu: unhandled write of 0x%x to CAUSE register", v)
		}
	default:
		panicFmt("cpu: unhandled cop0 register 0x%x", copR)
	}
}

// Add Immediate Unsigned and check for overflow
func (cpu *CPU) OpADDI(instruction Instruction) {
	i := int32(instruction.ImmSE())
	t := instruction.T()
	s := instruction.S()

	si := int32(cpu.Reg(s))
	v, err := add32Overflow(si, i)
	if err != nil {
		cpu.Exception(EXCEPTION_OVERFLOW)
		return
	}

	cpu.SetReg(t, uint32(v))
}

// Load Word
func (cpu *CPU) OpLW(instruction Instruction) {
	i := instruction.ImmSE()
	t := instruction.T()
	s := instruction.S()

	addr := cpu.Reg(s) + i

	// address must be 32 bit aligned
	if addr%4 == 0 {
		v := cpu.Load32(addr)
		// put the load in the delay slot
		cpu.Load[0] = t
		cpu.Load[1] = v
	} else {
		cpu.Exception(EXCEPTION_LOAD_ADDRESS_ERROR)
	}
}

// Set on Less Than Unsigned
func (cpu *CPU) OpSLTU(instruction Instruction) {
	d := instruction.D()
	s := instruction.S()
	t := instruction.T()

	// set register d to 1 if register s is smaller than register t
	// TODO: check if this is correct
	var v uint32
	if cpu.Reg(s) < cpu.Reg(t) {
		v = 1
	}
	cpu.SetReg(d, v)
}

// Add Unsigned
func (cpu *CPU) OpADDU(instruction Instruction) {
	s := instruction.S()
	t := instruction.T()
	d := instruction.D()

	v := cpu.Reg(s) + cpu.Reg(t)
	cpu.SetReg(d, v)
}

// Store Halfword
func (cpu *CPU) OpSH(instruction Instruction) {
	i := instruction.ImmSE()
	t := instruction.T()
	s := instruction.S()

	addr := cpu.Reg(s) + i

	// address must be 16 bit aligned
	if addr%2 == 0 {
		v := cpu.Reg(t)
		cpu.Store16(addr, uint16(v))
	} else {
		cpu.Exception(EXCEPTION_STORE_ADDRESS_ERROR)
	}
}

// Jump And Link
func (cpu *CPU) OpJAL(instruction Instruction) {
	// store return address in $ra ($31)
	ra := cpu.NextPC
	cpu.OpJ(instruction)
	cpu.SetReg(31, ra)
	// `cpu.BranchOccured = true` is set by `cpu.OpJ` above
}

// Store Byte
func (cpu *CPU) OpSB(instruction Instruction) {
	i := instruction.ImmSE()
	t := instruction.T()
	s := instruction.S()

	addr := cpu.Reg(s) + i
	cpu.Store8(addr, uint8(cpu.Reg(t)))
}

// Jump Register
func (cpu *CPU) OpJR(instruction Instruction) {
	s := instruction.S()
	cpu.NextPC = cpu.Reg(s)
	cpu.BranchOccured = true
}

// Jump And Link Register
func (cpu *CPU) OpJALR(instruction Instruction) {
	d := instruction.D()
	s := instruction.S()

	ra := cpu.NextPC
	cpu.NextPC = cpu.Reg(s)

	// store return address in `d`
	cpu.SetReg(d, ra)
	cpu.BranchOccured = true
}

// Load Byte
func (cpu *CPU) OpLB(instruction Instruction) {
	i := instruction.ImmSE()
	t := instruction.T()
	s := instruction.S()

	addr := cpu.Reg(s) + i

	// cast to int8 to force sign extension
	v := int8(cpu.Load8(addr))

	// put the load in the delay slot
	cpu.Load[0] = t
	cpu.Load[1] = uint32(v)
}

// Branch if Equal
func (cpu *CPU) OpBEQ(instruction Instruction) {
	i := instruction.ImmSE()
	s := instruction.S()
	t := instruction.T()

	if cpu.Reg(s) == cpu.Reg(t) {
		cpu.Branch(i)
	}
}

// Move From Coprocessor 0
func (cpu *CPU) OpMFC0(instruction Instruction) {
	cpuR := instruction.T()
	copR := instruction.D()

	var v uint32
	switch copR {
	case 12:
		v = cpu.Cop0.SR
	case 13: // cause register
		v = cpu.Cop0.Cause
	case 14: // exception PC
		v = cpu.Cop0.Epc
	default:
		panicFmt("cpu: unhandled read from cop0r%d", copR)
	}

	cpu.Load[0] = cpuR
	cpu.Load[1] = v
}

// Add and generate an exception on overflow
func (cpu *CPU) OpADD(instruction Instruction) {
	s := instruction.S()
	t := instruction.T()
	d := instruction.D()

	si := int32(cpu.Reg(s))
	ti := int32(cpu.Reg(t))

	v, err := add32Overflow(si, ti)
	if err != nil {
		cpu.Exception(EXCEPTION_OVERFLOW)
		return
	}

	cpu.SetReg(d, uint32(v))
}

// Branch if Greater Than Zero
func (cpu *CPU) OpBGTZ(instruction Instruction) {
	i := instruction.ImmSE()
	s := instruction.S()

	// the comparison is done in signed integers
	v := int32(cpu.Reg(s))
	if v > 0 {
		cpu.Branch(i)
	}
}

// Branch if Less than or Equal to Zero
func (cpu *CPU) OpBLEZ(instruction Instruction) {
	i := instruction.ImmSE()
	s := instruction.S()

	// the comparison is done in signed integers
	v := int32(cpu.Reg(s))
	if v <= 0 {
		cpu.Branch(i)
	}
}

// Load Byte Unsigned
func (cpu *CPU) OpLBU(instruction Instruction) {
	i := instruction.ImmSE()
	t := instruction.T()
	s := instruction.S()

	addr := cpu.Reg(s) + i
	v := cpu.Load8(addr)

	// put the load in the delay slot
	cpu.Load[0] = t
	cpu.Load[1] = uint32(v)
}

// BGEZ, BLTZ, BGEZAL, BLTZAL. Bits 16 and 20 are used to figure out which
// one to use
func (cpu *CPU) OpBXX(instruction Instruction) {
	i := instruction.ImmSE()
	s := instruction.S()

	instU := uint32(instruction)
	isBGEZ := ((instU >> 16) & 1)
	isLink := (instU>>17)&0xf == 8
	v := int32(cpu.Reg(s))

	// test "less than zero"
	var test uint32
	if v < 0 {
		test = 1
	}

	// if the test is "greater than or equal to zero" we need to negate
	// the comparison above since ("a >= 0" <=> "!(a < 0)"). the XOR will
	// take care of that
	test ^= isBGEZ

	if isLink {
		ra := cpu.PC
		// store return address in R31
		cpu.SetReg(31, ra)
	}
	if test != 0 {
		cpu.Branch(i)
	}
}

// Set if Less Than Immediate (signed)
func (cpu *CPU) OpSLTI(instruction Instruction) {
	i := int32(instruction.ImmSE())
	s := instruction.S()
	t := instruction.T()

	if int32(cpu.Reg(s)) < i {
		cpu.SetReg(t, 1)
	} else {
		cpu.SetReg(t, 0)
	}
}

// Set if Less Than Immediate Unsigned
func (cpu *CPU) OpSLTIU(instruction Instruction) {
	i := instruction.ImmSE()
	s := instruction.S()
	t := instruction.T()

	if cpu.Reg(s) < i {
		cpu.SetReg(t, 1)
	} else {
		cpu.SetReg(t, 0)
	}
}

// Subtract Unsigned
func (cpu *CPU) OpSUBU(instruction Instruction) {
	s := instruction.S()
	t := instruction.T()
	d := instruction.D()

	v := cpu.Reg(s) - cpu.Reg(t)
	cpu.SetReg(d, v)
}

// Shift Right Arithmetic
func (cpu *CPU) OpSRA(instruction Instruction) {
	i := instruction.Shift()
	t := instruction.T()
	d := instruction.D()

	v := int32(cpu.Reg(t)) >> i
	cpu.SetReg(d, uint32(v))
}

// Divide (signed)
func (cpu *CPU) OpDIV(instruction Instruction) {
	s := instruction.S()
	t := instruction.T()

	n := int32(cpu.Reg(s))
	d := int32(cpu.Reg(t))

	if d == 0 {
		// division by zero, results are bogus
		cpu.Hi = uint32(n)

		if n >= 0 {
			cpu.Lo = 0xffffffff
		} else {
			cpu.Lo = 1
		}
	} else if uint32(n) == 0x80000000 && d == -1 {
		// result is not representable in a 32 bit signed integer
		cpu.Hi = 0
		cpu.Lo = 0x80000000
	} else {
		cpu.Hi = uint32(n % d)
		cpu.Lo = uint32(n / d)
	}
}

// Divide Unsigned
func (cpu *CPU) OpDIVU(instruction Instruction) {
	s := instruction.S()
	t := instruction.T()

	n := cpu.Reg(s)
	d := cpu.Reg(t)

	if d == 0 {
		// division by zero, results are bogus
		cpu.Hi = n
		cpu.Lo = 0xffffffff
	} else {
		cpu.Hi = n % d
		cpu.Lo = n / d
	}
}

// Move From LO
func (cpu *CPU) OpMFLO(instruction Instruction) {
	d := instruction.D()
	cpu.SetReg(d, cpu.Lo)
}

// Move From HI
func (cpu *CPU) OpMFHI(instruction Instruction) {
	d := instruction.D()
	cpu.SetReg(d, cpu.Hi)
}

// Shift Right Logical
func (cpu *CPU) OpSRL(instruction Instruction) {
	i := instruction.Shift()
	t := instruction.T()
	d := instruction.D()

	v := cpu.Reg(t) >> i
	cpu.SetReg(d, v)
}

// Set on Less Than (signed)
func (cpu *CPU) OpSLT(instruction Instruction) {
	d := instruction.D()
	s := instruction.S()
	t := instruction.T()

	var v uint32
	if int32(cpu.Reg(s)) < int32(cpu.Reg(t)) {
		v = 1
	}

	cpu.SetReg(d, v)
}

// Returns the register value at `index`. The first register is always zero
func (cpu *CPU) Reg(index uint32) uint32 {
	return cpu.Regs[index]
}

// Sets the value at the `index` register and sets the first register to zero
func (cpu *CPU) SetReg(index, val uint32) {
	cpu.OutRegs[index] = val
	// R0 should always remain 0, we can't change it
	cpu.OutRegs[0] = 0
}

// Trigger an exception
func (cpu *CPU) Exception(cause Exception) {
	handlerAddr := cpu.Cop0.EnterException(cause, cpu.CurrentPC, cpu.DelaySlot)

	// exceptions don't have a branch delay, jump directly into
	// the handler
	cpu.PC = handlerAddr
	cpu.NextPC = cpu.PC + 4
}

// System Call
func (cpu *CPU) OpSyscall() {
	cpu.Exception(EXCEPTION_SYSCALL)
}

// Move To LO
func (cpu *CPU) OpMTLO(instruction Instruction) {
	s := instruction.S()
	cpu.Lo = cpu.Reg(s)
}

// Move To HI
func (cpu *CPU) OpMTHI(instruction Instruction) {
	s := instruction.S()
	cpu.Hi = cpu.Reg(s)
}

// Return From Expression
func (cpu *CPU) OpRFE(instruction Instruction) {
	// there are other instructions with the same encoding, but all
	// are virtual memory related and the PlayStation doesn't implement
	// them. Still, we need to make sure we're not running buggy code
	if instruction&0x3f != 0b010000 {
		panicFmt("cpu: invalid cop0 rfe instruction 0x%x", instruction)
	}

	cpu.Cop0.ReturnFromException()
}

// Load Halfword Unsigned
func (cpu *CPU) OpLHU(instruction Instruction) {
	i := instruction.ImmSE()
	t := instruction.T()
	s := instruction.S()

	addr := cpu.Reg(s) + i
	// address must be 16 bit aligned
	if addr%2 == 0 {
		v := cpu.Load16(addr)

		// put the load in the delay slot
		cpu.Load[0] = t
		cpu.Load[1] = uint32(v)
	} else {
		cpu.Exception(EXCEPTION_LOAD_ADDRESS_ERROR)
	}
}

// Shift Left Logical Variable
func (cpu *CPU) OpSLLV(instruction Instruction) {
	d := instruction.D()
	s := instruction.S()
	t := instruction.T()

	// shift amount is truncated to 5 bits
	v := cpu.Reg(t) << (cpu.Reg(s) & 0x1f)
	cpu.SetReg(d, v)
}

// Load Halfword (signed)
func (cpu *CPU) OpLH(instruction Instruction) {
	i := instruction.ImmSE()
	t := instruction.T()
	s := instruction.S()

	addr := cpu.Reg(s) + i

	// cast to int16 to force sign extension
	v := int16(cpu.Load16(addr))

	// put the load in the delay slot
	cpu.Load[0] = t
	cpu.Load[1] = uint32(v)
}

// Bitwise Not Or
func (cpu *CPU) OpNOR(instruction Instruction) {
	d := instruction.D()
	s := instruction.S()
	t := instruction.T()

	v := ^(cpu.Reg(s) | cpu.Reg(t))
	cpu.SetReg(d, v)
}

// Shift Right Arithmetic Variable
func (cpu *CPU) OpSRAV(instruction Instruction) {
	d := instruction.D()
	s := instruction.S()
	t := instruction.T()

	// shift amount is truncated to 5 bits
	v := int32(cpu.Reg(t)) >> (cpu.Reg(s) & 0x1f)
	cpu.SetReg(d, uint32(v))
}

// Shift Right Logical Variable
func (cpu *CPU) OpSRLV(instruction Instruction) {
	d := instruction.D()
	s := instruction.S()
	t := instruction.T()

	// shift amount is truncated to 5 bits
	v := cpu.Reg(t) >> (cpu.Reg(s) & 0x1f)
	cpu.SetReg(d, v)
}

// Multiply Unsigned
func (cpu *CPU) OpMULTU(instruction Instruction) {
	s := instruction.S()
	t := instruction.T()

	a := uint64(cpu.Reg(s))
	b := uint64(cpu.Reg(t))
	v := a * b

	cpu.Hi = uint32(v >> 32)
	cpu.Lo = uint32(v)
}

// Bitwise eXclusive OR
func (cpu *CPU) OpXOR(instruction Instruction) {
	d := instruction.D()
	s := instruction.S()
	t := instruction.T()

	v := cpu.Reg(s) ^ cpu.Reg(t)
	cpu.SetReg(d, v)
}

// Break
func (cpu *CPU) OpBreak() {
	cpu.Exception(EXCEPTION_BREAK)
}

// Multiply (signed)
func (cpu *CPU) OpMULT(instruction Instruction) {
	s := instruction.S()
	t := instruction.T()

	a := int64(int32(cpu.Reg(s)))
	b := int64(int32(cpu.Reg(t)))

	v := uint64(a * b)
	cpu.Hi = uint32(v >> 32)
	cpu.Lo = uint32(v)
}

// Subtract and check for signed overflow
func (cpu *CPU) OpSUB(instruction Instruction) {
	s := instruction.S()
	t := instruction.T()
	d := instruction.D()

	si := int32(cpu.Reg(s))
	ti := int32(cpu.Reg(t))

	v, err := sub32Overflow(si, ti)
	if err != nil {
		cpu.Exception(EXCEPTION_OVERFLOW)
	} else {
		cpu.SetReg(d, uint32(v))
	}
}

// Bitwise eXclusive Or Immediate
func (cpu *CPU) OpXORI(instruction Instruction) {
	i := instruction.Imm()
	t := instruction.T()
	s := instruction.S()

	v := cpu.Reg(s) ^ i
	cpu.SetReg(t, v)
}

// Coprocessor 1 opcode (does not exist on the PlayStation)
func (cpu *CPU) OpCOP1() {
	cpu.Exception(EXCEPTION_COPROCESSOR_ERROR)
}

// Coprocessor 3 opcode (does not exist on the PlayStation)
func (cpu *CPU) OpCOP3() {
	cpu.Exception(EXCEPTION_COPROCESSOR_ERROR)
}

// Coprocessor 2 opcode (GTE)
func (cpu *CPU) OpCOP2(instruction Instruction) {
	panicFmt("cpu: unhandled GTE instruction %d", instruction)
}

// Load Word Left (little-endian only implementation)
func (cpu *CPU) OpLWL(instruction Instruction) {
	i := instruction.ImmSE()
	t := instruction.T()
	s := instruction.S()

	addr := cpu.Reg(s) + i

	// this instruction bypasses the load delay restriction;
	// it will merge the new contents with the value currently
	// being loaded if needed
	curV := cpu.OutRegs[t]

	// next, load the *aligned* word containing the first addressed byte
	// TODO: maybe there is a way to do this without casts?
	alignedAddr := uint32(int64(addr) & ^3)
	alignedWord := cpu.Load32(alignedAddr)

	// depending on the address alignment, fetch 1, 2, 3 or 4 *most*
	// significant bytes and put them in the target register
	var v uint32
	switch addr & 3 {
	case 0:
		v = (curV & 0x00ffffff) | (alignedWord << 24)
	case 1:
		v = (curV & 0x0000ffff) | (alignedWord << 16)
	case 2:
		v = (curV & 0x000000ff) | (alignedWord << 8)
	case 3:
		v = 0 | (alignedWord << 0)
	default:
		panic("cpu (lwl): unreachable")
	}

	// put the load in the delay slot
	cpu.Load[0] = t
	cpu.Load[1] = v
}

// Load Word Right (little-endian only implementation)
func (cpu *CPU) OpLWR(instruction Instruction) {
	i := instruction.ImmSE()
	t := instruction.T()
	s := instruction.S()

	addr := cpu.Reg(s) + i

	// this instruction bypasses the load delay restriction;
	// it will merge the new contents with the value currently
	// being loaded if needed
	curV := cpu.OutRegs[t]

	// next, load the *aligned* word containing the first addressed byte
	// TODO: maybe there is a way to do this without casts?
	alignedAddr := uint32(int64(addr) & ^3)
	alignedWord := cpu.Load32(alignedAddr)

	// depending on the address alignment, fetch 1, 2, 3 or 4 *least*
	// significant bytes and put them in the target register
	var v uint32
	switch addr & 3 {
	case 0:
		v = 0 | (alignedWord >> 0)
	case 1:
		v = (curV & 0xff000000) | (alignedWord >> 8)
	case 2:
		v = (curV & 0xffff0000) | (alignedWord >> 16)
	case 3:
		v = (curV & 0xffffff00) | (alignedWord >> 24)
	default:
		panic("cpu (lwr): unreachable")
	}

	// put the load in the delay slot
	cpu.Load[0] = t
	cpu.Load[1] = v
}

// Store Word Left (little-endian only implementation)
func (cpu *CPU) OpSWL(instruction Instruction) {
	i := instruction.ImmSE()
	t := instruction.T()
	s := instruction.S()

	addr := cpu.Reg(s) + i
	v := cpu.Reg(t)

	alignedAddr := uint32(int64(addr) & ^3)
	// load the current value for the aligned word at the target address
	curMem := cpu.Load32(alignedAddr)

	var mem uint32
	switch addr & 3 {
	case 0:
		mem = (curMem & 0xffffff00) | (v >> 24)
	case 1:
		mem = (curMem & 0xffff0000) | (v >> 16)
	case 2:
		mem = (curMem & 0xff000000) | (v >> 8)
	case 3:
		mem = 0 | (v >> 0)
	default:
		panic("cpu (swl): unreachable")
	}
	cpu.Store32(alignedAddr, mem)
}

// Store Word Right (little-endian only implementation)
func (cpu *CPU) OpSWR(instruction Instruction) {
	i := instruction.ImmSE()
	t := instruction.T()
	s := instruction.S()

	addr := cpu.Reg(s) + i
	v := cpu.Reg(t)

	alignedAddr := uint32(int64(addr) & ^3)
	// load the current value for the aligned word at the target address
	curMem := cpu.Load32(alignedAddr)

	var mem uint32
	switch addr & 3 {
	case 0:
		mem = 0 | (v << 0)
	case 1:
		mem = (curMem & 0x000000ff) | (v << 8)
	case 2:
		mem = (curMem & 0x0000ffff) | (v << 16)
	case 3:
		mem = (curMem & 0x00ffffff) | (v << 24)
	default:
		panic("cpu (swr): unreachable")
	}
	cpu.Store32(alignedAddr, mem)
}

// Load Word in Coprocessor 0 (not supported)
func (cpu *CPU) OpLWC0() {
	cpu.Exception(EXCEPTION_COPROCESSOR_ERROR)
}

// Load Word in Coprocessor 1 (not supported)
func (cpu *CPU) OpLWC1() {
	cpu.Exception(EXCEPTION_COPROCESSOR_ERROR)
}

// Load Word in Coprocessor 2
func (cpu *CPU) OpLWC2(instruction Instruction) {
	panicFmt("unhandled GTE LWC %d", instruction)
}

// Load Word in Coprocessor 3 (not supported)
func (cpu *CPU) OpLWC3() {
	cpu.Exception(EXCEPTION_COPROCESSOR_ERROR)
}

// Store Word in Coprocessor 0 (not supported)
func (cpu *CPU) OpSWC0() {
	cpu.Exception(EXCEPTION_COPROCESSOR_ERROR)
}

// Store Word in Coprocessor 1 (not supported)
func (cpu *CPU) OpSWC1() {
	cpu.Exception(EXCEPTION_COPROCESSOR_ERROR)
}

// Store Word in Coprocessor 2
func (cpu *CPU) OpSWC2(instruction Instruction) {
	panicFmt("unhandled GTE SWC %d", instruction)
}

// Store Word in Coprocessor 3 (not supported)
func (cpu *CPU) OpSWC3() {
	cpu.Exception(EXCEPTION_COPROCESSOR_ERROR)
}

func (cpu *CPU) OpIllegal(instruction Instruction) {
	fmt.Printf("cpu: illegal instruction 0x%x\n", instruction)
	cpu.Exception(EXCEPTION_ILLEGAL_INSTRUCTION)
}
