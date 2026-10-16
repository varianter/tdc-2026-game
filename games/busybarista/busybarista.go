// Package busybarista is a one-button timing game set at the conference coffee
// stand. Cups ride a conveyor: hold the button to pour and release on the
// fill line, then tap as the cup passes under the lid press. Later levels mix
// in tap-only espresso shots and tall cups, and the tempo keeps climbing.
package busybarista

import (
	"bytes"
	"embed"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"
	"math/rand"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"variant.dev/tdcgame/games/busybarista/assets"
	"variant.dev/tdcgame/tdcgame"
)

const (
	// levelDuration is how long each stepped level lasts. runCap is the hard
	// limit on a run; the levels are tuned so lives run out well before it.
	levelDuration = 12.0
	runCap        = 60.0

	// gameOverInputDelay swallows presses on the score screen for a moment,
	// so a player still hammering the button gets to read it.
	gameOverInputDelay = 1.25

	bannerDuration = 1.8
	// A level that introduces a new kind of cup holds its banner longer, so
	// the extra line can be read, and waits longer before the first cup.
	introBannerDuration = 3.2
	introBreather       = introBannerDuration
	// breather pushes the first event of a new level back a little, so the
	// banner can be read.
	breather = 0.9

	// A pour may start a little before the cup comes to rest, and has a
	// lenient grace period after: reaction to the cue isn't the skill, the
	// release is.
	holdOpenLead   = 0.15
	holdStartGrace = 0.5

	// Cups glide in over two beats, but never faster than minTravel allows.
	// Arrivals and shots land on half beats.
	approachBeats = 2.0
	minTravel     = 0.55

	firstEventDelay = 1.0

	// closeSettle is how long the game lingers after the last cup is done
	// once the shop has closed, so the final lid or spill can be seen.
	closeSettle = 0.6
)

// Layout, in 426×240 viewport pixels.
const (
	beltY       = 160.0 // belt surface; cup bottoms rest here
	spoutX      = 170.0
	pressX      = 262.0
	counterEndX = 322.0
	spawnX      = -20.0
	nozzleY     = 112.0
	pressRestY  = 112.0 // bottom of the lid press piston at rest
)

type level struct {
	name             string
	bpm              float64
	fillMin, fillMax float64 // seconds from starting a pour to hitting the line
	espresso, large  float64 // chance of each special cup
	minGap           float64 // guaranteed gap between one window closing and the next opening
	intro            *intro  // a new kind of cup this level brings in, if any
}

// intro announces a new kind of cup in the level banner, with a line on how
// to serve it. The level's first cup is one of them, sent once the banner has
// gone, so the player tries it straight after reading about it.
type intro struct {
	kind cupKind
	how  string
}

var levels = []level{
	{"OPENING", 96, 0.9, 0.9, 0, 0, 0.25, nil},
	{"MORNING RUSH", 112, 0.7, 0.8, 0, 0, 0.22, nil},
	{"KEYNOTE BREAK", 126, 0.6, 0.7, 0.4, 0, 0.2, &intro{cupEspresso, "TAP AS IT PASSES THE SPOUT"}},
	{"LUNCH PANIC", 142, 0.5, 0.75, 0.3, 0.3, 0.17, nil},
	{"CAFFEINE MELTDOWN", 160, 0.4, 0.6, 0.35, 0.3, 0.15, nil},
}

type gameState int

const (
	stateReady gameState = iota
	statePlaying
	stateGameOver
)

type cupKind int

const (
	cupStandard cupKind = iota
	cupLarge
	cupEspresso
)

// cupSize returns the cup's width and height in viewport pixels.
func (k cupKind) size() (w, h float64) {
	switch k {
	case cupLarge:
		return 20, 36
	case cupEspresso:
		return 13, 11
	}
	return 18, 27
}

type cupPhase int

const (
	phaseIncoming cupPhase = iota // gliding to the spout (espresso: through it)
	phaseWaiting                  // at the spout, waiting for the pour
	phasePouring                  // button held, coffee rising
	phaseToLid                    // on its way through the lid press
	phaseDone                     // finished or failed, leaving the belt
)

