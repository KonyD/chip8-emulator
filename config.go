package main

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	window_width     int32
	window_height    int32
	fgColor          uint32
	bgColor          uint32
	scale            uint32
	pixelOutlines    bool
	romName          string
	instsPerSecond   uint32 // CHIP8 CPU "clock rate" or hz
	volume           int16
	currentExtension Extension
	colorLerpRate    float32
}

type Extension int

const (
	CHIP_8 Extension = iota
	SUPERCHIP
	XOCHIP
)

func (config *Config) SetConfigFromArgs() {
	// defaults
	config.window_width, config.window_height = 64, 32
	config.fgColor, config.bgColor = 0xFFFFFFFF, 0x00000000 // WHITE & BLACK
	config.scale, config.pixelOutlines, config.romName = 10, false, "roms/tests/1-chip8-logo.ch8"
	config.instsPerSecond = 500
	config.volume = 3000
	config.currentExtension = CHIP_8
	config.colorLerpRate = 0.7

	for _, arg := range os.Args[1:] {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]

		switch key {
		case "name":
			config.romName = "roms/" + value
		case "scale":
			if v, err := strconv.Atoi(value); err == nil {
				config.scale = uint32(v)
			}
		case "pixelOutlines":
			if v, err := strconv.ParseBool(value); err == nil {
				config.pixelOutlines = v
			}
		case "instsPerSecond":
			if v, err := strconv.Atoi(value); err == nil {
				config.instsPerSecond = uint32(v)
			}
		case "colorLerpRate":
			if v, err := strconv.ParseFloat(value, 32); err == nil {
				config.colorLerpRate = float32(v)
			}
		default:
			log.Fatalf("Unknown parameter: %q", key)
		}
	}
}
