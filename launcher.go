package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"variant.dev/tdcgame/tdcgame"
)

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

// attractTime is a monotonic clock, in seconds, driving every launcher/options
// animation (scrolling grid, twinkling stars, breathing hero button). It's a
// package global so the Options screen — which reuses the launcher's backdrop —
// keeps animating while it's in front, even though the launcher's own Update
// isn't running then. Both scenes advance it by dt.
var attractTime float64

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
	attractTime += dt

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

	// Dark panels first so every label sits on a clean background, then the
	// crisp content on top of them.
	l.drawTitle(screen, scale)
	drawPanel(screen, scale, listCardX0, panelTop, listCardX1, panelBot, neonPurple)
	l.drawScoreboardPanel(screen, scale)
	l.drawGameList(screen, scale)
	l.drawText(screen, scale)
	l.drawScoreboardText(screen, scale)
	l.drawControls(screen, scale)

	// Fade-to-black overlay (covers text too so the whole launcher fades)
	if l.fadeAlpha > 0 {
		alpha := uint8(math.Min(l.fadeAlpha, 255))
		b := screen.Bounds()
		vector.FillRect(screen, 0, 0, float32(b.Dx()), float32(b.Dy()), color.RGBA{0, 0, 0, alpha}, false)
	}
}

const listPanelW = 290

// Shared row layout, used both when drawing the low-res list visuals and
// when drawing the crisp, device-resolution text on top of them.
// listStartY leaves room for the neon title band above the first row. listRowGap
// is the whitespace between one pill's bottom and the next one's top; listRowH
// (the pitch between row tops) is derived from it so the pill height itself
// (listPillH) stays fixed regardless of how much gap is configured.
const (
	listPillH  = 22
	listRowGap = 6
	listRowH   = listPillH + listRowGap
	// listStartY centers the four-row block vertically in the card so the top and
	// bottom margins match (and the selected row's growth never overruns the
	// frame). The block spans 4*listRowH-listRowGap tall.
	listStartY   = panelTop + (panelBot-panelTop-(4*listRowH-listRowGap))/2 + listRowGap
	listStartX   = 32
	listSwatch   = 10
	listFontSize = 8
)

// The games list and scoreboard are drawn as two aligned dark cards (panelTop..
// panelBot). The list card spans the full width of the game rows; the scoreboard
// card sits to its right.
const (
	panelTop = 52
	panelBot = 172
	// The card is centered on the game rows (whose center sits at x=155) so the
	// selected row — which grows a few px wider — keeps equal margins on both
	// sides of the purple frame instead of hugging one edge.
	listCardX0 = 14
	listCardX1 = 296
)

// cornerR is the shared corner radius (vp units) for panels and rows. Kept small
// so frames read as crisp 8-bit rectangles rather than soft bubbles.
const cornerR = 3

const (
	scoreboardX             = listPanelW + 32
	scoreboardPanelX0       = scoreboardX - 6
	scoreboardPanelX1       = ScreenW - listStartX
	scoreboardHeaderH       = 18
	scoreboardTitleFontSize = 5
	scoreboardRowH          = 16
	scoreboardRowsStartY    = panelTop + scoreboardHeaderH + 12
	scoreboardValuePadRight = 6
	scoreboardMaxRows       = 5
	scoreboardFontSize      = 7
)

// Neon synthwave palette. The hues are shared across the launcher and the
// options screen (same package) so the whole menu system reads as one piece.
// The *Text colors are deliberately bright so text stays crisp and legible on
// the near-black backdrop — the glow lives on the shapes (frames, button,
// floor), never smeared across the type.
var (
	neonCyan   = color.RGBA{45, 226, 230, 255}  // #2DE2E6
	neonPink   = color.RGBA{255, 56, 100, 255}  // #FF3864
	neonPurple = color.RGBA{122, 47, 245, 255}  // #7A2FF5
	cyanText   = color.RGBA{120, 245, 250, 255} // bright cyan for the title
	pinkText   = color.RGBA{255, 120, 150, 255} // bright pink for accents
	starColor  = color.RGBA{200, 230, 255, 255} // cool white twinkle
)