type cup struct {
	kind  cupKind
	phase cupPhase

	// Linear motion: at x0 at t0, heading for x1 at t1. With stop it rests at
	// x1 afterwards; without, it carries on at the same speed.
	x0, t0, x1, t1 float64
	stop           bool

	target float64 // arrival time at the spout, or the espresso shot time

	fillDur   float64 // pour time from start to the line
	lineFrac  float64 // fill line, as a fraction of the cup's height
	fill      float64 // current coffee level, 0..1
	holdStart float64
	lineT     float64 // when the coffee crosses the line on this pour
	lidT      float64 // when the cup is centred under the lid press

	lidded   bool
	spilled  bool
	failed   bool // falls off the end of the counter instead of being served
	missed   bool // any MISS on this cup; no item bonus
	notified bool // an espresso's successor has been scheduled

	fallY, fallVY, tilt float64
	steamT              float64
}

func (c *cup) x(t float64) float64 {
	if t <= c.t0 || c.t1 <= c.t0 {
		return c.x0
	}
	if c.stop && t >= c.t1 {
		return c.x1
	}
	return c.x0 + (c.x1-c.x0)*(t-c.t0)/(c.t1-c.t0)
}

func (c *cup) moveTo(now, toX, at float64, stop bool) {
	c.x0, c.t0, c.x1, c.t1, c.stop = c.x(now), now, toX, at, stop
}

// particle is a droplet, steam wisp or confetti speck in viewport space.
type particle struct {
	x, y, vx, vy float64
	gravity      float64
	life, max    float64
	size         float64
	clr          [4]float32
}

// figure is a customer in the queue, drawn from the shared TDC sprite.
type figure struct {
	x     float64
	phase float64
	tint  [3]float32
	hop   float64
}

// BusyBarista implements tdcgame.TdcGameWithPlayer, GameWithCustomDraw,
// TdcGameWithFinalScore and TdcGameWithCleanup.
type BusyBarista struct {
	state gameState
	now   float64 // seconds since the run started
	sc    scoring
	btn   button
	clock beatClock
	rng   *rand.Rand

	levelIdx      int
	bannerT       float64
	bannerLen     float64 // how long the current banner shows
	breatherUntil float64

	// forceNext makes the next scheduled cup a forcedKind, for a level intro.
	forceNext  bool
	forcedKind cupKind

	// closing is set when the run cap is reached: no more cups are sent, but
	// the ones already on the belt can still be finished.
	closing     bool
	settleTimer float64

	cups      []*cup
	popups    []popup
	particles []particle
	shake     float64

	served        int
	gameOverDelay float64

	pour *audio.Player

	// visuals
	baristaSheet *tdcgame.SpriteSheet
	crowdSheet   *tdcgame.SpriteSheet
	queue        []figure
	animClock    float64
	hop          float64 // barista hop timer
	pressT       float64 // time since the lid press last stamped
	pressDrop    float64 // how far the press travels down to this cup
	beltOffset   float64
	createdAt    time.Time
	viewport     *ebiten.Image
}

func New(assetsFS embed.FS) *BusyBarista {
	initSounds()
	img, _, err := image.Decode(bytes.NewReader(assets.Barista))
	if err != nil {
		log.Fatalf("busybarista: decode barista: %v", err)
	}
	baristaImg := ebiten.NewImageFromImage(img)

	return newGame(baristaImg, tdcgame.LoadSpriteSheet(tdcgame.LoadAssets(assetsFS), 64, 64))
}

func newGame(baristaImg *ebiten.Image, crowd *tdcgame.SpriteSheet) *BusyBarista {
	g := &BusyBarista{
		sc:           newScoring(),
		clock:        beatClock{bpm: levels[0].bpm},
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
		baristaSheet: &tdcgame.SpriteSheet{Image: baristaImg, FrameW: 64, FrameH: 64, Columns: 10},
		crowdSheet:   crowd,
		pressT:       10,
		createdAt:    time.Now(),
		queue: []figure{
			{x: 352, phase: 0.0, tint: [3]float32{0.8, 0.9, 1.05}},
			{x: 380, phase: 1.1, tint: [3]float32{1.05, 1.0, 0.75}},
			{x: 408, phase: 2.3, tint: [3]float32{0.9, 1.05, 0.85}},
		},
	}
	// A cup waits at the spout on the title screen.
	g.cups = []*cup{{kind: cupStandard, phase: phaseWaiting, x0: spoutX, lineFrac: 0.85}}
	return g
}

