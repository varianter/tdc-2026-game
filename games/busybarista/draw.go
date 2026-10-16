package busybarista

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"variant.dev/tdcgame/tdcgame"
)

var (
	wallClr     = color.RGBA{62, 41, 35, 255}
	wainscotClr = color.RGBA{88, 57, 40, 255}
	woodClr     = color.RGBA{126, 82, 52, 255}
	woodDark    = color.RGBA{98, 62, 40, 255}
	woodLight   = color.RGBA{170, 120, 80, 255}
	beltClr     = color.RGBA{38, 36, 42, 255}
	beltDash    = color.RGBA{70, 68, 76, 255}
	chrome      = color.RGBA{176, 180, 190, 255}
	chromeDark  = color.RGBA{110, 114, 126, 255}
	chromeLight = color.RGBA{226, 230, 236, 255}
	coffeeClr   = color.RGBA{92, 52, 30, 255}
	streamClr   = color.RGBA{150, 90, 48, 255}
	cremaClr    = color.RGBA{196, 140, 80, 255}
	glassEdge   = color.RGBA{200, 222, 232, 255}
	glassFill   = color.NRGBA{60, 70, 80, 90}
	lidClr      = color.RGBA{238, 232, 220, 255}
	lineClr     = color.RGBA{255, 210, 80, 255}
	outlineClr  = color.RGBA{24, 18, 20, 255}
	ceramicClr  = color.RGBA{240, 238, 232, 255}
)

func rect(dst *ebiten.Image, x, y, w, h float64, clr color.Color) {
	vector.FillRect(dst, float32(math.Round(x)), float32(math.Round(y)), float32(math.Round(w)), float32(math.Round(h)), clr, false)
}

// cupScratch is reused to draw each cup before blitting it, so falling cups
// can be rotated as one piece. Cups are drawn bottom-centred at cupOriginX,Y.
var cupScratch *ebiten.Image

const (
	cupScratchW, cupScratchH = 34, 50
	cupOriginX, cupOriginY   = 17.0, 47.0
)

