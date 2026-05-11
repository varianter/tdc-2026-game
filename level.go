package main

import (
	"image/color"
	"log"
	"math"
)

type GameObjectType int

const (
	Platform GameObjectType = iota
	Coin
	Flag
)

var stateName = map[GameObjectType]string{
	Platform: "platform",
	Coin:     "coin",
	Flag:     "flag",
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
	case Flag:
		return color.RGBA{96, 247, 57, 255}
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

func newFlag(x, y float64) GameObject {
	return GameObject{t: Flag, p: Position{x: x, y: y}, w: float64(4), h: float64(64 + 20)}
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

		newPlatform(760, 140),

		newPlatform(920, 200),
		newFlag(920+80, 200+32),

		newCoin(760+45, 240+32+13),
		newPlatform(760, 240),

		newPlatform(560, 300),
		newFlag(560+80, 300+32),

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
	reverse  bool
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
	for i, _ := range l.gameObjects {
		obj := &l.gameObjects[i]
		if obj.removed {
			continue
		}

		gp := obj.p
		gpw := obj.w
		gph := obj.h

		feetOffset := float64(13)

		// TODO: Collide on stuff above us

		// Collide with stuff below us
		if (py <= gp.y+gph && py+ph > gp.y+gph) && // falling past the top
			((px > gp.x && px+feetOffset < gp.x+gpw) || // body is inside, use feetoffset to make feet have to touch
				(px+pw-feetOffset > gp.x && px+pw < gp.x+gpw)) {
			collideY := gp.y + gph - pYOffset
			l.calcCollision(&collision, collideY, obj, true)
		}

		if px < gp.x+gpw && px+pw > gp.x && // body passing into box
			((py >= gp.y-10 && py <= gp.y-10+gph) || // feet colliding with top
				(py+ph >= gp.y-10 && py+ph <= gp.y-10+gph)) { // head colliding with bottom
			l.calcCollision(&collision, collideX(px, pw, gp.x, gpw), obj, false)
		}

		// Collide with ground
		if py <= 0 {
			collision.collideY.collision = true
			collision.collideY.cord = 0
		}

		if px < 0 {
			collision.collideX.collision = true
			collision.collideX.cord = 0
			collision.reverse = true
		}

	}
	return collision
}

func collideX(px, pw, gpx, gpw float64) float64 {
	if px < gpx+(gpw/2) { // going right
		return gpx - pw - 1
	} else { // going left
		return gpx + gpw + 1
	}
}

// Returns true if the game object is to be removed
func (l *Level) calcCollision(collision *CollisionResult, cord float64, gobj *GameObject, yCoords bool) {
	switch gobj.t {
	case Platform:
		{
			if yCoords {
				collision.collideY.collision = true
				collision.collideY.cord = math.Max(cord, collision.collideY.cord)
			} else {
				collision.collideX.collision = true
				collision.collideX.cord = math.Max(cord, collision.collideX.cord)
			}
		}
	case Coin:
		{
			collision.coin = append(collision.coin, gobj.p)
			gobj.removed = true
		}
	case Flag:
		{
			if !yCoords {
				collision.collideX.cord = cord
			}
			collision.reverse = true
			gobj.removed = true
		}

	default:
		log.Printf("calcCollision: Unknown GameObjectType %s \n", gobj.t.String())
	}
}
