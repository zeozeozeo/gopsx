package emulator

type Instruction uint32

// Return bits [31:26] of the instruction
func (op Instruction) Function() uint32 {
	return uint32(op) >> 26
}

// Return register index in bits [25:21]
func (op Instruction) S() uint32 {
	return (uint32(op) >> 21) & 0x1f
}

// Return register index in bits [20:16]
func (op Instruction) T() uint32 {
	return (uint32(op) >> 16) & 0x1f
}

// Return immediate value in bits [16:0]
func (op Instruction) Imm() uint32 {
	return uint32(op) & 0xffff
}