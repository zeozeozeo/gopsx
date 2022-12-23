package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/zeozeozeo/gopsx/emulator"
)

var (
	width, height = 1024, 512
	gpu           *emulator.GPU
	currentFrame  = ebiten.NewImage(width, height)
	wg            sync.WaitGroup
	prevFrameTime = time.Now()
	deltaTime     float64
	showFps       *bool
	showCycles    *bool
	cpu           *emulator.CPU
	didPanic      bool
	panicString   string
	doRecover     *bool
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

	wg.Wait()
	screen.DrawImage(currentFrame, op)

	// draw error message if there was a panic
	if didPanic {
		ebitenutil.DebugPrintAt(screen, panicString, 8, 48)
	}
}

func (g *ebitenGame) Layout(insideWidth, insideHeight int) (int, int) {
	return width, height
}

func (g *ebitenGame) drawFrame() {
	wg.Add(1)
	defer wg.Done()

	// calculate delta time
	deltaTime = time.Since(prevFrameTime).Seconds()

	// create renderer if it's nil
	if g.renderer == nil {
		g.renderer = gpu.NewEbitenRenderer()
	}

	// clear previous frame and draw the new one
	// FIXME: for some reason, the image is flickering after the GPU timings were implemented
	currentFrame.Clear()
	g.renderer.Draw(currentFrame)
	if *showFps {
		ebitenutil.DebugPrintAt(currentFrame, fmt.Sprintf("%f fps", 1/deltaTime), 8, 8)
	}
	if *showCycles {
		ebitenutil.DebugPrintAt(currentFrame, fmt.Sprintf("%d cycles", cpu.Th.Cycles), 8, 24)
	}

	prevFrameTime = time.Now()
}

func startEbitenWindow(g *ebitenGame) {
	ebiten.SetWindowSize(width, height)
	ebiten.SetWindowTitle("gopsx")

	if err := ebiten.RunGame(g); err != nil {
		panic(err)
	}
}

func main() {
	// parse arguments
	biosPath := flag.String("bios", "SCPH1001.BIN", "path to the BIOS file")
	showFps = flag.Bool("fps", true, "show FPS value")
	showCycles = flag.Bool("cycles", true, "show amount of CPU cycles")
	doRecover = flag.Bool("recover", true, "recover from emulator panics")
	flag.Parse()

	g := &ebitenGame{}
	go startEmulator(g, *biosPath)
	startEbitenWindow(g)
}

func startEmulator(g *ebitenGame, biosPath string) {
	// start emulator
	bios := loadBios(biosPath)
	ram := emulator.NewRAM()
	gpu = emulator.NewGPU(emulator.HARDWARE_NTSC)
	gpu.SetFrameEnd(g.drawFrame)
	inter := emulator.NewInterconnect(bios, ram, gpu)
	cpu = emulator.NewCPU(inter)

	defer func() {
		if *doRecover {
			if r := recover(); r != nil {
				fmt.Printf("\nrecovered from panic: %s\n\n%s\n", r, debug.Stack())
				didPanic = true
				panicString = fmt.Sprintf("recovered from panic:\n%s", r)
			}
		}
	}()

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
