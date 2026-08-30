package bounce

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	bounceassets "variant.dev/tdcgame/games/bounce/assets"
	"variant.dev/tdcgame/tdcgame"
)

const (
	screenW = 426
	screenH = 240
	// The player is drawn 40% larger than the original 32px.
	playerDrawSize = 45

	// The sprite frame is mostly empty above the head: the artwork occupies
	// rows 15..63 of every 64px frame (measured from assets/tdcgjenger.png).
	// Bouncing and collisions work off the artwork box rather than the frame,
	// so the player rebounds exactly when their head meets the canopy instead
	// of a head's height short of it.
	spriteFrameSize = 64.0
	spriteArtTop    = 15.0
	playerBodyOffY  = spriteArtTop / spriteFrameSize * playerDrawSize
	playerBodyH     = (spriteFrameSize - spriteArtTop) / spriteFrameSize * playerDrawSize

	// The hitbox is the full height of the body but only the torso's width, so
	// swinging arms and legs don't count as a hit.
	playerHitW    = 22.5
	playerHitOffX = 11.25
	playerHitOffY = playerBodyOffY
	playerHitH    = playerBodyH

	// wallThickness is both the collision inset and the height of the canopy /
	// bounce-mattress bands, so what you see is exactly what you bounce off.
	wallThickness = 14

	// fieldTop/fieldBottom bound the visible arena (used for spawning);
	// playTop/playBottom bound the sprite's top-left corner so that the body
	// inside it stays within the arena.
	fieldTop    = float64(wallThickness)
	fieldBottom = float64(screenH - wallThickness)
	playTop     = fieldTop - playerBodyOffY
	playBottom  = fieldBottom - playerBodyOffY - playerBodyH

	baseScrollSpeed = 60.0
	maxScrollSpeed  = 250.0
	// speedIncreaseRate ramps the scroll toward maxScrollSpeed as you travel.
	// At 1.6 the cap was ~90s away, i.e. barely reachable inside a run; this
	// gets there in around 40s so the run actually builds to top speed.
	speedIncreaseRate = 3.5
	// Dashing also drags the world past a little faster while you hold it, so
	// the lunge feels like it covers ground rather than just sliding right.
	chargeBoostsScroll   = true
	chargeScrollMultiMax = 1.4
	chargeScrollDrag     = 0.985

	// The player bounces between canopy and mattress on its own, faster the
	// further you get. Vertical movement isn't something you steer directly.
	bounceSpeedBase = 90.0
	bounceSpeedMax  = 330.0
	bounceSpeedRamp = 0.042 // extra px/s of bounce per metre travelled

	// Holding the button shoots you forward through the castle; letting go
	// drifts you back to where you started. That forward lunge is how you pick
	// which gap in the boulders you arrive at.
	homeX        = 50.0
	dashMaxX     = 190.0
	dashAccel    = 520.0
	dashMaxSpeed = 300.0
	dashReturn   = 2.4 // how briskly you drift home once you let go

	obstacleBaseInterval = 280.0
	obstacleMinInterval  = 70.0

	doubleTapEnabled = false
	doubleTapWindow  = 0.25

	maxSwitches        = 3
	switchRechargeTime = 5.0

	// Air pressure is the dash's fuel. It drains while you hold the button and
	// only refills once you let go, so the button can't simply be held down:
	// the game is a rhythm of lunges and drifting back.
	pumpMax          = 2.6
	pumpRefillRate   = 0.6
	pumpRestartLevel = 0.8

	// distanceScoreDivisor turns metres travelled into points, so staying alive
	// is what scores rather than farming powerups.
	distanceScoreDivisor = 20.0
	powerupScore         = 5
	smashScore           = 20

	// A shield is a fixed budget: picking one up stacks time up to a cap, and
	// smashing a boulder spends some of it. It can never extend itself.
	shieldDuration      = 5.0
	shieldStackDuration = 3.0
	shieldMaxDuration   = 10.0
	shieldSmashCost     = 1.5

	powerupDrawSize     = 20.0
	powerupCollideSize  = 20.0
	powerupBaseInterval = 600.0
	powerupMinInterval  = 300.0
	powerupMargin       = 8.0
)

type obstacle struct {
	x, y, w, h float64
	// seed drives the boulder's cracks so each rock looks different but stays
	// stable from frame to frame.
	seed uint32
	// left and right are the drawn silhouette's half-widths at each facet
	// corner, computed once at spawn. Drawing and collision both read them, so
	// the rock you see is exactly the rock you hit.
	left, right boulderChain
}

func newObstacle(x, y, w, h float64) obstacle {
	seed := rand.Uint32()
	return obstacle{
		x: x, y: y, w: w, h: h,
		seed:  seed,
		left:  boulderEdge(w/2, h, seed, 0),
		right: boulderEdge(w/2, h, seed, 64),
	}
}

// rows is the number of 1px scanlines the boulder is drawn with; collision
// walks the same rows.
func (o obstacle) rows() int {
	r := int(math.Ceil(o.h))
	if r < 2 {
		r = 2
	}
	return r
}

