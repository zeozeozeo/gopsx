package emulator

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

var emptyImage = ebiten.NewImage(2, 2)

func init() {
	emptyImage.Fill(color.RGBA{255, 255, 255, 255})
}

// An Ebitengine renderer that implements Renderer
type EbitenRenderer struct {
	DrawData *DrawData
	Gpu      *GPU
}

// Returns a new Ebitengine renderer
func (gpu *GPU) NewEbitenRenderer() *EbitenRenderer {
	renderer := &EbitenRenderer{
		DrawData: gpu.DrawData,
		Gpu:      gpu,
	}
	return renderer
}

func (renderer *EbitenRenderer) Draw(screen *ebiten.Image) {
	// generate Ebiten vertices from draw data (TODO: maybe there's
	// a better way to do this?)
	verticesLen := len(renderer.DrawData.VtxBuffer)
	vertices := make([]ebiten.Vertex, verticesLen)
	indices := make([]uint16, verticesLen)

	for idx, vtx := range renderer.DrawData.VtxBuffer {
		vertices[idx].ColorR = float32(vtx.Color.R) / 255
		vertices[idx].ColorG = float32(vtx.Color.G) / 255
		vertices[idx].ColorB = float32(vtx.Color.B) / 255
		vertices[idx].ColorA = 1 // should always be 1
		x := float32(vtx.Position.X + renderer.Gpu.DrawingXOffset)
		y := float32(vtx.Position.Y + renderer.Gpu.DrawingYOffset)
		vertices[idx].DstX = x
		vertices[idx].DstY = y

		// FIXME
		vertices[idx].SrcX = 0
		vertices[idx].SrcY = 0

		indices[idx] = uint16(idx)
	}

	op := &ebiten.DrawTrianglesOptions{}
	screen.DrawTriangles(
		vertices,
		indices,
		emptyImage,
		op,
	)

	// reset vertices
	renderer.DrawData.VtxBuffer = nil
}
