package main

import (
	"embed"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

//go:embed assets/chick.png
//go:embed assets/ground.png
var assets embed.FS

type Assets struct {
	Sprites     map[string]*ebiten.Image
	Backgrounds map[string]*ebiten.Image
}

func LoadAssets() *Assets {
	a := &Assets{
		Sprites:     make(map[string]*ebiten.Image),
		Backgrounds: make(map[string]*ebiten.Image),
	}
	a.Sprites["player"], _, _ = ebitenutil.NewImageFromFileSystem(assets, "assets/chick.png")
	a.Sprites["ground"], _, _ = ebitenutil.NewImageFromFileSystem(assets, "assets/ground.png")
	return a
}

type SpriteSheet struct {
	Image   *ebiten.Image
	FrameW  int
	FrameH  int
	Columns int
}

func LoadSpriteSheet(assets *Assets, frameW, frameH int) *SpriteSheet {
	img := assets.Sprites["player"]
	w, _ := img.Bounds().Dx(), img.Bounds().Dy()
	return &SpriteSheet{
		Image:   img,
		FrameW:  frameW,
		FrameH:  frameH,
		Columns: w / frameW,
	}
}

func (s *SpriteSheet) Frame(index int) *ebiten.Image {
	col := index % s.Columns
	row := index / s.Columns
	x := col * s.FrameW
	y := row * s.FrameH
	return s.Image.SubImage(image.Rect(x, y, x+s.FrameW, y+s.FrameH)).(*ebiten.Image)
}

type Animation struct {
	Sheet      *SpriteSheet
	StartFrame int // first frame of this animation row
	FrameCount int // how many frames in the animation
	FPS        float64
	elapsed    float64
}

func (a *Animation) Update(dt float64) {
	a.elapsed += dt
}

func (a *Animation) CurrentFrame() *ebiten.Image {
	frame := int(a.elapsed*a.FPS) % a.FrameCount
	return a.Sheet.Frame(a.StartFrame + frame)
}