// spanAt returns the boulder's drawn horizontal extent [x0, x1) on scanline i,
// which covers world rows [o.y+i, o.y+i+1).
func (o obstacle) spanAt(i int) (x0, x1 float64) {
	half := o.w / 2
	left := math.Max(1, o.left.at(i, o.rows()))
	right := math.Max(1, o.right.at(i, o.rows()))
	return o.x + half - left, o.x + half + right
}

// hits reports whether the player's hitbox touches the boulder's actual
// silhouette rather than its bounding box, so clearing a rock's notched corner
// by a pixel really does clear it.
func (o obstacle) hits(px, py, pw, ph float64) bool {
	if px >= o.x+o.w || px+pw <= o.x || py >= o.y+o.h || py+ph <= o.y {
		return false
	}
	rows := o.rows()
	i0 := int(math.Floor(py - o.y))
	i1 := int(math.Ceil(py+ph-o.y)) - 1
	if i0 < 0 {
		i0 = 0
	}
	if i1 > rows-1 {
		i1 = rows - 1
	}
	for i := i0; i <= i1; i++ {
		x0, x1 := o.spanAt(i)
		if px < x1 && px+pw > x0 {
			return true
		}
	}
	return false
}

type powerupKind int

const (
	powerupShield powerupKind = iota
	powerupAntiShield
)

type powerup struct {
	x, y float64
	kind powerupKind
}

type particle struct {
	x, y    float64
	vx, vy  float64
	life    float64
	r, g, b uint8
	size    float64
}

type Game struct {
	playerAnim     *tdcgame.Animation
	sokkerImg      *ebiten.Image
	sokkerGlow     *ebiten.Image
	antiSokkerImg  *ebiten.Image
	antiSokkerGlow *ebiten.Image

	wallImg   *ebiten.Image
	topImg    *ebiten.Image
	bottomImg *ebiten.Image
	turretImg *ebiten.Image

	cameraX  float64
	playerY  float64
	playerVY float64

	// playerX is the player's on-screen x: homeX at rest, further right while
	// dashing. bounceSpeed is the current automatic vertical speed.
	playerX     float64
	playerVX    float64
	bounceSpeed float64

	scrollSpeed float64
	distance    float64
	score       int
	bonusScore  int

	chargeTime  float64
	scrollBoost float64

	// pressure is the air left in the pump; pumpLocked is set when it runs dry
	// and cleared once it has refilled past pumpRestartLevel with the button
	// released.
	pressure   float64
	pumpLocked bool

	// promptTime animates the start prompt. Update isn't called until the
	// player presses the button, so the prompt can't lean on timeSinceStart.
	promptTime float64

	// started flips on the first Update. The framework's GameRunner freezes the
	// game (and so never calls Update) until the player presses the button, so
	// this doubles as "the player has pressed the button".
	started bool

	obstacles     []obstacle
	nextObstacleX float64

	powerups     []powerup
	nextPowerupX float64

	shieldTimer  float64
	hasHadShield bool

	lastReleaseTime float64
	timeSinceStart  float64
	spaceWasPressed bool

	switchesLeft   int
	switchRecharge float64

	particles []particle

	GameOver bool

	viewport *ebiten.Image
}

func New(assets embed.FS) *Game {
	playerImg, _, err := ebitenutil.NewImageFromFileSystem(assets, "assets/tdcgjenger.png")
	if err != nil {
		panic(err)
	}
	sheet := &tdcgame.SpriteSheet{
		Image:   playerImg,
		FrameW:  64,
		FrameH:  64,
		Columns: playerImg.Bounds().Dx() / 64,
	}
	anim := &tdcgame.Animation{
		Sheet:  sheet,
		Frames: []int{30, 31, 32, 33, 34, 35},
		FPS:    12,
	}

	sokImg := decodeImage(bounceassets.Sokker)
	antiSokImg := decodeImage(bounceassets.AntiSokker)

	initSounds()

	return &Game{
		playerAnim:     anim,
		sokkerImg:      ebiten.NewImageFromImage(sokImg),
		sokkerGlow:     buildSokkerGlow(sokImg, int(powerupDrawSize)),
		antiSokkerImg:  ebiten.NewImageFromImage(antiSokImg),
		antiSokkerGlow: buildSokkerGlow(antiSokImg, int(powerupDrawSize)),
		wallImg:        ebiten.NewImageFromImage(decodeImage(bounceassets.CastleWall)),
		topImg:         ebiten.NewImageFromImage(decodeImage(bounceassets.CastleTop)),
		bottomImg:      ebiten.NewImageFromImage(decodeImage(bounceassets.CastleBottom)),
		turretImg:      ebiten.NewImageFromImage(decodeImage(bounceassets.CastleTurret)),
		playerY:        float64(screenH)/2 - float64(playerDrawSize)/2,
		playerVY:       -bounceSpeedBase,
		playerX:        homeX,
		bounceSpeed:    bounceSpeedBase,
		scrollSpeed:    baseScrollSpeed,
		nextObstacleX:  float64(screenW) + 100,
		nextPowerupX:   float64(screenW) + 400,
		switchesLeft:   maxSwitches,
		pressure:       pumpMax,
	}
}

