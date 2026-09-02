package main

import (
	"embed"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"variant.dev/tdcgame/tdcgame"
)

//go:embed assets/variant_logo.png
var logoFS embed.FS

var logoImage *ebiten.Image

func init() {
	var err error
	logoImage, _, err = ebitenutil.NewImageFromFileSystem(logoFS, "assets/variant_logo.png")
	if err != nil {
		log.Fatal(err)
	}
}

type launcherState int

const (
	launcherBrowsing launcherState = iota
	launcherFading
)

const holdToStartDuration = 0.85

// tapMaxDuration is how long the button may be held and still count as a tap
// that advances the selection, rather than the beginning of a hold. At 0.15s
// an ordinary press easily overshot it and did nothing at all; at 0.25s a
// slightly-long tap while cycling still flashed the fill, so it sits at 0.33s.
const tapMaxDuration = 0.33

// holdIndicatorDelay is the grace period before the hold-progress bar appears.
// Without it, every quick click while browsing flashes the bar on and off. It
// matches tapMaxDuration, so the bar appearing is exactly the signal that
// letting go now aborts instead of selecting.
const holdIndicatorDelay = tapMaxDuration

type gameEntry struct {
	name     string
	gameName string // internal key scores are stored under, e.g. "tdcrunner"
	color    color.RGBA
	newScene func() Scene
	key      ebiten.Key
}

var launcherGames []gameEntry

// launcherSelected remembers the highlighted game across launcher instances, so
// returning from a game (which builds a fresh LauncherScene) lands back on the
// item you launched instead of resetting to the top of the list.
var launcherSelected int

type LauncherScene struct {
	selected  int
	state     launcherState
	holdTimer float64
	fadeAlpha float64
	viewport  *ebiten.Image

	// holdArmed guards against a button that's still held when this scene
	// becomes active (e.g. returning from the options screen) from being read
	// as a fresh hold and launching a game. It's cleared when the button is
	// released.
	holdArmed bool
}

func NewLauncherScene() *LauncherScene {
	sel := launcherSelected
	if sel < 0 || sel >= len(launcherGames) {
		sel = 0
	}
	return &LauncherScene{holdArmed: true, selected: sel}
}

