package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/zeozeozeo/gopsx/emulator"
)

var (
	width, height = 1024, 512
	gpu           *emulator.GPU
	currentFrame  = ebiten.NewImage(width, height)
)

type ebitenGame struct {
	renderer *emulator.EbitenRenderer
}

func (g *ebitenGame) Update() error {
	return nil
}

func (g *ebitenGame) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}

	// scale rendered frame to fit window
	fx := currentFrame.Bounds().Dx()
	fy := currentFrame.Bounds().Dy()
	scaleX := float64(width) / float64(fx)
	scaleY := float64(height) / float64(fy)
	op.GeoM.Scale(scaleX, scaleY)

	screen.DrawImage(currentFrame, op)
}

func (g *ebitenGame) Layout(insideWidth, insideHeight int) (int, int) {
	return width, height
}

func (g *ebitenGame) drawFrame() {
	// create renderer if it's nil
	if g.renderer == nil {
		g.renderer = gpu.NewEbitenRenderer()
	}

	// clear previous frame and draw the new one
	currentFrame.Clear()
	g.renderer.Draw(currentFrame)
}

func startEbitenWindow(g *ebitenGame) {
	ebiten.SetWindowSize(width, height)
	ebiten.SetWindowTitle("gopsx")

	if err := ebiten.RunGame(g); err != nil {
		panic(err)
	}
}

func main() {
	g := &ebitenGame{}
	go startEmulator(g)
	startEbitenWindow(g)
}

func startEmulator(g *ebitenGame) {
	// parse arguments
	biosPath := flag.String("bios", "SCPH1001.BIN", "path to the BIOS file")
	flag.Parse()

	// start emulator
	bios := loadBios(*biosPath)
	ram := emulator.NewRAM()
	gpu = emulator.NewGPU()
	gpu.SetFrameEnd(g.drawFrame)
	inter := emulator.NewInterconnect(bios, ram, gpu)
	cpu := emulator.NewCPU(inter)

	for {
		cpu.RunNextInstruction()
	}
}

func loadBios(path string) *emulator.BIOS {
	log.Printf("loading bios \"%s\"", path)
	start := time.Now()

	// read bios
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// load bios
	bios, err := emulator.LoadBIOS(file)
	if err != nil {
		panic(err)
	}

	log.Printf("loaded bios in %s", time.Since(start))
	return bios
}
