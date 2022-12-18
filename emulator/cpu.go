package emulator

// CPU state
type CPU struct {
	// The program counter register
	PC uint32
	// Next instruction to be executed, used to simulate the branch delay slot
	NextInstruction Instruction
	// General purpose registers. The first value must always be 0
	Regs [32]uint32
	// Memory interface
	Inter *Interconnect
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
	cpu.DecodeAndExecute(instruction)
}

// Returns a 32bit little endian value at `addr`. Panics if the address does not exist
func (cpu *CPU) Load32(addr uint32) uint32 {
	return cpu.Inter.Load32(addr)
}

func (cpu *CPU) Store32(addr, val uint32) {
	cpu.Inter.Store32(addr, val)
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
		default:
			panicFmt("cpu: unhandled instruction 0x%x", instruction)
		}
	case 0b001001: // Add Immediate Unsigned
		cpu.OpAddIU(instruction)
	case 0b000010: // Jump
		cpu.OpJ(instruction)
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
	i := instruction.ImmSE()
	t := instruction.T()
	s := instruction.S()

	addr := cpu.Reg(s) + i
	cpu.Store32(addr, cpu.Reg(t))
}

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

func (cpu *CPU) OpCOP0(instruction Instruction) {
	todo()
	/*
	switch instruction.COPOpcode() {
	case 0b00100:
		cpu.OPMTC0(instruction)
	default:
		panicFmt("unhandled cop0 instruction 0x%x", instruction)
	}
	*/
}

// Returns the register value at `index`. The first register is always zero
func (cpu *CPU) Reg(index uint32) uint32 {
	return cpu.Regs[index]
}

// Sets the value at the `index` register and sets the first register to zero
func (cpu *CPU) SetReg(index, val uint32) {
	cpu.Regs[index] = val
	// R0 should always remain 0, we can't change it
	cpu.Regs[0] = 0
}
