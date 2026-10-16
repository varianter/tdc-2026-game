// Package assets holds Busy Barista's embedded artwork and sound effects.
// Everything here is generated: run gen_barista.go for the barista sprite and
// gen_sounds.go for the audio.
package assets

import _ "embed"

// Barista is the TDC character's idle row re-dressed as a barista: 10 frames
// of 64×64.
//
//go:embed barista.png
var Barista []byte

// Sound effects: 16-bit stereo WAV at 44100 Hz.
var (
	//go:embed lid.wav
	LidSound []byte

	//go:embed shot.wav
	ShotSound []byte

	//go:embed spill.wav
	SpillSound []byte

	//go:embed miss.wav
	MissSound []byte

	//go:embed levelup.wav
	LevelUpSound []byte

	//go:embed gameover.wav
	GameOverSound []byte
)
