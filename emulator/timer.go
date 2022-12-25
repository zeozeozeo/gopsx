package emulator

// Represents a timer clock source
type Clock uint8

const (
	CLOCK_SYSCLOCK      Clock = iota // CPU clock at 33.8685MHz
	CLOCK_SYSCLOCK_DIV8 Clock = iota // CPU clock divided by 8 (~4.2335625MHz)
	CLOCK_GPU_DOTCLOCK  Clock = iota // GPU's dotclock (~53MHz)
	CLOCK_GPU_HSYNC     Clock = iota // GPU's HSync signal
)

// Returns true if it's a GPU clock
func (c Clock) NeedsGPU() bool {
	return c == CLOCK_GPU_DOTCLOCK || c == CLOCK_GPU_HSYNC
}

// All timers use different values for sysclocks for some reason
var ClockSourceLookupTable = [][]Clock{
	// timer 0
	{
		CLOCK_SYSCLOCK, CLOCK_GPU_DOTCLOCK,
		CLOCK_SYSCLOCK, CLOCK_GPU_DOTCLOCK,
	},
	// timer 1
	{
		CLOCK_SYSCLOCK, CLOCK_GPU_HSYNC,
		CLOCK_SYSCLOCK, CLOCK_GPU_HSYNC,
	},
	// timer 2
	{
		CLOCK_SYSCLOCK, CLOCK_SYSCLOCK,
		CLOCK_SYSCLOCK_DIV8, CLOCK_SYSCLOCK_DIV8,
	},
}

type ClockSource uint8

func ClockSourceFromField(field uint16) ClockSource {
	if uint16(int32(field) & ^3) != 0 {
		panicFmt("invalid clock source %d", field)
	}
	return ClockSource(field)
}

func (cs ClockSource) Clock(instance Peripheral) Clock {
	switch instance {
	case PERIPHERAL_TIMER0:
		return ClockSourceLookupTable[0][cs]
	case PERIPHERAL_TIMER1:
		return ClockSourceLookupTable[1][cs]
	case PERIPHERAL_TIMER2:
		return ClockSourceLookupTable[2][cs]
	}
	panic("timer: invalid peripheral for Clock()")
}

// Represents timer synchronization modes when `FreeRun` is false
type TSync uint16

const (
	// Timer 1, timer 2: pause during HBlank/VBlank.
	// Timer 3: stop counter.
	TSYNC_PAUSE TSync = 0
	// Timer 1, timer 2: reset counter at HBlank/VBlank.
	// Timer 3: free run.
	TSYNC_RESET TSync = 1
	// Timer 1, timer 2: wait for HBlank/VBlank and then free run.
	// Timer 3: stop counter.
	TSYNC_RESET_AND_PAUSE TSync = 2
)

func TSyncFromField(field uint16) TSync {
	if field > uint16(TSYNC_RESET_AND_PAUSE) {
		panic("timer: invalid field value for TSyncFromField")
	}
	return TSync(field)
}

type Timer struct {
	Instance Peripheral // 0, 1 or 2
	Counter  uint16     // Timer counter
	FreeRun  bool       // If true, the timer does not synchronize with an external signal
	Target   uint16     // Timer counter target
	TSync    TSync      // Synchronization mode when `FreeRun` is false
	// If true, the counter is reset when it reaches `Target`, otherwise it counts to 0xffff
	TargetWrap      bool
	TargetIrq       bool        // Specifies whether to raise an interrupt when `Target` is reached
	WrapIrq         bool        // Raises an interrupt when `TargetIrq` wraps after 0xffff
	RepeatIrq       bool        // If true, the interrupt is automatically cleared
	NegateIrq       bool        // When true, the IRQ signal is inverted after each interrupt
	ClockSource     ClockSource // Each timer can use a different clock source
	TargetReached   bool        // True if `Target` has been reached since the last read
	OverflowReached bool        // True when the counter overflowed 0xffff
	Period          FracCycles  // Period of a counter tick, the GPU can be used as a source
	Phase           FracCycles  // Current position in the counter tick
	Interrupt       bool        // True if an interrupt is active
}

// Returns a new Timer instance
func NewTimer(instance Peripheral) *Timer {
	return &Timer{
		Instance:    instance,
		TSync:       TSyncFromField(0),
		ClockSource: ClockSourceFromField(0),
		Period:      FracCyclesFromFixed(1),
		Phase:       FracCyclesFromFixed(0),
	}
}