// bgColor is the launcher's near-black backdrop; the rest of the shared palette
// vars are retinted for the neon look. headerTextColor stays dark because it's
// the text/icon color painted over the bright accent fill as a hold charges.
var bgColor = color.RGBA{10, 6, 18, 255}              // #0A0612
var whiteTextColor = color.RGBA{236, 240, 255, 255}   // cool near-white
var rowBorderColor = color.RGBA{74, 58, 110, 255}     // dim purple outline
var selectedRowColor = neonCyan                       // accent for the focused row
var headerTextColor = color.RGBA{10, 6, 18, 255}      // dark text over bright fills
var panelFillColor = color.RGBA{18, 11, 32, 235}      // near-opaque dark card
var footerLabelColor = color.RGBA{159, 232, 234, 255} // soft cyan

// --- Animated backdrop --------------------------------------------------------

// gridHorizonY is the vanishing-point height of the synthwave floor grid. It
// sits below the content cards so the grid and its glow never bleed behind the
// game list or scoreboard; stars live above it, the grid sweeps from it down.
const gridHorizonY = 182

type star struct {
	x, y  float32
	phase float64
	size  float32
}

var stars []star

func init() {
	// A fixed seed keeps the starfield identical every launch (and across the
	// many redraws), so stars twinkle in place rather than reshuffling.
	r := rand.New(rand.NewSource(1337))
	for i := 0; i < 44; i++ {
		s := star{
			x:     float32(r.Float64()) * ScreenW,
			y:     float32(r.Float64()) * (gridHorizonY - 14),
			phase: r.Float64() * 2 * math.Pi,
			size:  1,
		}
		if r.Float64() > 0.88 {
			s.size = 2
		}
		stars = append(stars, s)
	}
}

func (l *LauncherScene) drawBackground(vp *ebiten.Image) {
	vp.Fill(bgColor)
	drawStars(vp)
	drawHorizonGlow(vp)
	drawFloorGrid(vp)
}

// drawStars scatters twinkling dots across the sky. Each star's alpha breathes
// on its own phase so the field shimmers instead of blinking in unison. They're
// kept dim so they never fight the content cards drawn on top.
func drawStars(vp *ebiten.Image) {
	for _, s := range stars {
		tw := 0.5 + 0.5*math.Sin(attractTime*2+s.phase)
		a := uint8(40 + 110*tw)
		vector.FillRect(vp, s.x, s.y, s.size, s.size, withAlpha(starColor, a), false)
	}
}

// drawHorizonGlow lays a hot-pink band along the horizon that fades upward, the
// classic synthwave "sunset" bleeding into the sky above the grid.
func drawHorizonGlow(vp *ebiten.Image) {
	const bands = 6
	for i := 0; i < bands; i++ {
		a := uint8(float64(80) * (1 - float64(i)/bands))
		y := float32(gridHorizonY) - float32(i)*2
		vector.FillRect(vp, 0, y, ScreenW, 2, withAlpha(neonPink, a), false)
	}
	vector.FillRect(vp, 0, gridHorizonY, ScreenW, 1, neonPink, false)
}

// drawFloorGrid draws the receding synthwave floor: verticals converging on the
// vanishing point and horizontal lines that accelerate toward the viewer as
// attractTime advances, so the floor appears to scroll forever.
func drawFloorGrid(vp *ebiten.Image) {
	horizon := float32(gridHorizonY)
	bottom := float32(ScreenH)
	vanishX := float32(ScreenW) / 2

	for j := -9; j <= 9; j++ {
		bx := vanishX + float32(j)*48
		vector.StrokeLine(vp, vanishX, horizon, bx, bottom, 1, withAlpha(neonPurple, 120), false)
	}

	// Horizontal lines sweep downward; a quadratic in t spaces them wider as they
	// near the bottom (perspective), and the fractional scroll makes them flow.
	frac := attractTime * 0.35
	frac -= math.Floor(frac)
	const lines = 8
	for k := 0; k < lines; k++ {
		t := (float64(k) + frac) / lines
		y := horizon + (bottom-horizon)*float32(t*t)
		a := uint8(50 + 170*t)
		vector.StrokeLine(vp, 0, y, ScreenW, y, 1, withAlpha(neonCyan, a), false)
	}
}

