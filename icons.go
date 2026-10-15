package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Pixel-art button-action icons, one character per logical pixel ('.' is
// transparent). Both are the same finger over a shared arcade button,
// animated in a loop: a quick tap, or a press held for about a second.
var (
	pointingHand = &pixelArt{rows: []string{
		"......ooooooooo.....",
		".....ossssssssso....",
		".....ossssssssso....",
		"...ooossssssssso....",
		"..ossSssssssssso....",
		"..ossSssssssssso....",
		"..ossSssssssssso....",
		"..ossSssssosSsSso...",
		"...oooossSosSsSso...",
		".....ossSoooooooo...",
		".....ossSo..........",
		".....ossSo..........",
		".....ossSo..........",
		".....ossSo..........",
		"......ooo...........",
	}}
	// 1.2s loop: down for a brief tap.
	tapIcon = &pressIcon{hand: pointingHand, period: 1.2, down: 0.45, up: 0.65, move: 0.06}
	// Same motion as the tap, but holds the button down for over a second.
	holdIcon = &pressIcon{hand: pointingHand, period: 2.4, down: 0.45, up: 1.65, move: 0.06}

	// The button, drawn buttonTop rows down the icon. Pressed, the dome sinks
	// a pixel.
	buttonUp = &pixelArt{rows: []string{
		".....rrrrrrrrrr.....",
		"..rrwwrrrrrrrrrrrr..",
		".rrrrrrrrrrrrrrrrrr.",
		".dddddddddddddddddd.",
		"gggggggggggggggggggg",
		"gggggggggggggggggggg",
		"GGGGGGGGGGGGGGGGGGGG",
	}}
	buttonDown = &pixelArt{rows: []string{
		"....................",
		"....rwwrrrrrrrrr....",
		".rrrrrrrrrrrrrrrrrr.",
		".dddddddddddddddddd.",
		"gggggggggggggggggggg",
		"gggggggggggggggggggg",
		"GGGGGGGGGGGGGGGGGGGG",
	}}
)

const (
	buttonTop = 15 // icon row where the button starts
	handLift  = 3  // icon pixels the hand rises off the button between presses
)

// pressIcon is a hand pressing the button on a loop of period seconds: it
// moves down over [down, down+move], stays pressed until up, then lifts over
// [up, up+move].
type pressIcon struct {
	hand                   *pixelArt
	period, down, up, move float64
}

// pressAmount returns how far the hand is down at time t: 0 lifted, 1 pressing.
func (p *pressIcon) pressAmount(t float64) float64 {
	t = math.Mod(t, p.period)
	switch {
	case t < p.down:
		return 0
	case t < p.down+p.move:
		return (t - p.down) / p.move
	case t < p.up:
		return 1
	case t < p.up+p.move:
		return 1 - (t-p.up)/p.move
	}
	return 0
}

// Draw draws the icon at time t, top-left at device-pixel (x, y), each icon
// pixel px device pixels wide. The hand sits 1px lower when pressing, to
// follow the sunken dome.
func (p *pressIcon) Draw(screen *ebiten.Image, x, y, px, t float64) {
	d := p.pressAmount(t)
	button := buttonUp
	if d == 1 {
		button = buttonDown
	}
	button.Draw(screen, x, y+buttonTop*px, px)
	handY := 1 - math.Round(handLift*(1-d))
	p.hand.Draw(screen, x, y+handY*px, px)
}

var iconPalette = map[byte]color.RGBA{
	'o': {70, 40, 25, 255},    // hand outline
	's': {255, 204, 150, 255}, // skin
	'S': {214, 160, 110, 255}, // skin crease
	'r': {214, 58, 52, 255},   // button red
	'w': {242, 242, 242, 255}, // highlight
	'd': {140, 30, 30, 255},   // button skirt
	'g': {110, 110, 110, 255}, // housing
	'G': {70, 70, 70, 255},    // housing shadow
}

// pixelArt is a small sprite defined as rows of iconPalette characters. Its
// image is built on first draw, once the graphics context exists.
type pixelArt struct {
	rows []string
	img  *ebiten.Image
}

// Draw draws the sprite with its top-left at device-pixel (x, y), each sprite
// pixel px device pixels wide, using nearest-neighbour filtering to stay crisp.
func (p *pixelArt) Draw(screen *ebiten.Image, x, y, px float64) {
	if p.img == nil {
		rgba := image.NewRGBA(image.Rect(0, 0, len(p.rows[0]), len(p.rows)))
		for py, row := range p.rows {
			for px := 0; px < len(row); px++ {
				if c, ok := iconPalette[row[px]]; ok {
					rgba.SetRGBA(px, py, c)
				}
			}
		}
		p.img = ebiten.NewImageFromImage(rgba)
	}
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Scale(px, px)
	op.GeoM.Translate(x, y)
	screen.DrawImage(p.img, op)
}
