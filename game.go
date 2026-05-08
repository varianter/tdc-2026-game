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

func (c *Canvas) TilingGround(img *ebiten.Image, cameraX, cameraY, worldWidth float64) {
	w := float64(img.Bounds().Dx()) // w is 32 for our ground texture
	startX := math.Floor(cameraX/w) * w

	for x := startX; x < cameraX+ScreenW; x += w {
		c.DrawImage(img, x-cameraX, (GroundY-GroundDrawOffset)-cameraY)
	}
}

type Camera struct {
	x, y float64
}

func (cam *Camera) Follow(player *Player, screenW, screenH int) {
	// camera stays on player
	cam.x = player.x - float64(screenW)/2 + float64(player.sheet.FrameW)/2
	cam.y = -player.y
}

type Game struct {
	player *Player
	camera Camera
	assets *Assets
	level  *Level
}

func (g *Game) Update() error {
	dt := 1.0 / float64(ebiten.TPS()) // calculate deltatime based on TPS, ~0.0166 at 60 TPS
	err := g.player.Update(dt, *g.level)
	if err != nil {
		return err
	}
	g.camera.Follow(g.player, ScreenH, ScreenW)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{30, 30, 30, 255})

	c := &Canvas{
		screen: screen,
	}
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()

	c.Rect(0, 0, float32(w), float32((h)), color.RGBA{135, 206, 235, 255}) // light blue

	c.TilingGround(g.assets.Sprites["ground"], g.camera.x, g.camera.y, 5000)

	g.DrawImage(c, g.player.current.CurrentFrame(), g.player.x, g.player.y)

	for _, gameObject := range g.level.gameObjects {
		// TODO: Hard coded size
		w := float32(64)
		h := float32(32)
		p := gameObject.p
		g.RectXY(c, float32(p.x), float32(p.y), w, h, color.RGBA{27, 130, 0, 255})
	}
}

func (g *Game) DrawImage(c *Canvas, img *ebiten.Image, x float64, y float64) {
	zeroY := GroundY - 32 - y // Zero on grid system is players feet (ish)
	c.DrawImage(img, x-g.camera.x, zeroY-g.camera.y)
}

func (g *Game) RectXY(c *Canvas, x, y, w, h float32, clr color.Color) {
	zeroY := GroundY - 32 - y // Zero on grid system is players feet (ish)
	vector.FillRect(c.screen, x-float32(g.camera.x), zeroY-float32(g.camera.y), w, h, clr, false)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenW, ScreenH
}