func (l *LauncherScene) Update(dt float64) (Scene, error) {
	for _, k := range inpututil.AppendJustPressedKeys(nil) {
		for i, g := range launcherGames {
			if g.key != ebiten.KeyMax && g.key == k {
				launcherSelected = i
				return g.newScene(), nil
			}
		}
	}

	// O opens the options screen. A dedicated hotkey (like the per-game keys)
	// instead of a triple-press, which players triggered accidentally while
	// cycling. IsKeyJustPressed keeps a held O from immediately reopening it.
	if inpututil.IsKeyJustPressed(ebiten.KeyO) {
		return NewOptionsScene(l), nil
	}

	switch l.state {
	case launcherBrowsing:
		pressed := ebiten.IsKeyPressed(ebiten.KeyEnter)
		if !l.holdArmed {
			// Ignore a button carried over held (e.g. from the options screen)
			// until it's released, so it can't immediately launch a game.
			if !pressed {
				l.holdArmed = true
				l.holdTimer = 0
			}
			break
		}

		if pressed {
			l.holdTimer += dt
			if l.holdTimer >= holdToStartDuration {
				l.state = launcherFading
			}
		} else {
			// A quick tap (released before the hold bar appears) advances the
			// selection; releasing after the bar has shown just aborts the hold.
			if inpututil.IsKeyJustReleased(ebiten.KeyEnter) && l.holdTimer <= tapMaxDuration {
				l.selected = (l.selected + 1) % len(launcherGames)
				launcherSelected = l.selected
			}
			l.holdTimer = 0
		}
	case launcherFading:
		l.fadeAlpha += dt * 255
		if l.fadeAlpha >= 255 {
			entry := launcherGames[l.selected]
			if entry.newScene != nil {
				launcherSelected = l.selected
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

	l.drawBackground(vp)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(vp, op)

	l.drawGameList(screen, scale)
	l.drawScoreboardPanel(screen, scale)
	l.drawText(screen, scale)
	l.drawScoreboardText(screen, scale)
	l.drawFooterHints(screen, scale)
	l.drawOptionsHint(screen, scale)
	l.drawLogo(screen, scale)

	// Fade-to-black overlay (covers text too so the whole launcher fades)
	if l.fadeAlpha > 0 {
		alpha := uint8(math.Min(l.fadeAlpha, 255))
		b := screen.Bounds()
		vector.FillRect(screen, 0, 0, float32(b.Dx()), float32(b.Dy()), color.RGBA{0, 0, 0, alpha}, false)
	}
}

const listPanelW = 290
const logoHeaderH = 24

// Shared row layout, used both when drawing the low-res list visuals and
// when drawing the crisp, device-resolution text on top of them.
// listStartY leaves listTopPad of breathing room below the logo header,
// above the first row's pill top, instead of butting it right up against
// the header. listRowGap is the whitespace between one pill's bottom and
// the next one's top; listRowH (the pitch between row tops) is derived from
// it so the pill height itself (listPillH) stays fixed regardless of how
// much gap is configured.
const (
	listPillH    = 22
	listRowGap   = 6
	listRowH     = listPillH + listRowGap
	listTopPad   = 32
	listStartY   = logoHeaderH + listTopPad + listRowGap
	listStartX   = 32
	listSwatch   = 10
	listFontSize = 8
)

const (
	scoreboardX             = listPanelW + 32
	scoreboardPanelX0       = scoreboardX - 6
	scoreboardPanelX1       = ScreenW - listStartX
	scoreboardPanelY0       = listStartY - listRowGap
	scoreboardHeaderH       = 18
	scoreboardTitleFontSize = 5
	scoreboardRowH          = 16
	scoreboardRowsStartY    = scoreboardPanelY0 + scoreboardHeaderH + 6
	scoreboardValuePadRight = 6
	scoreboardMaxRows       = 5
	scoreboardFontSize      = 7
)

var (
	headerBgColor   = color.RGBA{242, 242, 242, 255} // #F2F2F2
	headerTextColor = color.RGBA{40, 40, 40, 255}    // #282828
)

// bgColor is the launcher's dark background.
var bgColor = color.RGBA{40, 40, 40, 255}            // #282828
var whiteTextColor = color.RGBA{242, 242, 242, 255}  // #F2F2F2
var rowBorderColor = color.RGBA{106, 106, 106, 255}  // #6A6A6A
var selectedRowColor = color.RGBA{255, 212, 47, 255} // #FFD42F

func (l *LauncherScene) drawBackground(vp *ebiten.Image) {
	vp.Fill(bgColor)
	vector.FillRect(vp, 0, 0, ScreenW, logoHeaderH, headerBgColor, false)
}

const logoPadY = 8

// drawLogo draws the Variant logo at device-pixel resolution, top-left and
// vertically centered within the header strip. It's scaled down from a
// higher-resolution source image (not the low-res viewport) so it stays
// crisp at any window size.
func (l *LauncherScene) drawLogo(screen *ebiten.Image, scale float64) {
	if logoImage == nil {
		return
	}
	targetH := float64(logoHeaderH-2*logoPadY) * scale
	imgScale := targetH / float64(logoImage.Bounds().Dy())

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Scale(imgScale, imgScale)
	op.GeoM.Translate(float64(listStartX)*scale, (float64(logoHeaderH)*scale-targetH)/2)
	screen.DrawImage(logoImage, op)
}

const (
	selectedRowExtraH = 4
	selectedRowExtraW = 8
)

// selectedRowStrokeW is the outline thickness (in vp px) of the resting selected
// row, before the hold fill sweeps in. Thicker than an unselected row's 1px
// border so it reads as the active item.
const selectedRowStrokeW = 2

const selectedNameFontSize = 9

// rowRect returns row i's pill rectangle in vp-space (unscaled): left edge,
// top edge, width, height. Shared by drawGameList and drawText so both agree
// on exactly where each row sits once the selected row's growth is applied.
func (l *LauncherScene) rowRect(i int) (x, y, w, h float64) {
	x = float64(listStartX - 6)
	w = float64(listPanelW - listStartX)
	h = float64(listPillH)
	y = float64(listStartY - listRowGap + i*listRowH)
	if i == l.selected {
		x -= selectedRowExtraW / 2
		w += selectedRowExtraW
		y -= selectedRowExtraH / 2
		h += selectedRowExtraH
	}
	return x, y, w, h
}

// drawGameList draws the pill-shaped selection highlight, each unselected
// game's color swatch, and (on the selected row) the hold-to-start hint, at
// device-pixel resolution so the rounded corners stay smooth regardless of
// window scale. The selected row drops its swatch and grows a bit taller and
// wider instead, so it reads as "the one you're on" rather than just another
// list item.
func (l *LauncherScene) drawGameList(screen *ebiten.Image, scale float64) {
	s := float32(scale)
	for i, game := range launcherGames {
		x, y, w, h := l.rowRect(i)
		rowX, rowY, rowW, rowH := float32(x)*s, float32(y)*s, float32(w)*s, float32(h)*s

		if i == l.selected {
			// The selected row rests as an outline; holding fills it with the
			// accent color left→right so it reads as filling up (not emptying),
			// landing on the solid pill exactly as the game launches. Clipping
			// the full-pill rounded rect to the filled sub-region keeps the left
			// rounded cap and leaves a clean straight wipe edge.
			strokeRoundedRect(screen, rowX, rowY, rowW, rowH, rowH/2, selectedRowStrokeW*s, selectedRowColor)
			if fillW := float32(l.holdProgress()) * rowW; fillW > 0 {
				clip := image.Rect(int(rowX), int(rowY), int(rowX+fillW), int(rowY+rowH+1))
				dst := screen.SubImage(clip).(*ebiten.Image)
				fillRoundedRect(dst, rowX, rowY, rowW, rowH, rowH/2, selectedRowColor)
			}
			continue
		}

		strokeRoundedRect(screen, rowX, rowY, rowW, rowH, rowH/2, s, rowBorderColor)
		swatch := float32(listSwatch) * s
		fillRoundedRect(screen, float32(listStartX)*s, rowY+7*s, swatch, swatch, swatch/3, dimColor(game.color, 0.45))
	}
}

// holdFill maps a hold timer to a fill fraction in [0,1]. It stays 0 until the
// timer passes holdIndicatorDelay, the same grace period Update uses to
// distinguish a click from the start of a hold, so a tap never flashes a
// partial fill. Shared by the launcher and options screens.
func holdFill(holdTimer float64) float64 {
	if holdTimer <= holdIndicatorDelay {
		return 0
	}
	p := (holdTimer - holdIndicatorDelay) / (holdToStartDuration - holdIndicatorDelay)
	if p > 1 {
		return 1
	}
	return p
}

// holdProgress returns the current hold-to-start fill fraction in [0,1].
func (l *LauncherScene) holdProgress() float64 {
	return holdFill(l.holdTimer)
}

// rankYellowColor marks the 1st-place rank/score; every other rank is plain
// white, so only the leader stands out.
var rankYellowColor = color.RGBA{255, 212, 47, 255} // #FFD42F

// scoreboardEmptyColor is used for the "0" placeholder value in an unfilled
// scoreboard slot, so it reads as unset rather than an achieved score.
var scoreboardEmptyColor = color.RGBA{106, 106, 106, 255} // #6A6A6A

func rankColor(rank int) color.RGBA {
	if rank == 0 {
		return rankYellowColor
	}
	return whiteTextColor
}

const scoreboardPanelPadBottom = 4
const scoreboardPanelRadius = 10
const scoreboardBorderWidth = 1

// drawScoreboardPanel draws the scoreboard as a card
func (l *LauncherScene) drawScoreboardPanel(screen *ebiten.Image, scale float64) {
	s := float32(scale)
	r := float32(scoreboardPanelRadius) * s

	x0 := float32(scoreboardPanelX0) * s
	x1 := float32(scoreboardPanelX1) * s
	panelY0 := float32(scoreboardPanelY0) * s
	panelY1 := float32(scoreboardRowsStartY+scoreboardMaxRows*scoreboardRowH+scoreboardPanelPadBottom) * s

	fillRoundedTopRect(screen, x0, panelY0, x1-x0, float32(scoreboardHeaderH)*s, r, headerBgColor)
	strokeRoundedRect(screen, x0, panelY0, x1-x0, panelY1-panelY0, r, float32(scoreboardBorderWidth)*s, headerBgColor)

	dividerX0 := float32(scoreboardX) * s
	dividerX1 := float32(scoreboardPanelX1-scoreboardValuePadRight) * s // matches the score value's right edge
	for i := 1; i < scoreboardMaxRows; i++ {
		y := float32(scoreboardRowsStartY+i*scoreboardRowH-scoreboardRowH/2) * s
		vector.StrokeLine(screen, dividerX0, y, dividerX1, y, s/2, rowBorderColor, false)
	}
}

// topScores returns up to scoreboardMaxRows scores for gameName, highest first.
func topScores(gameName string) []int {
	if scoreKeeper == nil {
		return nil
	}
	scores := scoreKeeper.GetScores(gameName)
	sort.Sort(sort.Reverse(sort.IntSlice(scores)))
	if len(scores) > scoreboardMaxRows {
		scores = scores[:scoreboardMaxRows]
	}
	return scores
}

// drawText draws every label at device-pixel resolution (screen, not vp) so
// it stays crisp instead of being upscaled with the rest of the pixel art.
func (l *LauncherScene) drawText(screen *ebiten.Image, scale float64) {
	for i, game := range launcherGames {
		x, y, w, h := l.rowRect(i)
		midY := (y + h/2) * scale

		if i == l.selected {
			nameX := (x + 6) * scale
			fontSize := float64(selectedNameFontSize) * scale
			tdcgame.WriteAt(screen, game.name, nameX, midY, fontSize, whiteTextColor, text.AlignStart, text.AlignCenter)
			// Over the hold-to-start accent fill the name flips to dark, so it
			// stays legible as the fill sweeps across. Clip the second pass to
			// the same filled sub-region the fill uses in drawGameList.
			s := float32(scale)
			rowX, rowY := float32(x)*s, float32(y)*s
			if fillW := float32(l.holdProgress()) * float32(w) * s; fillW > 0 {
				clip := image.Rect(int(rowX), int(rowY), int(rowX+fillW), int((y+h)*scale)+1)
				dst := screen.SubImage(clip).(*ebiten.Image)
				tdcgame.WriteAt(dst, game.name, nameX, midY, fontSize, headerTextColor, text.AlignStart, text.AlignCenter)
			}
			continue
		}

		nameX := float64(listStartX+listSwatch+8) * scale
		tdcgame.WriteAt(screen, game.name, nameX, midY, float64(listFontSize)*scale, whiteTextColor, text.AlignStart, text.AlignCenter)
	}
}

// drawScoreboardText renders the high-score list for the currently selected
// game
func (l *LauncherScene) drawScoreboardText(screen *ebiten.Image, scale float64) {
	if len(launcherGames) == 0 {
		return
	}

	x := int(float64(scoreboardX) * scale)
	titleMidY := (float64(scoreboardPanelY0) + float64(scoreboardHeaderH)/2) * scale
	tdcgame.WriteAt(screen, "HIGH SCORES", float64(x), titleMidY, float64(scoreboardTitleFontSize)*scale, headerTextColor, text.AlignStart, text.AlignCenter)

	scores := topScores(launcherGames[l.selected].gameName)
	valueX := float64(scoreboardPanelX1-scoreboardValuePadRight) * scale

	// Always render every slot (filled or not) so the card reads as a
	// consistent podium rather than collapsing to a single placeholder line.
	for i := 0; i < scoreboardMaxRows; i++ {
		y := scoreboardRowsStartY + i*scoreboardRowH
		clr := rankColor(i)
		rank := fmt.Sprintf("%d.", i+1)
		tdcgame.WriteAt(screen, rank, float64(x), float64(y-2)*scale, float64(scoreboardFontSize)*scale, clr, text.AlignStart, text.AlignStart)

		value, valueColor := "0", scoreboardEmptyColor
		if i < len(scores) {
			value, valueColor = fmt.Sprintf("%d", scores[i]), clr
		}
		tdcgame.WriteAt(screen, value, valueX, float64(y-2)*scale, float64(scoreboardFontSize)*scale, valueColor, text.AlignEnd, text.AlignStart)
	}
}

const (
	footerBottomPad = 16
	footerFontSize  = 5
	footerCircleR   = 4
	footerIconGap   = 7
	footerGroupGap  = 14
)

var footerLabelColor = color.RGBA{200, 200, 210, 255}

// drawFooterHints draws the input-hint legend
func (l *LauncherScene) drawFooterHints(screen *ebiten.Image, scale float64) {
	y := float32(ScreenH-footerBottomPad) * float32(scale)

	x := float64(listStartX)
	x = l.drawHint(screen, x, y, scale, false, "Press to move")
	x += footerGroupGap

	dividerX := float32(x) * float32(scale)
	vector.StrokeLine(screen, dividerX, y-6*float32(scale), dividerX, y+6*float32(scale), float32(scale), rowBorderColor, false)
	x += footerGroupGap

	l.drawHint(screen, x, y, scale, true, "Hold to start")
}

// toggleCRT flips a mode on/off: selecting the active option turns the CRT off,
// selecting the other switches to it (so Subtle/Full act as mutually exclusive
// on/off toggles). Shared with the options screen.
func toggleCRT(cur, clicked crtMode) crtMode {
	if cur == clicked {
		return crtOff
	}
	return clicked
}

// drawOptionsHint draws a right-anchored footer hint pointing at the options
// screen (reached with the O hotkey).
func (l *LauncherScene) drawOptionsHint(screen *ebiten.Image, scale float64) {
	y := float64(ScreenH-footerBottomPad) * scale
	fs := float64(footerFontSize) * scale
	tdcgame.WriteAt(screen, "PRESS O: OPTIONS", float64(scoreboardPanelX1)*scale, y,
		fs, footerLabelColor, text.AlignEnd, text.AlignCenter)
}

// drawHint draws one "○ label" hint
func (l *LauncherScene) drawHint(screen *ebiten.Image, x float64, y float32, scale float64, filled bool, label string) float64 {
	s := float32(scale)
	cx := float32(x+footerCircleR) * s
	if filled {
		vector.FillCircle(screen, cx, y, footerCircleR*s, rankYellowColor, true)
	} else {
		vector.StrokeCircle(screen, cx, y, footerCircleR*s, s, footerLabelColor, true)
	}

	textX := x + footerCircleR*2 + footerIconGap
	fontSize := footerFontSize * scale
	tdcgame.WriteAt(screen, label, textX*scale, float64(y), fontSize, footerLabelColor, text.AlignStart, text.AlignCenter)
	return textX + tdcgame.MeasureWidth(label, fontSize)/scale
}

func dimColor(c color.RGBA, f float64) color.RGBA {
	return color.RGBA{
		uint8(float64(c.R) * f),
		uint8(float64(c.G) * f),
		uint8(float64(c.B) * f),
		c.A,
	}
}