// --- Panels & crisp text ------------------------------------------------------

// drawPanel draws one dark content card: a near-opaque fill under a neon tube
// frame, so the crisp text placed on top always has clean contrast.
func drawPanel(screen *ebiten.Image, scale, x0, y0, x1, y1 float64, frame color.RGBA) {
	s := float32(scale)
	r := float32(cornerR) * s
	fillRoundedRect(screen, float32(x0)*s, float32(y0)*s, float32(x1-x0)*s, float32(y1-y0)*s, r, panelFillColor)
	strokeNeonRect(screen, float32(x0)*s, float32(y0)*s, float32(x1-x0)*s, float32(y1-y0)*s, r, frame, s)
}

// writeShadow draws crisp text with a single 1px dark drop-shadow for contrast
// over busy areas (title, hero floor). No halo — the shadow keeps the pixel
// font sharp instead of smeared.
func writeShadow(dst *ebiten.Image, msg string, x, y, size float64, clr color.RGBA, pa, sa text.Align, off float64) {
	tdcgame.WriteAt(dst, msg, x+off, y+off, size, color.RGBA{0, 0, 0, 210}, pa, sa)
	tdcgame.WriteAt(dst, msg, x, y, size, clr, pa, sa)
}

// drawTitle draws the crisp VARIANT / ARCADE wordmark centered along the top,
// with a neon underline. The space on the synthwave floor below is left open for
// a one-button animation landing in a later change.
func (l *LauncherScene) drawTitle(screen *ebiten.Image, scale float64) {
	cx := float64(ScreenW) / 2 * scale
	titleSize := 16 * scale

	writeShadow(screen, "VARIANT", cx, 19*scale, titleSize, cyanText, text.AlignCenter, text.AlignCenter, 1.5*scale)
	vw := tdcgame.MeasureWidth("VARIANT", titleSize)
	uy := float32(29 * scale)
	vector.StrokeLine(screen, float32(cx)-float32(vw)/2, uy, float32(cx)+float32(vw)/2, uy, float32(2*scale), neonCyan, false)
	writeShadow(screen, "A R C A D E", cx, 38*scale, 6*scale, pinkText, text.AlignCenter, text.AlignCenter, 1*scale)
}

// drawControls explains the one-button controls in the top-left corner, reusing
// the champion screen's animated press/hold icons. Rather than showing both at
// once (visually busy), it cycles through a single slot — "PRESS FOR NEXT GAME"
// fades out and "HOLD TO PLAY GAME" fades in — so only one hint shows at a time
// and the finger motion makes the tap/hold difference obvious.
func (l *LauncherScene) drawControls(screen *ebiten.Image, scale float64) {
	// phase is a whole number of each icon's animation loop (tap 1.2s, hold 2.4s)
	// so a hint always fades out between presses — 2 full hold loops — rather than
	// cutting off mid-press. fade is the in/out time.
	const phase, fade = 4.8, 0.5
	const controlsX, iconTop = 24.0, 16.0 // nudged right and down to sit by the logo

	cyc := math.Mod(attractTime, 2*phase)
	local := cyc
	icon, line1, line2, clr := tapIcon, "PRESS FOR", "NEXT GAME", cyanText
	if cyc >= phase {
		local = cyc - phase
		icon, line1, line2, clr = holdIcon, "HOLD TO", "PLAY GAME", pinkText
	}

	alpha := 1.0
	switch {
	case local < fade:
		alpha = local / fade
	case local > phase-fade:
		alpha = (phase - local) / fade
	}

	const iconLP = 0.9 // vp px per icon pixel
	px := scale * iconLP
	iconW, iconH := 20*iconLP, 22*iconLP
	// Drive the icon from the per-hint local clock so its press cycle restarts
	// when the hint appears and lands on a whole loop as it fades.
	icon.DrawAlpha(screen, controlsX*scale, iconTop*scale, px, local, alpha)

	// Two-line label to the right of the icon, both lines sharing the fade.
	labelX := (controlsX + iconW + 5) * scale
	mid := iconTop + iconH/2
	writeFade(screen, line1, labelX, (mid-4)*scale, 6*scale, clr, alpha, scale)
	writeFade(screen, line2, labelX, (mid+4)*scale, 6*scale, clr, alpha, scale)
}

