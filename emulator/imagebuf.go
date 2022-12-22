package emulator

import (
	"image"
	"image/color"
)

// Stores image data
type ImageBuffer struct {
	Position   Vec2U                    // Top-left coordinates in VRAM
	Resolution Vec2U                    // Image resolution
	Buffer     [VRAM_SIZE_PIXELS]uint16 // 1MB per image buffer (TODO: use a dynamic slice?)
	Index      uint32                   // Position in the buffer
}

// Returns a new image buffer instance
func NewImageBuffer() *ImageBuffer {
	return &ImageBuffer{}
}

// Resets the image buffer to zeros
func (buf *ImageBuffer) Clear() {
	buf.Position.X = 0
	buf.Position.Y = 0
	buf.Resolution.X = 0
	buf.Resolution.Y = 0
	buf.Index = 0
}

func (buf *ImageBuffer) PushWord(word uint32) {
	buf.Buffer[buf.Index] = uint16(word)
	buf.Buffer[buf.Index+1] = uint16(word >> 16)
	buf.Index += 2
}

func (buf *ImageBuffer) Reset(x, y, width, height uint16) {
	buf.Position.X = x
	buf.Position.Y = y
	buf.Resolution.X = width
	buf.Resolution.Y = height
	buf.Index = 0
}

// Returns the RGBA color value at `x`,`y`
func (buf *ImageBuffer) At(x, y int) color.Color {
	// TODO: make sure this works
	val := buf.Buffer[x+y]
	r := uint8(((val & 0b01111100_00000000) >> 7) | ((val & 0b01111100_00000000) >> 12))
	g := uint8(((val & 0b00000011_11100000) >> 2) | ((val & 0b00000011_11100000) >> 7))
	b := uint8(((val & 0b00011111) << 3) | ((val & 0b00011111) >> 2))
	return color.RGBA{r, g, b, 255}
}

// Converts the image to an image.RGBA
func (buf *ImageBuffer) ToImage() image.Image {
	width, height := int(buf.Resolution.X), int(buf.Resolution.Y)
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// set each pixel
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, buf.At(x, y))
		}
	}
	return img
}
