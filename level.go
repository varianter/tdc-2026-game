package main

import "math"

type GameObjectType int

const (
	Platform GameObjectType = iota
)

var stateName = map[GameObjectType]string{
	Platform: "platform",
}

func (ss GameObjectType) String() string {
	return stateName[ss]
}

type Position struct {
	x, y float64
}

type GameObject struct {
	p Position
	t GameObjectType
}

type Level struct {
	gameObjects []GameObject
}

func NewLevel() *Level {
	objs := LoadDefault()

	return &Level{gameObjects: objs}
}

func LoadDefault() []GameObject {
	objs := []GameObject{{t: Platform, p: Position{x: 60, y: 0}}, {t: Platform, p: Position{x: 60, y: 20}}}

	// objs := []GameObject{{t: Platform, p: Position{x: 60, y:20}}, {t: Platform, p: Position{x: 60, y: 0}}}
	return objs
}

type CollisionResult struct {
	collideX Collision
	collideY Collision
}

type Collision struct {
	cord      float64
	collision bool
}

func (l *Level) collide(px, py float64) CollisionResult {
	pw := float64(32)
	ph := float64(32)
	pYOffset := float64(1) // TODO: Chicken sprite has 1px of empty space at the bottom

	collision := CollisionResult{collideX: Collision{}, collideY: Collision{}}
	for _, obj := range l.gameObjects {
		gp := obj.p
		// TODO: Hard coded size
		gpw := float64(64)
		gph := float64(32)

		feetOffset := float64(13)

		// TODO: Collide on stuff above us
		if (py <= gp.y+gph) &&
			((px > gp.x && px+feetOffset < gp.x+gpw) ||
				(px+pw-feetOffset > gp.x && px+pw < gp.x+gpw)) {
			collideY := gp.y + gph - pYOffset
			collision.collideY.collision = true
			collision.collideY.cord = math.Max(collideY, collision.collideY.cord)
		}

		// Crashing into stuff going right
		if px < gp.x+gpw && px+pw > gp.x &&
			// We are on the right y coordinates
			((py >= gp.y-10 && py <= gp.y-10+gph) || // feet colliding with top
				(py+ph >= gp.y-10 && py+ph <= gp.y-10+gph)) { // head colliding with bottom
			if px < gp.x+(gpw/2) {

				collideX := gp.x - pw - 1
				collision.collideX.collision = true
				collision.collideX.cord = math.Max(collideX, collision.collideX.cord)
			} else {

				collideX := gp.x + gpw + 1
				collision.collideX.collision = true
				collision.collideX.cord = math.Max(collideX, collision.collideX.cord)
			}
		}

		// Collide with ground
		if py <= 0 {
			collision.collideY.collision = true
			collision.collideY.cord = 0
		}

	}
	return collision
}
