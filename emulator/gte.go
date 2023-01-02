package emulator

import (
	"fmt"
	"math"
)

// Geometry Transformation Engine (coprocessor 2)
type GTE struct {
	Rbk         int32          // Background color red component (signed 20.12)
	Gbk         int32          // Background color green component (signed 20.12)
	Bbk         int32          // Background color blue component (signed 20.12)
	Rfc         int32          // Far color red component (signed 28.4)
	Gfc         int32          // Far color green component (signed 28.4)
	Bfc         int32          // Far color blue component (signed 28.4)
	Ofx         int32          // Screen offset X (signed 16.16)
	Ofy         int32          // Screen offset Y (signed 16.16)
	H           uint16         // Projection plane distance
	Dqa         int16          // Depth queing coeffient (signed 8.8)
	Dqb         int32          // Depth queing offset (signed 8.24)
	Zsf3        int16          // Scale factor of the average of 3 Z values (triangle, signed 4.12)
	Zsf4        int16          // Scale factor of the average of 4 Z values (quad, signed 4.12)
	Matrices    [3][3][3]int16 // Rotation, light, color
	CtrlVectors [4][3]int32    // Background color, far color, zero
	Flags       uint32         // Overflow flags
	V           [4][3]int16    // Vectors
	Mac         [4]int32       // Accumulators (4 * word)
	Otz         uint16         // Average Z value
	Rgb         [4]uint8       // The last byte contains a GP0 command
	Ir          [4]int16       // Accumulators (4 * halfword)
	XyFifo      [4][2]int16    // XY fifo
	ZFifo       [4]uint16      // Z fifo
	RgbFifo     [3][4]uint8    // RGB fifo
	Lzcs        uint32         // Input value for `Lzcr`
	Lzcr        uint8          // Number of leading zeroes in `Lzcs`
}

// Returns a new GTE instance
func NewGTE() *GTE {
	return &GTE{
		Lzcr: 32,
	}
}

// Set value of a control register
func (gte *GTE) SetControl(reg, val uint32) {
	// TODO: there should be a store delay when setting a GTE register
	// TODO: make this cleaner :P
	switch reg {
	case 0:
		gte.Matrices[MATRIX_ROTATION][0][0] = int16(val)
		gte.Matrices[MATRIX_ROTATION][0][1] = int16(val >> 16)
	case 1:
		gte.Matrices[MATRIX_ROTATION][0][2] = int16(val)
		gte.Matrices[MATRIX_ROTATION][1][0] = int16(val >> 16)
	case 2:
		gte.Matrices[MATRIX_ROTATION][1][1] = int16(val)
		gte.Matrices[MATRIX_ROTATION][1][2] = int16(val >> 16)
	case 3:
		gte.Matrices[MATRIX_ROTATION][2][0] = int16(val)
		gte.Matrices[MATRIX_ROTATION][2][1] = int16(val >> 16)
	case 4:
		gte.Matrices[MATRIX_ROTATION][2][2] = int16(val)
	case 5, 6, 7:
		gte.CtrlVectors[CV_TRANSLATION][reg-5] = int32(val)
	case 8:
		gte.Matrices[MATRIX_LIGHT][0][0] = int16(val)
		gte.Matrices[MATRIX_LIGHT][0][1] = int16(val >> 16)
	case 9:
		gte.Matrices[MATRIX_LIGHT][0][2] = int16(val)
		gte.Matrices[MATRIX_LIGHT][1][0] = int16(val >> 16)
	case 10:
		gte.Matrices[MATRIX_LIGHT][1][1] = int16(val)
		gte.Matrices[MATRIX_LIGHT][1][2] = int16(val >> 16)
	case 11:
		gte.Matrices[MATRIX_LIGHT][2][0] = int16(val)
		gte.Matrices[MATRIX_LIGHT][2][1] = int16(val >> 16)
	case 12:
		gte.Matrices[MATRIX_LIGHT][2][2] = int16(val)
	case 13, 14, 15:
		gte.CtrlVectors[CV_BACKGROUNDCOLOR][reg-13] = int32(val)
	case 16:
		gte.Matrices[MATRIX_COLOR][0][0] = int16(val)
		gte.Matrices[MATRIX_COLOR][0][1] = int16(val >> 16)
	case 17:
		gte.Matrices[MATRIX_COLOR][0][2] = int16(val)
		gte.Matrices[MATRIX_COLOR][1][0] = int16(val >> 16)
	case 18:
		gte.Matrices[MATRIX_COLOR][1][1] = int16(val)
		gte.Matrices[MATRIX_COLOR][1][2] = int16(val >> 16)
	case 19:
		gte.Matrices[MATRIX_COLOR][2][0] = int16(val)
		gte.Matrices[MATRIX_COLOR][2][1] = int16(val >> 16)
	case 20:
		gte.Matrices[MATRIX_COLOR][2][2] = int16(val)
	case 21, 22, 23:
		gte.CtrlVectors[CV_FARCOLOR][reg-21] = int32(val)
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
	case 31:
		gte.Flags = val & 0x7ffff00
		msb := val&0x7f87e000 != 0
		gte.Flags |= oneIfTrue(msb) << 31
	default:
		panicFmt("gte: unhandled control register %d <- 0x%x", reg, val)
	}
}

