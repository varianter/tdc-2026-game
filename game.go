package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
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

func (c *Canvas) TilingGround(img *ebiten.Image, cameraX, cameraY, groundY float64, worldWidth float64) {
	w := float64(img.Bounds().Dx())
	startX := math.Floor(cameraX/w) * w
	for x := startX; x < cameraX+ScreenW; x += w {
		c.DrawImage(img, x-cameraX, (groundY-GroundDrawOffset)-cameraY)
	}
}

type Camera struct {
	x, y float64
}

func (cam *Camera) Follow(player *Player, screenW, screenH int) {
	cam.x = player.x - float64(screenW)/2 + float64(player.sheet.FrameW)/2
	cam.y = player.y + 60 - float64(screenH)/2

	maxCamY := GroundY - 190.0
	if cam.y > maxCamY {
		cam.y = maxCamY
	}
}

type Game struct {
	currentScene Scene
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

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenW, ScreenH
}