// CustomDraw renders the scene into the 426×240 viewport, scales it up, then
// draws text on top at device resolution so it stays crisp.
func (g *BusyBarista) CustomDraw(screen *ebiten.Image) {
	scale := float64(screen.Bounds().Dx()) / tdcgame.ScreenW
	if g.viewport == nil {
		g.viewport = ebiten.NewImage(tdcgame.ScreenW, tdcgame.ScreenH)
		cupScratch = ebiten.NewImage(cupScratchW, cupScratchH)
	}
	vp := g.viewport
	elapsed := time.Since(g.createdAt).Seconds()

	if g.state == stateReady && len(g.cups) > 0 {
		// Attract mode: the waiting cup fills and empties.
		g.cups[0].fill = 0.45 + 0.4*math.Sin(elapsed*1.6)
	}

	g.drawBackdrop(vp)
	g.drawBarista(vp)
	g.drawMachine(vp)
	g.drawGauge(vp)
	g.drawCounter(vp)
	g.drawCups(vp)
	g.drawPress(vp)
	g.drawRings(vp)
	g.drawParticles(vp)
	g.drawQueue(vp)
	g.drawHUDShapes(vp)

	switch g.state {
	case stateReady:
		rect(vp, 0, 0, tdcgame.ScreenW, tdcgame.ScreenH, color.RGBA{0, 0, 0, 110})
		panel(vp, 53, 32, 320, 146)
		drawTitleCups(vp, elapsed)
	case stateGameOver:
		rect(vp, 0, 0, tdcgame.ScreenW, tdcgame.ScreenH, color.RGBA{0, 0, 0, 90})
		panel(vp, 83, 36, 260, 150)
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	if g.shake > 0 {
		k := g.shake / 0.25
		op.GeoM.Translate(math.Sin(elapsed*90)*3*k*scale, math.Cos(elapsed*70)*1.5*k*scale)
	}
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(vp, op)

	// ── Text at device resolution ─────────────────────────────────────────
	if g.state == statePlaying {
		// Scene labels would show through the title and score panels.
		write(screen, "TDC", 170, 70, 6, chromeDark, text.AlignCenter, scale)
		g.drawChalkboardText(screen, scale)
	}
	write(screen, "BUSY BARISTA", 140, 196, 7, color.RGBA{70, 44, 28, 255}, text.AlignCenter, scale)
	write(screen, "TDC 2026", 140, 206, 5, color.RGBA{110, 70, 44, 255}, text.AlignCenter, scale)

	switch g.state {
	case stateReady:
		g.drawTitleText(screen, scale, elapsed)
	case statePlaying:
		g.drawHUDText(screen, scale)
		g.drawPopups(screen, scale)
		g.drawBanner(screen, scale)
	case stateGameOver:
		g.drawPopups(screen, scale)
		cx := float64(tdcgame.ScreenW / 2)
		write(screen, "GAME OVER", cx, 46, 12, lineClr, text.AlignCenter, scale)
		write(screen, fmt.Sprintf("SCORE: %d", g.sc.score), cx, 70, 14, color.White, text.AlignCenter, scale)
		write(screen, fmt.Sprintf("CUPS SERVED  %d", g.served), cx, 96, 7, color.RGBA{230, 210, 190, 255}, text.AlignCenter, scale)
		write(screen, fmt.Sprintf("BEST COMBO   %d", g.sc.maxCombo), cx, 108, 7, color.RGBA{230, 210, 190, 255}, text.AlignCenter, scale)
		write(screen, fmt.Sprintf("PERFECTS     %d", g.sc.counts[tierPerfect]), cx, 120, 7, color.RGBA{230, 210, 190, 255}, text.AlignCenter, scale)
		write(screen, "REACHED: "+g.lvl().name, cx, 134, 6, color.RGBA{180, 160, 140, 255}, text.AlignCenter, scale)
		if g.gameOverDelay <= 0 {
			tdcgame.DrawEndHints(screen, cx*scale, 152*scale, scale, elapsed, color.White)
		}
	}
}

// — scenery ———————————————————————————————————————————————————————————————

func (g *BusyBarista) drawBackdrop(vp *ebiten.Image) {
	vp.Fill(wallClr)
	// faint wallpaper stripes
	for x := 0.0; x < tdcgame.ScreenW; x += 16 {
		rect(vp, x, 0, 6, 120, color.RGBA{70, 47, 40, 255})
	}
	// wainscot panelling
	rect(vp, 0, 118, tdcgame.ScreenW, 2, woodLight)
	rect(vp, 0, 120, tdcgame.ScreenW, 120, wainscotClr)
	for x := 4.0; x < tdcgame.ScreenW; x += 26 {
		rect(vp, x, 126, 20, 26, color.RGBA{80, 51, 36, 255})
	}

	// shelf with mugs and jars
	rect(vp, 8, 54, 112, 3, woodLight)
	rect(vp, 8, 57, 112, 2, woodDark)
	mugs := []color.RGBA{{210, 80, 70, 255}, {236, 232, 220, 255}, {80, 140, 190, 255}, {236, 232, 220, 255}, {230, 180, 60, 255}}
	for i, m := range mugs {
		x := 14 + float64(i)*20
		rect(vp, x, 45, 9, 9, m)
		rect(vp, x+9, 47, 2, 5, m)
	}
	// coffee bean jars
	for i := range 3 {
		x := 18 + float64(i)*32
		rect(vp, x, 24, 14, 18, color.NRGBA{180, 200, 210, 120})
		rect(vp, x+1, 30, 12, 12, color.RGBA{80, 44, 26, 255})
		rect(vp, x-1, 22, 16, 3, woodDark)
	}
	rect(vp, 8, 42, 112, 2, woodLight)

	// chalkboard menu
	rect(vp, 328, 36, 90, 80, woodDark)
	rect(vp, 331, 39, 84, 74, color.RGBA{30, 40, 36, 255})

	// pendant lamps with a warm glow
	for _, lx := range []float64{128, 304} {
		rect(vp, lx, 0, 1, 14, color.RGBA{20, 16, 16, 255})
		vector.FillCircle(vp, float32(lx), 22, 34, color.NRGBA{255, 190, 110, 14}, true)
		vector.FillCircle(vp, float32(lx), 22, 18, color.NRGBA{255, 200, 120, 18}, true)
		rect(vp, lx-6, 14, 13, 6, color.RGBA{40, 60, 52, 255})
		rect(vp, lx-3, 20, 7, 2, color.RGBA{255, 230, 170, 255})
	}
}

func (g *BusyBarista) drawChalkboardText(screen *ebiten.Image, scale float64) {
	chalk := color.RGBA{230, 230, 220, 255}
	dim := color.RGBA{170, 180, 170, 255}
	write(screen, "MENU", 373, 43, 7, chalk, text.AlignCenter, scale)
	items := []struct{ name, price string }{
		{"LATTE", "42"}, {"FLAT WHITE", "45"}, {"ESPRESSO", "30"}, {"BUG FIX", "?"}, {"OAT DEPLOY", "48"},
	}
	for i, it := range items {
		y := 58 + float64(i)*10
		write(screen, it.name, 336, y, 4.5, dim, text.AlignStart, scale)
		write(screen, it.price, 410, y, 4.5, chalk, text.AlignEnd, scale)
	}
}

func (g *BusyBarista) drawBarista(vp *ebiten.Image) {
	const x, feet, sc = 92.0, 184.0, 1.4
	yOff := 0.0
	if g.hop > 0 {
		yOff = -6 * math.Sin(math.Pi*(1-g.hop/0.22))
	}
	frame := g.baristaSheet.Frame(int(g.animClock*6) % 10)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(sc, sc)
	op.GeoM.Translate(x-32*sc, feet+yOff-64*sc)
	vp.DrawImage(frame, op)
}

// drawMachine draws the espresso machine as an arch over the belt.
func (g *BusyBarista) drawMachine(vp *ebiten.Image) {
	const x, y, w, h = 146.0, 62.0, 48.0, 50.0
	// side legs down to the belt
	rect(vp, x, y+h, 6, beltY-y-h, chromeDark)
	rect(vp, x+w-6, y+h, 6, beltY-y-h, chromeDark)
	// body
	rect(vp, x-1, y-1, w+2, h+2, outlineClr)
	rect(vp, x, y, w, h, chrome)
	rect(vp, x, y, w, 6, chromeDark)
	rect(vp, x+3, y+8, 2, h-12, chromeLight)
	rect(vp, x+w-6, y+8, 2, h-12, chromeLight)
	// cups warming on top
	for i := range 3 {
		rect(vp, x+8+float64(i)*12, y-6, 7, 5, ceramicClr)
	}
	// pressure dial
	vector.FillCircle(vp, float32(x+12), float32(y+24), 5, outlineClr, false)
	vector.FillCircle(vp, float32(x+12), float32(y+24), 4, ceramicClr, false)
	ang := -2.2 + 0.4*math.Sin(g.animClock*2)
	if g.pouringCup() != nil {
		ang = 0.6 + 0.2*math.Sin(g.animClock*30)
	}
	vector.StrokeLine(vp, float32(x+12), float32(y+24), float32(x+12+3.5*math.Cos(ang)), float32(y+24+3.5*math.Sin(ang)), 1, color.RGBA{200, 40, 40, 255}, false)
	// group head and nozzle
	rect(vp, spoutX-8, y+h, 16, 4, chromeDark)
	rect(vp, spoutX-3, y+h+4, 6, 4, outlineClr)
	// steam wand
	rect(vp, x+w-12, y+h, 2, 14, chromeLight)

	// pour stream
	if c := g.pouringCup(); c != nil {
		_, ch := c.kind.size()
		surface := beltY - 1 - (ch-2)*c.fill
		top := nozzleY + 8
		rect(vp, spoutX-1, top, 3, surface-top, streamClr)
		for sy := top + math.Mod(g.animClock*140, 5); sy < surface; sy += 5 {
			rect(vp, spoutX, sy, 1, 2, cremaClr)
		}
	}
}

// gaugeCup is the cup the fill gauge reads: the one at or heading for the
// spout.
func (g *BusyBarista) gaugeCup() *cup {
	for _, c := range g.cups {
		if c.kind != cupEspresso && (c.phase == phaseIncoming || c.phase == phaseWaiting || c.phase == phasePouring) {
			return c
		}
	}
	return nil
}

// drawGauge draws a big fill meter beside the machine mirroring the cup at
// the spout, with the judgement zones around the line.
func (g *BusyBarista) drawGauge(vp *ebiten.Image) {
	const x, top, w, h = 202.0, 62.0, 10.0, 90.0
	rect(vp, x-1, top-1, w+2, h+2, outlineClr)
	rect(vp, x, top, w, h, color.RGBA{26, 22, 24, 255})
	c := g.gaugeCup()
	if c == nil || c.fillDur == 0 {
		return
	}
	at := func(frac float64) float64 { return top + h - h*frac }
	rate := c.lineFrac / c.fillDur
	zone := func(win float64, clr color.RGBA) {
		lo := math.Max(0, c.lineFrac-win*rate)
		hi := math.Min(1, c.lineFrac+win*rate)
		rect(vp, x, at(hi), w, at(lo)-at(hi), clr)
	}
	zone(holdWindows[2], color.RGBA{50, 80, 60, 255})
	zone(holdWindows[1], color.RGBA{70, 120, 80, 255})
	zone(holdWindows[0], color.RGBA{150, 130, 60, 255})
	if c.fill > 0 {
		rect(vp, x+2, at(c.fill), w-4, h*c.fill, coffeeClr)
		rect(vp, x+2, at(c.fill), w-4, 1, cremaClr)
	}
	rect(vp, x-3, at(c.lineFrac), w+6, 1, lineClr)
}

func (g *BusyBarista) drawCounter(vp *ebiten.Image) {
	// belt
	rect(vp, 0, beltY, counterEndX, 5, beltClr)
	for x := -8 + g.beltOffset; x < counterEndX-4; x += 8 {
		rect(vp, x, beltY+1, 3, 1, beltDash)
	}
	vector.FillCircle(vp, float32(counterEndX-3), float32(beltY+2.5), 3, chromeDark, false)
	// counter front
	rect(vp, 0, beltY+5, counterEndX, 2, woodLight)
	rect(vp, 0, beltY+7, counterEndX, tdcgame.ScreenH-beltY-7, woodClr)
	for x := 0.0; x < counterEndX; x += 24 {
		rect(vp, x, beltY+7, 1, tdcgame.ScreenH-beltY-7, woodDark)
	}
	rect(vp, counterEndX-2, beltY+5, 2, tdcgame.ScreenH-beltY-5, woodDark)
	// sign
	rect(vp, 78, 189, 124, 26, woodDark)
	rect(vp, 80, 191, 120, 22, color.RGBA{232, 214, 180, 255})
	// floor beyond the counter
	rect(vp, counterEndX, 234, tdcgame.ScreenW-counterEndX, 6, color.RGBA{50, 34, 28, 255})
}

// — cups ——————————————————————————————————————————————————————————————————

func (g *BusyBarista) drawCups(vp *ebiten.Image) {
	for _, c := range g.cups {
		x := c.x(g.now)
		if x < -16 || g.state != stateReady && g.now < c.t0 {
			continue
		}
		cupScratch.Clear()
		drawCupShape(cupScratch, c)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-cupOriginX, -cupOriginY)
		op.GeoM.Rotate(c.tilt)
		op.GeoM.Translate(math.Round(x), beltY+c.fallY)
		vp.DrawImage(cupScratch, op)
	}
}

