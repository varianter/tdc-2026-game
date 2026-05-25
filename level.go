package main

import (
	"image/color"
	"math"
)

type GameObjectType int

const (
	Platform GameObjectType = iota
	Coin
	Flag
)

var gameObjectName = map[GameObjectType]string{
	Platform: "platform",
	Coin:     "coin",
	Flag:     "flag",
}

func (ss GameObjectType) String() string {
	return gameObjectName[ss]
}

type Square struct {
	p Position
	w float64
	h float64
}

type MovingSquare struct {
	*Square
	*Moving
}

type Moving struct {
	vy, vx, direction float64
	onground          bool
}

// Calculate next position based on velocity
func (ps *MovingSquare) nextPos(dt float64) {
	x := ps.p.x + (ps.vx * dt * ps.direction)
	y := ps.p.y + (ps.vy * dt)
	ps.p = Position{x: x, y: y}
}

func (sq *Square) top() float64 {
	return sq.p.y + sq.h
}

func (sq *Square) btm() float64 {
	return sq.p.y
}

func (sq *Square) left() float64 {
	return sq.p.x
}

func (sq *Square) right() float64 {
	return sq.p.x + sq.w
}

func (sq *Square) center_y() float64 {
	return sq.p.y + sq.h/2
}

func (sq *Square) center_x() float64 {
	return sq.p.x + sq.w/2
}

func (sq *Square) collides(psq Square) bool {
	if psq.p.x < sq.p.x+sq.w && // left side of player is at the left of the right side of obj
		psq.p.x+psq.w > sq.p.x && // right side of player is at the right of the left side of obj
		psq.p.y+psq.h > sq.p.y && // top side of player is above the bottom side of ob
		psq.p.y < sq.p.y+sq.h { // the bottom side of player is underneath the top side of obj
		return true
	} else {
		return false
	}
}

func (obj *Square) get_overlap(p Square) Square {
	yTop := math.Min(obj.p.y+obj.h, p.p.y+p.h)
	yBottom := math.Max(obj.p.y, p.p.y)

	xLeft := math.Max(obj.p.x, p.p.x)
	xRight := math.Min(obj.p.x+obj.w, p.p.x+p.w)

	w := xRight - xLeft
	h := yTop - yBottom

	return Square{
		p: Position{x: xLeft, y: yBottom},
		w: w, h: h,
	}
}

type Position struct {
	x, y float64
}

type GameObject struct {
	t       GameObjectType
	s       Square
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
	return GameObject{t: Platform, s: Square{p: Position{x: x, y: y}, w: float64(100), h: float64(32)}}
}

func newCoin(x, y float64) GameObject {
	return GameObject{t: Coin, s: Square{p: Position{x: x, y: y}, w: float64(10), h: float64(10)}}
}

func newFlag(x, y float64) GameObject {
	return GameObject{t: Flag, s: Square{p: Position{x: x, y: y}, w: float64(4), h: float64(64 + 20)}}
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

		newPlatform(920, 190),
		newFlag(920+80, 190+32),

		newCoin(760+45, 400+32+13),
		newPlatform(760, 400),

		newPlatform(560, 270),
		newFlag(560+80, 270+32),

		newPlatform(600, 80),
		newCoin(600+45, 80+32+13),
		newPlatform(700, 20),
		newCoin(700+45, 20+32+13),
	}

	return objs
}

type Col struct {
	idx    int
	solved map[int]struct{}
	t      GameObjectType
}

func (b *Col) next(l Level, pSquare Square, coins bool) bool {
	for i := range l.gameObjects {
		if _, ok := b.solved[i]; ok {
			continue
		}
		obj := &l.gameObjects[i]
		if obj.removed {
			continue
		}

		if obj.s.collides(pSquare) {
			b.idx = i
			b.t = obj.t
			return true
		}
	}

	return false
}

func (l *Level) collideObj(pSquare Square, gameObjIdx int) bool {
	obj := &l.gameObjects[gameObjIdx]
	if obj.removed || obj.t != Platform {
		return false
	}

	if obj.s.collides(pSquare) {
		return true
	}
	return false
}

func (l *Level) overlap(pSquare Square, gambObjIdx int) (Square, Square) {
	s := l.gameObjects[gambObjIdx].s

	return s.get_overlap(pSquare), s
}

func (l *Level) register_collision(gameObjIdx int) {
	obj := &l.gameObjects[gameObjIdx]
	if obj.t == Coin || obj.t == Flag {
		obj.removed = true
	}
}
