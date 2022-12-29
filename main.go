package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/zeozeozeo/gopsx/emulator"
)

var (
	width, height = 1024, 512
	gpu           *emulator.GPU
	currentFrame  = ebiten.NewImage(1024, 512)
	wg            sync.WaitGroup
	prevFrameTime = time.Now()
	showFps       *bool
	showCycles    *bool
	cpu           *emulator.CPU
	didPanic      bool
	panicString   string
	doRecover     *bool
	frameDt       float64
	disc          *emulator.Disc
)

// Gamepad button can be binded to multiple keys
var keyboardGamepadBindings = map[emulator.Button][]ebiten.Key{
	emulator.BUTTON_START:    {ebiten.KeyBackspace},
	emulator.BUTTON_SELECT:   {ebiten.KeyShiftRight},
	emulator.BUTTON_DUP:      {ebiten.KeyUp},
	emulator.BUTTON_DRIGHT:   {ebiten.KeyRight},
	emulator.BUTTON_DDOWN:    {ebiten.KeyDown},
	emulator.BUTTON_DLEFT:    {ebiten.KeyLeft},
	emulator.BUTTON_L2:       {ebiten.KeyKPDivide},
	emulator.BUTTON_R2:       {ebiten.KeyKPMultiply},
	emulator.BUTTON_L1:       {ebiten.KeyKP7},
	emulator.BUTTON_R1:       {ebiten.KeyKP9},
	emulator.BUTTON_TRIANGLE: {ebiten.KeyKP8},
	emulator.BUTTON_CIRCLE:   {ebiten.KeyKP6},
	emulator.BUTTON_CROSS:    {ebiten.KeyKP2},
	emulator.BUTTON_SQUARE:   {ebiten.KeyKP4},
}

type ebitenGame struct {
	renderer   *emulator.EbitenRenderer
	gamepadIDs map[ebiten.GamepadID]struct{}
	axes       map[ebiten.GamepadID][]float64
}

func (g *ebitenGame) Update() error {
	if cpu == nil {
		return nil
	}
	pad := cpu.Inter.PadMemCard.Pad1
	g.handleConnectedGamepads()
	g.handleGamepadInput(pad)
	handleKeyboard(pad)

	return nil
}

func handleKeyboard(pad *emulator.Gamepad) {
	for _, button := range emulator.GamepadButtons {
		keys := keyboardGamepadBindings[button]
		for _, key := range keys {
			if ebiten.IsKeyPressed(key) {
				pad.SetButtonState(button, emulator.BUTTON_STATE_PRESSED)
			} else if inpututil.IsKeyJustReleased(key) {
				pad.SetButtonState(button, emulator.BUTTON_STATE_RELEASED)
			}
			break
		}
	}

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		os.Exit(0)
	}
}

func (g *ebitenGame) handleConnectedGamepads() {
	if g.gamepadIDs == nil {
		g.gamepadIDs = map[ebiten.GamepadID]struct{}{}
	}

	gamepadsConnected := inpututil.AppendJustConnectedGamepadIDs(nil)
	for _, id := range gamepadsConnected {
		fmt.Printf("main: gamepad connected: id: %d, SDL ID: %s\n", id, ebiten.GamepadSDLID(id))
		g.gamepadIDs[id] = struct{}{}
	}

	for id := range g.gamepadIDs {
		if inpututil.IsGamepadJustDisconnected(id) {
			fmt.Printf("main: gamepad disconnected: id: %d\n", id)
			delete(g.gamepadIDs, id)
		}
	}
}

func (g *ebitenGame) handleGamepadInput(pad *emulator.Gamepad) {
	g.axes = map[ebiten.GamepadID][]float64{}

	for id := range g.gamepadIDs {
		maxAxis := ebiten.GamepadAxisCount(id)
		for a := 0; a < maxAxis; a++ {
			v := ebiten.GamepadAxisValue(id, a)
			g.axes[id] = append(g.axes[id], v)
		}

		maxButton := ebiten.GamepadButton(ebiten.GamepadButtonCount(id))

		for b := ebiten.GamepadButton(id); b < maxButton; b++ {
			// log button events
			if inpututil.IsGamepadButtonJustPressed(id, b) {
				fmt.Printf("main: button pressed: id: %d, button: %d\n", id, b)
				pad.SetButtonState(buttonFromId(int(b)), emulator.BUTTON_STATE_PRESSED)
			}
			if inpututil.IsGamepadButtonJustReleased(id, b) {
				fmt.Printf("main: button released: id: %d, button: %d\n", id, b)
				pad.SetButtonState(buttonFromId(int(b)), emulator.BUTTON_STATE_RELEASED)
			}
		}
	}
}

