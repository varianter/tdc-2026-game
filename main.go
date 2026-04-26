package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	WalkSpeed        = 100.0 // px/sec
	JumpSpeed        = 60
	Gravity          = 600  // px/sec^2
	JumpForce        = -250 // negative means up
	GroundY          = 148  // 0,0 is top left
	AirControl       = 0.5
	AnimIdleFPS      = 5.0
	AnimWalkFPS      = 12.0
	AnimRunFPS       = 14.0
	ScreenW          = 426
	ScreenH          = 240
	GroundDrawOffset = 6.0
)

func main() {
	assets := LoadAssets()
	sheet := LoadSpriteSheet(assets, 32, 32)
	player := Newplayer(sheet)

	game := &Game{
		player: player,
		assets: assets,
	}
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSize(426*4, 240*4)
	ebiten.SetWindowTitle("GOTY2026")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
