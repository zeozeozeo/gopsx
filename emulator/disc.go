package emulator

import (
	"fmt"
	"io"
)

// CD sector size in bytes
const SECTOR_SIZE uint64 = 2352

// Represents a disc region
type Region int

const (
	REGION_JAPAN         Region = iota // Japan (NTSC): SCEI
	REGION_NORTH_AMERICA Region = iota // North America (NTSC): SCEA
	REGION_EUROPE        Region = iota // Europe (PAL): SCEE
)

func GetHardwareFromRegion(region Region) HardwareType {
	switch region {
	case REGION_JAPAN, REGION_NORTH_AMERICA:
		return HARDWARE_NTSC
	case REGION_EUROPE:
		return HARDWARE_PAL
	}
	return HARDWARE_NTSC
}

// A PlayStation disc
type Disc struct {
	File   io.ReadSeeker // BIN reader
	Region Region
}

// Creates a new disc instance
func NewDisc(r io.ReadSeeker) (*Disc, error) {
	disc := &Disc{
		File: r,
	}
	err := disc.IdentifyRegion()
	if err != nil {
		return nil, err
	}
	return disc, nil
}

func (disc *Disc) RegionString() string {
	switch disc.Region {
	case REGION_JAPAN:
		return "Japan"
	case REGION_NORTH_AMERICA:
		return "North America"
	case REGION_EUROPE:
		return "Europe"
	}
	return ""
}

// Identifies the region of the disc
func (disc *Disc) IdentifyRegion() error {
	// sector 00:02:04 should contain the "Licensed by"... string
	msf := MsfFromBcd(0x00, 0x02, 0x04)
	sector, err := disc.ReadDataSector(msf)
	if err != nil {
		panic(err)
	}

	licenseData := sector.DataBytes()[0:76]

	// only leave characters A-z
	var license string
	for _, char := range licenseData {
		if char >= 'A' && char <= 'z' {
			license += string(char)
		}
	}

	switch license {
	case "LicensedbySonyComputerEntertainmentInc": // Japan
		disc.Region = REGION_JAPAN
	case "LicensedbySonyComputerEntertainmentAmerica": // North America
		disc.Region = REGION_NORTH_AMERICA
	case "LicensedbySonyComputerEntertainmentEurope": // Europe
		disc.Region = REGION_EUROPE
	default:
		return fmt.Errorf("invalid disc region string \"%s\"", license)
	}
	return nil
}

func (disc *Disc) ReadDataSector(msf Msf) (*XaSector, error) {
	sector, err := disc.ReadSector(msf)
	if err != nil {
		return nil, err
	}
	sector.ValidateMode1Or2(msf)
	return sector, nil
}

func (disc *Disc) ReadSector(msf Msf) (*XaSector, error) {
	index := msf.SectorIndex() - 150 // TODO: parse cuesheet
	pos := uint64(index) * SECTOR_SIZE
	_, err := disc.File.Seek(int64(pos), io.SeekStart)
	if err != nil {
		return nil, err
	}

	sector := NewXaSector()
	nread := 0

	for uint64(nread) < SECTOR_SIZE {
		n, err := disc.File.Read(sector.Data[nread:])
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, fmt.Errorf("0 length sector read at 0x%x", nread)
		}
		nread += n
	}

	return sector, nil
}
