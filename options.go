package main

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"variant.dev/tdcgame/tdcgame"
)

// OptionsScene is a one-button settings screen, reached with the O hotkey on
// the launcher. It follows the same input grammar as the launcher: tap the
// button to move between rows, hold it to activate the focused row (toggle a
// CRT mode, or Back to return).
type OptionsScene struct {
	prev      *LauncherScene
	focus     int
	holdTimer float64
	consumed  bool // the current hold already fired; wait for release before firing again
	viewport  *ebiten.Image
}

func NewOptionsScene(prev *LauncherScene) *OptionsScene {
	return &OptionsScene{prev: prev}
}

const (
	optSubtle = iota
	optFull
	optBack
	optRowCount
)

// Options list layout (vp-space units).
const (
	optStartY = logoHeaderH + 44
	optRowH   = 26
	optPillX  = listStartX - 6
	optPillW  = 220
	optPillH  = 20
	optBoxSz  = 9
)

func (o *OptionsScene) Update(dt float64) (Scene, error) {
	if ebiten.IsKeyPressed(ebiten.KeyEnter) {
		o.holdTimer += dt
		if !o.consumed && o.holdTimer >= holdToStartDuration {
			o.consumed = true
			switch o.focus {
			case optSubtle:
				currentCRT = toggleCRT(currentCRT, crtSubtle)
			case optFull:
				currentCRT = toggleCRT(currentCRT, crtFull)
			case optBack:
				// The button is still held; disarm the launcher so this hold
				// doesn't carry over into launching a game.
				o.prev.holdArmed = false
				return o.prev, nil
			}
		}
	} else {
		if !o.consumed && inpututil.IsKeyJustReleased(ebiten.KeyEnter) && o.holdTimer <= holdIndicatorDelay {
			o.focus = (o.focus + 1) % optRowCount
		}
		o.holdTimer = 0
		o.consumed = false
	}
	return nil, nil
}

func (o *OptionsScene) Draw(screen *ebiten.Image) {
	scale := float64(screen.Bounds().Dx()) / ScreenW
	if o.viewport == nil {
		o.viewport = ebiten.NewImage(ScreenW, ScreenH)
	}
	vp := o.viewport
	o.prev.drawBackground(vp)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(vp, op)

	o.prev.drawLogo(screen, scale)

	tdcgame.WriteAt(screen, "CRT EFFECT", float64(listStartX)*scale, float64(logoHeaderH+18)*scale,
		10*scale, whiteTextColor, text.AlignStart, text.AlignStart)

	labels := [optRowCount]string{"Subtle", "Full", "Back"}
	for i := 0; i < optRowCount; i++ {
		y := float64(optStartY + i*optRowH)
		o.drawRow(screen, scale, i, y, labels[i])
	}

	fs := float64(footerFontSize) * scale
	fy := float64(ScreenH-footerBottomPad) * scale
	tdcgame.WriteAt(screen, "PRESS TO MOVE    HOLD TO SELECT", float64(listStartX)*scale, fy,
		fs, footerLabelColor, text.AlignStart, text.AlignCenter)
}

func (o *OptionsScene) drawRow(screen *ebiten.Image, scale float64, i int, y float64, label string) {
	s := float32(scale)
	focused := i == o.focus

	px, pw, ph := float32(optPillX)*s, float32(optPillW)*s, float32(optPillH)*s
	py := float32(y) * s
	midY := (y + optPillH/2) * scale

	// drawContent paints the row's checkbox + label in one color, so it can be
	// drawn once in the resting color and again (clipped to the hold fill) in
	// the inverted color as the fill sweeps across.
	drawContent := func(dst *ebiten.Image, clr color.Color) {
		textX := float64(listStartX + 4)
		if i == optSubtle || i == optFull {
			checked := (i == optSubtle && currentCRT == crtSubtle) || (i == optFull && currentCRT == crtFull)
			bx := float32(textX) * s
			by := float32(midY) - float32(optBoxSz)*s/2
			box := float32(optBoxSz) * s
			strokeRoundedRect(dst, bx, by, box, box, 2*s, s, clr)
			if checked {
				inset := 2 * s
				// On an unfocused row the check keeps the accent color so it
				// still reads as active; on the focused row it inverts with
				// the text as the fill passes over it.
				fillClr := clr
				if !focused {
					fillClr = selectedRowColor
				}
				fillRoundedRect(dst, bx+inset, by+inset, box-2*inset, box-2*inset, 1*s, fillClr)
			}
			textX += optBoxSz + 6
		}
		tdcgame.WriteAt(dst, label, textX*scale, midY, 9*scale, clr, text.AlignStart, text.AlignCenter)
	}

	if !focused {
		strokeRoundedRect(screen, px, py, pw, ph, ph/2, s, rowBorderColor)
		drawContent(screen, whiteTextColor)
		return
	}

	// The focused row rests as an outline; holding fills it with the accent
	// color left→right (same as the launcher), landing on the solid pill as the
	// action fires. Text/checkbox flip to dark over the fill for contrast.
	strokeRoundedRect(screen, px, py, pw, ph, ph/2, selectedRowStrokeW*s, selectedRowColor)
	drawContent(screen, whiteTextColor)
	if fillW := float32(holdFill(o.holdTimer)) * pw; fillW > 0 {
		clip := image.Rect(int(px), int(py), int(px+fillW), int(py+ph+1))
		dst := screen.SubImage(clip).(*ebiten.Image)
		fillRoundedRect(dst, px, py, pw, ph, ph/2, selectedRowColor)
		drawContent(dst, headerTextColor)
	}
}
