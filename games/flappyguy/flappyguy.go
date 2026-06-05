// Package flappyguy implements a flappy-bird-style game.
package flappyguy

import "variant.dev/tdcgame/tdcgame"

const (
	pipeWidth   = 48
	gapHeight   = 100
	pipeSpacing = 160
	firstPipeX  = 180
	numPipes    = 100
	gravity     = 500
	flapForce   = 160
	maxY        = 170
)

type FlappyGuy struct {
	gameState    tdcgame.GameState
	runtime      float64
	currentScore int
	wasPressed   bool
	audio        *tdcgame.Audio
}

func New() *FlappyGuy {
	return &FlappyGuy{
		audio: loadAudio(),
	}
}

func (g *FlappyGuy) setGameOver() {
	if g.gameState == tdcgame.GameOver {
		return
	}
	g.gameState = tdcgame.GameOver
	g.audio.Play(soundScream)
}

func (g *FlappyGuy) GetCurrentScore() int {
	return g.currentScore
}

func (g *FlappyGuy) GetGameState() tdcgame.GameState {
	return g.gameState
}

func (g *FlappyGuy) GetGameObjects() []tdcgame.GameObject {
	return buildLevel()
}

func (g *FlappyGuy) GetGameParameters() *tdcgame.GameParameters {
	return &tdcgame.GameParameters{
		WalkSpeed:                90.0,
		JumpSpeed:                60,
		Gravity:                  gravity,
		JumpForce:                flapForce,
		AirControl:               1,
		AnimIdleFPS:              5.0,
		AnimWalkFPS:              10.0,
		AnimRunFPS:               14.0,
		ShouldCameraFollowPlayer: true,
		IsFlying:                 true,
		StartY:                   70,
	}
}

func (g *FlappyGuy) GetPlayerUpdateFunc() tdcgame.PlayerUpdate {
	return func(buttonpressed bool, dt float64, level tdcgame.Level, p *tdcgame.MovingSquare) {
		if g.runtime == 0 {
			g.wasPressed = buttonpressed
		}
		g.runtime += dt
		if g.runtime >= 120 {
			g.setGameOver()
			return
		}

		flap := buttonpressed && !g.wasPressed
		g.wasPressed = buttonpressed
		if flap {
			p.Vy = flapForce
			g.audio.Play(soundWingFlap)
		}

		p.Onground = false

		iter := level.NewCollisionIterator(p, dt)

		coins := 0
		for iter.Next(p.Square) {
			c := iter.CollisionResult
			switch c.T {
			case tdcgame.Coin:
				coins++
				iter.RegisterCollision()
				g.audio.Play(soundCoinCollect)
			case tdcgame.Platform:
				g.setGameOver()
				return
			}
		}

		if p.P.Y <= 0 {
			g.setGameOver()
			return
		}
		if p.P.Y+p.H > maxY {
			g.setGameOver()
			return
		}

		p.Vy -= gravity * dt

		g.currentScore += coins
	}
}

func buildLevel() []tdcgame.GameObject {
	gapBottoms := []float64{30, 55, 40, 70, 45, 65, 35, 60, 50, 75}

	objs := make([]tdcgame.GameObject, 0, numPipes*3)
	for i := 0; i < numPipes; i++ {
		x := firstPipeX + float64(i)*pipeSpacing
		gapBottom := gapBottoms[i%len(gapBottoms)]
		gapTop := gapBottom + gapHeight

		objs = append(objs, tdcgame.NewBox(x, 0, pipeWidth, gapBottom))
		objs = append(objs, tdcgame.NewBox(x, gapTop, pipeWidth, 160))
		objs = append(objs, tdcgame.NewCoin(x+pipeWidth/2-5, gapBottom+gapHeight/2-5))
	}

	return objs
}