// drawCupShape draws a cup bottom-centred at the scratch origin.
func drawCupShape(dst *ebiten.Image, c *cup) {
	w, h := c.kind.size()
	cx, bottom := cupOriginX, cupOriginY

	if c.kind == cupEspresso {
		x := cx - w/2
		rect(dst, x-1, bottom-h-1, w+2, h+2, outlineClr)
		rect(dst, x, bottom-h, w, h, ceramicClr)
		rect(dst, x+w+1, bottom-h+2, 2, 4, ceramicClr) // handle
		rect(dst, x-3, bottom-1, w+6, 2, ceramicClr)   // saucer
		if c.fill > 0 {
			rect(dst, x+1, bottom-h, w-2, 2, coffeeClr)
			rect(dst, x+1, bottom-h, w-2, 1, cremaClr)
		}
		return
	}

	// Clear takeaway cup, slightly narrower at the base, so the coffee level
	// is visible.
	for i := 0.0; i < h; i++ {
		rw := math.Round(w - 3*i/(h-1))
		y := bottom - 1 - i // i counts up from the bottom row
		x := math.Round(cx - rw/2)
		lvl := i / (h - 1)
		switch {
		case lvl < c.fill && lvl >= c.fill-1/(h-1):
			rect(dst, x+1, y, rw-2, 1, cremaClr)
		case lvl < c.fill:
			rect(dst, x+1, y, rw-2, 1, coffeeClr)
		default:
			rect(dst, x+1, y, rw-2, 1, glassFill)
		}
		rect(dst, x, y, 1, 1, glassEdge)
		rect(dst, x+rw-1, y, 1, 1, glassEdge)
	}
	rect(dst, cx-(w-3)/2, bottom-1, w-3, 1, glassEdge)

	if c.spilled {
		// coffee running down the outside
		rect(dst, cx-w/2-1, bottom-h, 1, h*0.7, coffeeClr)
		rect(dst, cx+w/2, bottom-h, 1, h*0.4, coffeeClr)
		rect(dst, cx-w/2-3, bottom-1, w+6, 1, coffeeClr)
	}

	if !c.lidded && c.lineFrac > 0 {
		ly := math.Round(bottom - 1 - c.lineFrac*(h-1))
		for x := cx - w/2 + 1; x < cx+w/2-1; x += 2 {
			rect(dst, x, ly, 1, 1, color.NRGBA{255, 210, 80, 170})
		}
		rect(dst, cx-w/2-2, ly, 2, 1, lineClr)
		rect(dst, cx+w/2, ly, 2, 1, lineClr)
	}

	if c.lidded {
		rect(dst, cx-w/2-1, bottom-h-2, w+2, 2, lidClr)
		rect(dst, cx-w/2+1, bottom-h-4, w-2, 2, lidClr)
		rect(dst, cx-1, bottom-h-5, 3, 1, lidClr)
		rect(dst, cx-w/2-1, bottom-h-1, w+2, 1, color.RGBA{200, 192, 176, 255})
	}
}

