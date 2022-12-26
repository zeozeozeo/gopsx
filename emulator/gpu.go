package emulator

import (
	"fmt"
	"image/color"
)

// Represents the depth of the pixel values in a texture page
type TextureDepth uint8

const (
	TEXTURE_DEPTH_4BIT  TextureDepth = 0 // 4 bits per pixel
	TEXTURE_DEPTH_8BIT  TextureDepth = 1 // 8 bits per pixel
	TEXTURE_DEPTH_15BIT TextureDepth = 2 // 15 bits per pixel
)

// VRAM dimensions
const (
	VRAM_WIDTH_PIXELS  = 1024
	VRAM_HEIGHT_PIXELS = 512
	VRAM_SIZE_PIXELS   = VRAM_WIDTH_PIXELS * VRAM_HEIGHT_PIXELS
)

// Interlaced output splits each frame in two fields
type Field uint8

const (
	FIELD_TOP    Field = 1 // Top field (odd lines)
	FIELD_BOTTOM Field = 0 // Bottom field (even lines)
)

// Use 16 bits for the fractional part of the clock ratio to get good precision
const CLOCK_RATIO_FRAC = 0x10000

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

type HardwareType uint8

const (
	HARDWARE_NTSC HardwareType = 0 // NTSC: 480i60Hz
	HARDWARE_PAL  HardwareType = 1 // PAL: 576i50Hz
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

type GP0CommandHandler func()

// Possible states for the GP0 command register
type GP0Mode uint8

const (
	GP0_MODE_COMMAND    GP0Mode = iota // Default mode: handling commands
	GP0_MODE_IMAGE_LOAD GP0Mode = iota // Loading an image into VRAM
)

// Graphics Processing Unit state
type GPU struct {
	DrawData  *DrawData // Stores the vertex buffers, etc.
	FrameEnd  func()    // If not nil, this function is called after rendering the frame
	PageBaseX uint8     // Texture page base X coordinate (4 bits, 64 byte increment)
	PageBaseY uint8     // Texture page base Y coordinate (1 bit, 256 line increment)
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
	Interlaced            bool              // Output interlaced video signal instead of progressive
	DisplayDisabled       bool              // Disable the display
	GP0Interrupt          bool              // True when the  GP0interrupt is active
	DmaDirection          DmaDirection      // DMA request direction
	RectangleTextureXFlip bool              // Mirror textured rectangles along the X axis
	RectangleTextureYFlip bool              // Mirror textured rectangles along the Y axis
	TextureWindowXMask    uint8             // Texture window X mask (8 pixel steps)
	TextureWindowYMask    uint8             // Texture window Y mask (8 pixel steps)
	TextureWindowXOffset  uint8             // Texture window X offset (8 pixel steps)
	TextureWindowYOffset  uint8             // Texture window Y offset (8 pixel steps)
	DrawingAreaLeft       uint16            // Left-most column of the drawing area
	DrawingAreaTop        uint16            // Top−most line of the drawing area
	DrawingAreaRight      uint16            // Right−most column of the drawing area
	DrawingAreaBottom     uint16            // Bottom−most line of the drawing area
	DrawingXOffset        int16             // Horizontal drawing offset applied to all vertex
	DrawingYOffset        int16             // Vertical drawing offset applied to all vertex
	DisplayVRamXStart     uint16            // First column of the display area in VRAM
	DisplayVRamYStart     uint16            // First line of the display area in VRAM
	DisplayHorizStart     uint16            // Display output horizontal start relative to HSYNC
	DisplayHorizEnd       uint16            // Display output horizontal end relative to HSYNC
	DisplayLineStart      uint16            // Display output first line relative to VSYNC
	DisplayLineEnd        uint16            // Display output last line relative to VSYNC
	GP0Command            CommandBuffer     // Buffer containing the current GP0 command
	GP0WordsRemaining     uint32            // Remaining words for the current GP0 command
	GP0Handler            GP0CommandHandler // Method implementing the current GP0 command
	GP0Mode               GP0Mode           // Current mode of the GP0 register
	LoadBuffer            *ImageBuffer      // GP0 ImageLoad buffer
	ClockFrac             uint16            // Fractional GPU cycle remainder from CPU clock
	DisplayLine           uint16            // Currently displayed video output line
	DisplayLineTick       uint16            // Current GPU clock tick for the current line
	VBlankInterrupt       bool              // True if the VBLANK interrupt is high
	Hardware              HardwareType      // PAL or NTSC
	ClockPhase            uint16            // Clock CPU/GPU time conversion in CPU periods
}

func NewGPU(hardware HardwareType) *GPU {
	// not sure what the reset values are, the BIOS should set them anyway
	gpu := &GPU{
		DrawData:          NewDrawData(),
		TextureDepth:      TEXTURE_DEPTH_4BIT,
		Field:             FIELD_TOP,
		HRes:              HResFromFields(0, 0),
		VRes:              VRES_240_LINES,
		VMode:             VMODE_NTSC,
		DisplayDepth:      DISPLAY_DEPTH_15BITS,
		DisplayDisabled:   true,
		DmaDirection:      DD_DMA_OFF,
		GP0Mode:           GP0_MODE_COMMAND,
		LoadBuffer:        NewImageBuffer(),
		DisplayHorizStart: 0x200,
		DisplayHorizEnd:   0xc00,
		DisplayLineStart:  0x10,
		DisplayLineEnd:    0x100,
		Hardware:          hardware,
	}
	return gpu
}

// Handle writes to the GP0 command register
func (gpu *GPU) GP0(val uint32) {
	if gpu.GP0WordsRemaining == 0 {
		// start a new GP0 command
		opcode := (val >> 24) & 0xff

		var length uint32
		var handler GP0CommandHandler

		switch opcode {
		case 0x00:
			length, handler = 1, gpu.GP0Nop
		case 0x01:
			length, handler = 1, gpu.GP0ClearCache
		case 0x02:
			length, handler = 3, gpu.GP0FillRect
		case 0x28:
			length, handler = 5, gpu.GP0QuadMonoOpaque
		case 0x2c:
			length, handler = 9, gpu.GP0QuadTextureBlendOpaque
		case 0x2d:
			length, handler = 9, gpu.GP0QuadTextureRawOpaque
		case 0x30:
			length, handler = 6, gpu.GP0TriangleShadedOpaque
		case 0x38:
			length, handler = 8, gpu.GP0QuadShadedOpaque
		case 0x65:
			length, handler = 4, gpu.GP0RectTextureRawOpaque
		case 0xa0:
			length, handler = 3, gpu.GP0ImageLoad
		case 0xc0:
			length, handler = 3, gpu.GP0ImageStore
		case 0xe1:
			length, handler = 1, gpu.GP0DrawMode
		case 0xe2:
			length, handler = 1, gpu.GP0TextureWindow
		case 0xe3:
			length, handler = 1, gpu.GP0DrawingAreaTopLeft
		case 0xe4:
			length, handler = 1, gpu.GP0DrawingAreaBottomRight
		case 0xe5:
			length, handler = 1, gpu.GP0DrawingOffset
		case 0xe6:
			length, handler = 1, gpu.GP0MaskBitSetting
		default:
			panicFmt("gpu: unhandled GP0 command 0x%x", val)
		}

		gpu.GP0WordsRemaining = length
		gpu.GP0Handler = handler
		gpu.GP0Command.Clear()
	}

	// continue current command
	gpu.GP0WordsRemaining--

	switch gpu.GP0Mode {
	case GP0_MODE_COMMAND:
		gpu.GP0Command.PushWord(val)

		if gpu.GP0WordsRemaining == 0 {
			// we have all the parameters, now we can run the method
			gpu.GP0Handler()
		}
	case GP0_MODE_IMAGE_LOAD:
		gpu.GP0HandleImageLoad(val)
	}
}

// GP0(0x02): Fill Rectangle
func (gpu *GPU) GP0FillRect() {
	// TODO: this should be affected by the mask
	clr := ColorFromGP0(gpu.GP0Command.Get(0))
	topLeft := Vec2FromGP0(gpu.GP0Command.Get(1))
	size := Vec2FromGP0(gpu.GP0Command.Get(2))

	gpu.DrawData.PushQuad(
		NewVertex(topLeft, clr),
		NewVertex(Vec2{topLeft.X + size.X, topLeft.Y}, clr),
		NewVertex(Vec2{topLeft.X, topLeft.Y + size.Y}, clr),
		NewVertex(Vec2{topLeft.X + size.X, topLeft.Y + size.Y}, clr),
	)
}

// GP0(0x2D): Raw Textured Opaque Quadrilateral
func (gpu *GPU) GP0QuadTextureRawOpaque() {
	// FIXME: we don't support textures at this point, so the color is just red
	clr := color.RGBA{255, 0, 0, 255}

	gpu.DrawData.PushQuad(
		NewVertex(Vec2FromGP0(gpu.GP0Command.Get(1)), clr),
		NewVertex(Vec2FromGP0(gpu.GP0Command.Get(3)), clr),
		NewVertex(Vec2FromGP0(gpu.GP0Command.Get(5)), clr),
		NewVertex(Vec2FromGP0(gpu.GP0Command.Get(7)), clr),
	)
}

// GP0(0x65): Opaque rectange with raw texture
func (gpu *GPU) GP0RectTextureRawOpaque() {
	// TODO: this should be affected by the mask
	clr := ColorFromGP0(gpu.GP0Command.Get(0))
	topLeft := Vec2FromGP0(gpu.GP0Command.Get(1))
	size := Vec2FromGP0(gpu.GP0Command.Get(3))

	gpu.DrawData.PushQuad(
		NewVertex(topLeft, clr),
		NewVertex(Vec2{topLeft.X + size.X, topLeft.Y}, clr),
		NewVertex(Vec2{topLeft.X, topLeft.Y + size.Y}, clr),
		NewVertex(Vec2{topLeft.X + size.X, topLeft.Y + size.Y}, clr),
	)
}

// GP0(0xA0): Image Load
func (gpu *GPU) GP0ImageLoad() {
	// the top-left corner location in VRAM
	pos := gpu.GP0Command.Get(1)

	gpu.LoadBuffer.Position.X = uint16(pos)
	gpu.LoadBuffer.Position.Y = uint16(pos >> 16)

	// parameter 2 contains the image resolution
	res := gpu.GP0Command.Get(2)
	width := res & 0xffff
	height := res >> 16
	gpu.LoadBuffer.Resolution.X = uint16(width)
	gpu.LoadBuffer.Resolution.Y = uint16(height)

	// size of the image in 16 bit pixels
	imgSize := width * height

	// if we have an odd number of pixels we must round up since we
	// transfer 32 bits at a time. there'll be 16 bits of padding in
	// the last word
	imgSize = uint32(int64(imgSize+1) & ^1)

	// store number of words expected for this image
	gpu.GP0WordsRemaining = imgSize / 2

	if gpu.GP0WordsRemaining == 0 {
		panic("gpu: 0 size image load")
	}

	// put the GP0 state machine in ImageLoad mode
	gpu.GP0Mode = GP0_MODE_IMAGE_LOAD
}

func (gpu *GPU) GP0HandleImageLoad(word uint32) {
	gpu.LoadBuffer.PushWord(word)

	if gpu.GP0WordsRemaining == 0 {
		// load done, switch back to command mode
		gpu.GP0Mode = GP0_MODE_COMMAND
		// TODO: load image here
		fmt.Println("gpu: unhandled image load")
		gpu.LoadBuffer.Clear()
	}
}

// GP0(0xC0): Image Store
func (gpu *GPU) GP0ImageStore() {
	// parameter 2 contains the image resolution
	res := gpu.GP0Command.Get(2)
	width := res & 0xffff
	height := res >> 16

	fmt.Printf("gpu: unhandled image store: %dx%d\n", width, height)
}

// GP0(0x28): Monochrome Opaque Quadliteral
func (gpu *GPU) GP0QuadMonoOpaque() {
	clr := ColorFromGP0(gpu.GP0Command.Get(0))
	gpu.DrawData.PushQuad(
		NewVertex(Vec2FromGP0(gpu.GP0Command.Get(1)), clr),
		NewVertex(Vec2FromGP0(gpu.GP0Command.Get(2)), clr),
		NewVertex(Vec2FromGP0(gpu.GP0Command.Get(3)), clr),
		NewVertex(Vec2FromGP0(gpu.GP0Command.Get(4)), clr),
	)
}

// GP0(0x38): Shaded Opaque Quadliteral
func (gpu *GPU) GP0QuadShadedOpaque() {
	gpu.DrawData.PushQuad(
		NewVertex(Vec2FromGP0(gpu.GP0Command.Get(1)), ColorFromGP0(gpu.GP0Command.Get(0))),
		NewVertex(Vec2FromGP0(gpu.GP0Command.Get(3)), ColorFromGP0(gpu.GP0Command.Get(2))),
		NewVertex(Vec2FromGP0(gpu.GP0Command.Get(5)), ColorFromGP0(gpu.GP0Command.Get(4))),
		NewVertex(Vec2FromGP0(gpu.GP0Command.Get(7)), ColorFromGP0(gpu.GP0Command.Get(6))),
	)
}

// GP0(0x30): Shaded Opaque Triangle
func (gpu *GPU) GP0TriangleShadedOpaque() {
	gpu.DrawData.PushVertices(
		NewVertex(Vec2FromGP0(gpu.GP0Command.Get(1)), ColorFromGP0(gpu.GP0Command.Get(0))),
		NewVertex(Vec2FromGP0(gpu.GP0Command.Get(3)), ColorFromGP0(gpu.GP0Command.Get(2))),
		NewVertex(Vec2FromGP0(gpu.GP0Command.Get(5)), ColorFromGP0(gpu.GP0Command.Get(4))),
	)
}

// GP0(0x2C): Textured Opaque Quadliteral
func (gpu *GPU) GP0QuadTextureBlendOpaque() {
	// FIXME: we don't support textures at this point, so the color is just red
	clr := color.RGBA{255, 0, 0, 255}
	gpu.DrawData.PushQuad(
		NewVertex(Vec2FromGP0(gpu.GP0Command.Get(1)), clr),
		NewVertex(Vec2FromGP0(gpu.GP0Command.Get(3)), clr),
		NewVertex(Vec2FromGP0(gpu.GP0Command.Get(5)), clr),
		NewVertex(Vec2FromGP0(gpu.GP0Command.Get(7)), clr),
	)
}

// GP0(0xE1) command
func (gpu *GPU) GP0DrawMode() {
	val := gpu.GP0Command.Get(0)

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

// GP0(0x00): No Operation
func (gpu *GPU) GP0Nop() {
	// NOP
}

// GP0(0x01): Clear Cache
func (gpu *GPU) GP0ClearCache() {
	// not implemented
}

// GP0(0xE3): Set Drawing Area Top Left
func (gpu *GPU) GP0DrawingAreaTopLeft() {
	val := gpu.GP0Command.Get(0)
	gpu.DrawingAreaTop = uint16((val >> 10) & 0x3ff)
	gpu.DrawingAreaLeft = uint16(val & 0x3ff)
}

// GP0(0xE4): Set Drawing Area BottomRight
func (gpu *GPU) GP0DrawingAreaBottomRight() {
	val := gpu.GP0Command.Get(0)
	gpu.DrawingAreaBottom = uint16((val >> 10) & 0x3ff)
	gpu.DrawingAreaRight = uint16(val & 0x3ff)
}

// GP0(0xE5): Set Drawing Offset
func (gpu *GPU) GP0DrawingOffset() {
	val := gpu.GP0Command.Get(0)
	x := uint16(val & 0x7ff)
	y := uint16((val >> 11) & 0x7ff)

	// values are 11 bit *signed* two's complement values, we need to
	// shift the value to 16 bits to force sign extension
	gpu.DrawingXOffset = (int16(x << 5)) >> 5
	gpu.DrawingYOffset = (int16(y << 5)) >> 5
}

// GP0(0xE2): Set Texture Window
func (gpu *GPU) GP0TextureWindow() {
	val := gpu.GP0Command.Get(0)
	gpu.TextureWindowXMask = uint8(val & 0x1f)
	gpu.TextureWindowYMask = uint8((val >> 5) & 0x1f)
	gpu.TextureWindowXOffset = uint8((val >> 10) & 0x1f)
	gpu.TextureWindowYOffset = uint8((val >> 15) & 0x1f)
}

// GP0(0xE6): Set Mask Bit Setting
func (gpu *GPU) GP0MaskBitSetting() {
	val := gpu.GP0Command.Get(0)
	gpu.ForceSetMaskBit = (val & 1) != 0
	gpu.PreserveMaskedPixels = (val & 2) != 0
}

// Handle writes to the GP1 command register
func (gpu *GPU) GP1(val uint32, th *TimeHandler, irqState *IrqState, timers *Timers) {
	opcode := (val >> 24) & 0xff

	switch opcode {
	case 0x00:
		gpu.GP1Reset(th, irqState)
		timers.VideoTimingsChanged(th, irqState, gpu)
	case 0x01:
		gpu.GP1ResetCommandBuffer()
	case 0x02:
		gpu.GP1AcknowledgeIrq()
	case 0x03:
		gpu.GP1DisplayEnable(val)
	case 0x04:
		gpu.GP1DmaDirection(val)
	case 0x05:
		gpu.GP1DisplayVRAMStart(val)
	case 0x06:
		gpu.GP1DisplayHorizontalRange(val)
	case 0x07:
		gpu.GP1DisplayVerticalRange(val, th, irqState)
	case 0x08:
		gpu.GP1DisplayMode(val, th, irqState)
		timers.VideoTimingsChanged(th, irqState, gpu)
	default:
		panicFmt("gpu: unhandled GP1 command 0x%x", val)
	}
}

// GP1(0x00): soft reset
func (gpu *GPU) GP1Reset(th *TimeHandler, irqState *IrqState) {
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
	gpu.Field = FIELD_TOP
	gpu.DisplayLine = 0
	gpu.DisplayLineTick = 0
	gpu.DrawingXOffset = 0
	gpu.DrawingYOffset = 0
	gpu.GP1ResetCommandBuffer()
	gpu.GP1AcknowledgeIrq()
	gpu.Sync(th, irqState)
	// FIXME: should also clear the FIFO when it's implemented
	// FIXME: should also invalidate GPU cache when it's implemented
}

// GP1(0x80): display mode
func (gpu *GPU) GP1DisplayMode(val uint32, th *TimeHandler, irqState *IrqState) {
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

	// TODO: should we reset the field here?
	gpu.Field = FIELD_TOP

	if val&0x80 != 0 {
		panicFmt("gpu: unsupported display mode 0x%x", val)
	}

	gpu.Sync(th, irqState)
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
func (gpu *GPU) GP1DisplayVerticalRange(val uint32, th *TimeHandler, irqState *IrqState) {
	gpu.DisplayLineStart = uint16(val & 0x3ff)
	gpu.DisplayLineEnd = uint16((val >> 10) & 0x3ff)
	gpu.Sync(th, irqState)
}

// GP1(0x03): Display Enable
func (gpu *GPU) GP1DisplayEnable(val uint32) {
	gpu.DisplayDisabled = val&1 != 0
}

// GP1(0x02): Acknowledge Interrupt
func (gpu *GPU) GP1AcknowledgeIrq() {
	gpu.GP0Interrupt = false
}

// GP1(0x01): Reset Command Buffer
func (gpu *GPU) GP1ResetCommandBuffer() {
	gpu.GP0Command.Clear()
	gpu.GP0WordsRemaining = 0
	gpu.GP0Mode = GP0_MODE_COMMAND
	// FIXME: this should also clear the command FIFO, when we implement it
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
	r |= oneIfTrue(gpu.GP0Interrupt) << 24

	// for now, we pretend that the GPU is always ready:
	// ready to receive command
	r |= 1 << 26
	// ready to send VRAM to CPU
	r |= 1 << 27
	// ready to receive DMA block
	r |= 1 << 28

	r |= uint32(gpu.DmaDirection) << 29

	// bit 31 is 1 if the currently displayed VRAM line is odd, 0 if it's even or if
	// we're in vertical blanking
	if !gpu.InVBlank() {
		r |= uint32(gpu.DisplayedVRamLine()&1) << 31
	}

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

// Sets the function that will be called when the frame is rendered
func (gpu *GPU) SetFrameEnd(end func()) {
	gpu.FrameEnd = end
}

// Convert GPU clock ratio to CPU clock ratio
func (gpu *GPU) GPUToCPUClockRatio() FracCycles {
	// convert delta into GPU clock periods
	var cpuClock float32 = 33.8685
	var gpuClock float32
	switch gpu.Hardware {
	case HARDWARE_NTSC:
		gpuClock = 53.69
	case HARDWARE_PAL:
		gpuClock = 53.20
	}

	return FracCyclesFromF32(gpuClock / cpuClock)
}

// Returns the number of GPU clock cycles per line, and the number of lines
// in a frame, depending of `VMode`
func (gpu *GPU) GetVModeTimings() (uint16, uint16) {
	switch gpu.VMode {
	case VMODE_NTSC:
		return 3412, 263
	case VMODE_PAL:
		return 3404, 314
	}
	return 0, 0
}

// Returns the number of GPU clock cycles per line, and the number of lines
// in a frame, depending of `VMode` in a 64 bit unsigned integer
func (gpu *GPU) GetVModeTimingsU64() (uint64, uint64) {
	ticksPerLine, linesPerFrame := gpu.GetVModeTimings()
	return uint64(ticksPerLine), uint64(linesPerFrame)
}

// Returns true if the GPU is in the blanking period
func (gpu *GPU) InVBlank() bool {
	return gpu.DisplayLine < gpu.DisplayLineStart || gpu.DisplayLine >= gpu.DisplayLineEnd
}

// Synchronizes the GPU state
func (gpu *GPU) Sync(th *TimeHandler, irqState *IrqState) {
	delta := th.Sync(PERIPHERAL_GPU)
	delta = uint64(gpu.ClockPhase) + delta*gpu.GPUToCPUClockRatio().GetFixed()

	// the low 16 bits are the new fractional part
	gpu.ClockPhase = uint16(delta)
	delta >>= 16 // make delta an integer again

	ticksPerLine, linesPerFrame := gpu.GetVModeTimingsU64()

	lineTick := uint64(gpu.DisplayLineTick) + delta
	line := uint64(gpu.DisplayLine) + lineTick/ticksPerLine

	gpu.DisplayLineTick = uint16(lineTick % ticksPerLine)

	if line > linesPerFrame {
		// new frame
		if gpu.Interlaced {
			// update field
			nframes := line / linesPerFrame
			if (nframes+uint64(gpu.Field))&1 != 0 {
				gpu.Field = FIELD_TOP
			} else {
				gpu.Field = FIELD_BOTTOM
			}
		}

		gpu.DisplayLine = uint16(line % linesPerFrame)
	} else {
		gpu.DisplayLine = uint16(line)
	}

	vblankInterrupt := gpu.InVBlank()

	if !gpu.VBlankInterrupt && vblankInterrupt {
		irqState.SetHigh(INTERRUPT_VBLANK)
	}

	if gpu.VBlankInterrupt && !vblankInterrupt {
		// end of vertical blanking, do the FrameEnd callback

		// FIXME: the FrameEnd() call here causes the screen to flicker
		// HACK: as a workaround, I check if the draw data has any vertices.
		//       I have no idea why this happens :(
		if gpu.FrameEnd != nil && len(gpu.DrawData.VtxBuffer) > 0 {
			gpu.FrameEnd()
		}
	}

	gpu.VBlankInterrupt = vblankInterrupt
	gpu.PredictNextSync(th)
}

func (gpu *GPU) PredictNextSync(th *TimeHandler) {
	ticksPerLine, linesPerFrame := gpu.GetVModeTimingsU64()
	var delta uint64 = 0
	currLine := uint64(gpu.DisplayLine)
	displayLineStart := uint64(gpu.DisplayLineStart)
	displayLineEnd := uint64(gpu.DisplayLineEnd)

	// number of ticks to get to the start of the next line
	delta += ticksPerLine - uint64(gpu.DisplayLineTick)

	if currLine >= displayLineEnd {
		// in vertical blanking at the end of the frame, synchronize
		// at the end of the blanking at the beginning of the next frame

		// number of ticks to get to the next frame
		delta += (linesPerFrame - currLine) * ticksPerLine
		// number of ticks to get to the end of the vblank in the next frame
		delta += (displayLineStart - 1) * ticksPerLine
	} else if currLine < displayLineStart {
		// in vertical blanking in the beginning of the frame, synchronize
		// at the end of the blanking in the current frame
		delta += (displayLineStart - 1 - currLine) * ticksPerLine
	} else {
		// not in blanking; synchronize at the beginning of the vertical blanking period
		delta += (displayLineEnd - 1 - currLine) * ticksPerLine
	}

	// convert delta to CPU clock periods
	delta <<= FRAC_CYCLES_FRAC_BITS
	// remove the current fractional cycle
	delta -= uint64(gpu.ClockPhase)

	// make sure we're never triggered too early
	ratio := gpu.GPUToCPUClockRatio().GetFixed()
	delta = (delta + ratio - 1) / ratio

	th.SetNextSyncDelta(PERIPHERAL_GPU, delta)
}

// Returns the index of the currently displayed VRAM line
func (gpu *GPU) DisplayedVRamLine() uint16 {
	var offset uint16
	if gpu.Interlaced {
		offset = gpu.DisplayLine*2 + uint16(gpu.Field)
	} else {
		offset = gpu.DisplayLine
	}

	// the VRAM wraps around, so incase of overflow truncate it to 9 bits
	return (gpu.DisplayVRamYStart + offset) & 0x1ff
}

func (gpu *GPU) Load(offset uint32, th *TimeHandler, irqState *IrqState) uint32 {
	gpu.Sync(th, irqState)

	switch offset {
	case 0:
		return gpu.Read()
	case 4:
		return gpu.Status()
	default:
		panicFmt("gpu: unhandled GPU read (offset %d)", offset)
	}
	return 0
}

func (gpu *GPU) Store(offset uint32, val uint32, th *TimeHandler, irqState *IrqState, timers *Timers) {
	gpu.Sync(th, irqState)

	switch offset {
	case 0:
		gpu.GP0(val)
	case 4:
		gpu.GP1(val, th, irqState, timers)
	default:
		panicFmt("gpu: unhandled GPU write 0x%x <- 0x%x\n", offset, val)
	}
}

func (hres HorizontalRes) DotclockDivider() uint8 {
	hr1 := (hres >> 1) & 0x3
	hr2 := hres&1 != 0

	if hr2 {
		return 7 // ~368 pixels
	} else {
		switch hr1 {
		case 0:
			return 10 // ~256 pixels
		case 1:
			return 8 // ~320 pixels
		case 2:
			return 5 // ~512 pixels
		case 3:
			return 4 // ~640 pixels
		default:
			panic("gpu: unreachable")
		}
	}
}

// Period of the dotclock in CPU cycles
func (gpu *GPU) DotclockPeriod() FracCycles {
	gpuClockPeriod := gpu.GPUToCPUClockRatio()
	dotclockDivider := gpu.HRes.DotclockDivider()

	period := gpuClockPeriod.GetFixed() * uint64(dotclockDivider)
	return FracCyclesFromFixed(period)
}

// Phase of the GPU dotclock relative to the CPU clock
func (gpu *GPU) DotclockPhase() FracCycles {
	panic("gpu: dotclock phase is not implemented")
	// return FracCyclesFromCycles(uint64(gpu.ClockPhase))
}

func (gpu *GPU) HSyncPeriod() FracCycles {
	ticksPerLine, _ := gpu.GetVModeTimings()
	lineLen := FracCyclesFromCycles(uint64(ticksPerLine))
	return lineLen.Divide(gpu.GPUToCPUClockRatio()) // GPU to CPU cycles
}

func (gpu *GPU) HSyncPhase() FracCycles {
	phase := FracCyclesFromCycles(uint64(gpu.DisplayLineTick))
	clockPhase := FracCyclesFromFixed(uint64(gpu.ClockPhase))
	phase = phase.Add(clockPhase)
	return phase.Multiply(gpu.GPUToCPUClockRatio()) // GPU to CPU cycles
}
