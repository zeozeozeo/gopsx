package emulator

type ButtonState int

const (
	BUTTON_STATE_PRESSED  ButtonState = 0
	BUTTON_STATE_RELEASED ButtonState = 1
)

type Button uint

const (
	BUTTON_SELECT   Button = 0
	BUTTON_START    Button = 3
	BUTTON_DUP      Button = 4
	BUTTON_DRIGHT   Button = 5
	BUTTON_DDOWN    Button = 6
	BUTTON_DLEFT    Button = 7
	BUTTON_L2       Button = 8
	BUTTON_R2       Button = 9
	BUTTON_L1       Button = 10
	BUTTON_R1       Button = 11
	BUTTON_TRIANGLE Button = 12
	BUTTON_CIRCLE   Button = 13
	BUTTON_CROSS    Button = 14
	BUTTON_SQUARE   Button = 15
)

// All gamepad buttons
var GamepadButtons = []Button{
	BUTTON_SELECT,
	BUTTON_START,
	BUTTON_DUP,
	BUTTON_DRIGHT,
	BUTTON_DDOWN,
	BUTTON_DLEFT,
	BUTTON_L2,
	BUTTON_R2,
	BUTTON_L1,
	BUTTON_R1,
	BUTTON_TRIANGLE,
	BUTTON_CIRCLE,
	BUTTON_CROSS,
	BUTTON_SQUARE,
}

type GamepadType int

const (
	GAMEPAD_TYPE_DISCONNECTED GamepadType = iota // Gamepad is not connected
	GAMEPAD_TYPE_DIGITAL      GamepadType = iota // SCPH-1080: Digital Joypad
)

// Gamepad
type Gamepad struct {
	Profile Profile // Implements Profile
	Seq     uint8   // Current position in reply sequence
	Active  bool    // If false, the current command is done processing
}

func (gp *Gamepad) Select() {
	// prepare for command
	gp.Active = true
	gp.Seq = 0
}

func (gp *Gamepad) SendCommand(cmd uint8) (uint8, bool) {
	if !gp.Active {
		return 0xff, false
	}

	resp, dsr := gp.Profile.HandleCommand(gp.Seq, cmd)
	gp.Active = dsr
	gp.Seq++

	return resp, dsr
}

// Shortcut for gp.Profile.SetButtonState(button, state)
func (gp *Gamepad) SetButtonState(button Button, state ButtonState) {
	gp.Profile.SetButtonState(button, state)
}

// Returns a new Gamepad instance
func NewGamepad(profileType GamepadType) *Gamepad {
	gp := &Gamepad{Active: true}
	switch profileType {
	case GAMEPAD_TYPE_DISCONNECTED:
		gp.Profile = NewDummyPad()
	case GAMEPAD_TYPE_DIGITAL:
		gp.Profile = NewDigitalPad()
	}
	return gp
}

// Interface for controller profiles
type Profile interface {
	HandleCommand(seq, cmd uint8) (uint8, bool)      // Handles commands
	SetButtonState(button Button, state ButtonState) // Handles button events
}

// Empty gamepad slot that implements Profile
type DummyPadProfile struct{}

func (profile *DummyPadProfile) HandleCommand(seq, cmd uint8) (uint8, bool) {
	return 0xff, false
}

func (profile *DummyPadProfile) SetButtonState(button Button, state ButtonState) {
	// NOP
}

// Returns a new instance of DummyPadProfile
func NewDummyPad() *DummyPadProfile {
	return &DummyPadProfile{}
}

// SCPH-1080: Digital Joypad (implements Profile)
type DigitalPadProfile struct {
	State uint16 // Only 1 bit per button, 2 bytes
}

func (profile *DigitalPadProfile) HandleCommand(seq, cmd uint8) (uint8, bool) {
	switch seq {
	case 0: // 0xff: does the command target a controller?
		return 0xff, cmd == 0x01
	case 1: // 0x41: are we a digital contoller?
		return 0x41, cmd == 0x42
	case 2: // 0x5a: ID byte
		return 0x5a, true
	case 3: // cross, start, select
		return uint8(profile.State), true
	case 4: // shoulder and shape buttons
		return uint8(profile.State >> 8), false
	default: // edge cases
		return 0xff, false
	}
}

func (profile *DigitalPadProfile) SetButtonState(button Button, state ButtonState) {
	s := profile.State
	mask := int32(1 << uint(button))

	switch state {
	case BUTTON_STATE_PRESSED:
		// TODO: check if this works
		profile.State = uint16(int32(s) & ^mask)
	case BUTTON_STATE_RELEASED:
		profile.State = s | uint16(mask)
	}
}

// SCPH-1080: Digital Joypad
func NewDigitalPad() *DigitalPadProfile {
	return &DigitalPadProfile{
		State: 0xffff,
	}
}
