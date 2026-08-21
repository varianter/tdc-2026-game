package vclicker

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"variant.dev/tdcgame/tdcgame"
)

const (
	clickerDuration = 8.0

	// gameOverInputDelay is a grace period after time runs out during which
	// button presses are ignored. Without it, a player still mashing at the
	// final tick blows straight through the score screen and back to the
	// launcher without ever reading it.
	gameOverInputDelay = 1.25

	btnW             = 70.0
	btnH             = 50.0
	particleLifetime = 1.2
)

type gameState int

const (
	stateReady gameState = iota
	statePlaying
	stateGameOver
)

type particle struct {
	x, y   float64
	vx, vy float64
	life   float64
	label  string
	clr    color.RGBA
}

var particleLabels = []string{
	"</>", "{}", "PR", "UX", "MVP", "Rust",
	"git", "// TODO", "BUG", "YAGNI", "AI",
	"nil", "WIP", "API", "404", "EOF",
	"Jira", "LGTM", "YOLO", "cookie", "npm",
	"deploy", "figma", "scrum", "agile",
}

var particleColors = []color.RGBA{
	{255, 60, 60, 255},
	{60, 255, 100, 255},
	{60, 140, 255, 255},
	{255, 255, 60, 255},
	{255, 60, 220, 255},
	{60, 240, 240, 255},
	{255, 160, 40, 255},
}

// VClicker implements TdcGameWithPlayer and GameWithCustomDraw.
// It takes full control of rendering via CustomDraw and manages
// its own 15-second button-mashing game loop.
type VClicker struct {
	state       gameState
	score       int
	timeLeft    float64
	createdAt   time.Time
	btnX, btnY  float64
	phaseX      float64
	phaseY      float64
	squishTimer float64
	// gameOverDelay counts down the input grace period on the score screen.
	gameOverDelay float64
	particles     []particle
	rng           *rand.Rand
	viewport      *ebiten.Image
}

func NewVClickerScene() *VClicker {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &VClicker{
		state:     stateReady,
		timeLeft:  clickerDuration,
		btnX:      tdcgame.ScreenW / 2.0,
		btnY:      tdcgame.ScreenH / 2.0,
		phaseX:    rng.Float64() * 2 * math.Pi,
		phaseY:    rng.Float64() * 2 * math.Pi,
		rng:       rng,
		createdAt: time.Now(),
	}
}

// — TdcGame ——————————————————————————————————————————————————————————————

func (v *VClicker) GetGameObjects() []tdcgame.GameObject { return nil }

func (v *VClicker) GetGameParameters() *tdcgame.GameParameters {
	return &tdcgame.GameParameters{ShouldCameraFollowPlayer: false}
}

// GetGameState holds off reporting the game as over until the grace period has
// elapsed. The framework's GameRunner acts on the button the moment it sees
// GameOver, so staying "running" for a beat is what keeps stray mashing from
// skipping the score screen. The screen itself is already showing, because
// CustomDraw keys off the internal state rather than this.
func (v *VClicker) GetGameState() tdcgame.GameState {
	if v.state == stateGameOver && v.gameOverDelay <= 0 {
		return tdcgame.GameOver
	}
	return tdcgame.Running
}

func (v *VClicker) GetCurrentScore() int { return v.score }

// — TdcGameWithPlayer —————————————————————————————————————————————————————

// GetPlayerUpdateFunc returns the game-logic callback invoked every frame by
// the GameRunner, starting after the player presses SPACE to begin.
func (v *VClicker) GetPlayerUpdateFunc() tdcgame.PlayerUpdate {
	return func(buttonpressed bool, dt float64, level tdcgame.Level, player *tdcgame.MovingSquare) {
		// First call = game starts
		if v.state == stateReady {
			v.state = statePlaying
			v.timeLeft = clickerDuration
		}
		// The score screen keeps animating while it swallows input.
		if v.state == stateGameOver {
			v.gameOverDelay = max(0, v.gameOverDelay-dt)
			v.updateParticles(dt)
			return
		}
		if v.state != statePlaying {
			return
		}

		v.timeLeft -= dt
		if v.timeLeft <= 0 {
			v.timeLeft = 0
			v.state = stateGameOver
			v.gameOverDelay = gameOverInputDelay
			return
		}

		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			v.score++
			v.squishTimer = 0.15
			v.spawnParticles()
		}

		if v.squishTimer > 0 {
			v.squishTimer = max(0, v.squishTimer-dt)
		}

		speedMult := 1.0
		if v.timeLeft < 5.0 {
			speedMult = 1.0 + 1.5*(1.0-v.timeLeft/5.0)
		}
		v.phaseX += 1.8 * speedMult * dt
		v.phaseY += 1.3 * speedMult * dt
		v.btnX = float64(tdcgame.ScreenW)/2.0 + 80.0*math.Sin(v.phaseX)
		v.btnY = float64(tdcgame.ScreenH)/2.0 + 55.0*math.Sin(v.phaseY)

		v.updateParticles(dt)
	}
}

