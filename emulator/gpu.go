package emulator

// Represents the depth of the pixel values in a texture page
type TextureDepth uint8

const (
	TEXTURE_DEPTH_4BIT  TextureDepth = 0 // 4 bits per pixel
	TEXTURE_DEPTH_8BIT  TextureDepth = 1 // 8 bits per pixel
	TEXTURE_DEPTH_15BIT TextureDepth = 2 // 15 bits per pixel
)

// Interlaced output splits each frame in two fields
type Field uint8

const (
	FIELD_TOP    Field = 1 // Top field (odd lines)
	FIELD_BOTTOM Field = 0 // Bottom field (even lines)
)

// Video output horizontal resolution
type HorizontalRes uint8

// Create a new HorizontalRes instance from the 2 bit field `hr1` and the one
// bit field `hr2`
func HResFromFields(hr1, hr2 uint8) HorizontalRes {
	hr := (hr2 & 1) | ((hr1 & 3) << 1)
	return HorizontalRes(hr)
}

// Return value of bits [18:16] of the status register
func (hr HorizontalRes) IntoStatus() uint32 {
	return uint32(hr) << 16
}

// Video output vertical resolution
type VerticalRes uint8

const (
	VRES_240_LINES VerticalRes = 0 // 240 lines
	VRES_480_LINES VerticalRes = 1 // 480 lines (only available for interlaced output)
)

// Represents a video mode (NTSC/PAL)
type VMode uint8

const (
	VMODE_NTSC VMode = 0 // NTSC: 480i60Hz
	VMODE_PAL  VMode = 1 // PAL: 576i50Hz
)

// Display area color depth
type DisplayDepth uint8

const (
	DISPLAY_DEPTH_15BITS DisplayDepth = 0 // 15 bits per pixel
	DISPLAY_DEPTH_24BITS DisplayDepth = 1 // 24 bits per pixel
)

// Represents the requested DMA direction
type DmaDirection uint8

const (
	DD_DMA_OFF     DmaDirection = 0
	DD_DMA_FIFO    DmaDirection = 1
	DD_CPU_TO_GP0  DmaDirection = 2
	DD_VRAM_TO_CPU DmaDirection = 3
)

type GPU struct {
	PageBaseX uint8 // Texture page base X coordinate (4 bits, 64 byte increment)
	PageBaseY uint8 // Texture page base Y coordinate (1 bit, 256 line increment)
	// Semi-transparency. Not entirely how to handle that value yet, it seems to
	// describe how to blend the source and the destination colors
	SemiTransparency uint8
	TextureDepth     TextureDepth // Texture page color depth
	Dithering        bool         // Enable dithering from 24 to 15 bits RGB
	DrawToDisplay    bool         // Allow drawing to the display area
	// Force "mask" bit of the pixel to 1 when writing to VRAM (otherwise, don't
	// modify it)
	ForceSetMaskBit      bool
	PreserveMaskedPixels bool // Don't draw to pixels which have the "mask" bit set
	// Currently displayed field. For progressive output this is always FIELD_TOP
	Field          Field
	TextureDisable bool          // When true, all textures are disabled
	VRes           VerticalRes   // Video output vertical resolution
	HRes           HorizontalRes // Video output horizontal resolution
	VMode          VMode         // Video mode
	// Display depth. The GPU itself always draws 15 bit RGB, 24 bit output must
	// use external assets (pre-rendered textures, MDEC, etc.)
	DisplayDepth          DisplayDepth
	Interlaced            bool          // Output interlaced video signal instead of progressive
	DisplayDisabled       bool          // Disable the display
	Interrupt             bool          // True when the interrupt is active
	DmaDirection          DmaDirection  // DMA request direction
	RectangleTextureXFlip bool          // Mirror textured rectangles along the X axis
	RectangleTextureYFlip bool          // Mirror textured rectangles along the Y axis
	TextureWindowXMask    uint8         // Texture window X mask (8 pixel steps)
	TextureWindowYMask    uint8         // Texture window Y mask (8 pixel steps)
	TextureWindowXOffset  uint8         // Texture window X offset (8 pixel steps)
	TextureWindowYOffset  uint8         // Texture window Y offset (8 pixel steps)
	DrawingAreaLeft       uint16        // Left-most column of the drawing area
	DrawingAreaTop        uint16        // Top−most line of the drawing area
	DrawingAreaRight      uint16        // Right−most column of the drawing area
	DrawingAreaBottom     uint16        // Bottom−most line of the drawing area
	DrawingXOffset        int16         // Horizontal drawing offset applied to all vertex
	DrawingYOffset        int16         // Vertical drawing offset applied to all vertex
	DisplayVRamXStart     uint16        // First column of the display area in VRAM
	DisplayVRamYStart     uint16        // First line of the display area in VRAM
	DisplayHorizStart     uint16        // Display output horizontal start relative to HSYNC
	DisplayHorizEnd       uint16        // Display output horizontal end relative to HSYNC
	DisplayLineStart      uint16        // Display output first line relative to VSYNC
	DisplayLineEnd        uint16        // Display output last line relative to VSYNC
	GP0Command            CommandBuffer // Buffer containing the current GP0 command
	GP0CommandRemaining   uint32        // Remaining words for the current GP0 command
	GP0CommandMethod      func()        // Method implementing the current GP0 command
}

