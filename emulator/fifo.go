package emulator

// Used to store command arguments
type FIFO struct {
	Buffer   [16]byte
	WritePtr uint8 // Write pointer (4 bits and carry)
	ReadPtr  uint8 // Read pointer (4 bits and carry)
}

// Returns a new FIFO instance
func NewFIFO() *FIFO {
	return &FIFO{}
}

// Returns true if the FIFO is empty
func (fifo *FIFO) IsEmpty() bool {
	// if the read and write pointers are the same, the FIFO is empty
	return (fifo.WritePtr^fifo.ReadPtr)&0x1f == 0
}

// Returns true if the FIFO is full
func (fifo *FIFO) IsFull() bool {
	// if both pointers point to the same address, but have a different
	// carry
	return (fifo.ReadPtr^fifo.WritePtr^0x10)&0x1f == 0
}

// Resets the FIFO
func (fifo *FIFO) Clear() {
	fifo.ReadPtr = fifo.WritePtr
	for i := 0; i < len(fifo.Buffer); i++ {
		fifo.Buffer[i] = 0
	}
}

// Pushes a value to the FIFO
func (fifo *FIFO) Push(val uint8) {
	fifo.Buffer[fifo.WritePtr&0xf] = val
	fifo.WritePtr++
}

// Increments the read pointer of the FIFO and returns the value at
// that pointer
func (fifo *FIFO) Pop() uint8 {
	fifo.ReadPtr++
	return fifo.Buffer[fifo.ReadPtr&0xf]
}

// Returns the amount of elements in the FIFO. The maximum value
// is 31, and it can overflow
func (fifo *FIFO) Length() uint8 {
	return (fifo.WritePtr - fifo.ReadPtr) & 0x1f
}