func decodeImage(data []byte) image.Image {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		panic(err)
	}
	return img
}

func (g *Game) Update(dt float64, spacePressed bool, spaceJustPressed bool) {
	if g.GameOver {
		return
	}

	g.started = true
	g.timeSinceStart += dt
	g.playerAnim.Update(dt)
	g.updateParticles(dt)

	if doubleTapEnabled {
		if g.switchesLeft < maxSwitches {
			g.switchRecharge += dt
			if g.switchRecharge >= switchRechargeTime {
				g.switchRecharge -= switchRechargeTime
				g.switchesLeft++
			}
		}

		if spaceJustPressed {
			if g.timeSinceStart-g.lastReleaseTime < doubleTapWindow && g.lastReleaseTime > 0 && g.switchesLeft > 0 {
				g.playerVY = -g.playerVY
				g.switchesLeft--
				g.switchRecharge = 0
			}
		}
		if g.spaceWasPressed && !spacePressed {
			g.lastReleaseTime = g.timeSinceStart
		}
		g.spaceWasPressed = spacePressed
	}

	// Dashing only works while there's pressure left, and pressure only comes
	// back once the button is released — so holding it down forever does
	// nothing but empty the tank.
	dashing := spacePressed && !g.pumpLocked && g.pressure > 0
	if dashing {
		g.pressure -= dt
		if g.pressure <= 0 {
			g.pressure = 0
			g.pumpLocked = true
		}
		g.chargeTime += dt
		g.playerVX = math.Min(dashMaxSpeed, g.playerVX+dashAccel*dt)
		g.playerX = math.Min(dashMaxX, g.playerX+g.playerVX*dt)
	} else {
		g.chargeTime = 0
		g.playerVX = 0
		// Ease back to the starting position rather than snapping.
		g.playerX += (homeX - g.playerX) * math.Min(1, dashReturn*dt)
		if math.Abs(g.playerX-homeX) < 0.05 {
			g.playerX = homeX
		}
		if !spacePressed {
			g.pressure = math.Min(pumpMax, g.pressure+pumpRefillRate*dt)
			if g.pumpLocked && g.pressure >= pumpRestartLevel {
				g.pumpLocked = false
			}
		}
	}

	// The bounce runs itself, quickening as the run goes on.
	g.bounceSpeed = math.Min(bounceSpeedMax, bounceSpeedBase+g.distance*bounceSpeedRamp)
	if g.playerVY >= 0 {
		g.playerVY = g.bounceSpeed
	} else {
		g.playerVY = -g.bounceSpeed
	}

	g.playerY += g.playerVY * dt

	if g.playerY < playTop {
		g.playerY = playTop
		g.playerVY = -g.playerVY
		g.onBounce()
	}
	if g.playerY > playBottom {
		g.playerY = playBottom
		g.playerVY = -g.playerVY
		g.onBounce()
	}

	g.scrollSpeed = math.Min(maxScrollSpeed, baseScrollSpeed+g.distance*speedIncreaseRate/100)
	if chargeBoostsScroll {
		if g.chargeTime > 0 {
			targetBoost := math.Min(chargeScrollMultiMax-1.0, g.chargeTime*0.4)
			g.scrollBoost += (targetBoost - g.scrollBoost) * dt * 3
		} else {
			g.scrollBoost *= math.Pow(chargeScrollDrag, 60*dt)
			if g.scrollBoost < 0.001 {
				g.scrollBoost = 0
			}
		}
	}
	effectiveScroll := g.scrollSpeed * (1.0 + g.scrollBoost)
	g.cameraX += effectiveScroll * dt
	g.distance += effectiveScroll * dt
	g.score = g.bonusScore + int(g.distance/distanceScoreDivisor)

	if g.shieldTimer > 0 {
		g.shieldTimer -= dt
	}

	g.spawnObstacles()
	g.spawnPowerups()
	g.pruneObstacles()
	g.prunePowerups()
	g.collectPowerups()
	g.checkCollisions()
}

func (g *Game) spawnObstacles() {
	for g.cameraX+float64(screenW)+150 > g.nextObstacleX {
		interval := math.Max(obstacleMinInterval, obstacleBaseInterval-g.distance/15)

		fieldH := fieldBottom - fieldTop
		h := 15 + rand.Float64()*math.Min(70, 20+g.distance/200)
		w := 15 + rand.Float64()*40
		y := fieldTop + rand.Float64()*(fieldH-h)

		g.obstacles = append(g.obstacles, newObstacle(g.nextObstacleX, y, w, h))

		if g.distance > 2000 && rand.Float64() < math.Min(0.5, g.distance/10000) {
			h2 := 15 + rand.Float64()*40
			y2 := fieldTop + rand.Float64()*(fieldH-h2)
			g.obstacles = append(g.obstacles, newObstacle(g.nextObstacleX+w+5, y2, w*0.7, h2))
		}

		g.nextObstacleX += interval + rand.Float64()*interval*0.4
	}
}

