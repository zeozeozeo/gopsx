package emulator

import "image/color"

// A 2 dimensional vector
type Vec2 struct {
	X, Y int16
}

// A single vertex with a position and color
type Vertex struct {
	Position Vec2
	Color    color.RGBA
}

// Stores the draw data
type DrawData struct {
	VtxBuffer []Vertex
}

// Pushes vertices to the vertex buffer
func (dd *DrawData) PushVertices(vertices ...Vertex) {
	dd.VtxBuffer = append(dd.VtxBuffer, vertices...)
}

func (dd *DrawData) PushQuad(vertices ...Vertex) {
	if len(vertices) != 4 {
		panicFmt("PushQuad takes 4 parameters, got %d", len(vertices))
	}

	// push the two triangles
	dd.PushVertices(vertices[0:3]...)
	dd.PushVertices(vertices[1:4]...)
}

// Parse position from a GP0 parameter
func Vec2FromGP0(val uint32) Vec2 {
	x := int16(val)
	y := int16(val >> 16)
	return Vec2{X: x, Y: y}
}

// Parse color from a GP0 parameter
func ColorFromGP0(val uint32) color.RGBA {
	r := uint8(val)
	g := uint8(val >> 8)
	b := uint8(val >> 16)
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

func NewVertex(pos Vec2, clr color.RGBA) Vertex {
	return Vertex{Position: pos, Color: clr}
}

func NewDrawData() *DrawData {
	return &DrawData{}
}
