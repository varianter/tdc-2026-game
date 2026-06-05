package tdcgame

import (
	"embed"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Assets struct {
	Sprites     map[string]*ebiten.Image
	Backgrounds map[string]*ebiten.Image
}

func LoadAssets(assets embed.FS) *Assets {
	a := &Assets{
		Sprites:     make(map[string]*ebiten.Image),
		Backgrounds: make(map[string]*ebiten.Image),
	}
	a.Sprites["player"], _, _ = ebitenutil.NewImageFromFileSystem(assets, "assets/tdcgjenger.png")
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
	Frames     []int // frames to use for this animation
	FPS        float64
	elapsed    float64
}

func NewAnimation(sheet *SpriteSheet, startFrame, frameCount int, fps float64) *Animation {
	frames := make([]int, frameCount)
	for i := 0; i < frameCount; i++ {
		frames[i] = startFrame + i
	}
	return &Animation{
		Sheet:      sheet,
		FPS:        fps,
		Frames:     frames,
	}
}

func (a *Animation) Update(dt float64) {
	a.elapsed += dt
}

func (a *Animation) CurrentFrame() *ebiten.Image {
	frame := int(a.elapsed*a.FPS) % len(a.Frames)
	return a.Sheet.Frame(a.Frames[frame])
}
