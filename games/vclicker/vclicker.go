package vclicker

import (
	"embed"
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

	particleLifetime = 1.2

	// Character rendering. The sprite is a 64x64 frame whose art sits ~15px
	// from the top; feet reach the frame bottom. Figures are anchored by their
	// feet on the floor line.
	spriteFrame   = 64.0
	spriteArtTop  = 15.0
	mainCharScale = 0.64

	// The jump is a parabolic hop with anticipation squash and apex stretch,
	// re-armed on every smash. Short and snappy so it keeps up with mashing.
	jumpDuration = 0.26
	jumpHeight   = 20.0

	// spawnCooldown throttles buzzword bursts. Every click still scores and
	// makes the guy hop, but words only spawn at most this often so fast
	// mashing doesn't bury the screen in unreadable text.
	spawnCooldown = 0.2

	floorY = float64(tdcgame.GroundY) // 186; feet line for the standup
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
	"No blockers", "LGTM", "Lunch soon",
	"MVP", "Post mortem", "Backlog", "Action items",
	"Basically done", "Q1", "Q4?", "Github down",
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

// figure is one colleague standing in the standup line. All are drawn from the
// single shared TDC sprite, differentiated by tint, horizontal flip and a
// desynced idle/bob phase so the crowd doesn't move in lockstep.
type figure struct {
	x, y  float64    // feet anchor in 426x240 viewport space
	scale float64    //
	phase float64    // idle-anim + bob phase offset (seconds)
	tint  [3]float32 // ColorScale RGB multipliers
	flip  bool
}

// VClicker implements TdcGameWithPlayer and GameWithCustomDraw.
// It takes full control of rendering via CustomDraw and manages its own
// 8-second button-mashing game loop, staged as a scrum standup: a line of
// colleagues with the TDC guy in the middle who jumps and shouts a buzzword
// on every smash.
type VClicker struct {
	state         gameState
	score         int
	timeLeft      float64
	createdAt     time.Time
	jumpTimer     float64 // counts a single hop down from jumpDuration
	spawnTimer    float64 // throttles buzzword bursts down from spawnCooldown
	gameOverDelay float64 // input grace period on the score screen
	particles     []particle
	rng           *rand.Rand
	viewport      *ebiten.Image

	// scene
	sheet     *tdcgame.SpriteSheet
	groundImg *ebiten.Image
	crowd     []figure
	animClock float64 // global idle time, advanced every frame

	// head anchor of the main character, updated each draw; particles emit here
	mainHeadX float64
	mainHeadY float64
}

func NewVClickerScene(assetsFS embed.FS) *VClicker {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	a := tdcgame.LoadAssets(assetsFS)
	v := &VClicker{
		state:     stateReady,
		timeLeft:  clickerDuration,
		rng:       rng,
		createdAt: time.Now(),
		sheet:     tdcgame.LoadSpriteSheet(a, 64, 64),
		groundImg: a.Sprites["ground"],
	}
	v.buildCrowd()
	return v
}

// buildCrowd lays out the colleagues as a straight line on the floor, leaving
// the center gap (x≈213) for the main character drawn separately. Mild scale,
// tint and flip variety makes them read as distinct people from one sprite.
func (v *VClicker) buildCrowd() {
	v.crowd = []figure{
		{x: 52, y: floorY, scale: 0.50, phase: 0.0, tint: [3]float32{1.05, 0.78, 0.72}, flip: false},
		{x: 110, y: floorY, scale: 0.52, phase: 1.3, tint: [3]float32{0.78, 0.90, 1.05}, flip: true},
		{x: 168, y: floorY, scale: 0.50, phase: 0.6, tint: [3]float32{0.85, 1.05, 0.80}, flip: false},
		{x: 258, y: floorY, scale: 0.50, phase: 2.1, tint: [3]float32{1.05, 1.00, 0.72}, flip: true},
		{x: 316, y: floorY, scale: 0.52, phase: 0.9, tint: [3]float32{0.90, 0.82, 1.05}, flip: false},
		{x: 374, y: floorY, scale: 0.50, phase: 2.6, tint: [3]float32{1.02, 0.86, 0.74}, flip: true},
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
// the GameRunner, starting after the player presses ENTER to begin.
func (v *VClicker) GetPlayerUpdateFunc() tdcgame.PlayerUpdate {
	return func(buttonpressed bool, dt float64, level tdcgame.Level, player *tdcgame.MovingSquare) {
		// The crowd idles on every screen.
		v.animClock += dt

		// First call = game starts
		if v.state == stateReady {
			v.state = statePlaying
			v.timeLeft = clickerDuration
		}
		// The score screen keeps animating while it swallows input.
		if v.state == stateGameOver {
			v.gameOverDelay = max(0, v.gameOverDelay-dt)
			if v.jumpTimer > 0 {
				v.jumpTimer = max(0, v.jumpTimer-dt)
			}
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
			v.jumpTimer = jumpDuration // re-arm the hop (retrigger, no stacking)
			if v.spawnTimer <= 0 {
				v.spawnParticles()
				v.spawnTimer = spawnCooldown
			}
		}

		if v.jumpTimer > 0 {
			v.jumpTimer = max(0, v.jumpTimer-dt)
		}
		if v.spawnTimer > 0 {
			v.spawnTimer = max(0, v.spawnTimer-dt)
		}

		v.updateParticles(dt)
	}
}

// — GameWithCustomDraw ————————————————————————————————————————————————————

// CustomDraw renders the entire scene at fixed 426×240 resolution and scales it
// up to the device-pixel screen. Sprites and background go into the low-res
// viewport; text and particles are drawn afterwards at device resolution so
// they stay crisp.
func (v *VClicker) CustomDraw(screen *ebiten.Image) {
	scale := float64(screen.Bounds().Dx()) / tdcgame.ScreenW
	if v.viewport == nil {
		v.viewport = ebiten.NewImage(tdcgame.ScreenW, tdcgame.ScreenH)
	}
	vp := v.viewport

	// ── Stage into the low-res viewport ────────────────────────────────────
	v.drawScene(vp)

	elapsed := time.Since(v.createdAt).Seconds()
	switch v.state {
	case statePlaying:
		v.drawTimerBar(vp, elapsed)
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
	// "SPRINT BOARD" header label sits on the whiteboard behind the standup.
	writeCentered(screen, "SPRINT BOARD", 43, 6, scale)

	switch v.state {
	case stateReady:
		writeCentered(screen, "SCRUM SMASHER 3000", 14, 14, scale)
		writeCentered(screen, "SMASH BUTTON TO SHOUT BUZZWORDS AT STANDUP!", tdcgame.ScreenH-30, 7, scale)
		writeCentered(screen, "PRESS BUTTON TO START", tdcgame.ScreenH-16, 8, scale)

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
			writeCentered(screen, "DOUBLE PRESS BUTTON TO REPLAY", tdcgame.ScreenH/2+30, 6, scale)
			writeCentered(screen, "OR PRESS ONCE TO RETURN", tdcgame.ScreenH/2+41, 6, scale)
		}
	}
}

// drawScene renders the standup backdrop and cast into the viewport: sky,
// sprint board, floor, the colleague line, then the jumping main character.
func (v *VClicker) drawScene(vp *ebiten.Image) {
	vp.Fill(color.RGBA{135, 206, 235, 255}) // same sky as the other games
	v.drawSprintBoard(vp)
	v.drawGround(vp)
	v.drawCrowd(vp)
	v.drawMainCharacter(vp)
}

// — internals ——————————————————————————————————————————————————————————————

// spawnParticles fires the buzzwords out in an upward fan from just above the
// jumper's head, so the words arc over and clear him instead of raining down
// and occluding the sprite mid-jump. Gravity in updateParticles brings them
// back down for a fountain effect.
func (v *VClicker) spawnParticles() {
	count := 1 + v.rng.Intn(2)
	const spawnClearance = 12.0 // lift the burst above the head
	originX := v.mainHeadX
	originY := v.mainHeadY - spawnClearance
	for i := range count {
		// Screen +y is down, so angles in (π, 2π) point upward. Spread them
		// evenly across that upper arc (left → up → right) with a little jitter.
		frac := (float64(i) + 0.5) / float64(count)
		angle := math.Pi + frac*math.Pi + (v.rng.Float64()-0.5)*0.35
		speed := 70.0 + v.rng.Float64()*70.0
		label := particleLabels[v.rng.Intn(len(particleLabels))]
		clr := particleColors[v.rng.Intn(len(particleColors))]
		v.particles = append(v.particles, particle{
			x:     originX,
			y:     originY,
			vx:    math.Cos(angle) * speed,
			vy:    math.Sin(angle) * speed, // negative → upward
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

// drawGround tiles the shared ground texture along the floor line.
func (v *VClicker) drawGround(vp *ebiten.Image) {
	tileW := float64(v.groundImg.Bounds().Dx())
	for x := 0.0; x < tdcgame.ScreenW; x += tileW {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x, floorY-tdcgame.GroundDrawOffset) // 180
		op.Filter = ebiten.FilterNearest
		vp.DrawImage(v.groundImg, op)
	}
}

// drawSprintBoard paints a simple whiteboard with sticky notes on the back
// wall to sell the standup. The "SPRINT BOARD" label is drawn later at device
// resolution so it stays legible.
func (v *VClicker) drawSprintBoard(vp *ebiten.Image) {
	bx, by, bw, bh := float32(150), float32(40), float32(126), float32(58)
	// legs
	vector.FillRect(vp, bx+14, by+bh, 4, 42, color.RGBA{120, 120, 130, 255}, false)
	vector.FillRect(vp, bx+bw-18, by+bh, 4, 42, color.RGBA{120, 120, 130, 255}, false)
	// frame + board
	vector.FillRect(vp, bx-2, by-2, bw+4, bh+4, color.RGBA{90, 90, 100, 255}, false)
	vector.FillRect(vp, bx, by, bw, bh, color.RGBA{235, 238, 240, 255}, false)
	// header strip for the "SPRINT BOARD" label (text drawn later at device res)
	vector.FillRect(vp, bx, by, bw, 16, color.RGBA{90, 90, 100, 255}, false)
	// one row of sticky notes below the header
	notes := []color.RGBA{
		{255, 214, 90, 255}, {255, 150, 160, 255},
		{150, 220, 150, 255}, {150, 200, 255, 255},
	}
	for i, n := range notes {
		nx := bx + 12 + float32(i)*28
		ny := by + 26
		vector.FillRect(vp, nx, ny, 18, 18, n, false)
	}
}

// drawFigure draws one figure from the shared sprite, feet-anchored on its
// floor position, with idle animation, a gentle bob, optional flip and a tint.
func (v *VClicker) drawFigure(vp *ebiten.Image, x, y, scale, phase, yOff, sx, sy float64, tint [3]float32, flip bool) {
	frame := v.sheet.Frame(20 + int((v.animClock+phase)*6)%10) // 10 idle frames @6fps
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterNearest
	if flip {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(spriteFrame, 0)
	}
	op.GeoM.Scale(scale*sx, scale*sy)
	// Feet-anchor: frame bottom lands on y. Scale from the ground up so squash
	// keeps the feet planted.
	op.GeoM.Translate(x-spriteFrame*scale*sx/2, (y+yOff)-spriteFrame*scale*sy)
	op.ColorScale.Scale(tint[0], tint[1], tint[2], 1)
	vp.DrawImage(frame, op)
}

func (v *VClicker) drawCrowd(vp *ebiten.Image) {
	for _, f := range v.crowd {
		bob := math.Sin((v.animClock+f.phase)*3) * 1.0
		v.drawFigure(vp, f.x, f.y, f.scale, f.phase, bob, 1, 1, f.tint, f.flip)
	}
}

// drawMainCharacter draws the TDC guy in the center, in front of the crowd,
// hopping when jumpTimer is active. Also records the head anchor for particles.
func (v *VClicker) drawMainCharacter(vp *ebiten.Image) {
	yOff, sx, sy := 0.0, 1.0, 1.0
	if v.jumpTimer > 0 {
		t := 1.0 - v.jumpTimer/jumpDuration
		yOff = -jumpHeight * math.Sin(t*math.Pi) // 0 at both ends, peak mid
		st := math.Sin(t * math.Pi)
		sy = 1.0 + 0.14*st // stretch at apex
		sx = 1.0 - 0.10*st // narrow at apex
		if t < 0.15 {      // brief anticipation squash on takeoff
			k := t / 0.15
			sy = 1 - 0.12*(1-k)
			sx = 1 + 0.10*(1-k)
		}
	}

	const centerX = float64(tdcgame.ScreenW) / 2 // 213
	v.drawFigure(vp, centerX, floorY, mainCharScale, 0, yOff, sx, sy,
		[3]float32{1.08, 1.08, 1.02}, false)

	v.mainHeadX = centerX
	v.mainHeadY = (floorY + yOff) - (spriteFrame-spriteArtTop)*mainCharScale
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
