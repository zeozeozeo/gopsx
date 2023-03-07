package emulator

import "fmt"

// CDROM-XA sector sync pattern, used to validate the sector
var XA_SECTOR_SYNC_PATTERN = []byte{
	0x00,
	0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff,
	0x00,
}

type SectorMode int

const (
	SECTOR_M1       SectorMode = 0 // Mode 1
	SECTOR_M2_FORM1 SectorMode = 1 // Mode 2 form 1 XA type
	SECTOR_M2_FORM2 SectorMode = 2 // Mode 2 form 2 XA type
	SECTOR_INVALID  SectorMode = 3 // Hasn't been validated yet
)

// CDROM-XA sector
type XaSector struct {
	Data [SECTOR_SIZE]byte // Data
	Mode SectorMode        // Sector mode. Only set after validation
}

// Returns a new XaSector instance
func NewXaSector() *XaSector {
	return &XaSector{
		Mode: SECTOR_INVALID,
	}
}

// Returns the byte at `index`
func (sector *XaSector) DataByte(index uint16) byte {
	return sector.Data[index]
}

// Returns the sector data as a slice (2352 bytes)
func (sector *XaSector) DataBytes() []byte {
	return sector.Data[:]
}

// Returns the sector data, skipping the sync pattern
func (sector *XaSector) DataNoSyncPattern() []byte {
	return sector.Data[12:]
}

func (sector *XaSector) Mode2XaPayload() ([]byte, error) {
	switch sector.Mode {
	case SECTOR_M2_FORM1:
		return sector.Data[24:2072], nil
	case SECTOR_M2_FORM2:
		return sector.Data[24:2348], nil
	}
	return nil, fmt.Errorf("invalid sector mode %d", sector.Mode)
}

// Returns the sector MSF (stored in bytes 12,13,14)
func (sector *XaSector) Msf() *Msf {
	return MsfFromBcd(
		sector.Data[12],
		sector.Data[13],
		sector.Data[14],
	)
}

// Validate the sector (returns nil if successful)
func (sector *XaSector) ValidateMode1Or2(msf *Msf) error {
	// validate sync pattern
	for idx, v := range sector.Data[:12] {
		if v != XA_SECTOR_SYNC_PATTERN[idx] {
			return fmt.Errorf("invalid sync pattern at %s", msf)
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
		return fmt.Errorf("xa: unhandled mode %d at %s", mode, msf)
	}
}

// Validate mode 2
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

	if submode&0x20 != 0 {
		sector.Mode = SECTOR_M2_FORM2
		return sector.ValidateMode2Form2()
	}
	sector.Mode = SECTOR_M2_FORM1
	return sector.ValidateMode2Form1()
}

// Validate CRC
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
	// panic("xa: not implemented")
	// ignore this for now
	return nil
}
