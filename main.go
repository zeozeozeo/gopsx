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
	renderer       *emulator.EbitenRenderer
	gamepadIDs     map[ebiten.GamepadID]struct{}
	axes           map[ebiten.GamepadID][]float64
	pressedButtons map[ebiten.GamepadID][]int
}

func (g *ebitenGame) Update() error {
	if cpu == nil {
		return nil
	}
	pad := cpu.Inter.PadMemCard.Pad1
	g.handleConnectedGamepads()
	g.captureGamepadInput()
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
}

func (g *ebitenGame) handleConnectedGamepads() {
	if g.gamepadIDs == nil {
		g.gamepadIDs = map[ebiten.GamepadID]struct{}{}
	}
	if g.pressedButtons == nil {
		g.pressedButtons = map[ebiten.GamepadID][]int{}
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

func (g *ebitenGame) captureGamepadInput() {
	g.axes = map[ebiten.GamepadID][]float64{}

	for id := range g.gamepadIDs {
		maxAxis := ebiten.GamepadAxisCount(id)
		for a := 0; a < maxAxis; a++ {
			v := ebiten.GamepadAxisValue(id, a)
			g.axes[id] = append(g.axes[id], v)
		}

		maxButton := ebiten.GamepadButton(ebiten.GamepadButtonCount(id))

		for b := ebiten.GamepadButton(id); b < maxButton; b++ {
			if ebiten.IsGamepadButtonPressed(id, b) {
				g.pressedButtons[id] = append(g.pressedButtons[id], int(b))
			}

			// log button events
			if inpututil.IsGamepadButtonJustPressed(id, b) {
				fmt.Printf("main: button pressed: id: %d, button: %d\n", id, b)
			}
			if inpututil.IsGamepadButtonJustReleased(id, b) {
				fmt.Printf("main: button released: id: %d, button: %d\n", id, b)
			}
		}
	}
}

func (g *ebitenGame) handleGamepadInput(pad *emulator.Gamepad) {
	var padButton emulator.Button
	var state emulator.ButtonState

	for id := range g.gamepadIDs {
		buttons := g.pressedButtons[id]
		if len(buttons) == 0 {
			return
		}
		button := buttons[0]

		// HACK: I have no idea if Ebiten exposes any button name
		// constants, so I just tested them myself
		// TODO: make this configurable
		switch button {
		case 0: // A -> Cross
			padButton = emulator.BUTTON_CROSS
		case 1: // B -> Circle
			padButton = emulator.BUTTON_CIRCLE
		case 3: // X -> Square
			padButton = emulator.BUTTON_SQUARE
		case 4: // Y -> Triangle
			padButton = emulator.BUTTON_TRIANGLE
		case 15: // DPadUp
			padButton = emulator.BUTTON_DUP
		case 17: // DPadDown
			padButton = emulator.BUTTON_DDOWN
		case 18: // DPadLeft
			padButton = emulator.BUTTON_DLEFT
		case 16: // DPadRight
			padButton = emulator.BUTTON_DRIGHT
		case 11: // Start
			padButton = emulator.BUTTON_START
		case 12: // Back -> Select
			padButton = emulator.BUTTON_SELECT
		case 6: // LeftShoulder
			padButton = emulator.BUTTON_L1
		case 7: // RightShoulder
			padButton = emulator.BUTTON_R1
		case 8:
			padButton = emulator.BUTTON_R2
		case 9:
			padButton = emulator.BUTTON_L2
		}

		inputButton := ebiten.GamepadButton(button)
		if inpututil.IsGamepadButtonJustPressed(id, inputButton) {
			state = emulator.BUTTON_STATE_PRESSED
		} else if inpututil.IsGamepadButtonJustReleased(id, inputButton) {
			state = emulator.BUTTON_STATE_RELEASED
		}
	}

	pad.SetButtonState(padButton, state)
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