func (g *Game) overlapsAnyObstacle(px, py, pw, ph float64) bool {
	for _, o := range g.obstacles {
		if px < o.x+o.w+powerupMargin && px+pw+powerupMargin > o.x &&
			py < o.y+o.h+powerupMargin && py+ph+powerupMargin > o.y {
			return true
		}
	}
	return false
}

func (g *Game) spawnPowerups() {
	for g.cameraX+float64(screenW)+150 > g.nextPowerupX {
		fieldH := fieldBottom - fieldTop
		x := g.nextPowerupX

		kind := powerupShield
		antiChance := math.Min(0.6, 0.15+(g.distance-3000)/20000)
		if g.hasHadShield && g.distance > 3000 && rand.Float64() < antiChance {
			kind = powerupAntiShield
		}

		placed := false
		for attempt := 0; attempt < 10; attempt++ {
			y := fieldTop + rand.Float64()*(fieldH-powerupCollideSize)
			if !g.overlapsAnyObstacle(x, y, powerupCollideSize, powerupCollideSize) {
				g.powerups = append(g.powerups, powerup{x: x, y: y, kind: kind})
				placed = true
				break
			}
		}
		if !placed {
			y := fieldTop + rand.Float64()*(fieldH-powerupCollideSize)
			g.powerups = append(g.powerups, powerup{x: x + 40, y: y, kind: kind})
		}

		interval := math.Max(powerupMinInterval, powerupBaseInterval-g.distance/30)
		g.nextPowerupX += interval + rand.Float64()*interval*0.5
	}
}

func (g *Game) pruneObstacles() {
	alive := g.obstacles[:0]
	for _, o := range g.obstacles {
		if o.x+o.w > g.cameraX-50 {
			alive = append(alive, o)
		}
	}
	g.obstacles = alive
}

func (g *Game) prunePowerups() {
	alive := g.powerups[:0]
	for _, p := range g.powerups {
		if p.x+powerupCollideSize > g.cameraX-50 {
			alive = append(alive, p)
		}
	}
	g.powerups = alive
}

func (g *Game) collectPowerups() {
	px := g.cameraX + g.playerX + playerHitOffX
	py := g.playerY + playerHitOffY

	alive := g.powerups[:0]
	for _, p := range g.powerups {
		if px < p.x+powerupCollideSize && px+playerHitW > p.x && py < p.y+powerupCollideSize && py+playerHitH > p.y {
			g.applyPowerup(p)
			g.bonusScore += powerupScore
			g.spawnCollectParticles(p.x-g.cameraX+powerupCollideSize/2, p.y+powerupCollideSize/2)
		} else {
			alive = append(alive, p)
		}
	}
	g.powerups = alive
}

func (g *Game) applyPowerup(p powerup) {
	switch p.kind {
	case powerupShield:
		g.hasHadShield = true
		if g.shieldTimer > 0 {
			g.shieldTimer = math.Min(shieldMaxDuration, g.shieldTimer+shieldStackDuration)
		} else {
			g.shieldTimer = shieldDuration
		}
		g.playPickup()
	case powerupAntiShield:
		if g.shieldTimer > 0 {
			g.shieldTimer = 0
		}
		g.playAntiShield()
	}
}

func (g *Game) checkCollisions() {
	px := g.cameraX + g.playerX + playerHitOffX
	py := g.playerY + playerHitOffY

	alive := g.obstacles[:0]
	for _, o := range g.obstacles {
		if o.hits(px, py, playerHitW, playerHitH) {
			if g.shieldTimer > 0 {
				// Smashing spends shield time rather than granting it, so a
				// shield is a budget of a few boulders, not a self-feeding loop.
				g.spawnDestroyParticles(o)
				g.bonusScore += smashScore
				g.shieldTimer = math.Max(0, g.shieldTimer-shieldSmashCost)
				g.playSmash()
				continue
			}
			g.GameOver = true
			alive = append(alive, o)
			break
		}
		alive = append(alive, o)
	}
	if g.GameOver {
		return
	}
	g.obstacles = alive
}

func (g *Game) spawnDestroyParticles(o obstacle) {
	cx := o.x - g.cameraX + o.w/2
	cy := o.y + o.h/2
	count := 12 + rand.Intn(8)
	for i := 0; i < count; i++ {
		angle := rand.Float64() * 2 * math.Pi
		speed := 40 + rand.Float64()*120
		// Grey rubble rather than sparks, so a shattered boulder reads as stone.
		shade := uint8(90 + rand.Intn(90))
		g.particles = append(g.particles, particle{
			x:    cx + rand.Float64()*o.w*0.5 - o.w*0.25,
			y:    cy + rand.Float64()*o.h*0.5 - o.h*0.25,
			vx:   math.Cos(angle) * speed,
			vy:   math.Sin(angle) * speed,
			life: 0.4 + rand.Float64()*0.4,
			r:    shade,
			g:    shade - uint8(rand.Intn(10)),
			b:    shade + uint8(rand.Intn(14)),
			size: 2 + rand.Float64()*3,
		})
	}
}