// Set a value of a data register
func (gte *GTE) SetData(reg, val uint32) {
	switch reg {
	case 0:
		gte.V[0][0] = int16(val)
		gte.V[0][1] = int16(val >> 16)
	case 1:
		gte.V[0][2] = int16(val)
	case 2:
		gte.V[1][0] = int16(val)
		gte.V[1][1] = int16(val >> 16)
	case 3:
		gte.V[1][2] = int16(val)
	case 4:
		gte.V[2][0] = int16(val)
		gte.V[2][1] = int16(val >> 16)
	case 5:
		gte.V[2][2] = int16(val)
	case 6:
		gte.Rgb[0] = uint8(val)       // red
		gte.Rgb[1] = uint8(val >> 8)  // green
		gte.Rgb[2] = uint8(val >> 16) // blue
		gte.Rgb[3] = uint8(val >> 24) // gp0 command
	case 7:
		gte.Otz = uint16(val)
	case 8:
		gte.Ir[0] = int16(val)
	case 9:
		gte.Ir[1] = int16(val)
	case 10:
		gte.Ir[2] = int16(val)
	case 11:
		gte.Ir[3] = int16(val)
	case 12:
		gte.XyFifo[0][0] = int16(val)
		gte.XyFifo[0][1] = int16(val >> 16)
	case 13:
		gte.XyFifo[1][0] = int16(val)
		gte.XyFifo[1][1] = int16(val >> 16)
	case 14:
		x, y := int16(val), int16(val>>16)
		gte.XyFifo[2][0] = x
		gte.XyFifo[2][1] = y
		gte.XyFifo[3][0] = x
		gte.XyFifo[3][1] = y
	case 16:
		gte.ZFifo[0] = uint16(val)
	case 17:
		gte.ZFifo[1] = uint16(val)
	case 18:
		gte.ZFifo[2] = uint16(val)
	case 19:
		gte.ZFifo[3] = uint16(val)
	case 22:
		gte.RgbFifo[2][0] = uint8(val)
		gte.RgbFifo[2][1] = uint8(val >> 8)
		gte.RgbFifo[2][2] = uint8(val >> 16)
		gte.RgbFifo[2][3] = uint8(val >> 24)
	case 24:
		gte.Mac[0] = int32(val)
	case 25:
		gte.Mac[1] = int32(val)
	case 26:
		gte.Mac[2] = int32(val)
	case 27:
		gte.Mac[3] = int32(val)
	case 31:
		fmt.Println("gte: write to read-only register 31")
	default:
		panicFmt("gte: unhandled data register store %d <- 0x%x", reg, val)
	}
}

