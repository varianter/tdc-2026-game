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
	flyAnim         *Animation
	currentAnimation *Animation
	vx, vy           float64
	x, y             float64
	h, w             float64
	direction        float64
	onground         bool
	isFlying         bool
	movementScale    float64
	walkSpeed        float64

	autorun            bool
	collisionOffsetX   float64
	collisionOffsetTop float64
}

func Newplayer(sheet *SpriteSheet, params *GameParameters) *Player {
	player := &Player{
		sheet: sheet,
		walkRightAnim: NewAnimation(sheet, 0, 8, params.AnimWalkFPS),
		walkLeftAnim: NewAnimation(sheet, 8, 8, params.AnimWalkFPS),
		idleRightAnim: NewAnimation(sheet, 15, 1, 11),
		idleLeftAnim: NewAnimation(sheet, 15, 1, 11),
		flyAnim: &Animation{
			Sheet: sheet,
			Frames: []int{3,6,7},
			FPS: 20,
		},
		x: 0, y: params.StartY,
		h: 64, w: 64,
		movementScale: 1.0,
		direction:     1.0,
		autorun:       true,
		onground:      !params.IsFlying,
		isFlying:      params.IsFlying,
		walkSpeed:     params.WalkSpeed,

		// The sprite is 64x64 but the actual drawn pixels are smaller
		// These values are to make it look more correct
		collisionOffsetX:   15,
		collisionOffsetTop: 15,
	}
	if params.IsFlying {
		player.currentAnimation = player.flyAnim
	} else {
		player.currentAnimation = player.walkRightAnim
	}

	return player
}

func (p *Player) UpdateStartScreen(dt float64) {
	if p.isFlying {
		p.switchAnim(p.flyAnim)
	} else {
		p.switchAnim(p.walkRightAnim)
	}
	p.currentAnimation.Update(dt)
}

func (p *Player) switchAnim(anim *Animation) {
	if p.currentAnimation != anim {
		anim.elapsed = 0
		p.currentAnimation = anim
	}
}

func (p *Player) Update(dt float64, level Level, tdcgamePlayerUpdate PlayerUpdate) error {
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
	tdcgamePlayerUpdate(ebiten.IsKeyPressed(ebiten.KeySpace), dt, level, &pSquare)

	nextX := pSquare.P.X - p.collisionOffsetX

	// Update animation
	if(p.isFlying) {
		p.switchAnim(p.flyAnim)
	} else {
		if nextX > p.x {
			p.switchAnim(p.walkRightAnim)
		}
		if nextX < p.x {
			p.switchAnim(p.walkLeftAnim)
		}
	}

	p.x = nextX
	p.y = pSquare.P.Y
	p.vy = pSquare.Vy
	p.vx = pSquare.Vx
	p.direction = pSquare.Direction
	p.onground = pSquare.Onground

	p.currentAnimation.Update(dt)
	return nil
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
