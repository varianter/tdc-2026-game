package main

import "github.com/hajimehoshi/ebiten/v2"

type Player struct {
	sheet         *SpriteSheet
	walkRightAnim *Animation
	walkLeftAnim  *Animation
	idleRightAnim *Animation
	idleLeftAnim  *Animation
	current       *Animation
	orientation   rune
	x, y          float64
	vx, vy        float64
	onGround      bool
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
		x: ScreenW / 2, y: GroundY,
		orientation: 'r',
	}
	player.current = player.idleRightAnim

	return player
}

func (c *Player) switchAnim(anim *Animation) {
	if c.current != anim {
		anim.elapsed = 0
		c.current = anim
	}
}

func (c *Player) Update(dt float64) error {
	movementScale := 1.0
	if !c.onGround {
		movementScale = AirControl
	}

	// Calculate velocity
	c.vx = 0
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		c.vx = WalkSpeed * movementScale
		c.switchAnim(c.walkRightAnim) // TODO: Flying animation
		c.orientation = 'r'
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		c.vx = -WalkSpeed * movementScale
		c.switchAnim(c.walkLeftAnim) // TODO: Flying animation
		c.orientation = 'l'
	}
	if c.onGround && (ebiten.IsKeyPressed(ebiten.KeySpace) || ebiten.IsKeyPressed(ebiten.KeyUp)) {
		c.vy = JumpForce
		c.onGround = false
	}
	if c.vx == 0 {
		if c.orientation == 'l' {
			c.switchAnim(c.idleLeftAnim)
		} else {
			c.switchAnim(c.idleRightAnim)
		}
	}

	// Gravity
	if !c.onGround {
		c.vy += Gravity * dt
	}

	// Update position based on velocity
	c.x += c.vx * dt
	c.y += c.vy * dt

	// Collision detection
	// Collide with ground
	if c.y+float64(c.sheet.FrameH) >= GroundY {
		c.y = GroundY - float64(c.sheet.FrameH)
		c.vy = 0
		c.onGround = true
	}

	c.current.Update(dt)
	return nil
}

func (c *Player) Draw(canvas *Canvas) {
	frame := c.current.CurrentFrame()
	canvas.DrawImage(frame, c.x, c.y)
}
