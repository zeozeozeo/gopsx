package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/zeozeozeo/gopsx/emulator"
)

func main() {
	// parse arguments
	biosPath := flag.String("bios", "SCPH1001.BIN", "path to the BIOS file")
	flag.Parse()

	// start emulator
	bios := loadBios(*biosPath)
	ram := emulator.NewRAM()
	gpu := emulator.NewGPU()
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
