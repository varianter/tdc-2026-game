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
	vx, vy        float64
	x, y          float64
	h, w          float64
	direction     float64
	onground      bool
	movementScale float64

	coins              int
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
		coins:         0,
		autorun:       AutoRun,
		onground:      true,

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

	coins := level.resolveCollisions(&pSquare, dt)
	p.coins += coins

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