func NewGPU() *GPU {
	// not sure what the reset values are, the BIOS should set them anyway
	gpu := &GPU{
		TextureDepth:    TEXTURE_DEPTH_4BIT,
		Field:           FIELD_TOP,
		HRes:            HResFromFields(0, 0),
		VRes:            VRES_240_LINES,
		VMode:           VMODE_NTSC,
		DisplayDepth:    DISPLAY_DEPTH_15BITS,
		DisplayDisabled: true,
		DmaDirection:    DD_DMA_OFF,
	}
	return gpu
}

// Handle writes to the GP0 command register
func (gpu *GPU) GP0(val uint32) {
	opcode := (val >> 24) & 0xff

	switch opcode {
	case 0x00:
		// NOP
	case 0xe1:
		gpu.GP0DrawMode(val)
	case 0xe2:
		gpu.GP0TextureWindow(val)
	case 0xe3:
		gpu.GP0DrawingAreaTopLeft(val)
	case 0xe4:
		gpu.GP0DrawingAreaBottomRight(val)
	case 0xe5:
		gpu.GP0DrawingOffset(val)
	case 0xe6:
		gpu.GP0MaskBitSetting(val)
	default:
		panicFmt("gpu: unhandled GP0 command 0x%x", val)
	}
}

// GP0(0xE1) command
func (gpu *GPU) GP0DrawMode(val uint32) {
	gpu.PageBaseX = uint8(val & 0xf)
	gpu.PageBaseY = uint8((val >> 4) & 1)
	gpu.SemiTransparency = uint8((val >> 5) & 3)

	switch (val >> 7) & 3 {
	case 0:
		gpu.TextureDepth = TEXTURE_DEPTH_4BIT
	case 1:
		gpu.TextureDepth = TEXTURE_DEPTH_8BIT
	case 2:
		gpu.TextureDepth = TEXTURE_DEPTH_15BIT
	default:
		panicFmt("gpu: unhandled texture depth %d", (val>>7)&3)
	}

	gpu.Dithering = ((val >> 9) & 1) != 0
	gpu.DrawToDisplay = ((val >> 10) & 1) != 0
	gpu.TextureDisable = ((val >> 11) & 1) != 0
	gpu.RectangleTextureXFlip = ((val >> 12) & 1) != 0
	gpu.RectangleTextureYFlip = ((val >> 13) & 1) != 0
}

// GP0(0xE3): Set Drawing Area Top Left
func (gpu *GPU) GP0DrawingAreaTopLeft(val uint32) {
	gpu.DrawingAreaTop = uint16((val >> 10) & 0x3ff)
	gpu.DrawingAreaLeft = uint16(val & 0x3ff)
}

// GP0(0xE4): Set Drawing Area BottomRight
func (gpu *GPU) GP0DrawingAreaBottomRight(val uint32) {
	gpu.DrawingAreaBottom = uint16((val >> 10) & 0x3ff)
	gpu.DrawingAreaRight = uint16(val & 0x3ff)
}

