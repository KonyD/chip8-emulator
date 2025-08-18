package main

import (
	"unsafe"

	"github.com/jupiterrider/purego-sdl3/sdl"
)

type Speaker struct {
	stream             *sdl.AudioStream
	sampleRate         int32
	frequency          float64
	volume             *int16
	beeping            *bool
	runningSampleIndex uint32 // keeps track of wave position
}

func (sp *Speaker) Init(beeping *bool, volume *int16) {
	sp.sampleRate = 44100
	sp.frequency = 440.0
	sp.volume = volume
	sp.beeping = beeping

	// Request mono, 16-bit, 44.1 kHz
	spec := sdl.AudioSpec{
		Format:   sdl.AudioS16Le, // int16 little-endian
		Channels: 1,
		Freq:     sp.sampleRate,
	}

	cb := sdl.NewAudioStreamCallback(audioCallback)
	sp.stream = sdl.OpenAudioDeviceStream(sdl.AudioDeviceDefaultPlayback, &spec, cb, unsafe.Pointer(sp))
	if sp.stream == nil {
		panic(sdl.GetError())
	}

	// Newly opened device starts paused
	if sdl.AudioStreamDevicePaused(sp.stream) {
		sdl.ResumeAudioStreamDevice(sp.stream)
	} else {
		sdl.PauseAudioStreamDevice(sp.stream)
	}
}

func (sp *Speaker) Close() {
	if sp.stream != nil {
		sdl.DestroyAudioStream(sp.stream)
		sp.stream = nil
	}
}

// Callback shape must match NewAudioStreamCallback!
func audioCallback(userdata unsafe.Pointer, stream *sdl.AudioStream, additional, total int32) {
	const bytesPerSample = 2 // int16 mono
	n := int(additional / bytesPerSample)
	if n <= 0 {
		return
	}

	sp := (*Speaker)(userdata)
	buf := make([]int16, n)

	// compute square wave period in samples
	squareWavePeriod := sp.sampleRate / int32(sp.frequency)
	halfPeriod := squareWavePeriod / 2

	play := true
	if sp.beeping != nil {
		play = *sp.beeping
	}

	for i := 0; i < n; i++ {
		var sample int16
		if play {
			if (sp.runningSampleIndex/uint32(halfPeriod))%2 == 0 {
				sample = *sp.volume
			} else {
				sample = -*sp.volume
			}
			sp.runningSampleIndex++
		} else {
			sample = 0
		}
		buf[i] = sample
	}

	// Queue PCM into the stream
	sdl.PutAudioStreamData(stream, (*uint8)(unsafe.Pointer(&buf[0])), int32(len(buf))*bytesPerSample)
}
