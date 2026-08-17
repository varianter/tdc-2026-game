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
	"github.com/hajimehoshi/ebiten/v2/vector"
	"variant.dev/tdcgame/tdcgame"
)

//go:embed sokker.png
var sokkerData []byte

//go:embed anti_sokker.png
var antiSokkerData []byte

const (
	screenW        = 426
	screenH        = 240
	playerDrawSize = 32
	playerHitW     = 16.0
	playerHitH     = 22.0
	playerHitOffX  = 8.0
	playerHitOffY  = 6.0

	wallThickness = 6
	playTop       = float64(wallThickness)
	playBottom    = float64(screenH - wallThickness - playerDrawSize)

	baseScrollSpeed      = 60.0
	maxScrollSpeed       = 250.0
	speedIncreaseRate    = 1.6
	chargeBoostsScroll   = true
	chargeScrollMultiMax = 2.2
	chargeScrollDrag     = 0.985

	chargeAccelBase = 400.0
	chargeRampRate  = 500.0
	maxPlayerSpeed  = 800.0
	minPlayerSpeed  = 20.0
	idleDrag        = 0.992

	obstacleBaseInterval = 280.0
	obstacleMinInterval  = 70.0

	doubleTapEnabled = false
	doubleTapWindow  = 0.25

	maxSwitches        = 3
	switchRechargeTime = 5.0

	shieldDuration              = 5.0
	shieldStackDuration         = 3.0
	shieldMaxDuration           = 20.0
	shieldObstacleDurationAward = 1.0
	powerupDrawSize             = 20.0
	powerupCollideSize          = 20.0
	powerupBaseInterval         = 600.0
	powerupMinInterval          = 300.0
	powerupMargin               = 8.0
)

type obstacle struct {
	x, y, w, h float64
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

	cameraX  float64
	playerY  float64
	playerVY float64

	scrollSpeed float64
	distance    float64
	score       int
	bonusScore  int

	chargeTime  float64
	scrollBoost float64

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
		Frames: []int{0, 1, 2, 3, 4, 5, 6, 7},
		FPS:    12,
	}

	sokImg, _, err := image.Decode(bytes.NewReader(sokkerData))
	if err != nil {
		panic(err)
	}

	antiSokImg, _, err := image.Decode(bytes.NewReader(antiSokkerData))
	if err != nil {
		panic(err)
	}
	sokkerEbi := ebiten.NewImageFromImage(sokImg)
	antiSokkerEbi := ebiten.NewImageFromImage(antiSokImg)
	return &Game{
		playerAnim:     anim,
		sokkerImg:      sokkerEbi,
		sokkerGlow:     buildSokkerGlow(sokImg, int(powerupDrawSize)),
		antiSokkerImg:  antiSokkerEbi,
		antiSokkerGlow: buildSokkerGlow(antiSokImg, int(powerupDrawSize)),
		playerY:        float64(screenH)/2 - float64(playerDrawSize)/2,
		playerVY:       -30,
		scrollSpeed:    baseScrollSpeed,
		nextObstacleX:  float64(screenW) + 100,
		nextPowerupX:   float64(screenW) + 400,
		switchesLeft:   maxSwitches,
	}
}

