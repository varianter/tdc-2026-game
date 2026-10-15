package tdcgame

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Pixel-art button-action icons, one character per logical pixel ('.' is
// transparent). All are the same finger over a shared arcade button,
// animated in a loop: a quick tap, a double tap, or a press held for about a
// second.
var (
	pointingHand = &PixelArt{Rows: []string{
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
	// "×2", drawn beside the finger on the double tap icon.
	timesTwo = &PixelArt{Rows: []string{
		"....yyy",
		"......y",
		"y.y.yyy",
		".y..y..",
		"y.y.yyy",
	}}

	// 1.2s loop: down for a brief tap.
	TapIcon = &PressIcon{hand: pointingHand, period: 1.2, presses: []press{{0.45, 0.65}}, move: 0.06}
	// 1.6s loop: two quick taps back to back, then a pause.
	DoubleTapIcon = &PressIcon{hand: pointingHand, badge: timesTwo, badgeX: 17, badgeY: 9,
		period: 1.6, presses: []press{{0.45, 0.56}, {0.70, 0.81}}, move: 0.05}
	// Same motion as the tap, but holds the button down for over a second.
	HoldIcon = &PressIcon{hand: pointingHand, period: 2.4, presses: []press{{0.45, 1.65}}, move: 0.06}

	// The button, drawn buttonTop rows down the icon. Pressed, the dome sinks
	// a pixel.
	buttonUp = &PixelArt{Rows: []string{
		".....rrrrrrrrrr.....",
		"..rrwwrrrrrrrrrrrr..",
		".rrrrrrrrrrrrrrrrrr.",
		".dddddddddddddddddd.",
		"gggggggggggggggggggg",
		"gggggggggggggggggggg",
		"GGGGGGGGGGGGGGGGGGGG",
	}}
	buttonDown = &PixelArt{Rows: []string{
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

// press is one press in a PressIcon loop: the hand moves down over
// [down, down+move], stays pressed until up, then lifts over [up, up+move].
type press struct{ down, up float64 }

// PressIcon is a hand pressing the button on a loop of period seconds, with
// an optional badge drawn at icon pixel (badgeX, badgeY).
type PressIcon struct {
	hand           *PixelArt
	badge          *PixelArt
	badgeX, badgeY int
	period         float64
	presses        []press
	move           float64
}

// Width returns the icon's width in icon pixels, badge included.
func (p *PressIcon) Width() float64 {
	w := p.hand.Width()
	if p.badge != nil {
		w = max(w, float64(p.badgeX)+p.badge.Width())
	}
	return w
}

// pressAmount returns how far the hand is down at time t: 0 lifted, 1 pressing.
func (p *PressIcon) pressAmount(t float64) float64 {
	t = math.Mod(t, p.period)
	for _, pr := range p.presses {
		switch {
		case t < pr.down:
			return 0
		case t < pr.down+p.move:
			return (t - pr.down) / p.move
		case t < pr.up:
			return 1
		case t < pr.up+p.move:
			return 1 - (t-pr.up)/p.move
		}
	}
	return 0
}

// Draw draws the icon at time t, top-left at device-pixel (x, y), each icon
// pixel px device pixels wide. The hand sits 1px lower when pressing, to
// follow the sunken dome.
func (p *PressIcon) Draw(screen *ebiten.Image, x, y, px, t float64) {
	p.DrawAlpha(screen, x, y, px, t, 1)
}

// DrawAlpha is Draw with an opacity in [0,1], for fading the icon in and out.
func (p *PressIcon) DrawAlpha(screen *ebiten.Image, x, y, px, t, alpha float64) {
	d := p.pressAmount(t)
	button := buttonUp
	if d == 1 {
		button = buttonDown
	}
	button.DrawAlpha(screen, x, y+buttonTop*px, px, alpha)
	handY := 1 - math.Round(handLift*(1-d))
	p.hand.DrawAlpha(screen, x, y+handY*px, px, alpha)
	if p.badge != nil {
		p.badge.DrawAlpha(screen, x+float64(p.badgeX)*px, y+float64(p.badgeY)*px, px, alpha)
	}
}

// DrawEndHints draws the game-over controls, "[tap] RETURN  [double tap]
// REPLAY", centred on device x cx with the icons' top at device y. s is device
// pixels per logical pixel; the icons are 1.5 logical px per icon pixel and
// the labels 6 logical px tall, as on the champion screen.
func DrawEndHints(screen *ebiten.Image, cx, y, s, t float64, clr color.Color) {
	const iconScale, labelGap, groupGap, labelSize = 1.5, 5.0, 36.0, 6.0
	hints := []struct {
		icon  *PressIcon
		label string
	}{{TapIcon, "RETURN"}, {DoubleTapIcon, "REPLAY"}}

	widths := make([]float64, len(hints))
	total := groupGap * float64(len(hints)-1)
	for i, h := range hints {
		widths[i] = h.icon.Width()*iconScale + labelGap + MeasureWidth(h.label, labelSize*s)/s
		total += widths[i]
	}

	gx := cx/s - total/2
	labelY := y + 11*iconScale*s
	for i, h := range hints {
		h.icon.Draw(screen, gx*s, y, s*iconScale, t)
		WriteAt(screen, h.label, (gx+h.icon.Width()*iconScale+labelGap)*s, labelY, labelSize*s, clr, text.AlignStart, text.AlignCenter)
		gx += widths[i] + groupGap
	}
}

// IconPalette colors the button-action icons, and any PixelArt without its
// own palette.
var IconPalette = map[byte]color.RGBA{
	'o': {70, 40, 25, 255},    // hand outline
	's': {255, 204, 150, 255}, // skin
	'S': {214, 160, 110, 255}, // skin crease
	'r': {214, 58, 52, 255},   // button red
	'w': {242, 242, 242, 255}, // highlight
	'd': {140, 30, 30, 255},   // button skirt
	'g': {110, 110, 110, 255}, // housing
	'G': {70, 70, 70, 255},    // housing shadow
	'y': {255, 212, 47, 255},  // badge yellow
	'x': {255, 255, 255, 255}, // plain white, for tinting with DrawColored
}

// PixelArt is a small sprite defined as rows of palette characters. Its
// image is built on first draw, once the graphics context exists.
type PixelArt struct {
	Rows    []string
	Palette map[byte]color.RGBA // nil means IconPalette
	img     *ebiten.Image
}

// Width and Height return the sprite's size in sprite pixels.
func (p *PixelArt) Width() float64  { return float64(len(p.Rows[0])) }
func (p *PixelArt) Height() float64 { return float64(len(p.Rows)) }

// Draw draws the sprite with its top-left at device-pixel (x, y), each sprite
// pixel px device pixels wide, using nearest-neighbour filtering to stay crisp.
func (p *PixelArt) Draw(screen *ebiten.Image, x, y, px float64) {
	p.DrawAlpha(screen, x, y, px, 1)
}

// DrawAlpha is Draw with an opacity in [0,1], for fading the sprite in and out.
func (p *PixelArt) DrawAlpha(screen *ebiten.Image, x, y, px, alpha float64) {
	p.draw(screen, x, y, px, color.White, alpha)
}

// DrawColored is Draw with every pixel multiplied by clr, so art drawn in
// plain white ('x') takes on clr.
func (p *PixelArt) DrawColored(screen *ebiten.Image, x, y, px float64, clr color.Color) {
	p.draw(screen, x, y, px, clr, 1)
}

func (p *PixelArt) draw(screen *ebiten.Image, x, y, px float64, clr color.Color, alpha float64) {
	if p.img == nil {
		pal := p.Palette
		if pal == nil {
			pal = IconPalette
		}
		rgba := image.NewRGBA(image.Rect(0, 0, len(p.Rows[0]), len(p.Rows)))
		for py, row := range p.Rows {
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
