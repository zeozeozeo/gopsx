package emulator

import "fmt"

func panicFmt(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}

func todo() {
	panic("TODO")
}
