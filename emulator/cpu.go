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
}

// Creates a new CPU state
func NewCPU(inter *Interconnect) *CPU {

	cpu := &CPU{
		PC:              0xbfc00000,       // PC reset value at the beginning of the BIOS
		NextInstruction: Instruction(0x0), // NOP
		Inter:           inter,
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

	// we reset the load to target register 0 for the next instruction
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

// Store 32 bit value into memory
func (cpu *CPU) Store32(addr, val uint32) {
	cpu.Inter.Store32(addr, val)
}

// Store 16 bit value into memory
func (cpu *CPU) Store16(addr uint32, val uint16) {
	cpu.Inter.Store16(addr, val)
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
		case 0b100101: // Bitwise OR
			cpu.OpOR(instruction)
		case 0b101011: // Set on Less Than Unsigned
			cpu.OpSLTU(instruction)
		case 0b100001: // Add Unsigned
			cpu.OpADDU(instruction)
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
	case 0b101001:
		cpu.OpSH(instruction)
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

// Coprocessor 0 opcode
func (cpu *CPU) OpCOP0(instruction Instruction) {
	switch instruction.S() {
	case 0b00100:
		cpu.OpMTC0(instruction)
	default:
		panicFmt("unhandled cop0 instruction 0x%x", instruction)
	}
}

func (cpu *CPU) OpMTC0(instruction Instruction) {
	cpuR := instruction.T()
	copR := instruction.D()

	v := cpu.Reg(cpuR)

	switch copR {
	case 3, 5, 6, 7, 9, 11: // breakpoints registers
		if v != 0 {
			panicFmt("unhandled write of 0x%x to cop0r%d", v, copR)
		}
	case 12: // status register
		cpu.SR = v
	case 13: // cause register
		if v != 0 {
			panicFmt("unhandled write of 0x%x to CAUSE register", v)
		}
	default:
		panicFmt("unhandled cop0 register 0x%x", copR)
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
		panic("ADDI overflow")
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
