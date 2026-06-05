package tdcgame

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type Player struct {
	sheet            *SpriteSheet
	walkRightAnim    *Animation
	walkLeftAnim     *Animation
	idleRightAnim    *Animation
	idleLeftAnim     *Animation
	currentAnimation *Animation
	vx, vy           float64
	x, y             float64
	h, w             float64
	direction        float64
	onground         bool
	movementScale    float64
	walkSpeed        float64

	autorun            bool
	collisionOffsetX   float64
	collisionOffsetTop float64
}

func Newplayer(sheet *SpriteSheet, params *GameParameters) *Player {
	player := &Player{
		sheet: sheet,
		walkRightAnim: &Animation{
			Sheet:      sheet,
			StartFrame: 0,
			FrameCount: 8,
			FPS:        params.AnimWalkFPS,
		},
		walkLeftAnim: &Animation{
			Sheet:      sheet,
			StartFrame: 8,
			FrameCount: 8,
			FPS:        params.AnimWalkFPS,
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
		autorun:       true,
		onground:      true,
		walkSpeed:     params.WalkSpeed,

		// The sprite is 64x64 but the actual drawn pixels are smaller
		// These values are to make it look more correct
		collisionOffsetX:   15,
		collisionOffsetTop: 15,
	}
	player.currentAnimation = player.idleRightAnim

	return player
}

func (p *Player) switchAnim(anim *Animation) {
	if p.currentAnimation != anim {
		anim.elapsed = 0
		p.currentAnimation = anim
	}
}

func (p *Player) Update(dt float64, level Level, tdcgamePlayerUpdate PlayerUpdate) (int, error) {
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		p.autorun = true
	}
	if ebiten.IsKeyPressed(ebiten.KeyM) {
		p.autorun = false
	}
	if p.autorun {
		p.vx = p.walkSpeed * p.movementScale
	} else {
		p.vx = 0
		if ebiten.IsKeyPressed(ebiten.KeyRight) {
			p.vx = p.walkSpeed * p.movementScale
		}
		if ebiten.IsKeyPressed(ebiten.KeyLeft) {
			p.vx = -p.walkSpeed * p.movementScale
		}
	}

	pSquare := p.ToCollisionSquare()
	coins := tdcgamePlayerUpdate(ebiten.IsKeyPressed(ebiten.KeySpace), dt, level, &pSquare)

	nextX := pSquare.P.X - p.collisionOffsetX

	// Update animation
	if nextX > p.x {
		p.switchAnim(p.walkRightAnim)
	}
	if nextX < p.x {
		p.switchAnim(p.walkLeftAnim)
	}

	p.x = nextX
	p.y = pSquare.P.Y
	p.vy = pSquare.Vy
	p.vx = pSquare.Vx
	p.direction = pSquare.Direction
	p.onground = pSquare.Onground

	p.currentAnimation.Update(dt)
	return coins, nil
}

func (p *Player) ToCollisionSquare() MovingSquare {
	pSquare := &MovingSquare{
		&Square{P: Position{X: p.x + p.collisionOffsetX, Y: p.y}, W: p.w - (p.collisionOffsetX * 2), H: p.h - p.collisionOffsetTop},
		&Moving{Vy: p.vy, Vx: p.vx, Direction: p.direction, Onground: p.onground},
	}
	return *pSquare
}

func (p *Player) Draw(canvas *Canvas) {
	frame := p.currentAnimation.CurrentFrame()
	canvas.DrawImage(frame, p.x, p.y)
}