// GP0(0xE5): Set Drawing Offset
func (gpu *GPU) GP0DrawingOffset(val uint32) {
	x := uint16(val & 0x7ff)
	y := uint16((val >> 11) & 0x7ff)

	// values are 11 bit *signed* two's complement values, we need to
	// shift the value to 16 bits to force sign extension
	gpu.DrawingXOffset = (int16(x << 5)) >> 5
	gpu.DrawingYOffset = (int16(y << 5)) >> 5
}

// GP0(0xE2): Set Texture Window
func (gpu *GPU) GP0TextureWindow(val uint32) {
	gpu.TextureWindowXMask = uint8(val & 0x1f)
	gpu.TextureWindowYMask = uint8((val >> 5) & 0x1f)
	gpu.TextureWindowXOffset = uint8((val >> 10) & 0x1f)
	gpu.TextureWindowYOffset = uint8((val >> 15) & 0x1f)
}

// GP0(0xE6): Set Mask Bit Setting
func (gpu *GPU) GP0MaskBitSetting(val uint32) {
	gpu.ForceSetMaskBit = (val & 1) != 0
	gpu.PreserveMaskedPixels = (val & 2) != 0
}

// Handle writes to the GP1 command register
func (gpu *GPU) GP1(val uint32) {
	opcode := (val >> 24) & 0xff

	switch opcode {
	case 0x00:
		gpu.GP1Reset()
	case 0x04:
		gpu.GP1DmaDirection(val)
	case 0x05:
		gpu.GP1DisplayVRAMStart(val)
	case 0x06:
		gpu.GP1DisplayHorizontalRange(val)
	case 0x07:
		gpu.GP1DisplayVerticalRange(val)
	case 0x08:
		gpu.GP1DisplayMode(val)
	default:
		panicFmt("gpu: unhandled GP1 command 0x%x", val)
	}
}

// GP1(0x00): soft reset
func (gpu *GPU) GP1Reset() {
	gpu.Interrupt = false
	gpu.PageBaseX = 0
	gpu.PageBaseY = 0
	gpu.SemiTransparency = 0
	gpu.TextureDepth = TEXTURE_DEPTH_4BIT
	gpu.TextureWindowXMask = 0
	gpu.TextureWindowYMask = 0
	gpu.TextureWindowXOffset = 0
	gpu.TextureWindowYOffset = 0
	gpu.Dithering = false
	gpu.DrawToDisplay = false
	gpu.TextureDisable = false
	gpu.RectangleTextureXFlip = false
	gpu.RectangleTextureYFlip = false
	gpu.DrawingAreaLeft = 0
	gpu.DrawingAreaTop = 0
	gpu.DrawingAreaRight = 0
	gpu.DrawingAreaBottom = 0
	gpu.DrawingXOffset = 0
	gpu.DrawingYOffset = 0
	gpu.ForceSetMaskBit = false
	gpu.PreserveMaskedPixels = false
	gpu.DmaDirection = DD_DMA_OFF
	gpu.DisplayDisabled = true
	gpu.DisplayVRamXStart = 0
	gpu.DisplayVRamYStart = 0
	gpu.HRes = HResFromFields(0, 0)
	gpu.VRes = VRES_240_LINES
	gpu.VMode = VMODE_NTSC
	gpu.Interlaced = true
	gpu.DisplayHorizStart = 0x200
	gpu.DisplayHorizEnd = 0xc00
	gpu.DisplayLineStart = 0x10
	gpu.DisplayLineEnd = 0x100
	gpu.DisplayDepth = DISPLAY_DEPTH_15BITS
	// FIXME: should also clear the FIFO when it's implemented
	// FIXME: should also invalidate GPU cache when it's implemented
}

// GP1(0x80): display mode
func (gpu *GPU) GP1DisplayMode(val uint32) {
	hr1 := uint8(val & 3)
	hr2 := uint8((val >> 6) & 1)

	gpu.HRes = HResFromFields(hr1, hr2)

	if val&0x4 != 0 {
		gpu.VRes = VRES_480_LINES
	} else {
		gpu.VRes = VRES_240_LINES
	}

	if val&0x8 != 0 {
		gpu.VMode = VMODE_PAL
	} else {
		gpu.VMode = VMODE_NTSC
	}

	if val&0x10 != 0 {
		gpu.DisplayDepth = DISPLAY_DEPTH_15BITS
	} else {
		gpu.DisplayDepth = DISPLAY_DEPTH_15BITS
	}

	gpu.Interlaced = val&0x20 != 0

	if val&0x80 != 0 {
		panicFmt("gpu: unsupported display mode 0x%x", val)
	}
}

