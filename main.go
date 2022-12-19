package main

import (
	"log"
	"os"
	"time"

	"github.com/zeozeozeo/gopsx/emulator"
)

func main() {
	bios := loadBios()
	ram := emulator.NewRAM()
	inter := emulator.NewInterconnect(bios, ram)
	cpu := emulator.NewCPU(inter)

	for {
		cpu.RunNextInstruction()
	}
}

func loadBios() *emulator.BIOS {
	log.Println("loading bios")
	start := time.Now()

	// read bios
	file, err := os.Open("SCPH1001.BIN")
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
