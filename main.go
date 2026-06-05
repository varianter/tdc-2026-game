package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	game := &Game{currentScene: NewLauncherScene()}
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSize(426*4, 240*4)
	ebiten.SetWindowTitle("GOTY2026")
	if err := ebiten.RunGameWithOptions(game, &ebiten.RunGameOptions{InitUnfocused: true}); err != nil {
		log.Fatal(err)
	}
}
