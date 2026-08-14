package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"variant.dev/tdcgame/tdcgame"
)

type launcherState int

const (
	launcherBrowsing launcherState = iota
	launcherFading
)

const holdToStartDuration = 0.6

// holdIndicatorDelay is the grace period before the hold-progress bar appears.
// Without it, every quick click while browsing flashes the bar on and off.
const holdIndicatorDelay = 0.15

type gameEntry struct {
	name     string
	color    color.RGBA
	newScene func() Scene
	key      ebiten.Key
}

var launcherGames []gameEntry

type LauncherScene struct {
	selected  int
	state     launcherState
	holdTimer float64
	fadeAlpha float64
	viewport  *ebiten.Image
}

func NewLauncherScene() *LauncherScene {
	return &LauncherScene{}
}

func (l *LauncherScene) Update(dt float64) (Scene, error) {
	for _, k := range inpututil.AppendJustPressedKeys(nil) {
		for _, g := range launcherGames {
			if g.key != ebiten.KeyMax && g.key == k {
				return g.newScene(), nil
			}
		}
	}

	switch l.state {
	case launcherBrowsing:
		if ebiten.IsKeyPressed(ebiten.KeyEnter) {
			l.holdTimer += dt
			if l.holdTimer >= holdToStartDuration {
				l.state = launcherFading
			}
		} else {
			if inpututil.IsKeyJustReleased(ebiten.KeyEnter) && l.holdTimer <= holdIndicatorDelay {
				// Released before the hold bar even appeared: treat as a click, select next.
				// Releasing after the bar has shown just aborts the hold and stays put.
				l.selected = (l.selected + 1) % len(launcherGames)
			}
			l.holdTimer = 0
		}
	case launcherFading:
		l.fadeAlpha += dt * 255
		if l.fadeAlpha >= 255 {
			entry := launcherGames[l.selected]
			if entry.newScene != nil {
				return entry.newScene(), nil
			}
		}
	}
	return nil, nil
}

func (l *LauncherScene) Draw(screen *ebiten.Image) {
	scale := float64(screen.Bounds().Dx()) / ScreenW
	if l.viewport == nil {
		l.viewport = ebiten.NewImage(ScreenW, ScreenH)
	}
	vp := l.viewport

	// ── Non-text visuals drawn into the low-res viewport, then upscaled ────
	l.drawBackground(vp)
	l.drawGameList(vp)
	l.drawHoldIndicator(vp)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(vp, op)

	// ── Text drawn at device resolution so it stays crisp ──────────────────
	l.drawText(screen, scale)

	// Fade-to-black overlay (covers text too so the whole launcher fades)
	if l.fadeAlpha > 0 {
		alpha := uint8(math.Min(l.fadeAlpha, 255))
		b := screen.Bounds()
		vector.FillRect(screen, 0, 0, float32(b.Dx()), float32(b.Dy()), color.RGBA{0, 0, 0, alpha}, false)
	}
}

// listPanelW is the width reserved for the game list; the remainder of the
// screen is reserved for a future scoreboard panel.
const listPanelW = 260

// Shared row layout, used both when drawing the low-res list visuals and
// when drawing the crisp, device-resolution text on top of them.
const (
	listRowH     = 26
	listStartY   = 40
	listStartX   = 16
	listSwatch   = 10
	listFontSize = 12
)

func (l *LauncherScene) drawBackground(vp *ebiten.Image) {
	vp.Fill(color.RGBA{12, 12, 24, 255})
}

func (l *LauncherScene) drawGameList(vp *ebiten.Image) {
	for i, game := range launcherGames {
		y := listStartY + i*listRowH
		isSelected := i == l.selected

		clr := game.color
		if isSelected {
			clr = brightenColor(clr, 40)
			vector.FillRect(vp, float32(listStartX-6), float32(y-4), float32(listPanelW-listStartX), listRowH-4, color.RGBA{clr.R, clr.G, clr.B, 45}, false)
			vector.StrokeRect(vp, float32(listStartX-6), float32(y-4), float32(listPanelW-listStartX), listRowH-4, 1.5, clr, false)
		} else {
			clr = dimColor(clr, 0.45)
		}

		vector.FillRect(vp, float32(listStartX), float32(y+3), listSwatch, listSwatch, clr, false)
	}
}

func (l *LauncherScene) drawHoldIndicator(vp *ebiten.Image) {
	if l.holdTimer <= holdIndicatorDelay || len(launcherGames) == 0 {
		return
	}

	const (
		barX = 16
		barW = listPanelW - 32
		barH = 6
	)
	barY := float32(listStartY + len(launcherGames)*listRowH + 8)

	progress := (l.holdTimer - holdIndicatorDelay) / (holdToStartDuration - holdIndicatorDelay)
	if progress > 1 {
		progress = 1
	}

	vector.StrokeRect(vp, barX, barY, barW, barH, 1, color.RGBA{200, 200, 220, 255}, false)
	fillW := float32(progress) * (barW - 2)
	if fillW > 0 {
		vector.FillRect(vp, barX+1, barY+1, fillW, barH-2, launcherGames[l.selected].color, false)
	}
}

// drawText draws every label at device-pixel resolution (screen, not vp) so
// it stays crisp instead of being upscaled with the rest of the pixel art.
func (l *LauncherScene) drawText(screen *ebiten.Image, scale float64) {
	tdcgame.WriteCentered(screen, "SELECT YOUR GAME", int(10*scale), int(14*scale))

	for i, game := range launcherGames {
		y := listStartY + i*listRowH
		x := int(float64(listStartX+listSwatch+8) * scale)
		tdcgame.Write(screen, game.name, x, int(float64(y-2)*scale), int(float64(listFontSize)*scale))
	}
}

func brightenColor(c color.RGBA, d int) color.RGBA {
	clamp := func(v int) uint8 {
		if v > 255 {
			return 255
		}
		return uint8(v)
	}
	return color.RGBA{clamp(int(c.R) + d), clamp(int(c.G) + d), clamp(int(c.B) + d), c.A}
}

func dimColor(c color.RGBA, f float64) color.RGBA {
	return color.RGBA{
		uint8(float64(c.R) * f),
		uint8(float64(c.G) * f),
		uint8(float64(c.B) * f),
		c.A,
	}
}