// — TdcGame ———————————————————————————————————————————————————————————————

func (g *BusyBarista) GetGameObjects() []tdcgame.GameObject { return nil }

func (g *BusyBarista) GetGameParameters() *tdcgame.GameParameters {
	return &tdcgame.GameParameters{ShouldCameraFollowPlayer: false}
}

// GetGameState reports GameOver only after the input grace period, since the
// runner acts on the button as soon as it sees it.
func (g *BusyBarista) GetGameState() tdcgame.GameState {
	if g.state == stateGameOver && g.gameOverDelay <= 0 {
		return tdcgame.GameOver
	}
	return tdcgame.Running
}

func (g *BusyBarista) GetCurrentScore() int { return g.sc.score }

func (g *BusyBarista) ScoreFinal() bool { return g.state == stateGameOver }

func (g *BusyBarista) Cleanup() {
	g.stopPour()
}

func (g *BusyBarista) GetPlayerUpdateFunc() tdcgame.PlayerUpdate {
	return func(_ bool, dt float64, _ tdcgame.Level, _ *tdcgame.MovingSquare) {
		g.update(dt)
	}
}

// — game loop —————————————————————————————————————————————————————————————

func (g *BusyBarista) update(dt float64) {
	g.animClock += dt
	g.hop = max(0, g.hop-dt)
	g.pressT += dt
	g.shake = max(0, g.shake-dt)
	g.bannerT = max(0, g.bannerT-dt)
	for i := range g.queue {
		g.queue[i].hop = max(0, g.queue[i].hop-dt)
	}
	g.popups = updatePopups(g.popups, dt)
	g.updateParticles(dt)

	switch g.state {
	case stateReady:
		g.start()
	case stateGameOver:
		g.gameOverDelay = max(0, g.gameOverDelay-dt)
		g.updateCups(dt)
		return
	}

	g.now += dt
	g.clock.advance(dt)
	g.updateLevel()

	pressed, released := g.btn.poll()
	if pressed {
		g.onPress()
	}
	if released {
		g.onRelease()
	}
	g.checkTimeouts()
	g.updateCups(dt)

	switch {
	case g.sc.lives <= 0:
		g.endGame()
	case !g.closing && g.now >= runCap:
		g.closeShop()
	case g.closing && !g.hasPendingCup():
		g.settleTimer += dt
		if g.settleTimer >= closeSettle {
			g.endGame()
		}
	}
}

// closeShop stops new cups at the run cap. Cups that haven't started down the
// belt yet are cancelled; the rest play out, so the run never ends mid-pour.
func (g *BusyBarista) closeShop() {
	g.closing = true
	onBelt := g.cups[:0]
	for _, c := range g.cups {
		if g.onBelt(c) {
			onBelt = append(onBelt, c)
		}
	}
	g.cups = onBelt
}

// onBelt reports whether c has come into view. Cups scheduled but still off
// screen to the left can be cancelled without the player noticing.
func (g *BusyBarista) onBelt(c *cup) bool {
	w, _ := c.kind.size()
	return g.now >= c.t0 && c.x(g.now)+w/2 > 0
}

// hasPendingCup reports whether any cup still needs an input.
func (g *BusyBarista) hasPendingCup() bool {
	for _, c := range g.cups {
		switch {
		case c.phase == phaseIncoming, c.phase == phaseWaiting, c.phase == phasePouring:
			return true
		case c.phase == phaseToLid && !c.lidded:
			return true
		}
	}
	return false
}

func (g *BusyBarista) start() {
	g.state = statePlaying
	g.cups = nil
	g.bannerLen = bannerDuration
	g.bannerT = g.bannerLen
	g.scheduleNext(firstEventDelay)
}

func (g *BusyBarista) endGame() {
	g.state = stateGameOver
	g.gameOverDelay = gameOverInputDelay
	g.stopPour()
	play(gameOverPCM)
}

func (g *BusyBarista) lvl() level { return levels[g.levelIdx] }

