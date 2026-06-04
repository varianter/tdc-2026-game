package tdcgame

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
	P Position
	W float64
	H float64
}

type MovingSquare struct {
	*Square
	*Moving
}

type Moving struct {
	Vy, Vx, Direction float64
	Onground          bool
}

// Calculate next position based on velocity
func (ps *MovingSquare) NextPos(dt float64) {
	x := ps.P.X + (ps.Vx * dt * ps.Direction)
	y := ps.P.Y + (ps.Vy * dt)
	ps.P = Position{X: x, Y: y}
}

func (ps *MovingSquare) CollideX(s Square) {
	playerCenter := ps.CenterX()
	blockCenter := s.CenterX()
	if playerCenter < blockCenter { // player is to the left of block
		ps.P.X = s.Left() - ps.W
	} else { // player is to the right of block
		ps.P.X = s.Right()
	}
	ps.Vx = 0
}

func (ps *MovingSquare) CollideY(s Square) {
	playerCenter := ps.CenterY()
	blockCenter := s.CenterY()
	if playerCenter >= blockCenter { // player is above block
		ps.P.Y = s.Top()
		ps.Vy = 0
		ps.Onground = true
	} else { // player is below block
		ps.P.Y = s.Btm() - ps.H
		ps.Vy = 0
	}
}

func (sq *Square) Top() float64 {
	return sq.P.Y + sq.H
}

func (sq *Square) Btm() float64 {
	return sq.P.Y
}

func (sq *Square) Left() float64 {
	return sq.P.X
}

func (sq *Square) Right() float64 {
	return sq.P.X + sq.W
}

func (sq *Square) CenterY() float64 {
	return sq.P.Y + sq.H/2
}

func (sq *Square) CenterX() float64 {
	return sq.P.X + sq.W/2
}

func (sq *Square) collides(psq *Square) bool {
	if psq.P.X < sq.P.X+sq.W && // left side of player is at the left of the right side of obj
		psq.P.X+psq.W > sq.P.X && // right side of player is at the right of the left side of obj
		psq.P.Y+psq.H > sq.P.Y && // top side of player is above the bottom side of ob
		psq.P.Y < sq.P.Y+sq.H { // the bottom side of player is underneath the top side of obj
		return true
	} else {
		return false
	}
}

func (obj *Square) get_overlap(p *Square) *Square {
	yTop := math.Min(obj.P.Y+obj.H, p.P.Y+p.H)
	yBottom := math.Max(obj.P.Y, p.P.Y)

	xLeft := math.Max(obj.P.X, p.P.X)
	xRight := math.Min(obj.P.X+obj.W, p.P.X+p.W)

	w := xRight - xLeft
	h := yTop - yBottom

	return &Square{
		P: Position{X: xLeft, Y: yBottom},
		W: w, H: h,
	}
}

