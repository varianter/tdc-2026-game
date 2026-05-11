package main

import (
	"image/color"
	"math"
)

type GameObjectType int

const (
	Platform GameObjectType = iota
	Coin
)

var stateName = map[GameObjectType]string{
	Platform: "platform",
	Coin:     "coin",
}

func (ss GameObjectType) String() string {
	return stateName[ss]
}

type Position struct {
	x, y float64
}

type GameObject struct {
	p       Position
	t       GameObjectType
	w       float64
	h       float64
	removed bool
}

func (g *GameObject) Color() color.RGBA {
	switch g.t {
	case Platform:
		return color.RGBA{27, 130, 0, 255}
	case Coin:
		return color.RGBA{255, 201, 56, 255}
	default:
		panic("Unknown gameobjecttype")
	}
}

type Level struct {
	gameObjects []GameObject
}

func newPlatform(x, y float64) GameObject {
	return GameObject{t: Platform, p: Position{x: x, y: y}, w: float64(100), h: float64(32)}
}

func newCoin(x, y float64) GameObject {
	return GameObject{t: Coin, p: Position{x: x, y: y}, w: float64(10), h: float64(10)}
}

func NewLevel() *Level {
	objs := LoadLadder()

	return &Level{gameObjects: objs}
}

func LoadLadder() []GameObject {
	objs := []GameObject{
		newPlatform(100, 20),
		newCoin(100+45, 20+32+13),
		newPlatform(200, 80),
		newCoin(200+45, 80+32+13),
		newPlatform(300, 140),
		newCoin(300+45, 140+32+13),
		newPlatform(400, 200),
		newCoin(400+45, 200+32+13),
		newPlatform(500, 140),
		newCoin(500+45, 140+32+13),
		newPlatform(600, 80),
		newCoin(600+45, 80+32+13),
		newPlatform(700, 20),
		newCoin(700+45, 20+32+13),
	}

	return objs
}

type CollisionResult struct {
	collideX Collision
	collideY Collision
	coin     []Position
}

type Collision struct {
	cord      float64
	collision bool
}

func (l *Level) collide(px, py float64) CollisionResult {
	pw := float64(32)
	ph := float64(32)
	pYOffset := float64(1) // TODO: Chicken sprite has 1px of empty space at the bottom

	collision := CollisionResult{collideX: Collision{}, collideY: Collision{}, coin: []Position{}}
	for i, obj := range l.gameObjects {
		if obj.removed {
			continue
		}

		coin := obj.t == Coin
		gp := obj.p
		// TODO: Hard coded size
		gpw := obj.w
		gph := obj.h

		feetOffset := float64(13)

		// TODO: Collide on stuff above us
		//
		// Collide with stuff below us
		if (py <= gp.y+gph) &&
			((px > gp.x && px+feetOffset < gp.x+gpw) ||
				(px+pw-feetOffset > gp.x && px+pw < gp.x+gpw)) {
			collideY := gp.y + gph - pYOffset
			if coin {
				collision.coin = append(collision.coin, gp)
				l.gameObjects[i].removed = true
			} else {

				collision.collideY.collision = true
				collision.collideY.cord = math.Max(collideY, collision.collideY.cord)
			}
		}

		if px < gp.x+gpw && px+pw > gp.x &&
			// We are on the right y coordinates
			((py >= gp.y-10 && py <= gp.y-10+gph) || // feet colliding with top
				(py+ph >= gp.y-10 && py+ph <= gp.y-10+gph)) { // head colliding with bottom
			if coin {
				collision.coin = append(collision.coin, gp)
				l.gameObjects[i].removed = true
			} else {
				if px < gp.x+(gpw/2) { // going right

					collideX := gp.x - pw - 1
					collision.collideX.collision = true
					collision.collideX.cord = math.Max(collideX, collision.collideX.cord)
				} else { // going left

					collideX := gp.x + gpw + 1
					collision.collideX.collision = true
					collision.collideX.cord = math.Max(collideX, collision.collideX.cord)
				}
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
