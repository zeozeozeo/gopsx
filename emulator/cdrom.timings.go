package emulator

// TODO: test the timings

const (
	TIMING_COMMAND_PENDING           uint32 = 9400     // Command start -> param transfer
	TIMING_COMMAND_PENDING_VARIATION uint32 = 6000     // Command start -> param transfer
	TIMING_PARAM_PUSH                uint32 = 1800     // Time to transfer 1 parameter
	TIMING_EXECUTION                 uint32 = 2000     // Last param push -> RX FIFO clear
	TIMING_RXFLUSH                   uint32 = 3500     // FIFO clear -> first response byte
	TIMING_RXPUSH                    uint32 = 1500     // Response byte push (after first byte)
	TIMING_BUSY_DELAY                uint32 = 3300     // Last response byte -> busy flag low
	TIMING_IRQ_DELAY                 uint32 = 2000     // Busy flag low -> IRQ trigger
	TIMING_GET_ID_ASYNC              uint32 = 15000    // CommandGetId -> RX clear
	TIMING_GET_ID_RX_PUSH            uint32 = 3100     // RX clear -> first GetId param push
	TIMING_READTOC_ASYNC             uint32 = 16000000 // Read table of contents
	TIMING_READTOC_RX_PUSH           uint32 = 1700     // RX clear -> ReadToc first param push
	TIMING_SEEKL_RX_PUSH             uint32 = 1700     // RX clear -> SeekL first param push
	TIMING_READ_RX_PUSH              uint32 = 1800     // RX clear -> ReadN/ReadS response
	TIMING_PAUSE_RX_PUSH             uint32 = 1700     // RX clear -> Pause response
	TIMING_INIT_RX_PUSH              uint32 = 1700     // RX clear -> Init param push
	TIMING_INIT                      uint32 = 900000   // CD-ROM init
)