// Returns the value of any data register
func (gte *GTE) Data(reg uint32) uint32 {
	switch reg {
	case 0:
		v0 := uint32(uint16(gte.V[0][0]))
		v1 := uint32(uint16(gte.V[0][1]))
		return v0 | v1<<16
	case 1:
		return uint32(gte.V[0][2])
	case 2:
		v0 := uint32(uint16(gte.V[1][0]))
		v1 := uint32(uint16(gte.V[1][1]))
		return v0 | v1<<16
	case 3:
		return uint32(gte.V[1][2])
	case 4:
		v0 := uint32(uint16(gte.V[2][0]))
		v1 := uint32(uint16(gte.V[2][1]))
		return v0 | v1<<16
	case 5:
		return uint32(gte.V[2][2])
	case 6:
		r := uint32(gte.Rgb[0])
		g := uint32(gte.Rgb[1])
		b := uint32(gte.Rgb[2])
		c := uint32(gte.Rgb[3]) // gp0 command
		return r | (g << 8) | (b << 16) | (c << 24)
	case 7:
		return uint32(gte.Otz)
	case 8:
		return uint32(gte.Ir[0])
	case 9:
		return uint32(gte.Ir[1])
	case 10:
		return uint32(gte.Ir[2])
	case 11:
		return uint32(gte.Ir[3])
	case 12:
		x := uint32(gte.XyFifo[0][0])
		y := uint32(gte.XyFifo[0][1])
		return x | (y << 16)
	case 13:
		x := uint32(gte.XyFifo[1][0])
		y := uint32(gte.XyFifo[1][1])
		return x | (y << 16)
	case 14:
		x := uint32(gte.XyFifo[2][0])
		y := uint32(gte.XyFifo[2][1])
		return x | (y << 16)
	case 15:
		x := uint32(gte.XyFifo[3][0])
		y := uint32(gte.XyFifo[3][1])
		return x | (y << 16)
	case 16:
		return uint32(gte.ZFifo[0])
	case 17:
		return uint32(gte.ZFifo[1])
	case 18:
		return uint32(gte.ZFifo[2])
	case 19:
		return uint32(gte.ZFifo[3])
	case 22:
		r := uint32(gte.RgbFifo[2][0])
		g := uint32(gte.RgbFifo[2][1])
		b := uint32(gte.RgbFifo[2][2])
		c := uint32(gte.RgbFifo[2][3]) // gp0 command
		r |= g << 8
		r |= b << 16
		r |= c << 24
		return r
	case 24:
		return uint32(gte.Mac[0])
	case 25:
		return uint32(gte.Mac[1])
	case 26:
		return uint32(gte.Mac[2])
	case 27:
		return uint32(gte.Mac[3])
	case 28, 29:
		v0 := saturate(gte.Ir[1] >> 7)
		v1 := saturate(gte.Ir[2] >> 7)
		v2 := saturate(gte.Ir[3] >> 7)
		return v0 | (v1 << 5) | (v2 << 10)
	case 31:
		return uint32(gte.Lzcr)
	default:
		panicFmt("gte: unhandled GTE data read %d", reg)
	}
	return 0
}

func saturate(v int16) uint32 {
	// clamp to 0..0x1f
	if v < 0 {
		return 0
	}
	if v > 0x1f {
		return 0x1f
	}
	return uint32(v)
}

