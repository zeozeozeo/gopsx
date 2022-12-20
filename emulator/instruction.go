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

// Returns the instruction as a string. If it is invalid, it returns "ILLEGAL"
func (op Instruction) String() string {
	switch op.Function() {
	case 0b001111: // Load Upper Immediate
		return "LUI"
	case 0b001101: // Bitwise Or Immediate
		return "ORI"
	case 0b101011: // Store Word
		return "SW"
	case 0b000000: // execute subfunction
		switch op.Subfunction() {
		case 0b000000: // Shift Left Logical
			return "SLL"
		case 0b000010: // Shift Right Logical
			return "SRL"
		case 0b100101: // Bitwise OR
			return "OR"
		case 0b100100: // Bitwise AND
			return "AND"
		case 0b101011: // Set on Less Than Unsigned
			return "SLTU"
		case 0b100001: // Add Unsigned
			return "ADDU"
		case 0b001000: // Jump Register
			return "JR"
		case 0b100000: // Add and generate an exception on overflow
			return "ADD"
		case 0b001001: // Jump And Link Register
			return "JALR"
		case 0b100011: // Subtract Unsigned
			return "SUBU"
		case 0b000011: // Shift Right Arithmetic
			return "SRA"
		case 0b011010: // Divide (signed)
			return "DIV"
		case 0b010010: // Move From LO
			return "MFLO"
		case 0b010000: // Move From HI
			return "MFHI"
		case 0b011011: // Divide Unsigned
			return "DIVU"
		case 0b101010: // Set on Less Than (signed)
			return "SLT"
		case 0b001100: // System Call
			return "Syscall"
		case 0b010011: // Move To LO
			return "MTLO"
		case 0b010001: // Move To HI
			return "MTHI"
		case 0b000100: // Shift Left Logical Variable
			return "SLLV"
		case 0b100111: // Bitwise Not Or
			return "NOR"
		case 0b000111: // Shift Right Arithmetic Variable
			return "SRAV"
		case 0b000110: // Shift Right Logical Variable
			return "SRLV"
		case 0b011001: // Multiply Unsigned
			return "MULTU"
		case 0b100110: // Bitwise eXclusive OR
			return "XOR"
		case 0b001101: // Break
			return "Break"
		case 0b011000: // Multiply (signed)
			return "MULT"
		case 0b100010: // Subtract and check for signed overflow
			return "SUB"
		}
	case 0b001001: // Add Immediate Unsigned
		return "ADDIU"
	case 0b000010: // Jump
		return "J"
	case 0b010000: // Coprocessor 0 opcode
		return "COP0"
	case 0b000101: // Branch if Not Equal
		return "BNE"
	case 0b001000: // Add Immediate Unsigned and check for overflow
		return "ADDI"
	case 0b100011: // Load Word
		return "LW"
	case 0b101001: // Store Halfword
		return "SH"
	case 0b000011: // Jump And Link
		return "JAL"
	case 0b001100: // Bitwise And Immediate
		return "ANDI"
	case 0b101000: // Store Byte
		return "SB"
	case 0b100000: // Load Byte
		return "LB"
	case 0b000100: // Branch if Equal
		return "BEQ"
	case 0b000111: // Branch if Greater Than Zero
		return "BGTZ"
	case 0b000110: // Branch if Less than or Equal to Zero
		return "BLEZ"
	case 0b100100: // Load Byte Unsigned
		return "LBU"
	case 0b000001: // BGEZ, BLTZ, BGEZAL, BLTZAL
		return "BXX"
	case 0b001010: // Set if Less Than Immediate (signed)
		return "SLTI"
	case 0b001011: // Set if Less Than Immediate Unsigned
		return "SLTIU"
	case 0b100101: // Load Halfword Unsigned
		return "LHU"
	case 0b100001: // Load Halfword (signed)
		return "LH"
	case 0b001110: // Bitwise eXclusive Or Immediate
		return "XORI"
	case 0b010001: // Coprocessor 1 opcode (does not exist on the PlayStation)
		return "COP1"
	case 0b010011: // Coprocessor 3 opcode (does not exist on the PlayStation)
		return "COP3"
	case 0b010010: // Coprocessor 2 opcode (GTE)
		return "COP2"
	case 0b100010: // Load Word Left
		return "LWL"
	case 0b100110: // Load Word Right
		return "LWR"
	case 0b101010: // Store Word Left
		return "SWL"
	case 0b101110: // Store Word Right
		return "SWR"
	case 0b110000: // Load Word in Coprocessor 0 (not supported)
		return "LWC0"
	case 0b110001: // Load Word in Coprocessor 1 (not supported)
		return "LWC1"
	case 0b110010: // Load Word in Coprocessor 2
		return "LWC2"
	case 0b110011: // Load Word in Coprocessor 3 (not supported)
		return "LWC3"
	case 0b111000: // Store Word in Coprocessor 0 (not supported)
		return "SWC0"
	case 0b111001: // Store Word in Coprocessor 1 (not supported)
		return "SWC1"
	case 0b111010: // Store Word in Coprocessor 2
		return "SWC2"
	case 0b111011: // Store Word in Coprocessor 3 (not supported)
		return "SWC3"
	}
	return "ILLEGAL"
}
