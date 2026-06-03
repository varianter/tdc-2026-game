package main

import (
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	args := os.Args[1:]

	activeGame := ""
	if len(args) > 0 {
		gameToRun := args[0]
		if len(gameToRun) > 0 {
			activeGame = gameToRun
		}
	}

	game := &Game{activeGame: activeGame}
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSize(426*4, 240*4)
	ebiten.SetWindowTitle("GOTY2026")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
