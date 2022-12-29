package emulator

import "fmt"

// Geometry Transformation Engine (coprocessor 2)
type GTE struct {
	Rbk  int32       // Background color red component (signed 20.12)
	Gbk  int32       // Background color green component (signed 20.12)
	Bbk  int32       // Background color blue component (signed 20.12)
	Rfc  int32       // Far color red component (signed 28.4)
	Gfc  int32       // Far color green component (signed 28.4)
	Bfc  int32       // Far color blue component (signed 28.4)
	Ofx  int32       // Screen offset X (signed 16.16)
	Ofy  int32       // Screen offset Y (signed 16.16)
	H    uint16      // Projection plane distance
	Dqa  int16       // Depth queing coeffient (signed 8.8)
	Dqb  int32       // Depth queing offset (signed 8.24)
	Zsf3 int16       // Scale factor of the average of 3 Z values (triangle, signed 4.12)
	Zsf4 int16       // Scale factor of the average of 4 Z values (quad, signed 4.12)
	Tr   [3]int32    // Translation vector (3x signed word)
	Lsm  [3][3]int16 // Light source matrix (3x3 signed 4.12)
	Lcm  [3][3]int16 // Light color matrix (3x3 signed 4.12)
	Rm   [3][3]int16 // Rotation matrix (3x3 signed 4.12)
	V0   [3]int16    // Vector 0 (3x signed 4.12)
	V1   [3]int16    // Vector 1 (3x signed 4.12)
	V2   [3]int16    // Vector 2 (3x signed 4.12)
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
	case 0:
		gte.Rm[0][0] = int16(val)
		gte.Rm[0][1] = int16(val >> 16)
	case 1:
		gte.Rm[0][2] = int16(val)
		gte.Rm[1][0] = int16(val >> 16)
	case 2:
		gte.Rm[1][1] = int16(val)
		gte.Rm[1][2] = int16(val >> 16)
	case 3:
		gte.Rm[2][0] = int16(val)
		gte.Rm[2][1] = int16(val >> 16)
	case 4:
		gte.Rm[2][2] = int16(val)
	case 5:
		gte.Tr[0] = int32(val)
	case 6:
		gte.Tr[1] = int32(val)
	case 7:
		gte.Tr[2] = int32(val)
	case 8:
		gte.Lsm[0][0] = int16(val)
		gte.Lsm[0][1] = int16(val >> 16)
	case 9:
		gte.Lsm[0][2] = int16(val)
		gte.Lsm[1][0] = int16(val >> 16)
	case 10:
		gte.Lsm[1][1] = int16(val)
		gte.Lsm[1][2] = int16(val >> 16)
	case 11:
		gte.Lsm[2][0] = int16(val)
		gte.Lsm[2][1] = int16(val >> 16)
	case 12:
		gte.Lsm[2][2] = int16(val)
	case 13:
		gte.Rbk = int32(val)
	case 14:
		gte.Gbk = int32(val)
	case 15:
		gte.Bbk = int32(val)
	case 16:
		gte.Lcm[0][0] = int16(val)
		gte.Lcm[0][1] = int16(val >> 16)
	case 17:
		gte.Lcm[0][2] = int16(val)
		gte.Lcm[1][0] = int16(val >> 16)
	case 18:
		gte.Lcm[1][1] = int16(val)
		gte.Lcm[1][2] = int16(val >> 16)
	case 19:
		gte.Lcm[2][0] = int16(val)
		gte.Lcm[2][1] = int16(val >> 16)
	case 20:
		gte.Lcm[2][2] = int16(val)
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
		panicFmt("gte: unhandled control register %d <- 0x%x", reg, val)
	}
}

// Set a value of a data register
func (gte *GTE) SetData(reg, val uint32) {
	fmt.Printf("gte: SetData: %d <- 0x%x\n", reg, val)

	switch reg {
	case 0:
		gte.V0[0] = int16(val)
		gte.V0[1] = int16(val >> 16)
	case 1:
		gte.V0[2] = int16(val)
	case 2:
		gte.V1[0] = int16(val)
		gte.V1[1] = int16(val >> 16)
	case 3:
		gte.V1[2] = int16(val)
	case 4:
		gte.V2[0] = int16(val)
		gte.V2[1] = int16(val >> 16)
	case 5:
		gte.V0[2] = int16(val)
	default:
		panicFmt("gte: unhandled data register store %d <- 0x%x", reg, val)
	}
}

// Execute command
func (gte *GTE) Command(cmd uint32) {
	panicFmt("gte: unhandled command 0x%x", cmd)
}