// drawPress draws the lid press; it slams down to the cup on a stamp.
func (g *BusyBarista) drawPress(vp *ebiten.Image) {
	drop := 0.0
	if g.pressT < 0.16 {
		k := g.pressT / 0.16
		drop = g.pressDrop * math.Sin(math.Pi*k)
	}
	const restY = pressRestY
	bottom := restY + drop
	// gantry
	rect(vp, pressX-26, 36, 52, 4, chromeDark)
	rect(vp, pressX-1, 40, 3, bottom-12-40, chrome)
	// piston head with hazard stripes
	rect(vp, pressX-10, bottom-13, 20, 13, outlineClr)
	rect(vp, pressX-9, bottom-12, 18, 8, color.RGBA{230, 190, 40, 255})
	for i := 0.0; i < 18; i += 6 {
		rect(vp, pressX-9+i, bottom-12, 3, 8, color.RGBA{40, 36, 30, 255})
	}
	rect(vp, pressX-9, bottom-4, 18, 3, chrome)
	// a fresh lid waiting in the jaws
	if g.pressT > 0.3 {
		rect(vp, pressX-7, bottom-1, 14, 2, lidClr)
	}
}

// drawRings draws a shrinking approach ring that closes on each tap target.
func (g *BusyBarista) drawRings(vp *ebiten.Image) {
	if g.state != statePlaying {
		return
	}
	ring := func(x, y, until float64) {
		const lead = 0.7
		if until < 0 || until > lead {
			return
		}
		r := 6 + until*34
		a := uint8(220 * (1 - until/lead))
		vector.StrokeCircle(vp, float32(x), float32(y), float32(r), 1.5, color.NRGBA{255, 230, 150, a}, true)
	}
	for _, c := range g.cups {
		_, h := c.kind.size()
		switch {
		case c.phase == phaseToLid && !c.lidded:
			ring(pressX, beltY-h/2, c.lidT-g.now)
		case c.kind == cupEspresso && c.phase == phaseIncoming:
			ring(spoutX, beltY-h/2, c.target-g.now)
		}
	}
}

