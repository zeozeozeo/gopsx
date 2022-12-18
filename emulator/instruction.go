package emulator

type Instruction uint32

// Return bits [31:26] of the instruction
func (op Instruction) Function() uint32 {
	return uint32(op) >> 26
}

// Return bits [5:0] of the instruction
func (op Instruction) Subfunction() uint32 {
	return uint32(op) & 0x3f
}

// Return register index in bits [25:21]
func (op Instruction) S() uint32 {
	return (uint32(op) >> 21) & 0x1f
}

// Return register index in bits [20:16]
func (op Instruction) T() uint32 {
	return (uint32(op) >> 16) & 0x1f
}

// Return register index in bits [15:11]
func (op Instruction) D() uint32 {
	return (uint32(op) >> 11) & 0x1f
}

// Return immediate value in bits [16:0]
func (op Instruction) Imm() uint32 {
	return uint32(op) & 0xffff
}

// Return immediate value in bits [16:0] as a sign-extended 32 bit value
func (op Instruction) ImmSE() uint32 {
	// TODO: check if this works properly
	v := int16(uint32(op) & 0xffff) // sign-extend v
	return uint32(v)
}

// Jump target stored in bits [25:0]
func (op Instruction) ImmJump() uint32 {
	return uint32(op) & 0x3ffffff
}

// Shift Immediate values are stored in bits [10:6]
func (op Instruction) Shift() uint32 {
	return (uint32(op) >> 6) & 0x1f
}
