package emulator

import "fmt"

var XA_SECTOR_SYNC_PATTERN = []byte{
	0x00,
	0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff,
	0x00,
}

type XaSector struct {
	Data       [SECTOR_SIZE]byte // Data
	DataOffset uint16            // Offset of the first data byte in the sector
}

func NewXaSector() *XaSector {
	return &XaSector{}
}

func (sector *XaSector) DataByte(index uint16) byte {
	index += sector.DataOffset
	return sector.Data[index]
}

func (sector *XaSector) DataBytes() []byte {
	return sector.Data[sector.DataOffset:]
}

func (sector *XaSector) Msf() Msf {
	return MsfFromBcd(
		sector.Data[12],
		sector.Data[13],
		sector.Data[14],
	)
}

func (sector *XaSector) ValidateMode1Or2(msf Msf) error {
	// validate sync pattern
	for idx, v := range sector.Data[0:12] {
		if v != XA_SECTOR_SYNC_PATTERN[idx] {
			return fmt.Errorf("invalid sector sync at %s", msf)
		}
	}

	// validate MSF
	sectorMsf := sector.Msf()
	if !msf.IsEqual(sectorMsf) {
		return fmt.Errorf("invalid msf (expected %s, got %s)", msf, sectorMsf)
	}

	mode := sector.Data[15]

	switch mode {
	case 2:
		return sector.ValidateMode2()
	default:
		panicFmt("xa: unhandled mode %d at %s", mode, msf)
	}
	return nil
}

func (sector *XaSector) ValidateMode2() error {
	// byte 16: File number
	// byte 17: Channel number
	// byte 18: Submode
	// byte 19: Coding information
	// byte 20: File number
	// byte 21: Channel number
	// byte 22: Submode
	// byte 23: Coding information
	// data...

	// check if submode copy is the same
	submode := sector.Data[18]
	submodeCopy := sector.Data[22]

	if submode != submodeCopy {
		return fmt.Errorf(
			"mode 2 mismatch at %s (%d and %d)",
			sector.Msf(), submode, submodeCopy,
		)
	}

	sector.DataOffset = 24

	if submode&0x20 != 0 {
		return sector.ValidateMode2Form2()
	}
	return sector.ValidateMode2Form1()
}

func (sector *XaSector) ValidateMode2Form1() error {
	// validate CRC
	crc := Crc32(sector.Data[16:2072])

	sectorCrc := uint32(sector.Data[2072]) |
		(uint32(sector.Data[2073]) << 8) |
		(uint32(sector.Data[2074]) << 16) |
		(uint32(sector.Data[2075]) << 24)
	if crc != sectorCrc {
		return fmt.Errorf("mode 2 form 1 CRC mismatch at %s", sector.Msf())
	}

	return nil
}

func (sector *XaSector) ValidateMode2Form2() error {
	panic("xa: not implemented")
}
