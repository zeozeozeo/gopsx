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

func NewFIFOFromBytes(data []byte) *FIFO {
	fifo := NewFIFO()
	for _, v := range data {
		fifo.Push(v)
	}
	return fifo
}

// Returns true if the FIFO is empty
func (fifo *FIFO) IsEmpty() bool {
	// if the read and write pointers are the same, the FIFO is empty
	return fifo.WritePtr == fifo.ReadPtr
}

// Returns true if the FIFO is full
func (fifo *FIFO) IsFull() bool {
	// if both pointers point to the same address, but have a different
	// carry
	return fifo.WritePtr == fifo.ReadPtr^0x10
}

// Resets the FIFO
func (fifo *FIFO) Clear() {
	fifo.ReadPtr = 0
	fifo.WritePtr = 0
	for i := 0; i < len(fifo.Buffer); i++ {
		fifo.Buffer[i] = 0
	}
}

// Pushes a value to the FIFO
func (fifo *FIFO) Push(val byte) {
	fifo.Buffer[fifo.WritePtr&0xf] = val
	fifo.WritePtr = (fifo.WritePtr + 1) & 0x1f
}

func (fifo *FIFO) PushSlice(data []byte) {
	for _, b := range data {
		fifo.Push(b)
	}
}

// Increments the read pointer of the FIFO and returns the value at
// that pointer
func (fifo *FIFO) Pop() byte {
	idx := fifo.ReadPtr & 0xf
	fifo.ReadPtr = (fifo.ReadPtr + 1) & 0x1f
	return fifo.Buffer[idx]
}

// Returns the amount of elements in the FIFO. The maximum value
// is 31, and it can overflow
func (fifo *FIFO) Length() uint8 {
	return (fifo.WritePtr - fifo.ReadPtr) & 0x1f
}
