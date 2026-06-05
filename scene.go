package main

import "github.com/hajimehoshi/ebiten/v2"

type Scene interface {
	Update(dt float64) (Scene, error)
	Draw(screen *ebiten.Image)
}