func (g *BusyBarista) updateLevel() {
	idx := min(len(levels)-1, int(g.now/levelDuration))
	if idx == g.levelIdx {
		return
	}
	g.levelIdx = idx
	g.clock.bpm = levels[idx].bpm
	g.bannerLen, g.breatherUntil = bannerDuration, g.now+breather
	if in := levels[idx].intro; in != nil {
		g.bannerLen, g.breatherUntil = introBannerDuration, g.now+introBreather
		g.introduce(in.kind)
	}
	g.bannerT = g.bannerLen
	play(levelUpPCM)
}

// introduce makes kind the next cup sent. Cups queued but not yet in view are
// pulled and rescheduled, so the new kind comes straight after the banner
// instead of after whatever was queued.
func (g *BusyBarista) introduce(kind cupKind) {
	g.forceNext, g.forcedKind = true, kind
	queued := false
	lastEnd := g.now // when the last input already on the belt is over
	onBelt := g.cups[:0]
	for _, c := range g.cups {
		if !g.onBelt(c) {
			queued = true
			continue
		}
		onBelt = append(onBelt, c)
		switch {
		case c.phase == phaseToLid && !c.lidded:
			lastEnd = max(lastEnd, c.lidT+tapWindows[2])
		case c.kind == cupEspresso && c.phase == phaseIncoming:
			lastEnd = max(lastEnd, c.target+tapWindows[2])
		}
	}
	g.cups = onBelt
	if queued {
		// The pulled cup's predecessor has already scheduled it and won't
		// again, so its replacement is scheduled here.
		g.scheduleNext(lastEnd + g.lvl().minGap)
	}
}

// scheduleNext queues the next cup so that its input window opens no earlier
// than earliestOpen. This is what guarantees inputs never overlap.
func (g *BusyBarista) scheduleNext(earliestOpen float64) {
	if g.closing {
		return
	}
	earliestOpen = max(earliestOpen, g.breatherUntil)
	l := g.lvl()
	beat := g.clock.beat()
	travel := approachBeats * beat

	kind := cupStandard
	switch r := g.rng.Float64(); {
	case g.forceNext:
		kind = g.forcedKind
		g.forceNext = false
	case r < l.espresso:
		kind = cupEspresso
	case r < l.espresso+l.large:
		kind = cupLarge
	}

	c := &cup{kind: kind, x0: spawnX, x1: spoutX}
	if kind == cupEspresso {
		c.target = g.clock.next(g.now, max(earliestOpen+tapWindows[2], g.now+minTravel), 2)
	} else {
		c.target = g.clock.next(g.now, max(earliestOpen+holdOpenLead, g.now+minTravel), 2)
		c.stop = true
		c.fillDur = l.fillMin + g.rng.Float64()*(l.fillMax-l.fillMin)
		if kind == cupLarge {
			c.fillDur *= 1.35
		}
		// Place the line so that the late edge of the OK window is exactly
		// the brim: overshoot past OK is literally a spill.
		c.lineFrac = c.fillDur / (c.fillDur + holdWindows[2])
	}
	c.t1 = c.target
	c.t0 = max(g.now, c.target-travel)
	g.cups = append(g.cups, c)
}

// activeCup returns the cup whose press window is open right now, if any.
func (g *BusyBarista) activeCup() *cup {
	for _, c := range g.cups {
		switch {
		case c.kind == cupEspresso && c.phase == phaseIncoming:
			if math.Abs(g.now-c.target) <= tapWindows[2] {
				return c
			}
		case c.phase == phaseIncoming || c.phase == phaseWaiting:
			if g.now >= c.target-holdOpenLead && g.now <= c.target+holdStartGrace {
				return c
			}
		case c.phase == phaseToLid && !c.lidded:
			if math.Abs(g.now-c.lidT) <= tapWindows[2] {
				return c
			}
		}
	}
	return nil
}

func (g *BusyBarista) pouringCup() *cup {
	for _, c := range g.cups {
		if c.phase == phasePouring {
			return c
		}
	}
	return nil
}

