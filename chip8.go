package main

import (
	"io"
	"log"
	"math/rand/v2"
	"os"

	"github.com/jupiterrider/purego-sdl3/sdl"
)

type sdl_t struct {
	window   *sdl.Window
	renderer *sdl.Renderer
}

type EmulatorState int

const (
	QUIT EmulatorState = iota
	RUNNING
	PAUSED
)

type Instruction struct {
	opcode uint16
	NNN    uint16 // 12 bit address/constant
	NN     uint8  // 8 bit constant
	N      uint8  // 4 bit constant
	X      uint8  // 4 bit register identifier
	Y      uint8  // 4 bit register identifier
}

type CHIP8 struct {
	state        EmulatorState
	ram          [4096]uint8
	entryPoint   uint32
	display      [64 * 32]bool   // Emulate original CHIP8 resolution pixels
	pixelColor   [64 * 32]uint32 // CHIP8 pixel colors to draw
	stack        [12]uint16      // Subroutine stack
	stackPointer uint8           // Stack pointer
	V            [16]uint8       // Data registers V0-VF
	I            uint16          // Index register
	PC           uint16          // Program counter
	delayTimer   uint8           // Decrements at 60hz when > 0
	soundTimer   uint8           // Decrements at 60hz and plays tone when > 0
	keypad       [16]bool        // Hexadecimal keypad 0x0-0xF
	romName      string          // Currently running ROM
	inst         Instruction     // Currently executing instruction
	renderer     Renderer
	keyboard     Keyboard
	speaker      Speaker
	beeping      bool
}

func (chip8 *CHIP8) Reset() {
	for i := range chip8.ram {
		chip8.ram[i] = 0
	}

	for i := range chip8.V {
		chip8.V[i] = 0
	}
	chip8.I = 0
	chip8.PC = uint16(chip8.entryPoint)
	chip8.stackPointer = 0
	for i := range chip8.stack {
		chip8.stack[i] = 0
	}

	for i := range chip8.display {
		chip8.display[i] = false
	}

	for i := range chip8.keypad {
		chip8.keypad[i] = false
	}

	chip8.delayTimer = 0
	chip8.soundTimer = 0

	chip8.speaker.Close()
}

func (chip8 *CHIP8) Init(romName string, config *Config, sdl_t sdl_t) {
	chip8.entryPoint = 0x200

	chip8.LoadFont()

	chip8.LoadRom(romName, chip8.entryPoint)

	chip8.state = RUNNING
	chip8.PC = uint16(chip8.entryPoint)
	chip8.romName = romName
	for i := range chip8.pixelColor {
		chip8.pixelColor[i] = config.bgColor
	}

	chip8.renderer = *NewRenderer(chip8, config, sdl_t)
	chip8.keyboard.Init(chip8, config, sdl_t)

	chip8.beeping = false
	chip8.speaker.Init(&chip8.beeping, &config.volume)
}

func (chip8 *CHIP8) LoadFont() {
	println("Loading font...")

	font := [...]uint8{
		0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
		0x20, 0x60, 0x20, 0x20, 0x70, // 1
		0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
		0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
		0x90, 0x90, 0xF0, 0x10, 0x10, // 4
		0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
		0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
		0xF0, 0x10, 0x20, 0x40, 0x40, // 7
		0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
		0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
		0xF0, 0x90, 0xF0, 0x90, 0x90, // A
		0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
		0xF0, 0x80, 0x80, 0x80, 0xF0, // C
		0xE0, 0x90, 0x90, 0x90, 0xE0, // D
		0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
		0xF0, 0x80, 0xF0, 0x80, 0x80, // F
	}

	copy(chip8.ram[:], font[:])

	println("Font loaded")
}

func (chip8 *CHIP8) LoadRom(romName string, entryPoint uint32) {
	println("Loading ROM...")

	// Open ROM file
	file, err := os.Open(romName)
	if err != nil {
		log.Fatalf("failed to open ROM: %v", err)
	}
	defer file.Close()

	// Read ROM into memory starting at 0x200
	romData, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("failed to read ROM: %v", err)
	}

	copy(chip8.ram[entryPoint:], romData)

	println("Loaded ROM:", romName)
}

func (chip8 *CHIP8) UpdateTimers() {
	if chip8.delayTimer > 0 {
		chip8.delayTimer--
	}

	if chip8.soundTimer > 0 {
		chip8.soundTimer--
		chip8.beeping = true
	} else {
		chip8.beeping = false
	}
}

