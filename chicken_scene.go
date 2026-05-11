package main

import "image/color"

import "github.com/hajimehoshi/ebiten/v2"

type ChickenScene struct {
	player *Player
	camera Camera
	assets *Assets
}

func NewChickenScene(assets *Assets) *ChickenScene {
	sheet := LoadSpriteSheet(assets, 32, 32)
	return &ChickenScene{
		player: Newplayer(sheet),
		assets: assets,
	}
}

func (s *ChickenScene) Update(dt float64) (Scene, error) {
	if err := s.player.Update(dt); err != nil {
		return nil, err
	}
	s.camera.Follow(s.player, ScreenH, ScreenW)
	return nil, nil
}

func (s *ChickenScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{30, 30, 30, 255})
	c := &Canvas{screen: screen}
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	c.Rect(0, 0, float32(w), float32(h), color.RGBA{135, 206, 235, 255})
	c.TilingGround(s.assets.Sprites["ground"], s.camera.x, s.camera.y, GroundY, 5000)
	frame := s.player.current.CurrentFrame()
	c.DrawImage(frame, s.player.x-s.camera.x, s.player.y-s.camera.y)
}