func (g *Game) spawnCollectParticles(cx, cy float64) {
	for i := 0; i < 10; i++ {
		angle := rand.Float64() * 2 * math.Pi
		speed := 30 + rand.Float64()*80
		g.particles = append(g.particles, particle{
			x: cx, y: cy,
			vx:   math.Cos(angle) * speed,
			vy:   math.Sin(angle) * speed,
			life: 0.3 + rand.Float64()*0.3,
			r:    50, g: 180, b: 255,
			size: 2 + rand.Float64()*2,
		})
	}
}

func (g *Game) updateParticles(dt float64) {
	alive := g.particles[:0]
	for i := range g.particles {
		p := &g.particles[i]
		p.life -= dt
		if p.life <= 0 {
			continue
		}
		p.x += p.vx * dt
		p.y += p.vy * dt
		p.vx *= 0.97
		p.vy *= 0.97
		alive = append(alive, *p)
	}
	g.particles = alive
}

// buildSokkerGlow creates an outline image from the sprite's alpha channel,
// scaled to the powerup draw size. Uses image.Image to avoid ebiten's
// ReadPixels restriction before the game loop starts.
func buildSokkerGlow(src image.Image, drawSize int) *ebiten.Image {
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()

	alpha := make([]bool, sw*sh)
	for py := 0; py < sh; py++ {
		for px := 0; px < sw; px++ {
			_, _, _, a := src.At(src.Bounds().Min.X+px, src.Bounds().Min.Y+py).RGBA()
			alpha[py*sw+px] = a > 0x8000
		}
	}

	outlineImg := image.NewRGBA(image.Rect(0, 0, sw, sh))
	for py := 0; py < sh; py++ {
		for px := 0; px < sw; px++ {
			if alpha[py*sw+px] {
				continue
			}
			neighbor := false
			for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}, {-1, -1}, {1, -1}, {-1, 1}, {1, 1}} {
				nx, ny := px+d[0], py+d[1]
				if nx >= 0 && nx < sw && ny >= 0 && ny < sh && alpha[ny*sw+nx] {
					neighbor = true
					break
				}
			}
			if neighbor {
				outlineImg.SetRGBA(px, py, color.RGBA{255, 255, 255, 255})
			}
		}
	}

	outline := ebiten.NewImageFromImage(outlineImg)
	scaled := ebiten.NewImage(drawSize, drawSize)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(drawSize)/float64(sw), float64(drawSize)/float64(sh))
	scaled.DrawImage(outline, op)
	return scaled
}

var (
	rockOutline = color.RGBA{18, 16, 26, 255}
	rockShadow  = color.RGBA{46, 42, 58, 255}
	rockBase    = color.RGBA{84, 80, 96, 255}
	rockLight   = color.RGBA{132, 128, 150, 255}
	rockCrack   = color.RGBA{30, 27, 40, 255}
)

// hash01 turns a seed and an index into a stable pseudo-random value in [0,1).
func hash01(seed uint32, n int) float64 {
	v := seed*2654435761 + uint32(n)*2246822519
	v ^= v >> 13
	v *= 3266489917
	v ^= v >> 16
	return float64(v%2048) / 2048
}

// boulderFacets is the number of straight edges per side of the silhouette.
// Few and long, so the rock reads as chipped stone with sharp corners rather
// than a smooth blob.
const boulderFacets = 5

// boulderChain is one side's facet-corner half-widths, from the top of the
// rock to the bottom.
type boulderChain [boulderFacets + 1]float64

// at linearly interpolates the chain at scanline i of rows, so the silhouette
// between corners is a straight edge.
func (c boulderChain) at(i, rows int) float64 {
	p := float64(i) / float64(rows-1) * boulderFacets
	k := int(p)
	if k >= boulderFacets {
		return c[boulderFacets]
	}
	return c[k] + (c[k+1]-c[k])*(p-float64(k))
}

// boulderEdge returns the half-width of the rock at each facet corner for one
// side, derived from an ellipse and then pulled in or out per corner. The
// alternating pull is what produces the jagged, spiky outline.
func boulderEdge(half, h float64, seed uint32, salt int) boulderChain {
	var ctl boulderChain
	for k := range ctl {
		// Inset from the very tips so the top and bottom are chunky rather
		// than needle-pointed.
		fy := (0.1 + 0.8*float64(k)/boulderFacets) * h
		t := (fy - h/2) / (h / 2)
		span := half * math.Sqrt(math.Max(0, 1-t*t))
		pull := 0.7 + 0.4*hash01(seed, salt+k)
		if k%2 == 1 {
			pull += 0.25 // every other corner juts out into a point
		}
		ctl[k] = math.Min(half, span*pull)
	}
	return ctl
}