type Position struct {
	X, Y float64
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

func NewPlatform(x, y float64) GameObject {
	return GameObject{t: Platform, s: Square{P: Position{X: x, Y: y}, W: float64(100), H: float64(32)}}
}

func NewCoin(x, y float64) GameObject {
	return GameObject{t: Coin, s: Square{P: Position{X: x, Y: y}, W: float64(10), H: float64(10)}}
}

func NewFlag(x, y float64) GameObject {
	return GameObject{t: Flag, s: Square{P: Position{X: x, Y: y}, W: float64(4), H: float64(64 + 20)}}
}

func NewLevelFromObjects(objs []GameObject) *Level {
	return &Level{gameObjects: objs, cell_size: 128, grid: buildGrid(objs, 128)}
}

func NewLevelBig(iterations int) *Level {
	objs := BigLadder(iterations)
	return NewLevelFromObjects(objs)
}

func BigLadder(iterations int) []GameObject {
	objs := []GameObject{}

	startIdx := float64(100)
	// TODO: Currently start struggling with 4000 iterations (= 84 000 game objects)
	for i := 0; i < iterations; i++ {
		objs = append(objs, NewPlatform(startIdx, 20))
		objs = append(objs, NewCoin(startIdx+45, 20+32+13))

		objs = append(objs, NewPlatform(startIdx+100, 80))
		objs = append(objs, NewCoin(startIdx+100+45, 80+32+13))

		objs = append(objs, NewPlatform(startIdx+200, 140))
		objs = append(objs, NewCoin(startIdx+200+45, 140+32+13))

		objs = append(objs, NewPlatform(startIdx+300, 200))
		objs = append(objs, NewCoin(startIdx+300+45, 200+32+13))

		objs = append(objs, NewPlatform(startIdx+400, 140))
		objs = append(objs, NewCoin(startIdx+400+45, 140+32+13))

		objs = append(objs, NewPlatform(startIdx+660, 140))

		objs = append(objs, NewPlatform(startIdx+820, 190))
		objs = append(objs, NewFlag(startIdx+820+80, 190+32))

		objs = append(objs, NewCoin(startIdx+660+45, 400+32+13))
		objs = append(objs, NewPlatform(startIdx+660, 400))

		objs = append(objs, NewPlatform(startIdx+460, 270))
		objs = append(objs, NewFlag(startIdx+460+80, 270+32))

		objs = append(objs, NewPlatform(startIdx+500, 80))
		objs = append(objs, NewCoin(startIdx+500+45, 80+32+13))
		objs = append(objs, NewPlatform(startIdx+500, 20))
		objs = append(objs, NewCoin(startIdx+500+45, 20+32+13))

		startIdx += 1200
	}

	return objs
}

// handleCollision returns 1 if a coin was collected, 0 otherwise

func buildGrid(gameObjects []GameObject, cell_size float64) map[Position][]int {
	grid := make(map[Position][]int)
	// Insert items into a grid system of size based on cell_size
	// Move this to the level constructor probably (then we need to recalc if we have moving game objects)
	for i := range gameObjects {
		obj := gameObjects[i]
		gx1 := math.Floor(obj.s.P.X / cell_size)
		gy1 := math.Floor(obj.s.P.Y / cell_size)
		gx2 := math.Floor((obj.s.P.X + obj.s.W) / cell_size)
		gy2 := math.Floor((obj.s.P.Y + obj.s.H) / cell_size)

		for gy := gy1; gy <= gy2; gy++ {
			for gx := gx1; gx <= gx2; gx++ {
				pos := Position{X: gx, Y: gy}
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
	grid_x1 := math.Floor(pSquare.P.X / l.cell_size)
	grid_y1 := math.Floor(pSquare.P.Y / l.cell_size)
	grid_x2 := math.Floor((pSquare.P.X + pSquare.W) / l.cell_size)
	grid_y2 := math.Floor((pSquare.P.Y + pSquare.H) / l.cell_size)
	for gy := grid_y1; gy <= grid_y2; gy++ {
		for gx := grid_x1; gx <= grid_x2; gx++ {
			pos := Position{X: gx, Y: gy}
			for _, otherIdx := range l.grid[pos] {
				if !l.gameObjects[otherIdx].removed {
					elements[otherIdx] = struct{}{}
				}
			}
		}
	}
	return elements
}

// func (l *Level) ResolveCollisions(pSquare *MovingSquare, dt float64) int {
// 	elements := l.findClosestElements(pSquare)
//
// 	pSquare.nextPos(dt)
// 	pc := CollisionIterator{idx: -1, elements: &elements}
// 	// We loop through the objects until we dont collide with anything
// 	for pc.next(l, *pSquare.Square, false) {
// 		overlap, objSquare := l.overlap(*pSquare.Square, pc.idx)
//
// 		pc.coins += l.handleCollision(pSquare, dt, overlap, objSquare, pc.t, pc.idx) // TODO: This should be passed in as a func so each game can handle collisions differently
//
// 		// delete(*pc.elements, pc.idx)
// 	}
//
// 	// TODO: Below should be outside this function, handled in the PlayerUpdate func in TdcGame
//
// 	return pc.coins
// }

type CollisionIterator struct {
	elements        *map[int]struct{}
	l               *Level
	CollisionResult *CollisionResult
}

type CollisionResult struct {
	Idx           int
	T             GameObjectType
	Overlap       *Square
	GameObjSquare Square
}

func (l *Level) NewCollisionIterator(pSquare *MovingSquare, dt float64) *CollisionIterator {
	elements := l.findClosestElements(pSquare)

	pSquare.NextPos(dt)

	ci := &CollisionIterator{elements: &elements, CollisionResult: nil, l: l}

	return ci
}

func (c *CollisionIterator) Register_collision() {
	obj := &c.l.gameObjects[c.CollisionResult.Idx]
	if obj.t == Coin || obj.t == Flag {
		obj.removed = true
	}

	delete(*c.elements, c.CollisionResult.Idx)
}

// Returns the first colliding object
func (c *CollisionIterator) Next(pSquare *Square) bool {
	for i := range *c.elements { // TODO: Might make more sense to start from idx instead of starting from the top every time
		obj := &c.l.gameObjects[i]

		if obj.s.collides(pSquare) {
			overlap := c.l.overlap(pSquare, i)
			c.CollisionResult = &CollisionResult{Idx: i, T: obj.t, GameObjSquare: obj.s, Overlap: overlap}
			return true
		}
	}

	return false
}

func (c *CollisionIterator) CheckCollisionObj(pSquare Square, gameObjIdx int) bool {
	obj := &c.l.gameObjects[gameObjIdx]
	if obj.removed || obj.t != Platform {
		return false
	}

	if obj.s.collides(&pSquare) {
		return true
	}
	return false
}

func (l *Level) ResolveCollisions(pSquare *MovingSquare, dt float64) {
	iter := l.NewCollisionIterator(pSquare, dt)
	for iter.Next(pSquare.Square) {
		obj := iter.CollisionResult
		if obj.T == Platform {
			if obj.Overlap.W > obj.Overlap.H {
				pSquare.CollideY(obj.GameObjSquare)
			} else {
				pSquare.CollideX(obj.GameObjSquare)
			}
		}
		iter.Register_collision()
	}
}

func (l *Level) overlap(pSquare *Square, gambObjIdx int) *Square {
	s := l.gameObjects[gambObjIdx].s

	return s.get_overlap(pSquare)
}
