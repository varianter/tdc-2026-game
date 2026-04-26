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

type Camera struct {
	x, y float64
}

func (cam *Camera) Follow(player *Player, screenW, screenH int) {
	cam.x = player.x - float64(screenW)/2 + float64(player.sheet.FrameW)/2
	cam.y = player.y + 60 - float64(screenH)/2

	maxCamY := GroundY - 190.0 // HACK: This is dumb but whatever
	if cam.y > maxCamY {
		cam.y = maxCamY
	}
}

type Game struct {
	player *Player
	camera Camera
	assets *Assets
}

func (g *Game) Update() error {
	dt := 1.0 / float64(ebiten.TPS()) // calculate deltatime based on TPS, ~0.0166 at 60 TPS
	err := g.player.Update(dt)
	if err != nil {
		return err
	}
	g.camera.Follow(g.player, ScreenH, ScreenW)
	return nil
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

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{30, 30, 30, 255})

	c := &Canvas{
		screen: screen,
	}
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()

	c.Rect(0, 0, float32(w), float32((h)), color.RGBA{135, 206, 235, 255}) // light blue

	c.TilingGround(g.assets.Sprites["ground"], g.camera.x, g.camera.y, GroundY, 5000)

	frame := g.player.current.CurrentFrame()
	c.DrawImage(frame, g.player.x-g.camera.x, g.player.y-g.camera.y)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenW, ScreenH
}