func (chip8 *CHIP8) ExecuteInstruction(config Config) {
	var carry bool

	// Get the next opcode from the ram
	chip8.inst.opcode = uint16(chip8.ram[chip8.PC])<<8 | uint16(chip8.ram[chip8.PC+1])
	chip8.PC += 2 // Pre-increment program counter for next opcode

	chip8.inst.NNN = chip8.inst.opcode & 0x0FFF
	chip8.inst.NN = uint8(chip8.inst.opcode) & 0x0FF
	chip8.inst.N = uint8(chip8.inst.opcode) & 0x0F
	chip8.inst.X = uint8((chip8.inst.opcode & 0x0F00) >> 8)
	chip8.inst.Y = uint8((chip8.inst.opcode & 0x00F0) >> 4)

	switch (chip8.inst.opcode >> 12) & 0x0F {
	case 0x00:
		switch chip8.inst.NNN {
		case 0x0E0:
			// 0x00E0: Clear the screen
			for i := range chip8.display {
				chip8.display[i] = false
			}
		case 0x0EE:
			// 0x00EE: Return from subroutine
			chip8.stackPointer--
			chip8.PC = chip8.stack[chip8.stackPointer]
		default:
			// Unimplemented/invalid opcode, may be 0xNNN for calling machine code routine for RCA1802
			break
		}
	case 0x01:
		// 0x1NNN: Jump to address NNN
		chip8.PC = chip8.inst.NNN
	case 0x02:
		// 0x2NNN: Call subroutine at NNN
		chip8.stack[chip8.stackPointer] = chip8.PC
		chip8.stackPointer++
		chip8.PC = chip8.inst.NNN
	case 0x03:
		// 0x3XNN: Check if VX == NN, if so, skip the next instruction
		if chip8.V[chip8.inst.X] == chip8.inst.NN {
			chip8.PC += 2
		}
	case 0x04:
		// 0x4XNN: Check if VX != NN, if so, skip the next instruction
		if chip8.V[chip8.inst.X] != chip8.inst.NN {
			chip8.PC += 2
		}
	case 0x05:
		// 0x5XY0: Check if VX == VY, if so, skip the next instruction
		if chip8.inst.N != 0 {
			break
		}

		if chip8.V[chip8.inst.X] == chip8.V[chip8.inst.Y] {
			chip8.PC += 2
		}
	case 0x06:
		// 0x6XNN: Set register VX to NN
		chip8.V[chip8.inst.X] = chip8.inst.NN
	case 0x07:
		// 0x7XNN: Set register VX += NN
		chip8.V[chip8.inst.X] += chip8.inst.NN
	case 0x08:
		switch chip8.inst.N {
		case 0:
			// 0x8XY0: Set register VX = VY
			chip8.V[chip8.inst.X] = chip8.V[chip8.inst.Y]
		case 1:
			// 0x8XY1: Set register VX |= VY
			chip8.V[chip8.inst.X] |= chip8.V[chip8.inst.Y]
			if config.currentExtension == CHIP_8 {
				chip8.V[0xF] = 0
			}
		case 2:
			// 0x8XY2: Set register VX &= VY
			chip8.V[chip8.inst.X] &= chip8.V[chip8.inst.Y]
			if config.currentExtension == CHIP_8 {
				chip8.V[0xF] = 0
			}
		case 3:
			// 0x8XY3: Set register VX ^= VY
			chip8.V[chip8.inst.X] ^= chip8.V[chip8.inst.Y]
			if config.currentExtension == CHIP_8 {
				chip8.V[0xF] = 0
			}
		case 4:
			// 0x8XY4: Set register VX += VY, set VF to 1 if carry, 0 if not
			carry = (uint16(chip8.V[chip8.inst.X]) + uint16(chip8.V[chip8.inst.Y])) > 255

			chip8.V[chip8.inst.X] += chip8.V[chip8.inst.Y]

			if carry {
				chip8.V[0xF] = 1
			} else {
				chip8.V[0xF] = 0
			}
		case 5:
			// 0x8XY5: Set register VX -= VY, set VF to 1 if there is not a borrow (result is positive/0)
			carry = chip8.V[chip8.inst.Y] <= chip8.V[chip8.inst.X]

			chip8.V[chip8.inst.X] -= chip8.V[chip8.inst.Y]

			if carry {
				chip8.V[0xF] = 1
			} else {
				chip8.V[0xF] = 0
			}
		case 6:
			// 0x8XY6: Set register VX >>= 1, store shifted off bit in VF
			var carry uint8

			if config.currentExtension == CHIP_8 {
				carry = chip8.V[chip8.inst.Y] & 1
				chip8.V[chip8.inst.X] = chip8.V[chip8.inst.Y] >> 1
			} else {
				carry = chip8.V[chip8.inst.X] & 1
				chip8.V[chip8.inst.X] >>= 1
			}

			chip8.V[0xF] = carry
		case 7:
			// 0x8XY7: Set register VX = VY - VX, set VF to 1 if there is not a borrow (result is positive/0)
			carry = chip8.V[chip8.inst.X] <= chip8.V[chip8.inst.Y]

			chip8.V[chip8.inst.X] = chip8.V[chip8.inst.Y] - chip8.V[chip8.inst.X]

			if carry {
				chip8.V[0xF] = 1
			} else {
				chip8.V[0xF] = 0
			}
		case 0xE:
			// 0x8XYE: Set register VX <<= 1, store shifted off bit in VF
			var carry uint8

			if config.currentExtension == CHIP_8 {
				carry = (chip8.V[chip8.inst.Y] & 0x80) >> 7
				chip8.V[chip8.inst.X] = chip8.V[chip8.inst.Y] << 1
			} else {
				carry = (chip8.V[chip8.inst.Y] & 0x80) >> 7
				chip8.V[chip8.inst.X] <<= 1
			}

			chip8.V[0xF] = carry
		default:
			// Wrong/unimplemented opcode
			break
		}
	case 0x09:
		// 0x9XY0: Check if VX != VY; Skip next instruction if so
		if chip8.inst.N != 0 {
			break
		}

		if chip8.V[chip8.inst.X] != chip8.V[chip8.inst.Y] {
			chip8.PC += 2
		}
	case 0x0A:
		// 0xANNN: Set index register I to NNN
		chip8.I = chip8.inst.NNN
	case 0x0B:
		// 0xBNNN: Jump to V0 + NNN
		chip8.PC = uint16(chip8.V[0]) + chip8.inst.NNN
	case 0x0C:
		// 0xCXNN: Sets register VX = rand() % 256 & NN (bitwise AND)
		chip8.V[chip8.inst.X] = uint8(rand.IntN(256)) & chip8.inst.NN
	case 0x0D:
		// 0xDXYN: Draw N-height sprite at coords X, Y: Read from memory location I
		//   Screen pixels are XOR'd with sprite bits,
		//   VF (Carry flag) is set if any screen pixels are set off; This is useful
		//   for collision detection or other reasons.
		xCoord := chip8.V[chip8.inst.X] % uint8(config.window_width)
		yCoord := chip8.V[chip8.inst.Y] % uint8(config.window_height)
		origX := xCoord // store original X for each row reset

		chip8.V[0xF] = 0 // reset VF (collision flag)

		for row := uint8(0); row < chip8.inst.N; row++ {
			// Get sprite byte from memory
			spriteData := chip8.ram[chip8.I+uint16(row)]
			xCoord = origX // reset x position for this row

			for col := int8(7); col >= 0; col-- {
				spriteBit := (spriteData & (1 << col)) != 0

				if chip8.renderer.SetPixel(xCoord, yCoord, spriteBit) {
					chip8.V[0xF] = 1
				}

				// Stop drawing row if we hit right edge
				xCoord++
				if xCoord >= uint8(config.window_width) {
					break
				}
			}

			// Stop drawing sprite if we hit bottom edge
			yCoord++
			if yCoord >= uint8(config.window_height) {
				break
			}
		}
	case 0x0E:
		switch chip8.inst.NN {
		case 0x9E:
			// 0xEX9E: Skip next instruction if key in VX is pressed
			if chip8.keypad[chip8.V[chip8.inst.X]] {
				chip8.PC += 2
			}
		case 0xA1:
			// 0xEX9E: Skip next instruction if key in VX is not pressed
			if !chip8.keypad[chip8.V[chip8.inst.X]] {
				chip8.PC += 2
			}
		}
	case 0x0F:
		switch chip8.inst.NN {
		case 0x0A:
			// 0xFX0A: VX = get_key(); Await until a keypress, and store in VX
			var anyKeyPressed bool = false
			var key uint8 = 0xFF

			for i := 0; key == 0xFF && i < len(chip8.keypad); i++ {
				if chip8.keypad[i] {
					key = uint8(i) // Save pressed key to check until it is released
					anyKeyPressed = true
					break
				}
			}

			// If no key has been pressed yet, keep getting the current opcode & running this instruction
			if !anyKeyPressed {
				chip8.PC -= 2
			} else {
				// A key has been pressed, also wait until it is released to set the key in VX
				if chip8.keypad[key] {
					chip8.PC -= 2
				} else {
					chip8.V[chip8.inst.X] = key // VX = key
					key = 0xFF                  // Reset key to not found
					anyKeyPressed = false       // Reset to nothing pressed yet
				}
			}
		case 0x1E:
			// 0xFX1E: I += VX; Add VX to register I. For non-Amiga CHIP8, does not affect VF
			chip8.I += uint16(chip8.V[chip8.inst.X])
		case 0x07:
			// 0xFX07: VX = delay timer
			chip8.V[chip8.inst.X] = chip8.delayTimer
		case 0x15:
			// 0xFX15: delay timer = VX
			chip8.delayTimer = chip8.V[chip8.inst.X]
		case 0x18:
			// 0xFX18: sound timer = VX
			chip8.soundTimer = chip8.V[chip8.inst.X]
		case 0x29:
			// 0xFX29: Set register I to sprite location in memory for character in VX (0x0-0xF)
			chip8.I = uint16(chip8.V[chip8.inst.X] * 5)
		case 0x33:
			// 0xFX33: Store BCD representation of VX at memory offset from I;
			//   I = hundred's place, I+1 = ten's place, I+2 = one's place
			var bcd uint8 = chip8.V[chip8.inst.X]
			chip8.ram[chip8.I+2] = bcd % 10
			bcd /= 10
			chip8.ram[chip8.I+1] = bcd % 10
			bcd /= 10
			chip8.ram[chip8.I] = bcd
		case 0x55:
			// 0xFX55: Register dump V0-VX inclusive to memory offset from I;
			//   SCHIP does not increment I, CHIP8 does increment I
			var i uint8
			for i = 0; i <= chip8.inst.X; i++ {
				if config.currentExtension == CHIP_8 {
					chip8.ram[chip8.I] = chip8.V[i]
					chip8.I++
				} else {
					chip8.ram[chip8.I+uint16(i)] = chip8.V[i]
				}
			}
		case 0x65:
			// 0xFX65: Register load V0-VX inclusive from memory offset from I;
			//   SCHIP does not increment I, CHIP8 does increment I
			var i uint8
			for i = 0; i <= chip8.inst.X; i++ {
				if config.currentExtension == CHIP_8 {
					chip8.V[i] = chip8.ram[chip8.I]
					chip8.I++
				} else {
					chip8.V[i] = chip8.ram[chip8.I+uint16(i)]
				}
			}
		}
	default:
		sdl.Log("Unknown opcode: 0x%X", chip8.inst.opcode)
	}
}

