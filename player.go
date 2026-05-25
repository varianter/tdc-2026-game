package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type Player struct {
	sheet         *SpriteSheet
	walkRightAnim *Animation
	walkLeftAnim  *Animation
	idleRightAnim *Animation
	idleLeftAnim  *Animation
	current       *Animation
	// TODO: Move these to a square thing
	vx, vy        float64
	x, y          float64
	h, w          float64
	direction     float64
	onground      bool
	movementScale float64

	coins              map[Position]struct{}
	autorun            bool
	collisionOffsetX   float64
	collisionOffsetTop float64
}

func Newplayer(sheet *SpriteSheet) *Player {
	player := &Player{
		sheet: sheet,
		walkRightAnim: &Animation{
			Sheet:      sheet,
			StartFrame: 0,
			FrameCount: 8,
			FPS:        AnimWalkFPS,
		},
		walkLeftAnim: &Animation{
			Sheet:      sheet,
			StartFrame: 8,
			FrameCount: 8,
			FPS:        AnimWalkFPS,
		},
		idleRightAnim: &Animation{
			Sheet:      sheet,
			StartFrame: 15,
			FrameCount: 1,
			FPS:        11,
		},
		idleLeftAnim: &Animation{
			Sheet:      sheet,
			StartFrame: 15,
			FrameCount: 1,
			FPS:        11,
		},
		x: 0, y: 0,
		h: 64, w: 64,
		movementScale: 1.0,
		direction:     1.0,
		coins:         make(map[Position]struct{}),
		autorun:       AutoRun,
		onground:      true,

		// collisionOffsetX: 0,
		collisionOffsetX:   15,
		collisionOffsetTop: 15,
	}
	player.current = player.idleRightAnim

	return player
}

func (p *Player) switchAnim(anim *Animation) {
	if p.current != anim {
		anim.elapsed = 0
		p.current = anim
	}
}

func (p *Player) Update(dt float64, level Level) error {
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		p.autorun = true
	}
	if ebiten.IsKeyPressed(ebiten.KeyM) {
		p.autorun = false
	}
	if p.autorun {
		p.vx = WalkSpeed * p.movementScale
	} else {
		p.vx = 0
		if ebiten.IsKeyPressed(ebiten.KeyRight) {
			p.vx = WalkSpeed * p.movementScale
		}
		if ebiten.IsKeyPressed(ebiten.KeyLeft) {
			p.vx = -WalkSpeed * p.movementScale
		}
	}

	if p.onground && (ebiten.IsKeyPressed(ebiten.KeySpace) || ebiten.IsKeyPressed(ebiten.KeyUp)) {
		p.vy = JumpForce
	}

	pSquare := p.ToCollisionSquare()
	pSquare.nextPos(dt)

	// Collide with platforms first
	pc := Col{idx: -1, solved: make(map[int]struct{})}
	for pc.next(level, *pSquare.Square, false) {
		overlap, objSquare := level.overlap(*pSquare.Square, pc.idx)

		if pc.t == Flag {
			pSquare.direction = pSquare.direction * -1.0

			playerCenter := pSquare.p.x + pSquare.w/2
			flagCenter := objSquare.p.x + objSquare.w/2
			if playerCenter < flagCenter { // player is to the left of block
				pSquare.p.x = objSquare.left() - pSquare.w - 1
			} else { // player is to the right of block
				pSquare.p.x = objSquare.right() + 1
			}
			level.register_collision(pc.idx)
			continue
		}
		if pc.t == Platform {
			if overlap.w > overlap.h { // y is shallowest so solve y collision first
				pSquare.collide_y(objSquare)
				pSquare.nextPos(dt) // dont do this,get new pos

				if level.collideObj(*pSquare.Square, pc.idx) {
					pSquare.collide_x(objSquare)
					pSquare.nextPos(dt)
				}
			} else {
				pSquare.collide_x(objSquare)
				pSquare.nextPos(dt)

				if level.collideObj(*pSquare.Square, pc.idx) {
					pSquare.collide_y(objSquare)

					pSquare.nextPos(dt)
				}
			}
			continue
		}

		if pc.t == Coin {
			p.coins[objSquare.p] = struct{}{}

			level.register_collision(pc.idx)
		}
		pc.solved[pc.idx] = struct{}{}
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

	// Collide with
	cc := Col{idx: -1, solved: make(map[int]struct{})}
	for cc.next(level, *pSquare.Square, false) {
	}

	nextX := pSquare.p.x - p.collisionOffsetX
	// Update animation
	if nextX > p.x {
		p.switchAnim(p.walkRightAnim)
	}
	if nextX < p.x {
		p.switchAnim(p.walkLeftAnim)
	}

	p.x = nextX
	p.y = pSquare.p.y
	p.vy = pSquare.vy
	p.vx = pSquare.vx
	p.direction = pSquare.direction
	p.onground = pSquare.onground

	p.current.Update(dt)
	return nil
}

func (ps *MovingSquare) collide_x(s Square) {
	playerCenter := ps.p.x + ps.w/2
	blockCenter := s.p.x + s.w/2
	if playerCenter < blockCenter { // player is to the left of block
		ps.p.x = s.left() - ps.w
	} else { // player is to the right of block
		ps.p.x = s.right()
	}
	ps.vx = 0
}

func (ps *MovingSquare) collide_y(s Square) {
	playerCenter := ps.p.y + ps.h/2
	blockCenter := s.p.y + s.h/2
	if playerCenter >= blockCenter { // player is above block
		ps.p.y = s.top()
		ps.vy = 0
		ps.onground = true
	} else { // player is below block
		ps.p.y = s.btm() - ps.h
		ps.vy = 0
	}
}

func (p *Player) ToCollisionSquare() MovingSquare {
	pSquare := &MovingSquare{
		&Square{p: Position{x: p.x + p.collisionOffsetX, y: p.y}, w: p.w - (p.collisionOffsetX * 2), h: p.h - p.collisionOffsetTop},
		&Moving{vy: p.vy, vx: p.vx, direction: p.direction, onground: false},
	}
	return *pSquare
}

func (p *Player) Draw(canvas *Canvas) {
	frame := p.current.CurrentFrame()
	canvas.DrawImage(frame, p.x, p.y)
}