func (g *Game) Update(dt float64, spacePressed bool, spaceJustPressed bool) {
	if g.GameOver {
		return
	}

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

	if spacePressed {
		g.chargeTime += dt
		accel := chargeAccelBase + chargeRampRate*g.chargeTime
		if math.Abs(g.playerVY) < 5 {
			g.playerVY = 80
		}
		if g.playerVY > 0 {
			g.playerVY += accel * dt
		} else {
			g.playerVY -= accel * dt
		}
		if g.playerVY > maxPlayerSpeed {
			g.playerVY = maxPlayerSpeed
		} else if g.playerVY < -maxPlayerSpeed {
			g.playerVY = -maxPlayerSpeed
		}
	} else {
		g.chargeTime = 0
		g.playerVY *= math.Pow(idleDrag, 60*dt)
	}

	g.playerY += g.playerVY * dt

	if g.playerY < playTop {
		g.playerY = playTop
		g.playerVY = -g.playerVY
	}
	if g.playerY > playBottom {
		g.playerY = playBottom
		g.playerVY = -g.playerVY
	}

	if math.Abs(g.playerVY) < minPlayerSpeed {
		if g.playerVY >= 0 {
			g.playerVY = minPlayerSpeed
		} else {
			g.playerVY = -minPlayerSpeed
		}
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
	g.score = g.bonusScore

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

		playH := playBottom - playTop
		h := 15 + rand.Float64()*math.Min(70, 20+g.distance/200)
		w := 15 + rand.Float64()*40
		y := playTop + rand.Float64()*(playH-h)

		g.obstacles = append(g.obstacles, obstacle{x: g.nextObstacleX, y: y, w: w, h: h})

		if g.distance > 2000 && rand.Float64() < math.Min(0.5, g.distance/10000) {
			h2 := 15 + rand.Float64()*40
			y2 := playTop + rand.Float64()*(playH-h2)
			g.obstacles = append(g.obstacles, obstacle{x: g.nextObstacleX + w + 5, y: y2, w: w * 0.7, h: h2})
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
		playH := playBottom - playTop
		x := g.nextPowerupX

		kind := powerupShield
		antiChance := math.Min(0.6, 0.15+(g.distance-3000)/20000)
		if g.hasHadShield && g.distance > 3000 && rand.Float64() < antiChance {
			kind = powerupAntiShield
		}

		placed := false
		for attempt := 0; attempt < 10; attempt++ {
			y := playTop + rand.Float64()*(playH-powerupCollideSize)
			if !g.overlapsAnyObstacle(x, y, powerupCollideSize, powerupCollideSize) {
				g.powerups = append(g.powerups, powerup{x: x, y: y, kind: kind})
				placed = true
				break
			}
		}
		if !placed {
			y := playTop + rand.Float64()*(playH-powerupCollideSize)
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
	px := g.cameraX + 50 + playerHitOffX
	py := g.playerY + playerHitOffY

	alive := g.powerups[:0]
	for _, p := range g.powerups {
		if px < p.x+powerupCollideSize && px+playerHitW > p.x && py < p.y+powerupCollideSize && py+playerHitH > p.y {
			g.applyPowerup(p)
			g.bonusScore += 1
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
	case powerupAntiShield:
		if g.shieldTimer > 0 {
			g.shieldTimer = 0
		}
	}
}

func (g *Game) checkCollisions() {
	px := g.cameraX + 50 + playerHitOffX
	py := g.playerY + playerHitOffY

	alive := g.obstacles[:0]
	for _, o := range g.obstacles {
		if px < o.x+o.w && px+playerHitW > o.x && py < o.y+o.h && py+playerHitH > o.y {
			if g.shieldTimer > 0 {
				g.spawnDestroyParticles(o)
				g.bonusScore += 3
				g.shieldTimer = math.Min(shieldMaxDuration, g.shieldTimer+shieldObstacleDurationAward)
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
		g.particles = append(g.particles, particle{
			x:    cx + rand.Float64()*o.w*0.5 - o.w*0.25,
			y:    cy + rand.Float64()*o.h*0.5 - o.h*0.25,
			vx:   math.Cos(angle) * speed,
			vy:   math.Sin(angle) * speed,
			life: 0.4 + rand.Float64()*0.4,
			r:    200 + uint8(rand.Intn(55)),
			g:    uint8(30 + rand.Intn(60)),
			b:    uint8(20 + rand.Intn(40)),
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

func (g *Game) Draw(screen *ebiten.Image) {
	// Render the game world into a fixed 426×240 viewport, then upscale it to
	// the device-pixel screen (keeping pixel art sharp). HUD text is drawn
	// afterwards directly on the screen so it stays crisp.
	scale := float64(screen.Bounds().Dx()) / screenW
	if g.viewport == nil {
		g.viewport = ebiten.NewImage(screenW, screenH)
	}
	vp := g.viewport
	vp.Fill(color.RGBA{12, 8, 20, 255})

	wallClr := color.RGBA{140, 140, 170, 255}
	if g.shieldTimer > 0 {
		pulse := uint8(40 + 20*math.Sin(g.timeSinceStart*8))
		wallClr = color.RGBA{50, 180, 255, pulse + 180}
	}
	vector.FillRect(vp, 0, 0, float32(screenW), float32(wallThickness), wallClr, false)
	vector.FillRect(vp, 0, float32(screenH-wallThickness), float32(screenW), float32(wallThickness), wallClr, false)

	gridClr := color.RGBA{25, 20, 40, 255}
	for y := wallThickness; y < screenH-wallThickness; y += 20 {
		vector.FillRect(vp, 0, float32(y), float32(screenW), 1, gridClr, false)
	}
	gridOffset := -math.Mod(g.cameraX, 40)
	for x := gridOffset; x < float64(screenW); x += 40 {
		vector.FillRect(vp, float32(x), float32(wallThickness), 1, float32(screenH-2*wallThickness), gridClr, false)
	}

	// obstacles
	for _, o := range g.obstacles {
		sx := float32(o.x - g.cameraX)
		if sx > float32(screenW)+10 || sx+float32(o.w) < -10 {
			continue
		}
		vector.FillRect(vp, sx, float32(o.y), float32(o.w), float32(o.h), color.RGBA{200, 40, 40, 255}, false)
		vector.StrokeRect(vp, sx, float32(o.y), float32(o.w), float32(o.h), 1, color.RGBA{255, 80, 80, 255}, false)
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

	// player with charge brightness
	frame := g.playerAnim.CurrentFrame()
	op := &ebiten.DrawImageOptions{}
	spriteScale := float64(playerDrawSize) / float64(g.playerAnim.Sheet.FrameW)
	op.GeoM.Scale(spriteScale, spriteScale)
	op.GeoM.Translate(50, g.playerY)
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
		cx := float32(50 + playerDrawSize/2)
		cy := float32(g.playerY + playerDrawSize/2)
		vector.StrokeCircle(vp, cx, cy, playerDrawSize/2+4, 1.5, color.RGBA{50, 180, 255, uint8(pulse * 255)}, true)
	}

	// charge bar
	if g.chargeTime > 0.05 {
		barW := float32(math.Min(g.chargeTime*40, 60))
		vector.FillRect(vp, 50, float32(g.playerY-6), barW, 3, color.RGBA{255, 200, 50, 220}, false)
	}

	// switch dots below player (only when double-tap is enabled)
	if doubleTapEnabled {
		dotY := float32(g.playerY + playerDrawSize + 4)
		dotStartX := float32(50 + playerDrawSize/2 - (maxSwitches*8-4)/2)
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

	// Game over overlay is drawn by the framework's GameRunner
}
