package main

import (
	"bytes"
	"embed"
	"fmt"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
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

type Game struct {
	activeGame string
	f          *tdcgame.GameRunner
}

const (
	ScreenW = 426
	ScreenH = 240
)

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenW, ScreenH
}

func (g *Game) Update() error {
	if g.f != nil {
		return g.f.Update()
	}
	return nil
}

// TODO: Type shenanigans
func createRunningGame(gameName string) tdcgame.TdcGameWithPlayer {
	switch gameName {
	case "tdcrunner":
		{
			return &tdcrunner.TdcRunner{}
		}
	default:
		panic(fmt.Sprintf("Unknown game: %s", gameName))
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.f != nil {
		g.f.Draw(screen)
		return
	}
	if g.activeGame != "" && g.f == nil {
		// TODO: Create new game here
		log.Println("Creating runner game")
		g.f = tdcgame.NewGameFrameworkWithPlayer(assets, createRunningGame(g.activeGame))
		g.f.Draw(screen)
		return
	}

	// Start screen or game select here
	screen.Fill(color.RGBA{30, 30, 30, 255})

	// Draw info
	msg := fmt.Sprintf("Please select a game by launching with an argument %f", ebiten.ActualTPS())
	op := &text.DrawOptions{}
	op.GeoM.Translate(0, 0) // top left of screen
	op.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, msg, &text.GoTextFace{
		Source: mplusFaceSource,
		Size:   8,
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
