package emulator

import "fmt"

type Msf struct {
	M, S, F uint8
}

// Creates a new Msf instance (all values are 0)
func NewMsf() Msf {
	return Msf{}
}

func (msf Msf) String() string {
	return fmt.Sprintf("%d:%d:%d", msf.M, msf.S, msf.F)
}

func (msf Msf) Values() (uint8, uint8, uint8) {
	return msf.M, msf.S, msf.F
}

func (msf Msf) Slice() []uint8 {
	return []uint8{msf.M, msf.S, msf.F}
}

func MsfFromBcd(m, s, f uint8) Msf {
	msf := Msf{m, s, f}

	// check if the MSF is valid
	for _, v := range msf.Slice() {
		if v > 0x99 || (v&0xf) > 0x9 {
			panicFmt("msf: invalid MSF: %s", msf)
		}
	}
	if s >= 0x60 || f >= 0x75 {
		panicFmt("msf: invalid MSF: %s", msf)
	}

	return msf
}

// Converts an MSF into a sector index
func (msf Msf) SectorIndex() uint32 {
	m := uint32((msf.M>>4)*10 + (msf.M & 0xf))
	s := uint32((msf.S>>4)*10 + (msf.S & 0xf))
	f := uint32((msf.F>>4)*10 + (msf.F & 0xf))
	return (60 * 75 * m) + (75 * s) + f
}

// Returns the MSF of the next sector
func (msf Msf) Next() Msf {
	m, s, f := msf.Values()

	if f < 0x74 {
		return Msf{m, s, incBcd(f)}
	}
	if s < 0x59 {
		return Msf{m, incBcd(s), f}
	}
	if m < 0x99 {
		return Msf{incBcd(m), s, f}
	}
	panic("msf: Next() overflow")
}

func incBcd(v uint8) uint8 {
	if v&0xf < 9 {
		return v + 1
	}
	return (v & 0xf0) + 0x10
}

func (msf Msf) ToU32() uint32 {
	m, s, f := msf.Values()
	return (uint32(m) << 16) | (uint32(s) << 8) | uint32(f)
}

func (msf Msf) IsEqual(msf2 Msf) bool {
	return msf.M == msf2.M && msf.S == msf2.S && msf.F == msf2.F
}