// writeFade draws crisp left-aligned text with a dark drop-shadow at a given
// opacity in [0,1], so a label can fade in and out.
func writeFade(screen *ebiten.Image, msg string, x, y, size float64, clr color.RGBA, alpha, scale float64) {
	tdcgame.WriteAt(screen, msg, x+scale, y+scale, size, color.RGBA{0, 0, 0, uint8(alpha * 200)}, text.AlignStart, text.AlignCenter)
	tdcgame.WriteAt(screen, msg, x, y, size, withAlpha(clr, uint8(alpha*255)), text.AlignStart, text.AlignCenter)
}

// drawWordmark draws a compact crisp "VARIANT" in the top-left corner, used by
// the options screen where the big centered title would crowd the content.
func (l *LauncherScene) drawWordmark(screen *ebiten.Image, scale float64) {
	x := float64(listStartX) * scale
	y := 14 * scale
	writeShadow(screen, "VARIANT", x, y, 10*scale, cyanText, text.AlignStart, text.AlignCenter, 1*scale)
}

// --- Game list ----------------------------------------------------------------

const (
	selectedRowExtraH = 4
	selectedRowExtraW = 8
)

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

// drawGameList draws each game's pill inside the list card. Unselected rows are
// a dim outline with the game's color swatch; the selected row glows brighter in
// neon cyan and fills left→right as the button is held, landing on a solid pill
// exactly as the game launches.
func (l *LauncherScene) drawGameList(screen *ebiten.Image, scale float64) {
	s := float32(scale)
	r := float32(cornerR) * s
	for i, game := range launcherGames {
		x, y, w, h := l.rowRect(i)
		rowX, rowY, rowW, rowH := float32(x)*s, float32(y)*s, float32(w)*s, float32(h)*s

		if i == l.selected {
			strokeNeonRect(screen, rowX, rowY, rowW, rowH, r, selectedRowColor, s)
			if fillW := float32(l.holdProgress()) * rowW; fillW > 0 {
				clip := image.Rect(int(rowX), int(rowY), int(rowX+fillW), int(rowY+rowH+1))
				dst := screen.SubImage(clip).(*ebiten.Image)
				fillRoundedRect(dst, rowX, rowY, rowW, rowH, r, selectedRowColor)
			}
			continue
		}

		strokeRoundedRect(screen, rowX, rowY, rowW, rowH, r, s, rowBorderColor)
		swatch := float32(listSwatch) * s
		fillRoundedRect(screen, float32(listStartX)*s, rowY+rowH/2-swatch/2, swatch, swatch, s, game.color)
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
var scoreboardEmptyColor = color.RGBA{90, 74, 130, 255} // dim purple

func rankColor(rank int) color.RGBA {
	if rank == 0 {
		return rankYellowColor
	}
	return whiteTextColor
}

// drawScoreboardPanel draws the scoreboard as a dark, neon-framed card aligned
// with the game-list card, with a divider under the HIGH SCORES header.
func (l *LauncherScene) drawScoreboardPanel(screen *ebiten.Image, scale float64) {
	s := float32(scale)
	drawPanel(screen, scale, scoreboardPanelX0, panelTop, scoreboardPanelX1, panelBot, neonCyan)

	x0 := float32(scoreboardPanelX0) * s
	x1 := float32(scoreboardPanelX1) * s
	hdrY := float32(panelTop+scoreboardHeaderH) * s
	vector.StrokeLine(screen, x0+6*s, hdrY, x1-6*s, hdrY, s, withAlpha(neonCyan, 150), false)

	dividerX0 := float32(scoreboardX) * s
	dividerX1 := float32(scoreboardPanelX1-scoreboardValuePadRight) * s // matches the score value's right edge
	for i := 1; i < scoreboardMaxRows; i++ {
		y := float32(scoreboardRowsStartY+i*scoreboardRowH-scoreboardRowH/2) * s
		vector.StrokeLine(screen, dividerX0, y, dividerX1, y, s/2, withAlpha(neonPurple, 110), false)
	}
}

// topScores returns up to scoreboardMaxRows scores for gameName, highest first.
// Equal scores keep save order, so an earlier score outranks a later tie.
func topScores(gameName string) []tdcgame.Score {
	if scoreKeeper == nil {
		return nil
	}
	scores := scoreKeeper.GetScores(gameName)
	sort.SliceStable(scores, func(i, j int) bool { return scores[i].Value > scores[j].Value })
	if len(scores) > scoreboardMaxRows {
		scores = scores[:scoreboardMaxRows]
	}
	return scores
}

// drawText draws each game's name at device-pixel resolution (screen, not vp) so
// it stays crisp instead of being upscaled with the rest of the pixel art.
func (l *LauncherScene) drawText(screen *ebiten.Image, scale float64) {
	for i, game := range launcherGames {
		x, y, w, h := l.rowRect(i)
		midY := (y + h/2) * scale

		if i == l.selected {
			nameX := (x + 12) * scale
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

// drawScoreboardText renders the high-score list for the currently selected game.
func (l *LauncherScene) drawScoreboardText(screen *ebiten.Image, scale float64) {
	if len(launcherGames) == 0 {
		return
	}

	x := float64(scoreboardX) * scale
	titleMidY := float64(panelTop+scoreboardHeaderH/2) * scale
	tdcgame.WriteAt(screen, "HIGH SCORES", x, titleMidY, float64(scoreboardTitleFontSize)*scale, neonCyan, text.AlignStart, text.AlignCenter)

	scores := topScores(launcherGames[l.selected].gameName)
	valueX := float64(scoreboardPanelX1-scoreboardValuePadRight) * scale

	// Always render every slot (filled or not) so the card reads as a
	// consistent podium rather than collapsing to a single placeholder line.
	for i := 0; i < scoreboardMaxRows; i++ {
		y := float64(scoreboardRowsStartY+i*scoreboardRowH-2) * scale
		clr := rankColor(i)
		rank := fmt.Sprintf("%d.", i+1)
		tdcgame.WriteAt(screen, rank, x, y, float64(scoreboardFontSize)*scale, clr, text.AlignStart, text.AlignStart)

		value, valueColor := "0", scoreboardEmptyColor
		if i < len(scores) {
			value, valueColor = fmt.Sprintf("%d", scores[i].Value), clr
		}
		tdcgame.WriteAt(screen, value, valueX, y, float64(scoreboardFontSize)*scale, valueColor, text.AlignEnd, text.AlignStart)

		// The name squeezes in between rank and score at a smaller size,
		// cut short when the card is too narrow for all of it.
		if i < len(scores) && scores[i].Name != "" {
			const nameFontSize, gap = 5.0, 3.0
			fontSize := float64(scoreboardFontSize) * scale
			nameX := x + tdcgame.MeasureWidth(rank, fontSize) + gap*scale
			avail := valueX - tdcgame.MeasureWidth(value, fontSize) - gap*scale - nameX
			name := scores[i].Name
			for name != "" && tdcgame.MeasureWidth(name, nameFontSize*scale) > avail {
				name = name[:len(name)-1]
			}
			tdcgame.WriteAt(screen, name, nameX, y+scale, nameFontSize*scale, clr, text.AlignStart, text.AlignStart)
		}
	}
}

const (
	footerBottomPad = 8
	footerFontSize  = 5
)

// toggleCRT flips a mode on/off: selecting the active option turns the CRT off,
// selecting the other switches to it (so Subtle/Full act as mutually exclusive
// on/off toggles). Shared with the options screen.
func toggleCRT(cur, clicked crtMode) crtMode {
	if cur == clicked {
		return crtOff
	}
	return clicked
}