// Returns the value of any control register
func (gte *GTE) Control(reg uint32) uint32 {
	switch reg {
	case 0:
		matrix := gte.Matrices[MATRIX_ROTATION]
		v0 := uint32(uint16(matrix[0][0]))
		v1 := uint32(uint16(matrix[0][1]))
		return v0 | v1<<16
	case 1:
		matrix := gte.Matrices[MATRIX_ROTATION]
		v0 := uint32(uint16(matrix[0][2]))
		v1 := uint32(uint16(matrix[1][0]))
		return v0 | v1<<16
	case 2:
		matrix := gte.Matrices[MATRIX_ROTATION]
		v0 := uint32(uint16(matrix[1][1]))
		v1 := uint32(uint16(matrix[1][2]))
		return v0 | v1<<16
	case 3:
		matrix := gte.Matrices[MATRIX_ROTATION]
		v0 := uint32(uint16(matrix[2][0]))
		v1 := uint32(uint16(matrix[2][1]))
		return v0 | v1<<16
	case 4:
		matrix := gte.Matrices[MATRIX_ROTATION]
		return uint32(uint16(matrix[2][2]))
	case 5, 6, 7:
		vector := gte.CtrlVectors[CV_TRANSLATION]
		return uint32(vector[reg-5])
	case 8:
		matrix := gte.Matrices[MATRIX_LIGHT]
		v0 := uint32(uint16(matrix[0][0]))
		v1 := uint32(uint16(matrix[0][1]))
		return v0 | v1<<16
	case 9:
		matrix := gte.Matrices[MATRIX_LIGHT]
		v0 := uint32(uint16(matrix[0][2]))
		v1 := uint32(uint16(matrix[1][0]))
		return v0 | v1<<16
	case 10:
		matrix := gte.Matrices[MATRIX_LIGHT]
		v0 := uint32(uint16(matrix[1][1]))
		v1 := uint32(uint16(matrix[1][2]))
		return v0 | v1<<16
	case 11:
		matrix := gte.Matrices[MATRIX_LIGHT]
		v0 := uint32(uint16(matrix[2][0]))
		v1 := uint32(uint16(matrix[2][1]))
		return v0 | v1<<16
	case 12:
		matrix := gte.Matrices[MATRIX_LIGHT]
		return uint32(uint16(matrix[2][2]))
	case 13, 14, 15:
		vector := gte.CtrlVectors[CV_BACKGROUNDCOLOR]
		return uint32(vector[reg-13])
	case 16:
		matrix := gte.Matrices[MATRIX_COLOR]
		v0 := uint32(uint16(matrix[0][0]))
		v1 := uint32(uint16(matrix[0][1]))
		return v0 | v1<<16
	case 17:
		matrix := gte.Matrices[MATRIX_COLOR]
		v0 := uint32(uint16(matrix[0][2]))
		v1 := uint32(uint16(matrix[1][0]))
		return v0 | v1<<16
	case 18:
		matrix := gte.Matrices[MATRIX_COLOR]
		v0 := uint32(uint16(matrix[1][1]))
		v1 := uint32(uint16(matrix[1][2]))
		return v0 | v1<<16
	case 19:
		matrix := gte.Matrices[MATRIX_COLOR]
		v0 := uint32(uint16(matrix[2][0]))
		v1 := uint32(uint16(matrix[2][1]))
		return v0 | v1<<16
	case 20:
		matrix := gte.Matrices[MATRIX_COLOR]
		return uint32(uint16(matrix[2][2]))
	case 21, 22, 23:
		vector := gte.CtrlVectors[CV_FARCOLOR]
		return uint32(vector[reg-21])
	case 24:
		return uint32(gte.Ofx)
	case 25:
		return uint32(gte.Ofy)
	case 26:
		return uint32(int16(gte.H))
	case 27:
		return uint32(gte.Dqa)
	case 28:
		return uint32(gte.Dqb)
	case 29:
		return uint32(gte.Zsf3)
	case 30:
		return uint32(gte.Zsf4)
	case 31:
		return gte.Flags
	default:
		panicFmt("gte: unhandled control register read %d", reg)
	}
	return 0
}

// Execute command
func (gte *GTE) Command(cmd uint32) {
	opcode := cmd & 0x3f
	gte.Flags = 0
	// fmt.Printf("gte: command 0x%x\n", opcode)

	switch opcode {
	case 0x06:
		gte.CommandNCLIP()
	case 0x13:
		config := CommandConfigFromCommand(cmd)
		gte.CommandNCDS(config)
	case 0x2d:
		gte.CommandAVSZ3()
	case 0x30:
		config := CommandConfigFromCommand(cmd)
		gte.CommandRTPT(config)
	default:
		panicFmt("gte: unhandled command 0x%x (opcode 0x%x)", cmd, opcode)
	}

	// flags MSB [30:23] + [18:13]
	msb := gte.Flags&0x7f87e000 != 0
	gte.Flags |= oneIfTrue(msb) << 31
}

// Normal clipping
func (gte *GTE) CommandNCLIP() {
	x0, y0 := int32(gte.XyFifo[0][0]), int32(gte.XyFifo[0][1])
	x1, y1 := int32(gte.XyFifo[1][0]), int32(gte.XyFifo[1][1])
	x2, y2 := int32(gte.XyFifo[2][0]), int32(gte.XyFifo[2][1])

	v0 := x0 * (y1 - y2)
	v1 := x1 * (y2 - y0)
	v2 := x2 * (y0 - y1)

	sum := int64(v0) + int64(v1) + int64(v2)
	gte.Mac[0] = gte.I64ToI32Result(sum)
}

// Normal color depth cue single vector
func (gte *GTE) CommandNCDS(config CommandConfig) {
	gte.DoNCD(config, 0)
}

