package busybarista

import (
	"math/rand"
	"testing"
)

const testDT = 1.0 / 60

func newTestGame(seed int64) *BusyBarista {
	return &BusyBarista{
		sc:     newScoring(),
		clock:  beatClock{bpm: levels[0].bpm},
		rng:    rand.New(rand.NewSource(seed)),
		queue:  make([]figure, 3),
		pressT: 10,
	}
}

// bot plays by reading the game's own targets, aiming offset seconds late.
type bot struct {
	g      *BusyBarista
	offset float64
	down   bool
}

func (b *bot) key() bool {
	g := b.g
	t := g.now + testDT // the time the next update will judge at
	if c := g.pouringCup(); c != nil {
		b.down = t < c.lineT+b.offset-testDT/2
		return b.down
	}
	if b.down {
		b.down = false // release a tap straight away
		return false
	}
	for _, c := range g.cups {
		var target float64
		switch {
		case c.kind == cupEspresso && c.phase == phaseIncoming:
			target = c.target
		case c.kind != cupEspresso && (c.phase == phaseIncoming || c.phase == phaseWaiting):
			target = c.target
		case c.phase == phaseToLid && !c.lidded:
			target = c.lidT
		default:
			continue
		}
		if t >= target+b.offset-testDT/2 {
			b.down = true
			return true
		}
		return false
	}
	return false
}

// runBot runs a full game with the bot and returns the finished game.
func runBot(t *testing.T, seed int64, offset float64, check func(g *BusyBarista)) *BusyBarista {
	t.Helper()
	g := newTestGame(seed)
	b := &bot{g: g, offset: offset}
	keyDown = b.key
	t.Cleanup(func() { keyDown = func() bool { return false } })

	for range int(70 / testDT) {
		g.update(testDT)
		if check != nil {
			check(g)
		}
		if g.state == stateGameOver {
			break
		}
	}
	return g
}