// drawBoulder draws a jagged chunk of rock: an angular silhouette built
// scanline by scanline from the obstacle's two facet chains, a lit upper-left
// face, a shadowed underside and a couple of cracks. Every pixel lands inside
// the same span obstacle.hits tests, so what you see is what you collide with.
func drawBoulder(dst *ebiten.Image, o obstacle, screenX float64) {
	rows := o.rows()
	dx := screenX - o.x // world -> screen

	for i := 0; i < rows; i++ {
		x0, x1 := o.spanAt(i)
		rowX := float32(x0 + dx)
		rowW := float32(x1 - x0)
		rowY := float32(o.y) + float32(i)
		h := o.h

		vector.FillRect(dst, rowX, rowY, rowW, 1, rockBase, false)

		// lit face on the upper left, shadow along the lower right
		if float64(i) < h*0.45 {
			vector.FillRect(dst, rowX+1, rowY, rowW*0.32, 1, rockLight, false)
		} else {
			vector.FillRect(dst, rowX+rowW*0.55, rowY, rowW*0.45-1, 1, rockShadow, false)
		}

		// hard rim so the rock separates from the vinyl behind it
		vector.FillRect(dst, rowX, rowY, 1, 1, rockOutline, false)
		vector.FillRect(dst, rowX+rowW-1, rowY, 1, 1, rockOutline, false)
		if i == 0 || i == rows-1 {
			vector.FillRect(dst, rowX, rowY, rowW, 1, rockOutline, false)
		}
	}

	// cracks
	for c := 0; c < 2; c++ {
		cx0 := float32(screenX + o.w*(0.3+0.35*hash01(o.seed, 900+c)))
		cy0 := float32(o.y + o.h*0.2)
		cx1 := cx0 + float32(o.w*0.25*(hash01(o.seed, 920+c)-0.5))
		cy1 := float32(o.y + o.h*(0.55+0.3*hash01(o.seed, 940+c)))
		vector.StrokeLine(dst, cx0, cy0, cx1, cy1, 1, rockCrack, false)
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Render the game world into a fixed 426×240 viewport, then upscale it to
	// the device-pixel screen (keeping pixel art sharp). HUD text is drawn
	// afterwards directly on the screen so it stays crisp.
	scale := float64(screen.Bounds().Dx()) / screenW
	if g.viewport == nil {
		g.viewport = ebiten.NewImage(screenW, screenH)
	}
	vp := g.viewport
	vp.Fill(color.RGBA{18, 16, 30, 255})

	// Inflatable vinyl wall behind everything, scrolling at half speed.
	wallW := float64(g.wallImg.Bounds().Dx())
	for x := -math.Mod(g.cameraX*0.5, wallW); x < float64(screenW); x += wallW {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x, wallThickness)
		vp.DrawImage(g.wallImg, op)
	}

	// Turrets further back still, dimmed so they read as scenery.
	const turretSpacing = 168.0
	turretH := float64(g.turretImg.Bounds().Dy())
	for x := -math.Mod(g.cameraX*0.25, turretSpacing); x < float64(screenW); x += turretSpacing {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x, float64(screenH-wallThickness)-turretH)
		op.ColorScale.Scale(0.5, 0.5, 0.56, 1)
		vp.DrawImage(g.turretImg, op)
	}

	// Canopy and bounce mattress: the surfaces the player rebounds off. They
	// scroll at full speed to sell the sense of motion, and tint blue while a
	// shield is up (the old solid walls did the same).
	bandW := float64(g.topImg.Bounds().Dx())
	bandOp := func() *ebiten.DrawImageOptions {
		op := &ebiten.DrawImageOptions{}
		if g.shieldTimer > 0 {
			pulse := float32(0.75 + 0.25*math.Sin(g.timeSinceStart*8))
			op.ColorScale.Scale(0.5, 0.8, 1.2, 1)
			op.ColorScale.Scale(pulse, pulse, pulse, 1)
		}
		return op
	}
	for x := -math.Mod(g.cameraX, bandW); x < float64(screenW); x += bandW {
		top := bandOp()
		top.GeoM.Translate(x, 0)
		vp.DrawImage(g.topImg, top)

		bottom := bandOp()
		bottom.GeoM.Translate(x, float64(screenH-wallThickness))
		vp.DrawImage(g.bottomImg, bottom)
	}

	// obstacles: boulders dragged into the castle
	for _, o := range g.obstacles {
		sx := o.x - g.cameraX
		if sx > float64(screenW)+10 || sx+o.w < -10 {
			continue
		}
		drawBoulder(vp, o, sx)
	}

	// powerups
	for _, p := range g.powerups {
		sx := float32(p.x - g.cameraX)
		if sx > float32(screenW)+10 || sx+powerupDrawSize < -10 {
			continue
		}
		bob := float32(math.Sin(g.timeSinceStart*3+p.x*0.1) * 3)
		drawY := float32(p.y) + bob

		spriteImg := g.sokkerImg
		glowImg := g.sokkerGlow
		glowClr := color.RGBA{100, 220, 255, 255}
		if p.kind == powerupAntiShield {
			spriteImg = g.antiSokkerImg
			glowImg = g.antiSokkerGlow
			glowClr = color.RGBA{255, 80, 80, 255}
		}

		glowPulse := 0.6 + 0.4*math.Sin(g.timeSinceStart*4+p.x*0.05)
		glowOp := &ebiten.DrawImageOptions{}
		glowOp.GeoM.Translate(float64(sx)-1, float64(drawY)-1)
		glowOp.ColorScale.Scale(float32(glowPulse), float32(glowPulse), float32(glowPulse), float32(glowPulse))
		glowOp.ColorScale.ScaleWithColor(glowClr)
		vp.DrawImage(glowImg, glowOp)

		op := &ebiten.DrawImageOptions{}
		spriteScale := powerupDrawSize / float64(spriteImg.Bounds().Dx())
		op.GeoM.Scale(spriteScale, spriteScale)
		op.GeoM.Translate(float64(sx), float64(drawY))
		vp.DrawImage(spriteImg, op)
	}

	// particles
	for _, p := range g.particles {
		alpha := uint8(math.Min(255, 255*(p.life/0.8)))
		sz := float32(p.size * math.Min(1, p.life/0.8))
		if sz < 0.5 {
			sz = 0.5
		}
		vector.FillRect(vp, float32(p.x)-sz/2, float32(p.y)-sz/2, sz, sz, color.RGBA{p.r, p.g, p.b, alpha}, false)
	}

	// speed lines trailing the dash, so the lunge forward reads as movement
	if g.playerVX > 20 {
		streak := float32(g.playerVX / dashMaxSpeed * 26)
		for i := 0; i < 4; i++ {
			ly := float32(g.playerY+playerBodyOffY) + float32(6+i*8)
			lx := float32(g.playerX) - streak - float32(i%2)*5
			alpha := uint8(150 - i*25)
			vector.FillRect(vp, lx, ly, streak, 1, color.RGBA{255, 255, 255, alpha}, false)
		}
	}

	// player with charge brightness
	frame := g.playerAnim.CurrentFrame()
	op := &ebiten.DrawImageOptions{}
	spriteScale := float64(playerDrawSize) / float64(g.playerAnim.Sheet.FrameW)
	op.GeoM.Scale(spriteScale, spriteScale)
	op.GeoM.Translate(g.playerX, g.playerY)
	chargeBright := math.Min(g.chargeTime*2.5, 1.5)
	r := float32(1.0 + chargeBright*0.4)
	gr := float32(1.0 + chargeBright*0.4)
	b := float32(1.0 + chargeBright*0.3)
	if r > 1.6 {
		r = 1.6
	}
	if gr > 1.6 {
		gr = 1.6
	}
	if b > 1.5 {
		b = 1.5
	}
	op.ColorScale.Scale(r, gr, b, 1)
	if g.shieldTimer > 0 {
		op.ColorScale.ScaleWithColor(color.RGBA{180, 230, 255, 255})
	}
	vp.DrawImage(frame, op)

	// shield aura
	if g.shieldTimer > 0 {
		pulse := float32(0.3 + 0.15*math.Sin(g.timeSinceStart*6))
		cx := float32(g.playerX + playerDrawSize/2)
		cy := float32(g.playerY + playerDrawSize/2)
		vector.StrokeCircle(vp, cx, cy, playerDrawSize/2+4, 1.5, color.RGBA{50, 180, 255, uint8(pulse * 255)}, true)
	}

	g.drawPressureGauge(vp)

	// switch dots below player (only when double-tap is enabled)
	if doubleTapEnabled {
		dotY := float32(g.playerY + playerDrawSize + 4)
		dotStartX := float32(g.playerX + playerDrawSize/2 - (maxSwitches*8-4)/2)
		for i := 0; i < maxSwitches; i++ {
			cx := dotStartX + float32(i*8)
			if i < g.switchesLeft {
				vector.FillCircle(vp, cx, dotY, 3, color.RGBA{255, 220, 50, 255}, true)
			} else {
				vector.FillCircle(vp, cx, dotY, 3, color.RGBA{80, 70, 40, 255}, true)
			}
		}
	}

	// Scale the viewport to fill the device-pixel screen (sharp pixel art).
	vpOp := &ebiten.DrawImageOptions{}
	vpOp.GeoM.Scale(scale, scale)
	vpOp.Filter = ebiten.FilterNearest
	screen.DrawImage(vp, vpOp)

	// HUD drawn at device resolution so it stays crisp.
	hudY := wallThickness + 4
	sz := int(8 * scale)
	tdcgame.Write(screen, fmt.Sprintf("SCORE: %d", g.score), int(8*scale), int(float64(hudY)*scale), sz)
	speedPct := int((g.scrollSpeed - baseScrollSpeed) / (maxScrollSpeed - baseScrollSpeed) * 100)
	tdcgame.Write(screen, fmt.Sprintf("SPEED: %d%%", speedPct), int(float64(screenW-94)*scale), int(float64(hudY)*scale), sz)

	if g.shieldTimer > 0 {
		tdcgame.WriteCenteredAt(screen, fmt.Sprintf("SHIELD: %.1fs", g.shieldTimer), int(float64(screenW/2)*scale), int(float64(hudY)*scale), sz)
	}

	if !g.started {
		g.drawStartPrompt(screen, scale)
	}

	// The framework's GameRunner skips its own overlays for custom-draw games,
	// so this game draws its own game-over screen.
	if g.GameOver {
		g.drawGameOver(screen, scale)
	}
}