// Average of 3 Z values
func (gte *GTE) CommandAVSZ3() {
	z1 := uint32(gte.ZFifo[1])
	z2 := uint32(gte.ZFifo[2])
	z3 := uint32(gte.ZFifo[3])
	sum := z1 + z2 + z3

	zsf3 := int64(gte.Zsf3)
	average := zsf3 * int64(sum)

	gte.Mac[0] = gte.I64ToI32Result(average)
	gte.Otz = gte.I64ToOTZ(average)
}

func (gte *GTE) CommandRTPT(config CommandConfig) {
	// transform vectors
	gte.DoRTP(config, 0)
	gte.DoRTP(config, 1)
	// do depth queuing on the Z vector
	projectionFactor := gte.DoRTP(config, 2)
	gte.DoDepthQueuing(projectionFactor)
}

func (gte *GTE) DoDepthQueuing(projectionFactor uint32) {
	factor := int64(projectionFactor)
	dqa := int64(gte.Dqa)
	dqb := int64(gte.Dqb)
	depth := dqb + dqa*factor

	gte.Mac[0] = gte.I64ToI32Result(depth)

	// set the 16 bit IR value
	depth >>= 12

	if depth < 0 {
		gte.SetFlag(12)
		gte.Ir[0] = 0
	} else if depth > 4096 {
		gte.SetFlag(12)
		gte.Ir[0] = 4096
	} else {
		gte.Ir[0] = int16(depth)
	}
}

func (gte *GTE) DoRTP(config CommandConfig, vectorIndex int) uint32 {
	// the result Z coordinate shifted by 12 bits
	var zShifted int32 = 0

	// step 1: compute "tr + vector * rm"
	rm := MATRIX_ROTATION
	tr := CV_TRANSLATION

	for r := 0; r < 3; r++ {
		res := int64(gte.CtrlVectors[tr][r]) << 12

		for c := 0; c < 3; c++ {
			v := int32(gte.V[vectorIndex][c])
			m := int32(gte.Matrices[rm][r][c])

			rot := v * m
			res = gte.I64ToI44(uint8(c), res+int64(rot))
		}

		gte.Mac[r+1] = int32(res >> int64(config.Shift))
		zShifted = int32(res >> 12)
	}

	// step 2: get camera coordinates from MAC and convert them to 16 bit
	// in IR
	val := gte.Mac[1]
	gte.Ir[1] = gte.I32ToI16Saturate(config, 0, val)
	val = gte.Mac[2]
	gte.Ir[2] = gte.I32ToI16Saturate(config, 1, val)

	// weird Z coordinate clamping behaviour
	max := int32(math.MaxInt16)
	min := int32(math.MinInt16)

	if zShifted > max || zShifted < min {
		gte.SetFlag(22)
	}

	// clamp the value
	if config.ClampNegative {
		min = 0
	}
	val = gte.Mac[3]
	if val < min {
		gte.Ir[3] = int16(min)
	} else if val > max {
		gte.Ir[3] = int16(max)
	} else {
		gte.Ir[3] = int16(val)
	}

	// step 3: clamp the shifted Z value to a saturated 16 bit
	// unsigned integer and push it to the Z fifo
	var zSaturated uint16 = 0
	if zShifted < 0 {
		gte.SetFlag(18)
		// it is 0
	} else if zShifted > math.MaxUint16 {
		gte.SetFlag(18)
		zSaturated = math.MaxUint16
	} else {
		zSaturated = uint16(zShifted)
	}

	// push it to the FIFO
	gte.ZFifo[0] = gte.ZFifo[1]
	gte.ZFifo[1] = gte.ZFifo[2]
	gte.ZFifo[2] = gte.ZFifo[3]
	gte.ZFifo[3] = zSaturated

	// step 3: perspective projection against the screen plane
	var projectionFactor uint32
	if zSaturated > gte.H/2 {
		projectionFactor = GTEDivide(gte.H, zSaturated)
	} else {
		// clip
		gte.SetFlag(17)
		projectionFactor = 0x1ffff
	}

	factor := int64(projectionFactor)
	x := int64(gte.Ir[1])
	y := int64(gte.Ir[2])
	ofx := int64(gte.Ofx)
	ofy := int64(gte.Ofy)

	// project X and Y onto the plane
	screenX := gte.I64ToI32Result(x*factor+ofx) >> 16
	screenY := gte.I64ToI32Result(y*factor+ofy) >> 16

	// push it to the XY fifo
	gte.XyFifo[3][0] = gte.I32ToI11Saturate(0, screenX)
	gte.XyFifo[3][1] = gte.I32ToI11Saturate(1, screenY)
	copy(gte.XyFifo[0][:], gte.XyFifo[1][:])
	copy(gte.XyFifo[1][:], gte.XyFifo[2][:])
	copy(gte.XyFifo[2][:], gte.XyFifo[3][:])

	return projectionFactor
}