// openWindows counts the input windows open right now.
func openWindows(g *BusyBarista) int {
	n := 0
	for _, c := range g.cups {
		switch {
		case c.kind == cupEspresso && c.phase == phaseIncoming:
			if abs(g.now-c.target) <= tapWindows[2] {
				n++
			}
		case c.kind != cupEspresso && (c.phase == phaseIncoming || c.phase == phaseWaiting):
			if g.now >= c.target-holdOpenLead && g.now <= c.target+holdStartGrace {
				n++
			}
		case c.phase == phasePouring:
			n++
		case c.phase == phaseToLid && !c.lidded:
			if abs(g.now-c.lidT) <= tapWindows[2] {
				n++
			}
		}
	}
	return n
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// A flawless player survives to the cap with nothing but PERFECTs, and the
// scheduler never asks for two inputs at once on the way.
func TestPerfectPlayReachesCapWithoutOverlap(t *testing.T) {
	for seed := int64(1); seed <= 20; seed++ {
		g := runBot(t, seed, 0, func(g *BusyBarista) {
			if n := openWindows(g); n > 1 {
				t.Fatalf("seed %d: %d input windows open at t=%.2f", seed, n, g.now)
			}
		})
		if g.sc.lives == 0 || g.now < runCap {
			t.Fatalf("seed %d: run ended at t=%.2f with %d lives", seed, g.now, g.sc.lives)
		}
		if g.sc.counts[tierMiss] != 0 || g.sc.counts[tierGood] != 0 || g.sc.counts[tierOK] != 0 {
			t.Fatalf("seed %d: expected only perfects, got %v", seed, g.sc.counts)
		}
		if g.levelIdx != len(levels)-1 {
			t.Fatalf("seed %d: only reached level %d", seed, g.levelIdx+1)
		}
	}
}

// Being consistently 100ms late is still inside OK on every event.
func TestLateButInsideOK(t *testing.T) {
	g := runBot(t, 3, 0.1, nil)
	if g.sc.counts[tierMiss] != 0 {
		t.Fatalf("expected no misses, got %v", g.sc.counts)
	}
	if g.sc.counts[tierOK] == 0 {
		t.Fatalf("expected OK judgements, got %v", g.sc.counts)
	}
}

// Overshooting a pour past the OK window spills.
func TestOvershootSpills(t *testing.T) {
	g := runBot(t, 5, 0.2, nil)
	if g.state != stateGameOver || g.sc.lives != 0 {
		t.Fatalf("expected the run to be lost, ended at t=%.2f with %d lives", g.now, g.sc.lives)
	}
	if g.sc.counts[tierMiss] != maxLives {
		t.Fatalf("expected %d misses, got %v", maxLives, g.sc.counts)
	}
}

// Doing nothing loses all lives quickly.
func TestIdleLoses(t *testing.T) {
	g := newTestGame(1)
	keyDown = func() bool { return false }
	for range int(20 / testDT) {
		g.update(testDT)
	}
	if g.state != stateGameOver || g.sc.lives != 0 {
		t.Fatalf("idle player still alive after 20s: state=%v lives=%d", g.state, g.sc.lives)
	}
}

// Mashing breaks the combo on every stray press, so it never builds a
// multiplier.
func TestMashingEarnsNoMultiplier(t *testing.T) {
	g := newTestGame(1)
	frame := 0
	keyDown = func() bool { return frame%6 < 3 }
	for range int(20 / testDT) {
		frame++
		g.update(testDT)
		if g.sc.multiplier() > 1 {
			t.Fatalf("mashing reached multiplier x%d at t=%.2f", g.sc.multiplier(), g.now)
		}
	}
}

// At the run cap no more cups are sent, but the ones already on the belt are
// played out: the run never ends mid-pour or with an input still due.
func TestCapLetsBeltFinish(t *testing.T) {
	for seed := int64(1); seed <= 20; seed++ {
		var onBelt map[*cup]bool
		g := runBot(t, seed, 0, func(g *BusyBarista) {
			if g.closing && onBelt == nil {
				onBelt = map[*cup]bool{}
				for _, c := range g.cups {
					onBelt[c] = true
				}
			}
			for _, c := range g.cups {
				if onBelt != nil && !onBelt[c] {
					t.Fatalf("seed %d: a cup was sent at t=%.2f after closing", seed, g.now)
				}
			}
			if g.state == stateGameOver && (g.pouringCup() != nil || g.hasPendingCup()) {
				t.Fatalf("seed %d: run ended at t=%.2f with a cup still in play", seed, g.now)
			}
		})
		if onBelt == nil {
			t.Fatalf("seed %d: the shop never closed", seed)
		}
		if g.state != stateGameOver || g.sc.counts[tierMiss] != 0 {
			t.Fatalf("seed %d: expected a clean finish, state=%v counts=%v", seed, g.state, g.sc.counts)
		}
	}
}

// The level that introduces espresso sends one as its first cup, and only once
// the longer banner and breather are over.
func TestEspressoIntroducedAfterBanner(t *testing.T) {
	introLevel := -1
	for i, l := range levels {
		if l.intro != nil && l.intro.kind == cupEspresso {
			introLevel = i
		}
	}
	if introLevel < 0 {
		t.Fatal("no level introduces espresso")
	}
	for seed := int64(1); seed <= 20; seed++ {
		start := -1.0
		seen := map[*cup]bool{}
		var first *cup
		runBot(t, seed, 0, func(g *BusyBarista) {
			if start < 0 && g.levelIdx == introLevel {
				start = g.now
				for _, c := range g.cups {
					if g.onBelt(c) {
						seen[c] = true // already in view
					}
				}
			}
			if start < 0 || first != nil {
				return
			}
			for _, c := range g.cups {
				if !seen[c] {
					first = c
					break
				}
			}
		})
		switch {
		case first == nil:
			t.Fatalf("seed %d: no cup sent after level %d started", seed, introLevel+1)
		case first.kind != cupEspresso:
			t.Fatalf("seed %d: first new cup was kind %d, not espresso", seed, first.kind)
		case first.target < start+introBreather:
			t.Fatalf("seed %d: espresso due at %.2f, before the breather ends at %.2f", seed, first.target, start+introBreather)
		}
	}
}