// GP1(0x04): DMA direction
func (gpu *GPU) GP1DmaDirection(val uint32) {
	switch val & 3 {
	case 0:
		gpu.DmaDirection = DD_DMA_OFF
	case 1:
		gpu.DmaDirection = DD_DMA_FIFO
	case 2:
		gpu.DmaDirection = DD_CPU_TO_GP0
	case 3:
		gpu.DmaDirection = DD_VRAM_TO_CPU
	default:
		panicFmt("gpu: unsupported DMA direction 0x%x", val)
	}
}

// GP1(0x05): Display VRAM Start
func (gpu *GPU) GP1DisplayVRAMStart(val uint32) {
	gpu.DisplayVRamXStart = uint16(val & 0x3fe)
	gpu.DisplayVRamYStart = uint16((val >> 10) & 0x1ff)
}

// GP1(0x06): Display Horizontal Range
func (gpu *GPU) GP1DisplayHorizontalRange(val uint32) {
	gpu.DisplayHorizStart = uint16(val & 0xfff)
	gpu.DisplayHorizEnd = uint16((val >> 12) & 0xfff)
}

// GP1(0x07): Display Vertical Range
func (gpu *GPU) GP1DisplayVerticalRange(val uint32) {
	gpu.DisplayLineStart = uint16(val & 0x3ff)
	gpu.DisplayLineEnd = uint16((val >> 10) & 0x3ff)
}

// Return value of the status register
func (gpu *GPU) Status() uint32 {
	var r uint32

	r |= uint32(gpu.PageBaseX) << 0
	r |= uint32(gpu.PageBaseY) << 4
	r |= uint32(gpu.SemiTransparency) << 5
	r |= uint32(gpu.TextureDepth) << 7
	r |= oneIfTrue(gpu.Dithering) << 9
	r |= oneIfTrue(gpu.DrawToDisplay) << 10
	r |= oneIfTrue(gpu.ForceSetMaskBit) << 11
	r |= oneIfTrue(gpu.PreserveMaskedPixels) << 12
	r |= uint32(gpu.Field) << 13
	// bit 14: not supported (when it's set on real hardware, it just messes up
	// the display in a weird way)
	r |= oneIfTrue(gpu.TextureDisable) << 15
	r |= gpu.HRes.IntoStatus()
	r |= uint32(gpu.VRes) << 19
	r |= uint32(gpu.VMode) << 20
	r |= uint32(gpu.DisplayDepth) << 21
	r |= oneIfTrue(gpu.Interlaced) << 22
	r |= oneIfTrue(gpu.DisplayDisabled) << 23
	r |= oneIfTrue(gpu.Interrupt) << 24

	// for now, we pretend that the GPU is always ready:
	// ready to recieve command
	r |= 1 << 26
	// ready to send VRAM to CPU
	r |= 1 << 27
	// ready to recieve DMA block
	r |= 1 << 28

	r |= uint32(gpu.DmaDirection) << 29

	// bit 31 should change depending on the currently drawn line (whether it's even,
	// odd or in the vblank apparently). we won't bother with it for now
	r |= 0 << 31

	// not sure about that, i'm guessing that it's the signal checked by the DMA
	// when sending data in Request synchronization mode, for now blindly follow
	// the Nocash spec
	var dmaRequest uint32
	switch gpu.DmaDirection {
	case DD_DMA_OFF: // always 0
		dmaRequest = 0
	case DD_DMA_FIFO: // should be 0 if FIFO is full, 1 otherwise
		dmaRequest = 1
	case DD_CPU_TO_GP0: // should be the same as status bit 28
		dmaRequest = (r >> 28) & 1
	case DD_VRAM_TO_CPU: // should be the same as status bit 27
		dmaRequest = (r >> 27) & 1
	}
	r |= dmaRequest << 25

	return r
}

// Return value of the `read` register
func (gpu *GPU) Read() uint32 {
	// FIXME: not implemented for now
	return 0
}