func (g *BusyBarista) onPress() {
	c := g.activeCup()
	if c == nil {
		g.sc.stray()
		g.addPopup(spoutX, nozzleY+10, "TOO EARLY", rgb(170, 160, 150), 6)
		return
	}
	switch {
	case c.kind == cupEspresso:
		t := judge(g.now-c.target, tapWindows)
		g.resolve(c, t, spoutX, "")
		if t != tierMiss {
			c.fill = 0.8
			c.lidded = true
			c.phase = phaseDone
			play(shotPCM)
			g.steamPuff(spoutX, nozzleY)
			g.finishItem(c, itemBonus/2)
		}
	case c.phase == phaseToLid:
		t := judge(g.now-c.lidT, tapWindows)
		g.resolve(c, t, pressX, "")
		if t != tierMiss {
			c.lidded = true
			g.pressT = 0
			_, h := c.kind.size()
			g.pressDrop = beltY - h - 3 - pressRestY
			play(lidPCM)
			g.finishItem(c, itemBonus)
		}
		c.phase = phaseDone
	default: // start the pour
		c.phase = phasePouring
		c.moveTo(g.now, spoutX, g.now, true)
		c.holdStart = g.now
		c.lineT = g.now + c.fillDur
		g.stopPour()
		g.pour = newPour(c.fillDur)
	}
}

func (g *BusyBarista) onRelease() {
	c := g.pouringCup()
	if c == nil {
		return
	}
	g.stopPour()
	err := g.now - c.lineT
	t := judge(err, holdWindows)
	label := ""
	if t == tierMiss {
		label = "UNDERFILLED"
	}
	g.resolve(c, t, spoutX, label)
	if t == tierMiss {
		g.failCup(c)
		g.scheduleNext(g.now + g.lvl().minGap)
		return
	}
	// On to the lid press, timed to land on a half beat.
	lead := max(g.lvl().minGap+tapWindows[2], 0.5*g.clock.beat())
	c.lidT = g.clock.next(g.now, g.now+lead, 2)
	c.phase = phaseToLid
	c.moveTo(g.now, pressX, c.lidT, false)
	g.scheduleNext(c.lidT + tapWindows[2] + g.lvl().minGap)
}

// checkTimeouts fails any event whose window has closed unanswered.
func (g *BusyBarista) checkTimeouts() {
	for _, c := range g.cups {
		switch {
		case c.kind == cupEspresso && c.phase == phaseIncoming:
			if g.now >= c.t0 && !c.notified {
				// The successor can be scheduled as soon as this shot is under
				// way, since its time is fixed.
				c.notified = true
				g.scheduleNext(c.target + tapWindows[2] + g.lvl().minGap)
			}
			if g.now > c.target+tapWindows[2] {
				g.resolve(c, tierMiss, spoutX, "MISSED SHOT")
				g.failCup(c)
			}
		case c.kind != cupEspresso && (c.phase == phaseIncoming || c.phase == phaseWaiting):
			if g.now >= c.target {
				c.phase = phaseWaiting
			}
			if g.now > c.target+holdStartGrace {
				g.resolve(c, tierMiss, spoutX, "TOO SLOW")
				g.failCup(c)
				g.scheduleNext(g.now + g.lvl().minGap)
			}
		case c.phase == phasePouring:
			c.fill = math.Min(1, (g.now-c.holdStart)*c.lineFrac/c.fillDur)
			if g.now > c.lineT+holdWindows[2] {
				g.stopPour()
				g.btn.disarm()
				c.spilled = true
				c.fill = 1
				play(spillPCM)
				g.splash(c)
				g.resolve(c, tierMiss, spoutX, "SPILL!")
				g.failCup(c)
				g.scheduleNext(g.now + g.lvl().minGap)
			}
		case c.phase == phaseToLid && !c.lidded:
			if g.now > c.lidT+tapWindows[2] {
				g.resolve(c, tierMiss, pressX, "NO LID")
				g.failCup(c)
			}
		}
	}
}

// resolve records a judgement and shows it at the action point.
func (g *BusyBarista) resolve(c *cup, t tier, x float64, missLabel string) {
	pts, lifeBack := g.sc.hit(t)
	_, h := c.kind.size()
	y := beltY - h - 14
	label := tierLabels[t]
	if t == tierMiss {
		c.missed = true
		g.shake = 0.25
		if missLabel != "" {
			label = missLabel
		}
		if !c.spilled {
			play(missPCM)
		}
	} else {
		g.hop = 0.22
	}
	g.addPopup(x, y, label, tierColors[t], 8)
	if pts > 0 {
		g.addPopup(x, y+9, "+"+strconv.Itoa(pts), rgb(255, 255, 255), 5)
	}
	if lifeBack {
		g.addPopup(x, y-10, "+1 LIFE", rgb(255, 140, 170), 7)
	}
}

