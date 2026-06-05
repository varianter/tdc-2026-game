package main

import (
	"flag"
	"log"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	gameFlag := flag.String("game", "", "debug: skip the launcher and start a specific game by name")
	flag.Parse()

	var initialScene Scene = NewLauncherScene()
	if *gameFlag != "" {
		for _, entry := range wheelGames {
			if strings.EqualFold(entry.name, *gameFlag) {
				initialScene = entry.newScene()
				break
			}
		}
	}

	game := &Game{currentScene: initialScene}
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSize(426*4, 240*4)
	ebiten.SetWindowTitle("GOTY2026")
	if err := ebiten.RunGameWithOptions(game, &ebiten.RunGameOptions{InitUnfocused: true}); err != nil {
		log.Fatal(err)
	}
}
