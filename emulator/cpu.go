package emulator

import "fmt"

// CPU state
type CPU struct {
	// The program counter register
	PC uint32
	// Next instruction to be executed, used to simulate the branch delay slot
	NextInstruction Instruction
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

	// COP0 register 12: Status Register
	SR uint32
	// HI register for division remainder and multiplication high result
	Hi uint32
	// LO register for division quotient and multiplication low result
	Lo uint32
}

// Creates a new CPU state
func NewCPU(inter *Interconnect) *CPU {
	cpu := &CPU{
		PC:              0xbfc00000,       // PC reset value at the beginning of the BIOS
		NextInstruction: Instruction(0x0), // NOP
		Inter:           inter,
		Hi:              0xdeadbeef, // junk
		Lo:              0xdeadbeef, // junk
	}

	// initialize registers to 0..32 (the values are not initialized on reset,
	// so we can put some garbage in them. note that the first value should
	// always be zero)
	for i := 0; i < len(cpu.Regs); i++ {
		cpu.Regs[i] = uint32(i)
	}

	return cpu
}

// Runs the instruction at the program counter and increments it
func (cpu *CPU) RunNextInstruction() {
	pc := cpu.PC

	// use previously loaded instruction
	instruction := cpu.NextInstruction

	// fetch instruction at PC
	cpu.NextInstruction = Instruction(cpu.Load32(pc))

	// increment PC to point to the next instruction (all instructions are 32 bit long)
	cpu.PC += 4 // wraps around: 0xfffffffc + 4 = 0

	// execute the pending load (if any, otherwise it will load $zero, which is a NOP)
	// `cpu.SetReg` only works on `cpu.OutRegs`, so this operation won't be visible by
	// the next instruction
	reg, val := cpu.Load[0], cpu.Load[1]
	cpu.SetReg(reg, val)

	// reset the load to target register 0 for the next instruction
	cpu.Load[0] = 0
	cpu.Load[1] = 0

	cpu.DecodeAndExecute(instruction)

	// copy the output registers as input for the next instruction
	// FIXME: this is copying 128 bytes of registers for each instruction,
	//        there could be a better way to do this
	cpu.Regs = cpu.OutRegs
}

// Returns a 32bit little endian value at `addr`. Panics if the address does not exist
func (cpu *CPU) Load32(addr uint32) uint32 {
	return cpu.Inter.Load32(addr)
}

// Returns the byte at `addr`
func (cpu *CPU) Load8(addr uint32) byte {
	return cpu.Inter.Load8(addr)
}

// Store 32 bit value into memory
func (cpu *CPU) Store32(addr, val uint32) {
	cpu.Inter.Store32(addr, val)
}

// Store 16 bit value into memory
func (cpu *CPU) Store16(addr uint32, val uint16) {
	cpu.Inter.Store16(addr, val)
}

// Store 32 bit value into memory
func (cpu *CPU) Store8(addr uint32, val uint8) {
	cpu.Inter.Store8(addr, val)
}

// Decodes and executes an instruction. Panics if the instruction is unhandled
func (cpu *CPU) DecodeAndExecute(instruction Instruction) {
	// http://problemkaputt.de/psx-spx.htm#cpuopcodeencoding
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
		default:
			panicFmt("cpu: unhandled instruction 0x%x", instruction)
		}
	case 0b001001: // Add Immediate Unsigned
		cpu.OpAddIU(instruction)
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
	default:
		panicFmt("cpu: unhandled instruction 0x%x", instruction)
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
	if cpu.SR&0x10000 != 0 {
		// cache is isolated, ignore write
		fmt.Println("cpu (sw): ignoring store while cache is isolated")
		return
	}

	i := instruction.ImmSE()
	t := instruction.T()
	s := instruction.S()

	addr := cpu.Reg(s) + i
	cpu.Store32(addr, cpu.Reg(t))
}

// Branch to immediate value `offset`
func (cpu *CPU) Branch(offset uint32) {
	// offset immediates are always shifted two places to the right since `PC`
	// addresses have to be aligned on 32 bits at all times
	offset <<= 2

	pc := cpu.PC
	pc += offset
	// we need to compensate for the hardcoded `cpu.PC += 4` in `RunNextInstruction`
	pc -= 4
	cpu.PC = pc
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
func (cpu *CPU) OpAddIU(instruction Instruction) {
	i := instruction.ImmSE()
	t := instruction.T()
	s := instruction.S()

	cpu.SetReg(t, cpu.Reg(s)+i)
}

// Jump
func (cpu *CPU) OpJ(instruction Instruction) {
	i := instruction.ImmJump()
	// the instructions must be aligned to a 32 bit boundary, so really
	// J encodes 28 bits of the target address (shifted by 2)
	// TODO: shouldn't we just call Branch()?
	cpu.PC = (cpu.PC & 0xf0000000) | (i << 2)
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
		cpu.SR = v
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
		panic("cpu: ADDI overflow")
	}

	cpu.SetReg(t, uint32(v))
}

// Load Word
func (cpu *CPU) OpLW(instruction Instruction) {
	if cpu.SR&0x10000 != 0 {
		// cache is isolated, ignore write
		fmt.Println("cpu (lw): ignoring store while cache is isolated")
		return
	}

	i := instruction.ImmSE()
	t := instruction.T()
	s := instruction.S()

	addr := cpu.Reg(s) + i

	// put the load in the delay slot
	cpu.Load[0] = t
	cpu.Load[1] = cpu.Load32(addr)
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
	if cpu.SR&0x10000 != 0 {
		fmt.Println("cpu (sh): ignoring store while cache is isolated")
		return
	}

	i := instruction.ImmSE()
	t := instruction.T()
	s := instruction.S()

	addr := cpu.Reg(s) + i
	v := cpu.Reg(t)
	cpu.Store16(addr, uint16(v))
}

// Jump And Link
func (cpu *CPU) OpJAL(instruction Instruction) {
	// store return address in $ra ($31)
	ra := cpu.PC
	cpu.SetReg(31, ra)
	cpu.OpJ(instruction)
}

// Store Byte
func (cpu *CPU) OpSB(instruction Instruction) {
	if cpu.SR&0x10000 != 0 {
		fmt.Println("cpu (sb): ignoring store while cache is isolated")
		return
	}

	i := instruction.ImmSE()
	t := instruction.T()
	s := instruction.S()

	addr := cpu.Reg(s) + i
	cpu.Store8(addr, uint8(cpu.Reg(t)))
}

// Jump Register
func (cpu *CPU) OpJR(instruction Instruction) {
	// TODO: i don't think this works correctly
	s := instruction.S()
	cpu.PC = cpu.Reg(s)
}

// Jump And Link Register
func (cpu *CPU) OpJALR(instruction Instruction) {
	// TODO: i don't think this works correctly
	d := instruction.D()
	s := instruction.S()

	ra := cpu.PC
	// store return address in `d`
	cpu.SetReg(d, ra)
	cpu.PC = cpu.Reg(s)
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
		v = cpu.SR
	case 13: // cause register
		panic("cpu: unhandled read from CAUSE register")
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
		panic("cpu: ADD overflow")
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
		// result is not representable in 32 bits

		// signed integer
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
