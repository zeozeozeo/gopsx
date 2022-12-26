package emulator

import "fmt"

// Geometry Transformation Engine (coprocessor 2)
type GTE struct {
	Rbk int32  // Background color red component (signed 20.12)
	Gbk int32  // Background color green component (signed 20.12)
	Bbk int32  // Background color blue component (signed 20.12)
	Rfc int32  // Far color red component (signed 28.4)
	Gfc int32  // Far color green component (signed 28.4)
	Bfc int32  // Far color blue component (signed 28.4)
	Ofx int32  // Screen offset X (signed 16.16)
	Ofy int32  // Screen offset Y (signed 16.16)
	H   uint16 // Projection plane distance
	Dqa int16  // Depth queing coeffient (signed 8.8)
	Dqb int32  // Depth queing offset (signed 8.24)
	// Scale factor when computing the average of 3 Z values
	// (triangle) (signed 4.12)
	Zsf3 int16
	// Scale factor when computing the average of 4 Z values
	// (quad) (signed 4.12)
	Zsf4 int16
}

// Returns a new GTE instance
func NewGTE() *GTE {
	return &GTE{}
}

// Set value of a control register
func (gte *GTE) SetControl(reg, val uint32) {
	// TODO: there should be a store delay when setting a GTE register
	fmt.Printf("gte: set control %d <- 0x%x\n", reg, val)

	switch reg {
	case 13:
		gte.Rbk = int32(val)
	case 14:
		gte.Gbk = int32(val)
	case 15:
		gte.Bbk = int32(val)
	case 21:
		gte.Rfc = int32(val)
	case 22:
		gte.Gfc = int32(val)
	case 23:
		gte.Bfc = int32(val)
	case 24:
		gte.Ofx = int32(val)
	case 25:
		gte.Ofy = int32(val)
	case 26:
		gte.H = uint16(val)
	case 27:
		gte.Dqa = int16(val)
	case 28:
		gte.Dqb = int32(val)
	case 29:
		gte.Zsf3 = int16(val)
	case 30:
		gte.Zsf4 = int16(val)
	default:
		panicFmt("gte: unhandled control register %d", reg)
	}
}
