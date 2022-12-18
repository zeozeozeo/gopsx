package emulator

// CPU state
type CPU struct {
	PC    uint32        // The program counter register
	Regs  [32]uint32    // General purpose registers. The first value must always be 0
	Inter *Interconnect // Memory interface
}

// Creates a new CPU state
func NewCPU(inter *Interconnect) *CPU {

	cpu := &CPU{
		PC:    0xbfc00000, // PC reset value at the beginning of the BIOS
		Inter: inter,
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

	// fetch instruction at PC
	instruction := Instruction(cpu.Load32(pc))

	// increment PC to point to the next instruction
	cpu.PC += 4 // wraps around: 0xfffffffc + 4 = 0
	cpu.DecodeAndExecute(instruction)
}

// Returns a 32bit little endian value at `addr`. Panics if the address does not exist
func (cpu *CPU) Load32(addr uint32) uint32 {
	return cpu.Inter.Load32(addr)
}

func (cpu *CPU) Store32(addr, val uint32) {
	// cpu.Inter.Store32(addr, val)
	todo()
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
	i := instruction.Imm()
	t := instruction.T()
	s := instruction.S()

	addr := cpu.Reg(s) + i
	cpu.Store32(addr, cpu.Reg(t))
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