func (g *BusyBarista) drawParticles(vp *ebiten.Image) {
	for _, p := range g.particles {
		k := float32(p.life / p.max)
		clr := color.RGBA{
			uint8(p.clr[0] * 255 * p.clr[3] * k), uint8(p.clr[1] * 255 * p.clr[3] * k),
			uint8(p.clr[2] * 255 * p.clr[3] * k), uint8(255 * p.clr[3] * k),
		}
		rect(vp, p.x, p.y, p.size, p.size, clr)
	}
}

// drawQueue draws the conference attendees waiting at the end of the counter.
func (g *BusyBarista) drawQueue(vp *ebiten.Image) {
	const feet, sc = 236.0, 0.9
	for _, f := range g.queue {
		yOff := math.Sin((g.animClock+f.phase)*3) * 0.8
		if f.hop > 0 {
			yOff -= 9 * math.Sin(math.Pi*(1-f.hop/0.3))
		}
		frame := g.crowdSheet.Frame(20 + int((g.animClock+f.phase)*6)%10)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(sc, sc)
		op.GeoM.Translate(f.x-32*sc, feet+yOff-64*sc)
		op.ColorScale.Scale(f.tint[0], f.tint[1], f.tint[2], 1)
		vp.DrawImage(frame, op)
	}
}

