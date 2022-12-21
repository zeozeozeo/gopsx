package emulator

// Buffer holding multi-word fixed-length GP0 command parameters
type CommandBuffer struct {
	// Command buffer: the longest possible command is GP0(0x3E)
	// which takes 12 parameters
	Buffer [12]uint32
	Len    uint8 // Number of words queued in the buffer
}

func NewCommandBuffer() *CommandBuffer {
	return &CommandBuffer{}
}

// Clears the command buffer
func (cmdbuf *CommandBuffer) Clear() {
	cmdbuf.Len = 0
}

// Pushes a word (32 bit unsigned integer) into the command buffer
func (cmdbuf *CommandBuffer) PushWord(word uint32) {
	cmdbuf.Buffer[cmdbuf.Len] = word
	cmdbuf.Len++
}

// Returns value at `index`
func (cmdbuf *CommandBuffer) Get(index uint8) uint32 {
	return cmdbuf.Buffer[index]
}