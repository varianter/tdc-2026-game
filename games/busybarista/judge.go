package busybarista

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// This file is the timing core: judging, scoring, lives and input edges.

type tier int

const (
	tierPerfect tier = iota
	tierGood
	tierOK
	tierMiss
)

// Judgement windows in seconds, as ± error from the target moment, for
// PERFECT, GOOD and OK. Anything wider is a MISS. Holds get slightly wider
// windows than taps, since the release is judged against a moving meter.
var (
	tapWindows  = [3]float64{0.030, 0.070, 0.120}
	holdWindows = [3]float64{0.040, 0.090, 0.150}
)

func judge(err float64, w [3]float64) tier {
	a := math.Abs(err)
	switch {
	case a <= w[0]:
		return tierPerfect
	case a <= w[1]:
		return tierGood
	case a <= w[2]:
		return tierOK
	}
	return tierMiss
}

var tierPoints = [...]int{tierPerfect: 300, tierGood: 150, tierOK: 50, tierMiss: 0}

var tierLabels = [...]string{tierPerfect: "PERFECT", tierGood: "GOOD", tierOK: "OK", tierMiss: "MISS"}

var tierColors = [...]color.RGBA{
	tierPerfect: {255, 222, 90, 255},
	tierGood:    {120, 230, 140, 255},
	tierOK:      {150, 190, 255, 255},
	tierMiss:    {255, 80, 70, 255},
}

const (
	maxLives = 3

	// lifeRegenStreak PERFECTs in a row restore a life (up to maxLives).
	lifeRegenStreak = 15

	// itemBonus is awarded, times the multiplier, for an item finished with no
	// MISS.
	itemBonus = 200
)

// scoring tracks score, combo and lives for one run.
type scoring struct {
	score         int
	combo         int
	maxCombo      int
	lives         int
	perfectStreak int
	counts        [4]int
}

func newScoring() scoring { return scoring{lives: maxLives} }

// multiplier steps up with the combo: ×1, ×2 at 10, ×3 at 25, ×4 at 50.
func (s *scoring) multiplier() int {
	switch {
	case s.combo >= 50:
		return 4
	case s.combo >= 25:
		return 3
	case s.combo >= 10:
		return 2
	}
	return 1
}

// hit records a judgement and returns the points it earned and whether it
// restored a life. A MISS costs a life and resets the combo.
func (s *scoring) hit(t tier) (points int, lifeRestored bool) {
	s.counts[t]++
	if t == tierMiss {
		s.combo = 0
		s.perfectStreak = 0
		s.lives = max(0, s.lives-1)
		return 0, false
	}
	s.combo++
	s.maxCombo = max(s.maxCombo, s.combo)
	points = tierPoints[t] * s.multiplier()
	s.score += points

	if t != tierPerfect {
		s.perfectStreak = 0
		return points, false
	}
	s.perfectStreak++
	if s.perfectStreak >= lifeRegenStreak {
		s.perfectStreak = 0
		if s.lives < maxLives {
			s.lives++
			return points, true
		}
	}
	return points, false
}

// stray is a press with nothing to hit: it breaks the combo but costs no life,
// so mashing doesn't pay.
func (s *scoring) stray() {
	s.combo = 0
	s.perfectStreak = 0
}

// bonus awards a flat bonus scaled by the multiplier and returns it.
func (s *scoring) bonus(base int) int {
	p := base * s.multiplier()
	s.score += p
	return p
}

// keyDown reports whether the one button is held. Tests swap it out to play
// the game.
var keyDown = func() bool { return ebiten.IsKeyPressed(ebiten.KeyEnter) }

// button turns the held state of the one button into press/release edges.
// It starts disarmed: the press that started the game is still held on the
// first frame, and must be let go before anything counts.
type button struct {
	down  bool
	armed bool
}

// poll returns this frame's edges. Disarmed, it reports nothing until the
// button has been released once.
func (b *button) poll() (pressed, released bool) {
	now := keyDown()
	if !b.armed {
		if !now {
			b.armed = true
		}
		b.down = now
		return false, false
	}
	pressed = now && !b.down
	released = !now && b.down
	b.down = now
	return pressed, released
}

// disarm swallows the rest of the current hold, e.g. when a pour is cut off
// by a spill while the button is still down.
func (b *button) disarm() { b.armed = false }

// popup is floating feedback text (judgements, points) in viewport space.
type popup struct {
	x, y    float64
	text    string
	clr     color.RGBA
	size    float64
	life    float64
	maxLife float64
}

func newPopup(x, y float64, text string, clr color.RGBA, size float64) popup {
	return popup{x: x, y: y, text: text, clr: clr, size: size, life: 0.7, maxLife: 0.7}
}

func updatePopups(ps []popup, dt float64) []popup {
	alive := ps[:0]
	for _, p := range ps {
		p.life -= dt
		p.y -= 22 * dt
		if p.life > 0 {
			alive = append(alive, p)
		}
	}
	return alive
}

// beatClock is the loose beat: a phase that advances with the current tempo.
// Events are scheduled onto upcoming beats.
type beatClock struct {
	phase float64 // beats since start
	bpm   float64
}

// advance moves the clock on by dt.
func (c *beatClock) advance(dt float64) {
	c.phase += dt * c.bpm / 60
}

// next returns the time of the first subdivision (1 = beats, 2 = half beats)
// at or after time t, assuming the current tempo holds until then.
func (c *beatClock) next(now, t float64, sub float64) float64 {
	bps := c.bpm / 60 * sub
	k := math.Ceil(c.phase*sub + (t-now)*bps - 1e-9)
	return now + (k-c.phase*sub)/bps
}

func (c *beatClock) beat() float64 { return 60 / c.bpm }
