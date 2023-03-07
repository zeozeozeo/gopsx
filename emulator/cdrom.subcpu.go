package emulator

// Sub-CPU sequence state
type SubCpuState int

const (
	SUBCPU_IDLE           SubCpuState = iota // Idle state
	SUBCPU_COMMANDPENDING SubCpuState = iota // A command is waiting to be executed
	SUBCPU_PARAMPUSH      SubCpuState = iota // Parameter transfer
	SUBCPU_EXECUTION      SubCpuState = iota // A command is being executed
	SUBCPU_RXFLUSH        SubCpuState = iota // Response FIFO is cleared
	SUBCPU_RXPUSH         SubCpuState = iota // Response FIFO transfer
	SUBCPU_BUSYDELAY      SubCpuState = iota // Waiting for busy flag
	SUBCPU_IRQDELAY       SubCpuState = iota // IRQ is waiting to be triggered
	SUBCPU_ASYNCRXPUSH    SubCpuState = iota // Asynchronous response transfer
)

// Sub-CPU asynchronous command handler
type AsyncResponseHandler func() uint32

// Sub-CPU asynchronous command response
type SubCpuResponse struct {
	Delay   uint32               // Amount of CPU cycles before the handler should be ran
	Handler AsyncResponseHandler // Command handler
}

func NewSubCpuResponse() *SubCpuResponse {
	return &SubCpuResponse{}
}

func (r *SubCpuResponse) Reset() {
	r.Delay = 0
	r.Handler = nil
}

func (r *SubCpuResponse) IsReady() bool {
	return r.Handler != nil
}

// The CD-ROM controllers' sub-CPU
type SubCpu struct {
	Sequence      SubCpuState     // Current command state
	Timer         uint32          // Time before the next sequence step
	Params        *FIFO           // Command FIFO
	Response      *FIFO           // Response FIFO
	IrqCode       IrqCode         // Command status
	AsyncResponse *SubCpuResponse // Asynchronous command response
}

func NewSubCpu() *SubCpu {
	return &SubCpu{
		Sequence:      SUBCPU_IDLE,
		Params:        NewFIFO(),
		Response:      NewFIFO(),
		IrqCode:       IRQ_CODE_OK,
		AsyncResponse: NewSubCpuResponse(),
	}
}

// Sets scpu.IrqCode
func (scpu *SubCpu) SetIrqCode(irqCode IrqCode) {
	scpu.IrqCode = irqCode
}

// Is a command being executed?
func (scpu *SubCpu) IsInCommand() bool {
	return scpu.Sequence != SUBCPU_IDLE
}

// Returns true if the async response handler is not nil
func (scpu *SubCpu) IsAsyncCommandPending() bool {
	return scpu.AsyncResponse.Handler != nil
}

// Returns the busy flag state
func (scpu *SubCpu) IsBusy() bool {
	return scpu.Sequence == SUBCPU_COMMANDPENDING ||
		scpu.Sequence == SUBCPU_PARAMPUSH ||
		scpu.Sequence == SUBCPU_EXECUTION ||
		scpu.Sequence == SUBCPU_RXFLUSH ||
		scpu.Sequence == SUBCPU_RXPUSH ||
		scpu.Sequence == SUBCPU_BUSYDELAY
}

// Starts a sub-CPU command with a delay
func (scpu *SubCpu) StartCommand(delay uint32) {
	if scpu.IsInCommand() {
		panic("subcpu: StartCommand() while another command is running")
	}
	if scpu.IsAsyncCommandPending() {
		panic("subcpu: StartCommand() while waiting for async response")
	}

	scpu.Sequence = SUBCPU_COMMANDPENDING
	scpu.Timer = delay
	scpu.Params.Clear()
	scpu.Response.Clear()

	// set the command state to be successful by default
	scpu.IrqCode = IRQ_CODE_OK
}

func (scpu *SubCpu) ScheduleAsyncResponse(handler AsyncResponseHandler, delay uint32) {
	if scpu.AsyncResponse.Handler != nil {
		panic("subcpu: tried to schedule async response with another response pending")
	}
	scpu.AsyncResponse.Handler = handler
}