func buttonFromId(id int) emulator.Button {
	switch id {
	case 0: // A -> Cross
		return emulator.BUTTON_CROSS
	case 1: // B -> Circle
		return emulator.BUTTON_CIRCLE
	case 3: // X -> Square
		return emulator.BUTTON_SQUARE
	case 4: // Y -> Triangle
		return emulator.BUTTON_TRIANGLE
	case 15: // DPadUp
		return emulator.BUTTON_DUP
	case 17: // DPadDown
		return emulator.BUTTON_DDOWN
	case 18: // DPadLeft
		return emulator.BUTTON_DLEFT
	case 16: // DPadRight
		return emulator.BUTTON_DRIGHT
	case 11: // Start
		return emulator.BUTTON_START
	case 12: // Back -> Select
		return emulator.BUTTON_SELECT
	case 6: // LeftShoulder
		return emulator.BUTTON_L1
	case 7: // RightShoulder
		return emulator.BUTTON_R1
	case 8:
		return emulator.BUTTON_R2
	case 9:
		return emulator.BUTTON_L2
	}
	return 0
}

func (g *ebitenGame) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear

	// scale rendered frame to fit window
	fx := currentFrame.Bounds().Dx()
	fy := currentFrame.Bounds().Dy()
	scaleX := float64(width) / float64(fx)
	scaleY := float64(height) / float64(fy)
	op.GeoM.Scale(scaleX, scaleY)

	wg.Wait()
	screen.DrawImage(currentFrame, op)

	if *showFps {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%f fps", 1/frameDt), 8, 8)
	}
	if *showCycles {
		ebitenutil.DebugPrintAt(
			screen,
			fmt.Sprintf("%d cycles\npc: 0x%x", cpu.Th.Cycles, cpu.PC),
			8, 24,
		)
	}

	// draw error message if there was a panic
	if didPanic {
		ebitenutil.DebugPrintAt(screen, panicString, 8, 48+24)
	}
}

func (g *ebitenGame) Layout(insideWidth, insideHeight int) (int, int) {
	return width, height
}

func (g *ebitenGame) drawFrame() {
	wg.Add(1)
	defer wg.Done()

	// calculate delta time
	frameDt = time.Since(prevFrameTime).Seconds()

	// create renderer if it's nil
	if g.renderer == nil {
		g.renderer = gpu.NewEbitenRenderer()
	}

	// clear previous frame and draw the new one
	// FIXME: for some reason, the image is flickering after the GPU timings were implemented
	currentFrame.Clear()
	g.renderer.Draw(currentFrame)

	prevFrameTime = time.Now()
}

func startEbitenWindow(g *ebitenGame) {
	ebiten.SetWindowSize(width, height)
	ebiten.SetWindowTitle("gopsx")
	ebiten.SetTPS(ebiten.SyncWithFPS)

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
	discPath := flag.String("disc", "", "disc .BIN path")
	flag.Parse()

	if *discPath != "" {
		// try to load disc
		file, err := os.Open(*discPath)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		disc, err = emulator.NewDisc(file)
		if err != nil {
			panic(err)
		}
		fmt.Printf("main: disc region: %s\n", disc.RegionString())
	}

	g := &ebitenGame{}
	go startEmulator(g, *biosPath)
	startEbitenWindow(g)
}

func startEmulator(g *ebitenGame, biosPath string) {
	// start emulator
	bios := loadBios(biosPath)
	ram := emulator.NewRAM()

	hardware := emulator.HARDWARE_NTSC
	if disc != nil {
		hardware = emulator.GetHardwareFromRegion(disc.Region)
	}
	gpu = emulator.NewGPU(hardware)

	gpu.SetFrameEnd(g.drawFrame)
	inter := emulator.NewInterconnect(bios, ram, gpu, disc)
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
	fmt.Printf("main: loading bios \"%s\"\n", path)
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

	fmt.Printf("main: loaded bios in %s\n", time.Since(start))
	return bios
}