// — HUD ——————————————————————————————————————————————————————————————————

func (g *BusyBarista) drawHUDShapes(vp *ebiten.Image) {
	if g.state == stateReady {
		return
	}
	// lives, as hearts
	for i := range maxLives {
		drawHeart(vp, 6+float64(i)*16, 5, i < g.sc.lives)
	}
	// level progress pips
	const pipW = 22.0
	x0 := tdcgame.ScreenW/2 - pipW*float64(len(levels))/2
	for i := range levels {
		x := x0 + float64(i)*pipW
		rect(vp, x+1, 17, pipW-2, 3, color.RGBA{40, 30, 28, 255})
		switch {
		case i < g.levelIdx:
			rect(vp, x+1, 17, pipW-2, 3, cremaClr)
		case i == g.levelIdx:
			k := math.Min(1, (g.now-float64(i)*levelDuration)/levelDuration)
			if i == len(levels)-1 {
				k = math.Min(1, (g.now-float64(i)*levelDuration)/(runCap-float64(i)*levelDuration))
			}
			rect(vp, x+1, 17, (pipW-2)*k, 3, lineClr)
		}
	}
}

func (g *BusyBarista) drawHUDText(screen *ebiten.Image, scale float64) {
	write(screen, g.lvl().name, tdcgame.ScreenW/2, 6, 6, color.RGBA{240, 220, 190, 255}, text.AlignCenter, scale)
	write(screen, fmt.Sprintf("%d", g.sc.score), tdcgame.ScreenW-6, 5, 12, color.White, text.AlignEnd, scale)
	if g.sc.combo > 1 {
		mult := g.sc.multiplier()
		clr := color.RGBA{230, 210, 190, 255}
		if mult > 1 {
			clr = lineClr
		}
		write(screen, fmt.Sprintf("COMBO %d  x%d", g.sc.combo, mult), tdcgame.ScreenW-6, 21, 6, clr, text.AlignEnd, scale)
	}
}

func (g *BusyBarista) drawPopups(screen *ebiten.Image, scale float64) {
	for _, p := range g.popups {
		a := math.Min(1, p.life/p.maxLife*1.6)
		shadow := color.RGBA{0, 0, 0, uint8(180 * a)}
		clr := color.RGBA{uint8(float64(p.clr.R) * a), uint8(float64(p.clr.G) * a), uint8(float64(p.clr.B) * a), uint8(255 * a)}
		write(screen, p.text, p.x+0.6, p.y+0.6, p.size, shadow, text.AlignCenter, scale)
		write(screen, p.text, p.x, p.y, p.size, clr, text.AlignCenter, scale)
	}
}