// Resets the timer internal state, gets called when the timer's
// configuration changes or when GPU timings change (if the timer
// relies on them)
func (timer *Timer) Reset(gpu *GPU, th *TimeHandler) {
	switch timer.ClockSource.Clock(timer.Instance) {
	case CLOCK_SYSCLOCK:
		timer.Period = FracCyclesFromCycles(1)
		timer.Phase = FracCyclesFromCycles(0)
	case CLOCK_SYSCLOCK_DIV8:
		timer.Period = FracCyclesFromCycles(8)
		timer.Phase = FracCyclesFromCycles(0)
	case CLOCK_GPU_DOTCLOCK:
		timer.Period = gpu.DotclockPeriod()
		timer.Phase = gpu.DotclockPhase()
	case CLOCK_GPU_HSYNC:
		timer.Period = gpu.HSyncPeriod()
		timer.Phase = gpu.HSyncPhase()
	}

	timer.PredictNextSync(th)
}

// Synchronizes this timer
func (timer *Timer) Sync(th *TimeHandler, irqState *IrqState) {
	delta := th.Sync(timer.Instance)
	if delta == 0 {
		return
	}

	deltaFrac := FracCyclesFromCycles(delta)
	ticks := deltaFrac.Add(timer.Phase)

	count := ticks.GetFixed() / timer.Period.GetFixed()
	phase := ticks.GetFixed() % timer.Period.GetFixed()

	// update current phase
	timer.Phase = FracCyclesFromFixed(phase)
	count += uint64(timer.Counter)
	targetPassed := false

	if (timer.Counter <= timer.Target) && (count > uint64(timer.Target)) {
		timer.TargetReached = true
		targetPassed = true
	}

	var wrap uint64
	overflow := false
	if timer.TargetWrap {
		wrap = uint64(timer.Target) + 1
	} else {
		wrap = 0x10000
	}

	if count >= wrap {
		count %= wrap

		if wrap == 0x10000 {
			timer.OverflowReached = true
			overflow = true
		}
	}

	timer.Counter = uint16(count)

	if (timer.WrapIrq && overflow) || (timer.TargetIrq && targetPassed) {
		var interrupt Interrupt
		switch timer.Instance {
		case PERIPHERAL_TIMER0:
			interrupt = INTERRUPT_TIMER0
		case PERIPHERAL_TIMER1:
			interrupt = INTERRUPT_TIMER1
		case PERIPHERAL_TIMER2:
			interrupt = INTERRUPT_TIMER2
		default:
			panic("timer: unreachable")
		}

		if timer.NegateIrq {
			// TODO
			panic("timer: negate IRQ is not implemented")
		} else {
			// start pulse
			irqState.SetHigh(interrupt)
			timer.Interrupt = true
		}
	} else if !timer.NegateIrq {
		// pulse is over
		timer.Interrupt = false
	}

	timer.PredictNextSync(th)
}

// Returns the value of the mode register
func (timer *Timer) Mode() uint16 {
	var r uint16

	r |= uint16(oneIfTrue(timer.FreeRun))
	r |= uint16(timer.TSync) << 1
	r |= uint16(oneIfTrue(timer.TargetWrap)) << 3
	r |= uint16(oneIfTrue(timer.TargetIrq)) << 4
	r |= uint16(oneIfTrue(timer.WrapIrq)) << 5
	r |= uint16(oneIfTrue(timer.RepeatIrq)) << 6
	r |= uint16(oneIfTrue(timer.NegateIrq)) << 7
	r |= uint16(timer.ClockSource) << 8
	r |= uint16(oneIfTrue(!timer.Interrupt)) << 10
	r |= uint16(oneIfTrue(timer.TargetReached)) << 11
	r |= uint16(oneIfTrue(timer.OverflowReached)) << 12

	// read resets the flags
	timer.TargetReached = false
	timer.OverflowReached = false

	return r
}

