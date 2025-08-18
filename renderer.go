package main

import (
	"github.com/jupiterrider/purego-sdl3/sdl"
)

type Renderer struct {
	chip8  *CHIP8
	config *Config
	sdl_t  sdl_t
}

func NewRenderer(chip8 *CHIP8, config *Config, sdl_t sdl_t) *Renderer {
	return &Renderer{
		chip8:  chip8,
		config: config,
		sdl_t:  sdl_t,
	}
}

func (r *Renderer) SetPixel(x, y uint8, spriteBit bool) bool {
	cols := uint8(r.config.window_width)
	rows := uint8(r.config.window_height)

	// Wrap around screen edges
	x %= cols
	y %= rows

	pixelIndex := int(y)*int(cols) + int(x)

	collision := spriteBit && r.chip8.display[pixelIndex]
	r.chip8.display[pixelIndex] = r.chip8.display[pixelIndex] != spriteBit

	return collision
}

func (r *Renderer) Render() {
	config := r.config
	chip8 := r.chip8
	renderer := r.sdl_t.renderer

	rect := sdl.FRect{X: 0, Y: 0, W: float32(config.scale), H: float32(config.scale)}

	bg_r := uint8((config.bgColor >> 24) & 0xFF)
	bg_g := uint8((config.bgColor >> 16) & 0xFF)
	bg_b := uint8((config.bgColor >> 8) & 0xFF)
	bg_a := uint8((config.bgColor >> 0) & 0xFF)

	for i := 0; i < len(chip8.display); i++ {
		rect.X = float32(i % int(config.window_width) * int(config.scale))
		rect.Y = float32(i / int(config.window_width) * int(config.scale))

		if chip8.display[i] {
			if chip8.pixelColor[i] != config.fgColor {
				chip8.pixelColor[i] = r.ColorLerp(chip8.pixelColor[i], config.fgColor, config.colorLerpRate)
			}

			red := uint8((chip8.pixelColor[i] >> 24) & 0xFF)
			green := uint8((chip8.pixelColor[i] >> 16) & 0xFF)
			blue := uint8((chip8.pixelColor[i] >> 8) & 0xFF)
			alpha := uint8((chip8.pixelColor[i] >> 0) & 0xFF)

			sdl.SetRenderDrawColor(renderer, red, green, blue, alpha)
			sdl.RenderFillRect(renderer, &rect)

			if config.pixelOutlines {
				sdl.SetRenderDrawColor(renderer, bg_r, bg_g, bg_b, bg_a)
				sdl.RenderRect(renderer, &rect)
			}
		} else {
			if chip8.pixelColor[i] != config.bgColor {
				chip8.pixelColor[i] = r.ColorLerp(chip8.pixelColor[i], config.bgColor, config.colorLerpRate)
			}

			red := uint8((chip8.pixelColor[i] >> 24) & 0xFF)
			green := uint8((chip8.pixelColor[i] >> 16) & 0xFF)
			blue := uint8((chip8.pixelColor[i] >> 8) & 0xFF)
			alpha := uint8((chip8.pixelColor[i] >> 0) & 0xFF)

			sdl.SetRenderDrawColor(renderer, red, green, blue, alpha)
			sdl.RenderFillRect(renderer, &rect)
		}
	}
}

func (r *Renderer) ClearScreen() {
	config := r.config
	renderer := r.sdl_t.renderer

	bg_r := uint8((config.bgColor >> 24) & 0xFF)
	bg_g := uint8((config.bgColor >> 16) & 0xFF)
	bg_b := uint8((config.bgColor >> 8) & 0xFF)
	bg_a := uint8((config.bgColor >> 0) & 0xFF)

	sdl.SetRenderDrawColor(renderer, bg_r, bg_g, bg_b, bg_a)
	sdl.RenderClear(renderer)
}

func (r *Renderer) ColorLerp(startColor, endColor uint32, t float32) uint32 {
	sr := float32((startColor >> 24) & 0xFF)
	sg := float32((startColor >> 16) & 0xFF)
	sb := float32((startColor >> 8) & 0xFF)
	sa := float32(startColor & 0xFF)

	er := float32((endColor >> 24) & 0xFF)
	eg := float32((endColor >> 16) & 0xFF)
	eb := float32((endColor >> 8) & 0xFF)
	ea := float32(endColor & 0xFF)

	retR := uint8((1-t)*sr + t*er)
	retG := uint8((1-t)*sg + t*eg)
	retB := uint8((1-t)*sb + t*eb)
	retA := uint8((1-t)*sa + t*ea)

	return (uint32(retR) << 24) | (uint32(retG) << 16) | (uint32(retB) << 8) | uint32(retA)
}
