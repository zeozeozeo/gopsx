package emulator

import "math"

// Represents a matrix index in the GTE's matrices
type Matrix int

const (
	MATRIX_ROTATION Matrix = 0 // Rotation matrix
	MATRIX_LIGHT    Matrix = 1 // Light matrix
	MATRIX_COLOR    Matrix = 2 // Color matrix
	MATRIX_INVALID  Matrix = 3 // Bogus operation
)

func MatrixFromCommand(cmd uint32) Matrix {
	switch (cmd >> 17) & 3 {
	case 0:
		return MATRIX_ROTATION
	case 1:
		return MATRIX_LIGHT
	case 2:
		return MATRIX_COLOR
	case 3:
		return MATRIX_INVALID
	default:
		panic("gte: unreachable")
	}
}

// Represents a control vector index in the GTE's control vectors
type ControlVector int

const (
	CV_TRANSLATION     ControlVector = 0
	CV_BACKGROUNDCOLOR ControlVector = 1
	CV_FARCOLOR        ControlVector = 2
	CV_ZERO            ControlVector = 3
)

func ControlVectorFromCommand(cmd uint32) ControlVector {
	switch (cmd >> 13) & 3 {
	case 0:
		return CV_TRANSLATION
	case 1:
		return CV_BACKGROUNDCOLOR
	case 2:
		return CV_FARCOLOR
	case 3:
		return CV_ZERO
	default:
		panic("gte: unreachable")
	}
}

type CommandConfig struct {
	Shift         uint8  // Right shift value
	ClampNegative bool   // Clamp negative results to 0
	Matrix        Matrix // MVMVA command matrix
	CtrlVector    ControlVector
}

func CommandConfigFromCommand(cmd uint32) CommandConfig {
	var shift uint8 = 0
	if cmd&(1<<19) != 0 {
		shift = 12
	}
	clampNegative := cmd&(1<<10) != 0

	return CommandConfig{
		Shift:         shift,
		ClampNegative: clampNegative,
		Matrix:        MatrixFromCommand(cmd),
		CtrlVector:    ControlVectorFromCommand(cmd),
	}
}

func (gte *GTE) SetFlag(bit uint8) {
	gte.Flags |= 1 << bit
}

func (gte *GTE) I64ToI44(flag uint8, val int64) int64 {
	if val > 0x7ffffffffff {
		gte.SetFlag(30 - flag)
	} else if val < -0x80000000000 {
		gte.SetFlag(27 - flag)
	}
	return (val << (64 - 44)) >> (64 - 44)
}

func (gte *GTE) I32ToI16Saturate(config CommandConfig, flag uint8, val int32) int16 {
	var max int32 = math.MaxInt16
	var min int32 = 0
	if !config.ClampNegative {
		min = math.MinInt16
	}

	// clamp
	if val > max {
		gte.SetFlag(24 - flag)
		return int16(max)
	}
	if val < min {
		gte.SetFlag(24 - flag)
		return int16(min)
	}
	return int16(val)
}

func (gte *GTE) I32ToI11Saturate(flag uint8, val int32) int16 {
	// clamp
	if val < -0x400 {
		gte.SetFlag(14 - flag)
		return -0x400
	}
	if val > 0x3ff {
		gte.SetFlag(14 - flag)
		return 0x3ff
	}
	return int16(val)
}

func (gte *GTE) I64ToI32Result(val int64) int32 {
	if val < -0x80000000 {
		gte.SetFlag(15)
	} else if val > 0x7fffffff {
		gte.SetFlag(16)
	}
	return int32(val)
}

func (gte *GTE) I64ToOTZ(average int64) uint16 {
	val := average >> 12

	if val < 0 {
		gte.SetFlag(18)
		return 0
	}
	if val > 0xffff {
		gte.SetFlag(18)
		return 0xffff
	}
	return uint16(val)
}