func (gte *GTE) DoNCD(config CommandConfig, vectorIndex int) {
	gte.MultiplyMatrixByVector(config, MATRIX_LIGHT, vectorIndex, CV_ZERO)
	gte.V[3][0] = gte.Ir[1]
	gte.V[3][1] = gte.Ir[2]
	gte.V[3][2] = gte.Ir[3]
	gte.MultiplyMatrixByVector(config, MATRIX_COLOR, 3, CV_BACKGROUNDCOLOR)

	r := gte.Rgb[0]
	g := gte.Rgb[1]
	b := gte.Rgb[2]
	col := []uint8{r, g, b}

	for i := 0; i < 3; i++ {
		fc := int64(gte.CtrlVectors[CV_FARCOLOR][i]) << 12
		ir := int32(gte.Ir[i+1])
		clr := int32(col[i]) << 4

		shading := int64(clr * ir)
		product := fc - shading
		tmp := gte.I64ToI32Result(product) >> int32(config.Shift)
		ir0 := int64(gte.Ir[0])
		m := int64(gte.I32ToI16Saturate(CommandConfigFromCommand(0), uint8(i), tmp))
		res := gte.I64ToI32Result(shading + ir0*m)

		gte.Mac[i+1] = res >> int32(config.Shift)
	}

	gte.MacToIr(config)
	gte.MacToRgbFifo()
}

func (gte *GTE) MultiplyMatrixByVector(
	config CommandConfig,
	matrix Matrix,
	vectorIndex int,
	ctrlVector ControlVector,
) {
	if matrix == MATRIX_INVALID {
		// TODO: this should output bogus results
		panic("gte: multiplication of invalid matrix")
	}
	if ctrlVector == CV_FARCOLOR {
		panic("gte: multiplication with far color vector") // TODO
	}

	// iterate over matrix rows
	for r := 0; r < 3; r++ {
		// add the control vector to the result
		res := int64(gte.CtrlVectors[ctrlVector][r]) << 12

		// iterate over matrix columns
		for c := 0; c < 3; c++ {
			v := int32(gte.V[vectorIndex][c])
			m := int32(gte.Matrices[matrix][r][c])

			product := v * m
			res = gte.I64ToI44(uint8(c), res+int64(product))
		}

		// store result in accumulator
		gte.Mac[r+1] = int32(res >> int64(config.Shift))
	}

	gte.MacToIr(config)
}

func (gte *GTE) MacToIr(config CommandConfig) {
	gte.Ir[1] = gte.I32ToI16Saturate(config, 0, gte.Mac[1])
	gte.Ir[2] = gte.I32ToI16Saturate(config, 1, gte.Mac[2])
	gte.Ir[3] = gte.I32ToI16Saturate(config, 2, gte.Mac[3])
}

func (gte *GTE) macToColor(mac int32, which uint8) uint8 {
	c := mac >> 4

	if c < 0 {
		gte.SetFlag(21 - which)
		return 0
	}
	if c > 0xff {
		gte.SetFlag(21 - which)
		return 0xff
	}
	return uint8(c)
}

func (gte *GTE) MacToRgbFifo() {
	mac1 := gte.Mac[1]
	mac2 := gte.Mac[2]
	mac3 := gte.Mac[3]

	r := gte.macToColor(mac1, 0)
	g := gte.macToColor(mac2, 1)
	b := gte.macToColor(mac3, 2)

	c := gte.Rgb[3]
	copy(gte.RgbFifo[0][:], gte.RgbFifo[1][:])
	copy(gte.RgbFifo[1][:], gte.RgbFifo[2][:])
	gte.RgbFifo[2][0] = r
	gte.RgbFifo[2][1] = g
	gte.RgbFifo[2][2] = b
	gte.RgbFifo[2][3] = c
}