// drawBanner slides the level name in across the middle of the screen.
func (g *BusyBarista) drawBanner(screen *ebiten.Image, scale float64) {
	if g.bannerT <= 0 {
		return
	}
	t := 1 - g.bannerT/g.bannerLen
	// Slide in and out over a fixed 0.27s, however long the banner stays.
	in, out := 0.27/g.bannerLen, 1-0.27/g.bannerLen
	x := float64(tdcgame.ScreenW) / 2
	switch {
	case t < in:
		x -= (1 - t/in) * 300
	case t > out:
		x += (t - out) / (1 - out) * 300
	}
	intro := g.lvl().intro
	h := 30.0
	if intro != nil {
		h = 48
	}
	vector.FillRect(screen, 0, float32(80*scale), float32(screen.Bounds().Dx()), float32(h*scale), color.RGBA{0, 0, 0, 150}, false)
	write(screen, fmt.Sprintf("LEVEL %d", g.levelIdx+1), x, 83, 6, cremaClr, text.AlignCenter, scale)
	write(screen, g.lvl().name, x, 92, 12, color.White, text.AlignCenter, scale)
	if intro != nil {
		g.drawIntroLine(screen, scale, x, 117, intro)
	}
}

// drawIntroLine draws "NEW: [tap icon] [cup] how to serve it", centred on x
// with its middle at y, all in viewport units.
func (g *BusyBarista) drawIntroLine(screen *ebiten.Image, scale, x, y float64, in *intro) {
	const iconPx, labelSize, howSize, gap = 0.7, 6.0, 5.0, 4.0
	const handleW = 4 // the espresso cup's handle sticks out past its width
	cupW, cupH := in.kind.size()
	label := "NEW:"
	labelW := tdcgame.MeasureWidth(label, labelSize*scale) / scale
	iconW := tdcgame.TapIcon.Width() * iconPx
	howW := tdcgame.MeasureWidth(in.how, howSize*scale) / scale
	total := labelW + gap + iconW + gap + cupW + handleW + gap + howW
	cx := x - total/2

	tdcgame.WriteAt(screen, label, cx*scale, y*scale, labelSize*scale, lineClr, text.AlignStart, text.AlignCenter)
	cx += labelW + gap
	tdcgame.TapIcon.Draw(screen, cx*scale, (y-11*iconPx)*scale, iconPx*scale, g.animClock)
	cx += iconW + gap

	// The cup, drawn the same way as on the belt, shown filled.
	cupScratch.Clear()
	drawCupShape(cupScratch, &cup{kind: in.kind, fill: 0.8, lidded: true})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(cx+cupW/2-cupOriginX, y+cupH/2-cupOriginY)
	op.GeoM.Scale(scale, scale)
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(cupScratch, op)
	cx += cupW + handleW + gap

	tdcgame.WriteAt(screen, in.how, cx*scale, y*scale, howSize*scale, color.White, text.AlignStart, text.AlignCenter)
}

func panel(vp *ebiten.Image, x, y, w, h float64) {
	rect(vp, x-2, y-2, w+4, h+4, cremaClr)
	rect(vp, x, y, w, h, color.NRGBA{30, 20, 18, 235})
}

// write draws text at viewport coordinates, scaled to device pixels.
func write(screen *ebiten.Image, msg string, x, y, size float64, clr color.Color, align text.Align, scale float64) {
	tdcgame.WriteAt(screen, msg, x*scale, y*scale, size*scale, clr, align, text.AlignStart)
}

// heartRows is the lives icon. x is the body, h a highlight, d the shaded
// lower edge.
var heartRows = []string{
	".xxxx...xxxx.",
	"xxhhxx.xxxxxx",
	"xhhxxxxxxxxxx",
	"xhxxxxxxxxxxx",
	"xxxxxxxxxxxxd",
	".xxxxxxxxxxd.",
	"..xxxxxxxxd..",
	"...xxxxxxd...",
	"....xxxxd....",
	".....xxd.....",
	"......d......",
}

