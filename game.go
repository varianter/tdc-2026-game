package main

import (
	"bytes"
	"embed"
	"fmt"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"variant.dev/tdcgame/games/tdcrunner"
	"variant.dev/tdcgame/tdcgame"
)

type Canvas struct {
	screen *ebiten.Image
}

func (c *Canvas) DrawImage(img *ebiten.Image, x, y float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	c.screen.DrawImage(img, op)
}

func (c *Canvas) Rect(x, y, w, h float32, clr color.Color) {
	vector.FillRect(c.screen, x, y, w, h, clr, false)
}

//go:embed assets/tdcgjenger.png
//go:embed assets/ground.png
var assets embed.FS

const (
	ScreenW = 426
	ScreenH = 240
)

type Game struct {
	currentScene Scene
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenW, ScreenH
}

func (g *Game) Update() error {
	dt := 1.0 / float64(ebiten.TPS())
	next, err := g.currentScene.Update(dt)
	if err != nil {
		return err
	}
	if next != nil {
		g.currentScene = next
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.currentScene.Draw(screen)
}

// GameRunnerScene wraps a tdcgame.GameRunner as a Scene.
// Pressing Q returns to the launcher.
type GameRunnerScene struct {
	runner *tdcgame.GameRunner
}

func (s *GameRunnerScene) Update(dt float64) (Scene, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		return NewLauncherScene(), nil
	}
	return nil, s.runner.Update()
}

func (s *GameRunnerScene) Draw(screen *ebiten.Image) {
	s.runner.Draw(screen)
}

func createGameFramework(gameName string) *tdcgame.GameRunner {
	switch gameName {
	case "tdcrunner":
		return tdcgame.NewGameFrameworkWithPlayer(assets, &tdcrunner.TdcRunner{})
	default:
		panic(fmt.Sprintf("Unknown game: %s", gameName))
	}
}

func init() {
	wheelGames = append(wheelGames, gameEntry{
		name:  "TDCRUNNER",
		color: color.RGBA{220, 60, 60, 255},
		newScene: func() Scene {
			return &GameRunnerScene{runner: createGameFramework("tdcrunner")}
		},
	})
}

func Write(s *ebiten.Image, msg string, x, y int, size int) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(color.White)
	text.Draw(s, msg, &text.GoTextFace{
		Source: mplusFaceSource,
		Size:   float64(size),
	}, op)
}

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.MPlus1pRegular_ttf))
	if err != nil {
		log.Fatal(err)
	}
	mplusFaceSource = s
}

var mplusFaceSource *text.GoTextFaceSource