// drawGameOver tells the player what happened and what the one button does
// next: a single press leaves, a double press replays (both handled by
// GameRunnerScene back in game.go).
func (g *Game) drawGameOver(screen *ebiten.Image, scale float64) {
	b := screen.Bounds()
	vector.FillRect(screen, 0, 0, float32(b.Dx()), float32(b.Dy()), color.RGBA{10, 8, 18, 205}, false)

	cx := float64(b.Dx()) / 2
	tdcgame.WriteAt(screen, "SPLAT!", cx, float64(screenH/2-52)*scale,
		30*scale, color.RGBA{240, 70, 60, 255}, text.AlignCenter, text.AlignStart)
	tdcgame.WriteAt(screen, "YOU HIT A ROCK", cx, float64(screenH/2-18)*scale,
		12*scale, color.RGBA{242, 242, 242, 255}, text.AlignCenter, text.AlignStart)
	tdcgame.WriteAt(screen, fmt.Sprintf("SCORE: %d", g.score), cx, float64(screenH/2+4)*scale,
		16*scale, color.RGBA{255, 212, 47, 255}, text.AlignCenter, text.AlignStart)

	hint := color.RGBA{200, 200, 210, 255}
	tdcgame.WriteAt(screen, "PRESS THE RED BUTTON ONCE TO RETURN TO MAIN SCREEN", cx,
		float64(screenH/2+34)*scale, 7*scale, hint, text.AlignCenter, text.AlignStart)
	tdcgame.WriteAt(screen, "DOUBLE CLICK IT TO PLAY AGAIN", cx,
		float64(screenH/2+48)*scale, 7*scale, hint, text.AlignCenter, text.AlignStart)
}