// drawHeart draws a life at x,y (top-left): red while it's still held, dark
// once it's lost.
func drawHeart(vp *ebiten.Image, x, y float64, alive bool) {
	body, hi, shade := color.RGBA{226, 38, 58, 255}, color.RGBA{255, 160, 170, 255}, color.RGBA{150, 18, 36, 255}
	if !alive {
		body, hi, shade = color.RGBA{64, 52, 58, 255}, color.RGBA{84, 72, 78, 255}, color.RGBA{44, 36, 42, 255}
	}
	for row, line := range heartRows {
		for col, c := range line {
			var clr color.RGBA
			switch c {
			case 'x':
				clr = body
			case 'h':
				clr = hi
			case 'd':
				clr = shade
			default:
				continue
			}
			rect(vp, x+float64(col), y+float64(row), 1, 1, clr)
		}
	}
}

// — instructions —————————————————————————————————————————————————————————

// The title screen explains the two stations as two steps, each with its
// button icon and a small cup acting it out in time with the icon.
const (
	titleStep1X, titleStep2X = 140.0, 286.0 // step centres
	titleIconY               = 70.0
	titleIconPx              = 1.2 // viewport px per icon pixel
	titleCupBottom           = 98.0
)

// titlePourFill is the step-1 cup's coffee level at time t, following the
// hold icon's loop: it fills while the finger is down and stops at the line.
func titlePourFill(t float64) float64 {
	const lineFrac, down, up, period = 0.8, 0.45, 1.65, 2.4
	k := math.Mod(t, period)
	switch {
	case k < down:
		return 0
	case k < up:
		return lineFrac * (k - down) / (up - down)
	}
	return lineFrac
}

// titleLidded reports whether the step-2 cup has its lid at time t, following
// the tap icon's loop: the lid goes on as the finger comes down.
func titleLidded(t float64) bool {
	return math.Mod(t, 1.2) >= 0.45
}

func drawTitleCups(vp *ebiten.Image, t float64) {
	pour := &cup{kind: cupStandard, lineFrac: 0.8, fill: titlePourFill(t)}
	lid := &cup{kind: cupStandard, lineFrac: 0.8, fill: 0.8, lidded: titleLidded(t)}
	for _, c := range []struct {
		c *cup
		x float64
	}{{pour, titleStep1X - 20}, {lid, titleStep2X - 20}} {
		cupScratch.Clear()
		drawCupShape(cupScratch, c.c)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(c.x-cupOriginX, titleCupBottom-cupOriginY)
		vp.DrawImage(cupScratch, op)
	}
}

func (g *BusyBarista) drawTitleText(screen *ebiten.Image, scale, t float64) {
	cx := float64(tdcgame.ScreenW) / 2
	write(screen, "BUSY BARISTA", cx, 40, 16, lineClr, text.AlignCenter, scale)

	px := titleIconPx * scale
	tdcgame.HoldIcon.Draw(screen, (titleStep1X+2)*scale, titleIconY*scale, px, t)
	tdcgame.TapIcon.Draw(screen, (titleStep2X+2)*scale, titleIconY*scale, px, t)
	write(screen, ">", cx, 80, 10, cremaClr, text.AlignCenter, scale)

	steps := []struct {
		x            float64
		name, detail string
	}{
		{titleStep1X, "1  POUR", "HOLD, LET GO ON THE LINE"},
		{titleStep2X, "2  LID", "TAP AT THE LID PRESS"},
	}
	for _, st := range steps {
		write(screen, st.name, st.x, 106, 8, color.White, text.AlignCenter, scale)
		write(screen, st.detail, st.x, 118, 5, color.RGBA{220, 200, 180, 255}, text.AlignCenter, scale)
	}

	// Start hint, styled like the end screen's.
	const startIconPx = 1.2
	label := "TO START"
	w := tdcgame.TapIcon.Width()*startIconPx + 5 + tdcgame.MeasureWidth(label, 7*scale)/scale
	x := cx - w/2
	tdcgame.TapIcon.Draw(screen, x*scale, 140*scale, startIconPx*scale, t)
	tdcgame.WriteAt(screen, label, (x+tdcgame.TapIcon.Width()*startIconPx+5)*scale, (140+11*startIconPx)*scale,
		7*scale, color.White, text.AlignStart, text.AlignCenter)
}
