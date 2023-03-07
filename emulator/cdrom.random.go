package emulator

// RNG that implements the algorithm from http://www.jstatsoft.org/v08/i14/paper
type CdRomRng struct {
	State uint32 // RNG state
}

func NewCdRomRng() *CdRomRng {
	return &CdRomRng{
		State: 1, // cannot be 0
	}
}

// Returns a next random number from the RNG. Will never be 0
func (rand *CdRomRng) Next() uint32 {
	rand.State ^= rand.State << 3
	rand.State ^= rand.State >> 5
	rand.State ^= rand.State << 25
	return rand.State
}