func (g *BusyBarista) finishItem(c *cup, base int) {
	if c.missed {
		return
	}
	pts := g.sc.bonus(base)
	_, h := c.kind.size()
	g.addPopup(c.x(g.now), beltY-h-36, "ORDER UP +"+strconv.Itoa(pts), rgb(255, 200, 120), 5)
}

func (g *BusyBarista) failCup(c *cup) {
	c.failed = true
	c.missed = true
	if c.phase != phaseToLid && c.kind != cupEspresso {
		// Shove it off down the belt.
		c.moveTo(g.now, counterEndX+30, g.now+0.6, false)
	}
	c.phase = phaseDone
}

func (g *BusyBarista) stopPour() {
	if g.pour != nil {
		g.pour.Pause()
		g.pour.Close()
		g.pour = nil
	}
}

// updateCups moves cups along, serves the finished ones at the counter end
// and drops the failed ones off it.
func (g *BusyBarista) updateCups(dt float64) {
	g.beltOffset = math.Mod(g.beltOffset+dt*40, 8)
	alive := g.cups[:0]
	for _, c := range g.cups {
		x := c.x(g.now)
		if c.lidded {
			c.steamT -= dt
			if c.steamT <= 0 {
				c.steamT = 0.25 + g.rng.Float64()*0.3
				_, h := c.kind.size()
				g.particles = append(g.particles, particle{
					x: x + (g.rng.Float64()-0.5)*4, y: beltY - h - 2, vx: (g.rng.Float64() - 0.5) * 4, vy: -10,
					life: 1, max: 1, size: 2, clr: [4]float32{1, 1, 1, 0.35},
				})
			}
		}
		if x >= counterEndX-8 && !c.failed {
			// Served: the front of the queue takes it.
			g.served++
			g.queue[0].hop = 0.3
			continue
		}
		if x >= counterEndX && c.failed {
			// Off the end of the counter and onto the floor.
			c.fallVY += 500 * dt
			c.fallY += c.fallVY * dt
			c.tilt += dt * 8
			if c.fallY > 120 {
				continue
			}
		}
		if x > tdcgame.ScreenW+40 {
			continue
		}
		alive = append(alive, c)
	}
	g.cups = alive
}

// — effects ———————————————————————————————————————————————————————————————

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{r, g, b, 255} }

func (g *BusyBarista) addPopup(x, y float64, text string, c color.RGBA, size float64) {
	g.popups = append(g.popups, newPopup(x, y, text, c, size))
}

func (g *BusyBarista) splash(c *cup) {
	_, h := c.kind.size()
	for range 18 {
		g.particles = append(g.particles, particle{
			x: spoutX + (g.rng.Float64()-0.5)*10, y: beltY - h,
			vx: (g.rng.Float64() - 0.5) * 90, vy: -40 - g.rng.Float64()*70, gravity: 300,
			life: 0.8, max: 0.8, size: 2, clr: [4]float32{0.42, 0.24, 0.13, 1},
		})
	}
}

func (g *BusyBarista) steamPuff(x, y float64) {
	for range 8 {
		g.particles = append(g.particles, particle{
			x: x + (g.rng.Float64()-0.5)*8, y: y,
			vx: (g.rng.Float64() - 0.5) * 30, vy: -15 - g.rng.Float64()*20,
			life: 0.7, max: 0.7, size: 3, clr: [4]float32{1, 1, 1, 0.5},
		})
	}
}

func (g *BusyBarista) updateParticles(dt float64) {
	alive := g.particles[:0]
	for _, p := range g.particles {
		p.vy += p.gravity * dt
		p.x += p.vx * dt
		p.y += p.vy * dt
		p.life -= dt
		if p.life > 0 {
			alive = append(alive, p)
		}
	}
	g.particles = alive
}