// Sets the value of the mode register
func (timer *Timer) SetMode(val uint16) {
	timer.FreeRun = (val & 1) == 0
	timer.TSync = TSyncFromField((val >> 1) & 3)
	timer.TargetWrap = (val>>3)&1 != 0
	timer.TargetIrq = (val>>4)&1 != 0
	timer.WrapIrq = (val>>5)&1 != 0
	timer.RepeatIrq = (val>>6)&1 != 0
	timer.NegateIrq = (val>>7)&1 != 0
	timer.ClockSource = ClockSourceFromField((val >> 8) & 3)

	// writing resets the counter and the interrupt flag
	timer.Counter = 0
	timer.Interrupt = false

	if timer.WrapIrq {
		panicFmt("timer (%d): WrapIrq is not implemented", timer.Instance)
	}
	if (timer.WrapIrq || timer.TargetIrq) && !timer.RepeatIrq {
		panicFmt("timer (%d): unsupported pulse timer interrupt", timer.Instance)
	}
	if !timer.FreeRun {
		panicFmt("timer (%d): sync mode is not supported", timer.Instance)
	}
}

func (timer *Timer) NeedsGPU() bool {
	if !timer.FreeRun {
		panic("timer: sync mode not supported")
	}
	return timer.ClockSource.Clock(timer.Instance).NeedsGPU()
}

func (timer *Timer) PredictNextSync(th *TimeHandler) {
	// TODO: add support for WrapIrq
	if !timer.TargetIrq {
		// we don't have an IRQ
		th.RemoveNextSync(timer.Instance)
		return
	}

	var countdown uint16
	if timer.Counter <= timer.Target {
		countdown = timer.Target - timer.Counter
	} else {
		countdown = 0xffff - timer.Counter + timer.Target
	}

	// convert timer counter to CPU cycles. the interrupt is generated
	// on the next cycle, so we add 1 to it
	delta := timer.Period.GetFixed() * (uint64(countdown) + 1)
	delta -= timer.Phase.GetFixed()
	// round to the next CPU cycle
	delta = FracCyclesFromFixed(delta).Ceil()

	th.SetNextSyncDelta(timer.Instance, delta)
}

type Timers struct {
	// Timer 0: GPU pixel clock.
	// Timer 1: GPU horizontal blanking.
	// Timer 2: System clock divided by 8
	Timers [3]*Timer
}

func NewTimers() *Timers {
	timers := &Timers{
		Timers: [3]*Timer{
			NewTimer(PERIPHERAL_TIMER0),
			NewTimer(PERIPHERAL_TIMER1),
			NewTimer(PERIPHERAL_TIMER2),
		},
	}
	return timers
}

func (timers *Timers) Load(size AccessSize, th *TimeHandler, offset uint32, irqState *IrqState) interface{} {
	if size != ACCESS_WORD && size != ACCESS_HALFWORD {
		panicFmt("timer: unsupported load size %d", size)
	}

	instance := offset >> 4
	timer := timers.Timers[instance]
	timer.Sync(th, irqState)

	var val uint16
	switch offset & 0xf {
	case 0:
		val = timer.Counter
	case 4:
		val = timer.Mode()
	case 8:
		val = timer.Target
	default:
		panicFmt("timer: unhandled register %d", offset&0xf)
	}

	return accessSizeU32(size, uint32(val))
}

func (timers *Timers) Store(
	size AccessSize,
	val interface{},
	th *TimeHandler,
	offset uint32,
	gpu *GPU,
	irqState *IrqState,
) {
	if size != ACCESS_WORD && size != ACCESS_HALFWORD {
		panicFmt("timer: unsupported store size %d", size)
	}

	valU16 := accessSizeToU16(size, val)
	instance := offset >> 4
	timer := timers.Timers[instance]
	timer.Sync(th, irqState)

	switch offset & 0xf {
	case 0:
		timer.Counter = valU16
	case 4:
		timer.SetMode(valU16)
	case 8:
		timer.Target = valU16
	default:
		panicFmt("timer: unhandled store register %d", offset&0xf)
	}

	if timer.NeedsGPU() {
		gpu.Sync(th, irqState)
	}
	timer.Reset(gpu, th)
}

func (timers *Timers) VideoTimingsChanged(th *TimeHandler, irqState *IrqState, gpu *GPU) {
	for _, timer := range timers.Timers {
		if timer.NeedsGPU() {
			timer.Sync(th, irqState)
			timer.Reset(gpu, th)
		}
	}
}

func (timers *Timers) Sync(th *TimeHandler, irqState *IrqState) {
	if th.NeedsSync(PERIPHERAL_TIMER0) {
		timers.Timers[0].Sync(th, irqState)
	}
	if th.NeedsSync(PERIPHERAL_TIMER1) {
		timers.Timers[1].Sync(th, irqState)
	}
	if th.NeedsSync(PERIPHERAL_TIMER2) {
		timers.Timers[2].Sync(th, irqState)
	}
}