// — GameWithCustomDraw ————————————————————————————————————————————————————

// CustomDraw renders the entire V-Clicker UI at fixed 426×240 resolution
// and scales it up to the device-pixel screen.
func (v *VClicker) CustomDraw(screen *ebiten.Image) {
	scale := float64(screen.Bounds().Dx()) / tdcgame.ScreenW
	if v.viewport == nil {
		v.viewport = ebiten.NewImage(tdcgame.ScreenW, tdcgame.ScreenH)
	}
	vp := v.viewport

	vp.Fill(color.RGBA{10, 10, 22, 255})

	// Pixel-grid overlay for 8-bit feel
	for x := 0; x < tdcgame.ScreenW; x += 4 {
		vector.FillRect(vp, float32(x), 0, 1, float32(tdcgame.ScreenH), color.RGBA{20, 20, 35, 255}, false)
	}
	for y := 0; y < tdcgame.ScreenH; y += 4 {
		vector.FillRect(vp, 0, float32(y), float32(tdcgame.ScreenW), 1, color.RGBA{20, 20, 35, 255}, false)
	}

	elapsed := time.Since(v.createdAt).Seconds()
	pulse := 1.0 + 0.06*math.Sin(elapsed*3.0)

	// ── Shapes into the low-res viewport ───────────────────────────────────
	btnPulse := -1.0 // <0 means no button label this frame
	switch v.state {
	case stateReady:
		v.drawButton(vp, pulse)
		btnPulse = pulse

	case statePlaying:
		v.drawTimerBar(vp, elapsed)
		v.drawButton(vp, 1.0)
		btnPulse = 1.0

	case stateGameOver:
		vector.FillRect(vp, float32(tdcgame.ScreenW/2)-110, float32(tdcgame.ScreenH/2)-42,
			220, 95, color.RGBA{10, 10, 22, 210}, false)
		drawBorder(vp, float32(tdcgame.ScreenW/2)-110, float32(tdcgame.ScreenH/2)-42, 220, 95)
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(vp, op)

	// ── Text at device resolution so it stays crisp ────────────────────────
	switch v.state {
	case stateReady:
		writeCentered(screen, "V-CLICKER", tdcgame.ScreenH/2-45, 14, scale)
		writeCentered(screen, "PRESS THE BIG RED BUTTON AS FAST AS YOU CAN!", tdcgame.ScreenH/2+38, 7, scale)
		writeCentered(screen, "PRESS THE BIG RED BUTTON TO START", tdcgame.ScreenH-18, 8, scale)

	case statePlaying:
		v.drawParticles(screen, scale)
		writeRight(screen, fmt.Sprintf("%d", v.score), tdcgame.ScreenW-6, 6, 16, scale)
		writeCentered(screen, fmt.Sprintf("%.1f", v.timeLeft), 6, 10, scale)

	case stateGameOver:
		v.drawParticles(screen, scale)
		cps := float64(v.score) / clickerDuration
		writeCentered(screen, "TIME'S UP!", tdcgame.ScreenH/2-34, 16, scale)
		writeCentered(screen, fmt.Sprintf("SCORE: %d", v.score), tdcgame.ScreenH/2-10, 14, scale)
		writeCentered(screen, fmt.Sprintf("%.1f CLICKS/SEC", cps), tdcgame.ScreenH/2+12, 10, scale)
		if v.gameOverDelay <= 0 {
			writeCentered(screen, "PRESS THE BIG RED BUTTON TO RETURN", tdcgame.ScreenH/2+32, 6, scale)
		}
	}

	if btnPulse > 0 {
		tdcgame.WriteAt(screen, "V", v.btnX*scale, (v.btnY-2)*scale, 24*btnPulse*scale,
			color.RGBA{230, 210, 255, 255}, text.AlignCenter, text.AlignCenter)
	}
}

// — internals ——————————————————————————————————————————————————————————————

func (v *VClicker) spawnParticles() {
	count := 4 + v.rng.Intn(3)
	for i := range count {
		angle := 2*math.Pi*float64(i)/float64(count) + (v.rng.Float64()-0.5)*0.8
		speed := 70.0 + v.rng.Float64()*70.0
		label := particleLabels[v.rng.Intn(len(particleLabels))]
		clr := particleColors[v.rng.Intn(len(particleColors))]
		v.particles = append(v.particles, particle{
			x:     v.btnX,
			y:     v.btnY,
			vx:    math.Cos(angle) * speed,
			vy:    math.Sin(angle) * speed,
			life:  particleLifetime,
			label: label,
			clr:   clr,
		})
	}
}

func (v *VClicker) updateParticles(dt float64) {
	alive := v.particles[:0]
	for i := range v.particles {
		p := &v.particles[i]
		p.x += p.vx * dt
		p.y += p.vy * dt
		p.vy += 50.0 * dt
		p.life -= dt
		if p.life > 0 {
			alive = append(alive, *p)
		}
	}
	v.particles = alive
}

func (v *VClicker) drawButton(vp *ebiten.Image, pulse float64) {
	squish := 0.0
	if v.squishTimer > 0 {
		t := 1.0 - (v.squishTimer / 0.15)
		squish = math.Sin(t * math.Pi)
	}

	w := (btnW*(1.0+0.22*squish) + 4) * pulse
	h := (btnH*(1.0-0.22*squish) + 4) * pulse
	x := float32(v.btnX - w/2)
	y := float32(v.btnY - h/2)

	glowAlpha := uint8(160 + 60*math.Sin(time.Since(v.createdAt).Seconds()*4))
	vector.FillRect(vp, x-2, y-2, float32(w)+4, float32(h)+4,
		color.RGBA{160, 100, 255, glowAlpha}, false)

	btnColor := color.RGBA{70, 30, 160, 255}
	if v.squishTimer > 0 {
		btnColor = color.RGBA{100, 50, 210, 255}
	}
	vector.FillRect(vp, x, y, float32(w), float32(h), btnColor, false)
	vector.FillRect(vp, x+2, y+2, float32(w)-4, 4, color.RGBA{140, 100, 220, 180}, false)
	// The "V" label is drawn at device resolution in CustomDraw (crisp text).
}

func (v *VClicker) drawTimerBar(vp *ebiten.Image, elapsed float64) {
	vector.FillRect(vp, 0, 0, float32(tdcgame.ScreenW), 5, color.RGBA{30, 30, 50, 255}, false)
	barW := float32(tdcgame.ScreenW) * float32(v.timeLeft/clickerDuration)
	barColor := color.RGBA{60, 220, 100, 255}
	if v.timeLeft < 5.0 {
		flash := uint8(180 + 75*math.Sin(elapsed*12))
		barColor = color.RGBA{flash, 40, 40, 255}
	}
	vector.FillRect(vp, 0, 0, barW, 5, barColor, false)
}

// drawParticles renders particle labels at device resolution. Viewport-space
// coordinates and sizes are multiplied by scale to stay crisp.
func (v *VClicker) drawParticles(screen *ebiten.Image, scale float64) {
	for _, p := range v.particles {
		t := p.life / particleLifetime
		size := (6.0 + 3.0*t) * scale
		tdcgame.WriteAt(screen, p.label, p.x*scale, p.y*scale, size,
			color.RGBA{p.clr.R, p.clr.G, p.clr.B, uint8(t * 220)},
			text.AlignCenter, text.AlignCenter)
	}
}

// writeCentered draws msg horizontally centered on the screen. y/size are in
// viewport space and scaled to device pixels.
func writeCentered(screen *ebiten.Image, msg string, y, size int, scale float64) {
	tdcgame.WriteAt(screen, msg, float64(tdcgame.ScreenW)/2*scale, float64(y)*scale, float64(size)*scale,
		color.White, text.AlignCenter, text.AlignStart)
}

// writeRight draws msg right-aligned at viewport-space x,y (scaled to device pixels).
func writeRight(screen *ebiten.Image, msg string, x, y, size int, scale float64) {
	tdcgame.WriteAt(screen, msg, float64(x)*scale, float64(y)*scale, float64(size)*scale,
		color.White, text.AlignEnd, text.AlignStart)
}

func drawBorder(screen *ebiten.Image, x, y, w, h float32) {
	clr := color.RGBA{120, 80, 200, 200}
	vector.FillRect(screen, x, y, w, 2, clr, false)
	vector.FillRect(screen, x, y+h-2, w, 2, clr, false)
	vector.FillRect(screen, x, y, 2, h, clr, false)
	vector.FillRect(screen, x+w-2, y, 2, h, clr, false)
}
