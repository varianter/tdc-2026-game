// Package assets holds the Bouncy Castle game's embedded artwork and sound
// effects. Everything here is generated: run gen_castle_assets.go for the
// scenery, gen_anti_shield.go for the anti-shield sprite and gen_sounds.go for
// the audio.
package assets

import _ "embed"

// Scenery.
var (
	//go:embed castle_wall.png
	CastleWall []byte

	//go:embed castle_top.png
	CastleTop []byte

	//go:embed castle_bottom.png
	CastleBottom []byte

	//go:embed castle_turret.png
	CastleTurret []byte
)

// Powerup sprites.
var (
	//go:embed sokker.png
	Sokker []byte

	//go:embed anti_sokker.png
	AntiSokker []byte
)

// Sound effects: 16-bit stereo WAV at 44100 Hz.
var (
	//go:embed bounce.wav
	BounceSound []byte

	//go:embed crush.wav
	CrushSound []byte

	//go:embed powerup.wav
	PowerupSound []byte

	//go:embed antishield.wav
	AntiShieldSound []byte
)
