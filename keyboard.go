package main

import "github.com/jupiterrider/purego-sdl3/sdl"

type Keyboard struct {
	chip8  *CHIP8
	config *Config
	sdl_t  sdl_t
}

func (k *Keyboard) Init(chip8 *CHIP8, config *Config, sdl_t sdl_t) {
	k.chip8 = chip8
	k.config = config
	k.sdl_t = sdl_t
}

// CHIP8 Keypad  QWERTY
// 123C          1234
// 456D          qwer
// 789E          asdf
// A0BF          zxcv
var keymap = map[sdl.Scancode]byte{
	sdl.Scancode1: 0x1,
	sdl.Scancode2: 0x2,
	sdl.Scancode3: 0x3,
	sdl.Scancode4: 0xC,

	sdl.ScancodeQ: 0x4,
	sdl.ScancodeW: 0x5,
	sdl.ScancodeE: 0x6,
	sdl.ScancodeR: 0xD,

	sdl.ScancodeA: 0x7,
	sdl.ScancodeS: 0x8,
	sdl.ScancodeD: 0x9,
	sdl.ScancodeF: 0xE,

	sdl.ScancodeZ: 0xA,
	sdl.ScancodeX: 0x0,
	sdl.ScancodeC: 0xB,
	sdl.ScancodeV: 0xF,
}

func (k *Keyboard) HandleInput() {
	var event sdl.Event

	for sdl.PollEvent(&event) {
		switch event.Type() {
		case sdl.EventQuit:
			// Exit window; End program
			k.chip8.state = QUIT
			return
		case sdl.EventKeyDown:
			k.OnKeyDown(event)
		case sdl.EventKeyUp:
			k.OnKeyUp(event)
		}
	}
}

func (k *Keyboard) OnKeyDown(event sdl.Event) {
	switch event.Key().Scancode {
	case sdl.ScancodeEscape:
		// Escape key; Exit window & End program
		k.chip8.state = QUIT
	case sdl.ScancodeSpace:
		// Space bar
		if k.chip8.state == RUNNING {
			k.chip8.state = PAUSED
			println("==== PAUSED ====")
		} else {
			k.chip8.state = RUNNING
			println("==== RESUMED ====")
		}
	case sdl.ScancodeR:
		// 'r': Reset CHIP8 machine for the current ROM
		println("==== RELOADING ROM ====")
		k.chip8.Reset()
		k.chip8.Init(k.chip8.romName, k.config, k.sdl_t)
	case sdl.ScancodeO:
		// 'o': Decrease Volume
		if k.config.volume > 0 {
			k.config.volume -= 500
		}
	case sdl.ScancodeP:
		// 'p': Increase Volume
		const maxInt16 = 32767
		if k.config.volume < maxInt16 {
			k.config.volume += 500
		}
	case sdl.ScancodeJ:
		// 'j': Decrease color lerp rate
		if k.config.colorLerpRate > 0.1 {
			k.config.colorLerpRate -= 0.1
		}
	case sdl.ScancodeK:
		// 'k': Increase color lerp rate
		if k.config.colorLerpRate < 1.0 {
			k.config.colorLerpRate += 0.1
		}
	default:
		if chip8Key, ok := keymap[event.Key().Scancode]; ok {
			k.chip8.keypad[chip8Key] = true
		}
	}
}

func (k *Keyboard) OnKeyUp(event sdl.Event) {
	if chip8Key, ok := keymap[event.Key().Scancode]; ok {
		k.chip8.keypad[chip8Key] = false
	}
}
