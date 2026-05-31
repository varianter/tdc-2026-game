package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	WalkSpeed        = 90.0 // px/sec
	JumpSpeed        = 60
	Gravity          = 700 // px/sec^2
	JumpForce        = 300
	GroundY          = 186
	AirControl       = 1
	AnimIdleFPS      = 5.0
	AnimWalkFPS      = 10.0
	AnimRunFPS       = 14.0
	ScreenW          = 426
	ScreenH          = 240
	GroundDrawOffset = 6.0
	AutoRun          = true
	GameEnd          = 1200 // TODO: Currently unused
)

func main() {
	assets := LoadAssets()
	sheet := LoadSpriteSheet(assets, 64, 64)
	player := Newplayer(sheet)

	game := &Game{
		player: player,
		assets: assets,
		level:  NewLevel(),
	}
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSize(426*4, 240*4)
	ebiten.SetWindowTitle("GOTY2026")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
