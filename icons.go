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
	p.DrawAlpha(screen, x, y, px, t, 1)
}

// DrawAlpha is Draw with an opacity in [0,1], for fading the icon in and out.
func (p *pressIcon) DrawAlpha(screen *ebiten.Image, x, y, px, t, alpha float64) {
	d := p.pressAmount(t)
	button := buttonUp
	if d == 1 {
		button = buttonDown
	}
	button.DrawAlpha(screen, x, y+buttonTop*px, px, alpha)
	handY := 1 - math.Round(handLift*(1-d))
	p.hand.DrawAlpha(screen, x, y+handY*px, px, alpha)
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
	'x': {255, 255, 255, 255}, // plain white, for tinting with DrawColored
}

// backIcon is a "return" arrow (like ⏎), drawn tinted to match the text
// around it.
var backIcon = &pixelArt{rows: []string{
	".......x",
	"..x....x",
	".xx....x",
	"xxxxxxxx",
	".xx.....",
	"..x.....",
}}

// pixelArt is a small sprite defined as rows of iconPalette characters. Its
// image is built on first draw, once the graphics context exists.
type pixelArt struct {
	rows    []string
	palette map[byte]color.RGBA // nil means iconPalette
	img     *ebiten.Image
}

// Draw draws the sprite with its top-left at device-pixel (x, y), each sprite
// pixel px device pixels wide, using nearest-neighbour filtering to stay crisp.
func (p *pixelArt) Draw(screen *ebiten.Image, x, y, px float64) {
	p.DrawAlpha(screen, x, y, px, 1)
}

// DrawAlpha is Draw with an opacity in [0,1], for fading the sprite in and out.
func (p *pixelArt) DrawAlpha(screen *ebiten.Image, x, y, px, alpha float64) {
	p.draw(screen, x, y, px, color.White, alpha)
}

// DrawColored is Draw with every pixel multiplied by clr, so art drawn in
// plain white ('x') takes on clr.
func (p *pixelArt) DrawColored(screen *ebiten.Image, x, y, px float64, clr color.Color) {
	p.draw(screen, x, y, px, clr, 1)
}

func (p *pixelArt) draw(screen *ebiten.Image, x, y, px float64, clr color.Color, alpha float64) {
	if p.img == nil {
		pal := p.palette
		if pal == nil {
			pal = iconPalette
		}
		rgba := image.NewRGBA(image.Rect(0, 0, len(p.rows[0]), len(p.rows)))
		for py, row := range p.rows {
			for px := 0; px < len(row); px++ {
				if c, ok := pal[row[px]]; ok {
					rgba.SetRGBA(px, py, c)
				}
			}
		}
		p.img = ebiten.NewImageFromImage(rgba)
	}
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Scale(px, px)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	op.ColorScale.ScaleAlpha(float32(alpha))
	screen.DrawImage(p.img, op)
}

// gameIconPalette colors the launcher's per-game icons.
var gameIconPalette = map[byte]color.RGBA{
	'r': {214, 58, 52, 255},   // red
	'y': {255, 212, 47, 255},  // yellow
	'b': {60, 140, 230, 255},  // blue
	'd': {60, 80, 160, 255},   // denim
	'o': {235, 130, 40, 255},  // ginger
	'p': {255, 150, 170, 255}, // pink
	'e': {110, 210, 90, 255},  // cat-eye green
	'k': {20, 14, 30, 255},    // near-black
	'w': {236, 240, 255, 255}, // white
	's': {255, 204, 150, 255}, // skin
	'h': {120, 70, 35, 255},   // hair
	'n': {170, 110, 60, 255},  // wood
	'N': {110, 65, 30, 255},   // wood shadow
	'g': {160, 160, 175, 255}, // steel
	'G': {105, 105, 120, 255}, // steel shadow
}

// Launcher icons, one per game, each listIconSize pixels square.
var (
	bouncyCastleIcon = &pixelArt{palette: gameIconPalette, rows: []string{
		".y........y.",
		"yyy......yyy",
		"rrry.yy.yrrr",
		"rrryyyyyyrrr",
		"rrrrrrrrrrrr",
		"rrrrrrrrrrrr",
		"rrrrbbbbrrrr",
		"rrrbbbbbbrrr",
		"rrrbbbbbbrrr",
		"rrrbbbbbbrrr",
		"yyyyyyyyyyyy",
		"yyyyyyyyyyyy",
	}}
	flappyGuyIcon = &pixelArt{palette: gameIconPalette, rows: []string{
		"....hhhh....",
		"....ssss....",
		"....sksk....",
		"....ssss....",
		"ww..bbbb..ww",
		"www.bbbb.www",
		".wwwbbbbwww.",
		"..wwbbbbww..",
		"....bbbb....",
		"....dddd....",
		"....d..d....",
		"....d..d....",
	}}
	catIcon = &pixelArt{palette: gameIconPalette, rows: []string{
		"o..........o",
		"oo........oo",
		"opo......opo",
		"oooooooooooo",
		"oooooooooooo",
		"ooeeooooeeoo",
		"ooekooooekoo",
		"oooooppooooo",
		"woooowwoooow",
		".oooooooooo.",
		"..oooooooo..",
		"............",
	}}
	hammerIcon = &pixelArt{palette: gameIconPalette, rows: []string{
		"y..........y",
		".gggggggggg.",
		".gwwwwwwwwg.",
		".gggggggggg.",
		".GGGGGGGGGG.",
		".....nN.....",
		".....nN.....",
		".....nN.....",
		".....nN.....",
		".....nN.....",
		".....nN.....",
		".....NN.....",
	}}
)
