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

func (ps *MovingSquare) collide_x(s Square) {
	playerCenter := ps.center_x()
	blockCenter := s.center_x()
	if playerCenter < blockCenter { // player is to the left of block
		ps.p.x = s.left() - ps.w
	} else { // player is to the right of block
		ps.p.x = s.right()
	}
	ps.vx = 0
}

func (ps *MovingSquare) collide_y(s Square) {
	playerCenter := ps.center_y()
	blockCenter := s.center_y()
	if playerCenter >= blockCenter { // player is above block
		ps.p.y = s.top()
		ps.vy = 0
		ps.onground = true
	} else { // player is below block
		ps.p.y = s.btm() - ps.h
		ps.vy = 0
	}
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
	grid        map[Position][]int
	cell_size   float64
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

func NewLevelSmall() *Level {
	objs := SmallLadder()
	return NewLevelFromObjects(objs)
}

func NewLevel() *Level {
	objs := SmallLadder()
	return NewLevelFromObjects(objs)
}

func NewLevelFromObjects(objs []GameObject) *Level {
	return &Level{gameObjects: objs, cell_size: 128, grid: buildGrid(objs, 128)}
}

func NewLevelBig(iterations int) *Level {
	objs := BigLadder(iterations)
	return NewLevelFromObjects(objs)
}

func SmallLadder() []GameObject {
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

func BigLadder(iterations int) []GameObject {
	objs := []GameObject{}

	startIdx := float64(100)
	// TODO: Currently start struggling with 4000 iterations (= 84 000 game objects)
	for i := 0; i < iterations; i++ {
		objs = append(objs, newPlatform(startIdx, 20))
		objs = append(objs, newCoin(startIdx+45, 20+32+13))

		objs = append(objs, newPlatform(startIdx+100, 80))
		objs = append(objs, newCoin(startIdx+100+45, 80+32+13))

		objs = append(objs, newPlatform(startIdx+200, 140))
		objs = append(objs, newCoin(startIdx+200+45, 140+32+13))

		objs = append(objs, newPlatform(startIdx+300, 200))
		objs = append(objs, newCoin(startIdx+300+45, 200+32+13))

		objs = append(objs, newPlatform(startIdx+400, 140))
		objs = append(objs, newCoin(startIdx+400+45, 140+32+13))

		objs = append(objs, newPlatform(startIdx+660, 140))

		objs = append(objs, newPlatform(startIdx+820, 190))
		objs = append(objs, newFlag(startIdx+820+80, 190+32))

		objs = append(objs, newCoin(startIdx+660+45, 400+32+13))
		objs = append(objs, newPlatform(startIdx+660, 400))

		objs = append(objs, newPlatform(startIdx+460, 270))
		objs = append(objs, newFlag(startIdx+460+80, 270+32))

		objs = append(objs, newPlatform(startIdx+500, 80))
		objs = append(objs, newCoin(startIdx+500+45, 80+32+13))
		objs = append(objs, newPlatform(startIdx+500, 20))
		objs = append(objs, newCoin(startIdx+500+45, 20+32+13))

		startIdx += 1200
	}

	return objs
}

// handleCollision returns 1 if a coin was collected, 0 otherwise
func (l *Level) handleCollision(pSquare *MovingSquare, dt float64, overlap Square, objSquare Square, objType GameObjectType, objIdx int) int {
	if objType == Flag {
		pSquare.direction = pSquare.direction * -1.0

		playerCenter := pSquare.center_x()
		flagCenter := objSquare.center_x()
		if playerCenter < flagCenter { // player is to the left of block
			pSquare.p.x = objSquare.left() - pSquare.w - 1
		} else { // player is to the right of block
			pSquare.p.x = objSquare.right() + 1
		}
		l.register_collision(objIdx)
	}
	if objType == Coin {
		l.register_collision(objIdx)
		return 1
	}
	if objType == Platform {
		if overlap.w > overlap.h { // y is shallowest so solve y collision first
			pSquare.collide_y(objSquare)
			pSquare.nextPos(dt) // dont do this,get new pos

			if l.collideObj(*pSquare.Square, objIdx) {
				pSquare.collide_x(objSquare)
				pSquare.nextPos(dt)
			}
		} else {
			pSquare.collide_x(objSquare)
			pSquare.nextPos(dt)

			if l.collideObj(*pSquare.Square, objIdx) {
				pSquare.collide_y(objSquare)

				pSquare.nextPos(dt)
			}
		}
	}
	return 0
}

func buildGrid(gameObjects []GameObject, cell_size float64) map[Position][]int {
	grid := make(map[Position][]int)
	// Insert items into a grid system of size based on cell_size
	// Move this to the level constructor probably (then we need to recalc if we have moving game objects)
	for i := range gameObjects {
		obj := gameObjects[i]
		gx1 := math.Floor(obj.s.p.x / cell_size)
		gy1 := math.Floor(obj.s.p.y / cell_size)
		gx2 := math.Floor((obj.s.p.x + obj.s.w) / cell_size)
		gy2 := math.Floor((obj.s.p.y + obj.s.h) / cell_size)

		for gy := gy1; gy <= gy2; gy++ {
			for gx := gx1; gx <= gx2; gx++ {
				pos := Position{x: gx, y: gy}
				arr, ok := grid[pos]
				if !ok {
					grid[pos] = []int{i}
				} else {
					grid[pos] = append(arr, i)
				}
			}
		}
	}

	return grid
}

func (l *Level) findClosestElements(pSquare *MovingSquare) map[int]struct{} {
	elements := make(map[int]struct{})
	grid_x1 := math.Floor(pSquare.p.x / l.cell_size)
	grid_y1 := math.Floor(pSquare.p.y / l.cell_size)
	grid_x2 := math.Floor((pSquare.p.x + pSquare.w) / l.cell_size)
	grid_y2 := math.Floor((pSquare.p.y + pSquare.h) / l.cell_size)
	for gy := grid_y1; gy <= grid_y2; gy++ {
		for gx := grid_x1; gx <= grid_x2; gx++ {
			pos := Position{x: gx, y: gy}
			for _, otherIdx := range l.grid[pos] {
				if !l.gameObjects[otherIdx].removed {
					elements[otherIdx] = struct{}{}
				}
			}
		}
	}
	return elements
}

func (l *Level) resolveCollisions(pSquare *MovingSquare, dt float64) int {
	elements := l.findClosestElements(pSquare)

	pSquare.nextPos(dt)
	pc := ColIterator{idx: -1, elements: &elements}
	// We loop through the objects until we dont collide with anything
	for pc.next(l, *pSquare.Square, false) {
		overlap, objSquare := l.overlap(*pSquare.Square, pc.idx)

		pc.coins += l.handleCollision(pSquare, dt, overlap, objSquare, pc.t, pc.idx)

		delete(*pc.elements, pc.idx)
	}

	// Collide with ground
	if pSquare.p.y <= 0 {
		pSquare.p.y = 0
		pSquare.vy = 0
		pSquare.onground = true
	}

	if !pSquare.onground {
		pSquare.vy -= Gravity * dt // Always apply gravity to avoid shenanigans
	}

	// Turn around at start flag
	if pSquare.p.x < 0 {
		pSquare.p.x = 0
		pSquare.direction = pSquare.direction * -1.0
	}

	return pc.coins
}

type ColIterator struct {
	idx      int
	elements *map[int]struct{}
	coins    int
	t        GameObjectType
}

// Returns the first colliding object
func (citer *ColIterator) next(l *Level, pSquare Square, coins bool) bool {
	for i := range *citer.elements { // TODO: Might make more sense to start from idx instead of starting from the top every time
		obj := &l.gameObjects[i]

		if obj.s.collides(pSquare) {
			citer.idx = i
			citer.t = obj.t
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
