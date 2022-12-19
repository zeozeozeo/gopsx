package emulator

type Exception uint32

const (
	EXCEPTION_SYSCALL             Exception = 0x8 // System call (caused by the SYSCALL opcode)
	EXCEPTION_OVERFLOW            Exception = 0xc // Arithmetic overflow
	EXCEPTION_LOAD_ADDRESS_ERROR  Exception = 0x4 // Address error on load
	EXCEPTION_STORE_ADDRESS_ERROR Exception = 0x5 // Address error on store
	EXCEPTION_BREAK               Exception = 0x9 // Breakpoint (caused by BREAK opcode)
	EXCEPTION_COPROCESSOR_ERROR   Exception = 0xb // Unsupported coprocessor operation
	EXCEPTION_ILLEGAL_INSTRUCTION Exception = 0xa // CPU encountered an unknown instruction
)
