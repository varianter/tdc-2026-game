package main

import (
	"testing"
	"time"
)

// Platform at (100, 20): left=100, right=200, btm=20, top=52

// NewTestSquare creates a player square with collision offsets applied.
// The resulting Square has: x = x+15, w = 34, h = 49
// So: left = x+15, right = x+49, btm = y, top = y+49

func clonePlayer(p *MovingSquare) *MovingSquare {
	sq := *p.Square
	mv := *p.Moving
	return &MovingSquare{&sq, &mv}
}

func runBothCollisionMethods(t *testing.T, player *MovingSquare, platform GameObject, dt float64, check func(t *testing.T, p *MovingSquare)) {
	t.Helper()

	l := NewLevelFromObjects([]GameObject{platform})
	p := clonePlayer(player)

	start := time.Now()
	l.resolveCollisionsInspectAll(p, dt)
	inspectAllDur := time.Since(start)

	t.Logf("resolveCollisionsInspectAll took %v", inspectAllDur)
	check(t, p)

	// Reset player for grid test
	l2 := NewLevelFromObjects([]GameObject{platform})
	p2 := clonePlayer(player)

	start = time.Now()
	l2.resolveCollisionsGrid(p2, dt)
	gridDur := time.Since(start)

	t.Logf("resolveCollisionsGrid took %v", gridDur)
	check(t, p2)
}

func TestCollisionXAxisFromLeft(t *testing.T) {
	// Player moving right (direction=1), approaching platform from the left.
	// Platform: x=100, w=100 => left=100, right=200
	// Place player so it just overlaps the left edge of the platform.
	// Player right edge (x+15+34 = x+49) should be slightly past platform left (100).
	// So x+49 = 101 => x = 52. Player left = 52+15 = 67.
	dt := 1.0 / 60
	platform := newPlatform(100, 20)

	runBothCollisionMethods(t, NewTestSquare(52, 20, 1, 0), platform, dt, func(t *testing.T, p *MovingSquare) {
		// After collision, player right edge should be pushed back to platform left (100).
		// player.p.x (left of collision square) should be 100 - 34 = 66
		expectedX := platform.s.left() - p.w
		if p.p.x != expectedX {
			t.Errorf("X from left: got p.x = %f; want %f", p.p.x, expectedX)
		}
		if p.vx != 0 {
			t.Errorf("X from left: vx should be 0 after collision, got %f", p.vx)
		}
	})
}

func TestCollisionXAxisFromRight(t *testing.T) {
	// Player moving left (direction=-1), approaching platform from the right.
	// Platform right=200. Player left edge (x+15) should be slightly past platform right (200).
	// So x+15 = 199 => x = 184.
	dt := 1.0 / 60
	platform := newPlatform(100, 20)

	runBothCollisionMethods(t, NewTestSquare(184, 20, -1, 0), platform, dt, func(t *testing.T, p *MovingSquare) {
		// After collision, player left edge pushed to platform right (200).
		expectedX := platform.s.right()
		if p.p.x != expectedX {
			t.Errorf("X from right: got p.x = %f; want %f", p.p.x, expectedX)
		}
		if p.vx != 0 {
			t.Errorf("X from right: vx should be 0 after collision, got %f", p.vx)
		}
	})
}

func TestCollisionYAxisFromAbove(t *testing.T) {
	// Player falling onto top of platform.
	// Platform: y=20, h=32, top=52.
	// Player with vy<0 (falling), center_y > platform center_y => lands on top.
	// Place player so btm (y) is just below platform top (52): y=50, top=50+49=99.
	// Player center_y = 50+24.5 = 74.5, platform center_y = 20+16 = 36 => player above platform.
	dt := 1.0 / 60
	platform := newPlatform(100, 20)

	runBothCollisionMethods(t, NewTestSquare(118, 50, 1, -50), platform, dt, func(t *testing.T, p *MovingSquare) {
		// After collision, player btm should be at platform top (52).
		expectedY := platform.s.top()
		if p.p.y != expectedY {
			t.Errorf("Y from above: got p.y = %f; want %f", p.p.y, expectedY)
		}
		if p.vy != 0 {
			t.Errorf("Y from above: vy should be 0 after landing, got %f", p.vy)
		}
		if !p.onground {
			t.Errorf("Y from above: player should be onground after landing")
		}
	})
}

func TestCollisionYAxisFromBelow(t *testing.T) {
	// Player jumping up and hitting the bottom of a platform.
	// Platform: y=100, h=32, btm=100, top=132.
	// Player with vy>0 (moving up), center_y < platform center_y => hits from below.
	// Place player so top (y+49) is just past platform btm (100): y=52, top=101.
	// Player center_y = 52+24.5 = 76.5, platform center_y = 100+16 = 116 => player below.
	dt := 1.0 / 60
	platform := newPlatform(100, 100)

	runBothCollisionMethods(t, NewTestSquare(118, 52, 1, 50), platform, dt, func(t *testing.T, p *MovingSquare) {
		// After collision, player top (p.y + p.h) should be pushed to platform btm (100).
		// i.e. p.p.y = platform.btm() - p.h
		expectedY := platform.s.btm() - p.h
		if p.p.y != expectedY {
			t.Errorf("Y from below: got p.y = %f; want %f", p.p.y, expectedY)
		}
		// vy is zeroed on collision but gravity is re-applied afterwards (onground=false),
		// so we just verify the player was pushed below the platform bottom.
		if p.p.y+p.h > platform.s.btm() {
			t.Errorf("Y from below: player top %f should not exceed platform btm %f", p.p.y+p.h, platform.s.btm())
		}
	})
}

func BenchmarkCollisionInspectAll(b *testing.B) {
	l := NewLevelBig(20000)
	p := NewTestSquare(0, 0, 1, 0)
	dt := 1.0 / 60

	for b.Loop() {
		l.resolveCollisionsInspectAll(p, dt)
	}
}

func BenchmarkCollisionGrid(b *testing.B) {
	l := NewLevelBig(20000)
	p := NewTestSquare(0, 0, 1, 0)
	dt := 1.0 / 60

	for b.Loop() {
		l.resolveCollisionsGrid(p, dt)
	}
}

func NewTestSquare(x, y, direction, vy float64) *MovingSquare {
	collisionOffset, w, h, vx := 15.0, 64.0, 64.0, 90.0
	return &MovingSquare{
		&Square{p: Position{x: x + collisionOffset, y: y}, w: w - (collisionOffset * 2), h: h - collisionOffset},
		&Moving{vy: vy, vx: vx, direction: direction, onground: false},
	}
}
