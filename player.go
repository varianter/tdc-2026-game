package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type Player struct {
	sheet         *SpriteSheet
	walkRightAnim *Animation
	walkLeftAnim  *Animation
	idleRightAnim *Animation
	idleLeftAnim  *Animation
	current       *Animation
	x, y          float64
	vx, vy        float64
	movementScale float64
	direction     float64
	coins         map[Position]struct{}
}

func Newplayer(sheet *SpriteSheet) *Player {
	player := &Player{
		sheet: sheet,
		walkRightAnim: &Animation{
			Sheet:      sheet,
			StartFrame: 0,
			FrameCount: 6,
			FPS:        AnimWalkFPS,
		},
		idleRightAnim: &Animation{
			Sheet:      sheet,
			StartFrame: 6,
			FrameCount: 6,
			FPS:        10,
		},
		walkLeftAnim: &Animation{
			Sheet:      sheet,
			StartFrame: 12,
			FrameCount: 6,
			FPS:        AnimWalkFPS,
		},
		idleLeftAnim: &Animation{
			Sheet:      sheet,
			StartFrame: 18,
			FrameCount: 6,
			FPS:        10,
		},
		x: 0, y: 0,
		movementScale: 1.0,
		direction:     1.0,
		coins:         make(map[Position]struct{}),
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

// Calculate next position based on velocity
func (p *Player) nextPos(dt float64) (x float64, y float64) {
	x = math.Round(p.x + (p.vx * dt * p.direction))
	y = math.Round(p.y + (p.vy * dt))
	return x, y
}

func (p *Player) Update(dt float64, level Level) error {
	// Calculate velocity

	if AutoRun {
		p.vx = WalkSpeed * p.movementScale

		if p.x+float64(p.sheet.FrameW) > GameEnd {
			p.direction = -1.0
			p.x = GameEnd - float64(p.sheet.FrameW)
		}

		if p.x < 0 {
			p.direction = 1.0
			p.x = 0
		}
	} else {
		p.vx = 0
		if ebiten.IsKeyPressed(ebiten.KeyRight) {
			p.vx = WalkSpeed * p.movementScale
		}
		if ebiten.IsKeyPressed(ebiten.KeyLeft) {
			p.vx = -WalkSpeed * p.movementScale
		}
	}

	if p.vy == 0 && (ebiten.IsKeyPressed(ebiten.KeySpace) || ebiten.IsKeyPressed(ebiten.KeyUp)) {
		p.vy = JumpForce
	}

	p.vy -= Gravity * dt // Always apply gravity to avoid shenanigans

	nextX, nextY := p.nextPos(dt)
	collision := level.collide(nextX, nextY)
	for _, coin := range collision.coin {
		p.coins[coin] = struct{}{}
	}
	if collision.collideX.collision {
		p.vx = 0
		p.x = collision.collideX.cord
	}

	if collision.collideY.collision {
		p.vy = 0
		p.y = collision.collideY.cord
	}

	// Update position with collision corrected velocity
	x, y := p.nextPos(dt)
	if x > p.x {
		p.switchAnim(p.walkRightAnim)
	}
	if x < p.x {
		p.switchAnim(p.walkLeftAnim)
	}
	p.x = x
	p.y = y

	p.current.Update(dt)
	return nil
}

func (p *Player) Draw(canvas *Canvas) {
	frame := p.current.CurrentFrame()
	canvas.DrawImage(frame, p.x, p.y)
}