func InitSDL(sdl_t *sdl_t, config Config) bool {
	if !sdl.Init(sdl.InitVideo | sdl.InitAudio | sdl.InitEvents) {
		sdl.Log("Could not initialize SDL subsystems! %s\n", sdl.GetError())
		return false
	}

	if sdl_t.window = sdl.CreateWindow("CHIP8 Emulator", config.window_width*int32(config.scale), config.window_height*int32(config.scale), 0); sdl_t.window == nil {
		sdl.Log("Couldn't create SDL window %s\n", sdl.GetError())
		return false
	}

	if sdl_t.renderer = sdl.CreateRenderer(sdl_t.window, ""); sdl_t.renderer == nil {
		sdl.Log("Couldn't create SDL renderer %s\n", sdl.GetError())
		return false
	}

	return true
}

func FinalCleanup(sdl_t sdl_t, sp Speaker) {
	sdl.DestroyWindow(sdl_t.window)
	sdl.DestroyRenderer(sdl_t.renderer)
	sp.Close()
	sdl.Quit()
}

func main() {
	var sdl_t sdl_t
	var config Config
	var chip8 CHIP8

	config.SetConfigFromArgs()

	if !InitSDL(&sdl_t, config) {
		panic("Something gone wrong when initializing SDL")
	}

	chip8.Init(config.romName, &config, sdl_t)

	chip8.renderer.ClearScreen()

	for chip8.state != QUIT {
		chip8.keyboard.HandleInput()

		if chip8.state == PAUSED {
			continue
		}

		startFrameTime := sdl.GetPerformanceCounter()

		var i uint32
		for i = 0; i < config.instsPerSecond/60; i++ {
			chip8.ExecuteInstruction(config)
		}

		endFrameTime := sdl.GetPerformanceCounter()

		timeElapsed := ((endFrameTime - startFrameTime) * 1000) / sdl.GetPerformanceFrequency()

		if timeElapsed < 16 {
			sdl.DelayNS((16 - timeElapsed) * 1_000_000) // convert ms -> ns
		}

		chip8.renderer.Render()

		sdl.RenderPresent(sdl_t.renderer)

		chip8.UpdateTimers()
	}

	FinalCleanup(sdl_t, chip8.speaker)
}