// The air supply rides along with the player as a round pressure dial, sitting
// diagonally below them. The wedge sweeps around the dial as a dash spends the
// air and winds back around as it refills, so you read your fuel without
// looking away from the player. It flashes red once empty, the cue to let go.
const (
	dialOffX = -6.0 // down and to the left of the body, clear of the sprite
	dialOffY = 6.0
	dialR    = 7.0
)

func (g *Game) drawPressureGauge(vp *ebiten.Image) {
	cx := float32(g.playerX + dialOffX)
	cy := float32(g.playerY + playerBodyOffY + playerBodyH + dialOffY)

	clr := color.RGBA{90, 220, 120, 255}
	if g.pumpLocked {
		flash := uint8(150 + 100*math.Sin(g.timeSinceStart*14))
		clr = color.RGBA{240, 70, 60, flash}
	}

	// dial face, so an almost-empty wedge still reads as a gauge
	vector.FillCircle(vp, cx, cy, dialR+1, color.RGBA{18, 16, 30, 190}, true)
	vector.StrokeCircle(vp, cx, cy, dialR+1, 1, color.RGBA{120, 120, 140, 200}, true)

	fill := g.pressure / pumpMax
	if fill <= 0 {
		return
	}

	// The wedge starts at 12 o'clock and turns clockwise as the dial fills.
	const start = -math.Pi / 2
	var path vector.Path
	path.MoveTo(cx, cy)
	path.Arc(cx, cy, dialR, start, float32(start+2*math.Pi*fill), vector.Clockwise)
	path.Close()

	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(clr)
	vector.FillPath(vp, &path, nil, op)
}

// drawStartPrompt covers the frozen first frame with the call to action. The
// framework holds the game still until the button is pressed, so this stays up
// until the player actually starts.
func (g *Game) drawStartPrompt(screen *ebiten.Image, scale float64) {
	b := screen.Bounds()
	vector.FillRect(screen, 0, 0, float32(b.Dx()), float32(b.Dy()), color.RGBA{10, 8, 18, 190}, false)

	g.promptTime += 1.0 / 60
	pulse := 0.75 + 0.25*math.Sin(g.promptTime*4)
	buttonClr := color.RGBA{uint8(210 * pulse), uint8(45 * pulse), uint8(40 * pulse), 255}

	// the big red button itself, so the prompt shows what to look for
	cx := float32(b.Dx()) / 2
	cy := float32(float64(screenH/2-46) * scale)
	r := float32(13 * scale)
	vector.FillCircle(screen, cx, cy+r*0.25, r, color.RGBA{40, 36, 48, 255}, true)
	vector.FillCircle(screen, cx, cy, r, buttonClr, true)
	vector.FillCircle(screen, cx-r*0.3, cy-r*0.35, r*0.28, color.RGBA{255, 170, 160, 200}, true)

	line1Y := float64(screenH/2-16) * scale
	line2Y := float64(screenH/2+8) * scale
	tdcgame.WriteAt(screen, "PRESS THE BIG RED BUTTON", float64(b.Dx())/2, line1Y,
		15*scale, color.RGBA{242, 242, 242, 255}, text.AlignCenter, text.AlignStart)
	tdcgame.WriteAt(screen, "TO START", float64(b.Dx())/2, line2Y,
		26*scale, color.RGBA{255, 212, 47, 255}, text.AlignCenter, text.AlignStart)

	tdcgame.WriteAt(screen, "HOLD TO SHOOT FORWARD - LET GO TO DRIFT BACK", float64(b.Dx())/2,
		float64(screenH/2+48)*scale, 7*scale, color.RGBA{200, 200, 210, 255}, text.AlignCenter, text.AlignStart)
}
