package emulator

// CD-DA audio mixer
type Mixer struct {
	CdLeftToSpuLeft   uint8
	CdLeftToSpuRight  uint8
	CdRightToSpuLeft  uint8
	CdRightToSpuRight uint8
}

func NewMixer() *Mixer {
	// TODO: what are the reset values?
	return &Mixer{}
}
