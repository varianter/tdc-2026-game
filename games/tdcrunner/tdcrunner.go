// Package tdcrunner
package tdcrunner

import "variant.dev/tdcgame/tdcgame"

type TdcRunner struct {
	gameState tdcgame.GameState
	runtime   float64
	currentScore int
}

func (r *TdcRunner) GetCurrentScore() int {
	return r.currentScore
}

func (r *TdcRunner) GetGameState() tdcgame.GameState {
	return r.gameState // TODO: gamestate can be running, dead, or game over
}

func (r *TdcRunner) GetGameObjects() []tdcgame.GameObject {
	objs := []tdcgame.GameObject{
		tdcgame.NewPlatform(100, 20),
		tdcgame.NewCoin(100+45, 20+32+13),
		tdcgame.NewPlatform(200, 80),
		tdcgame.NewCoin(200+45, 80+32+13),
		tdcgame.NewPlatform(300, 140),
		tdcgame.NewCoin(300+45, 140+32+13),
		tdcgame.NewPlatform(400, 200),
		tdcgame.NewCoin(400+45, 200+32+13),
		tdcgame.NewPlatform(500, 140),
		tdcgame.NewCoin(500+45, 140+32+13),

		tdcgame.NewPlatform(760, 140),

		tdcgame.NewPlatform(920, 190),
		tdcgame.NewFlag(920+80, 190+32),

		tdcgame.NewCoin(760+45, 400+32+13),
		tdcgame.NewPlatform(760, 400),

		tdcgame.NewPlatform(560, 270),
		tdcgame.NewFlag(560+80, 270+32),

		tdcgame.NewPlatform(600, 80),
		tdcgame.NewCoin(600+45, 80+32+13),
		tdcgame.NewPlatform(700, 20),
		tdcgame.NewCoin(700+45, 20+32+13),
	}

	return objs
}

func (r *TdcRunner) GetGameParameters() *tdcgame.GameParameters {
	return &tdcgame.GameParameters{
		WalkSpeed:   90.0,
		JumpSpeed:   60,
		Gravity:     700,
		JumpForce:   300,
		AirControl:  1,
		AnimIdleFPS: 5.0,
		AnimWalkFPS: 10.0,
		AnimRunFPS:  14.0,
	}
}

func (r *TdcRunner) GetPlayerUpdateFunc() tdcgame.PlayerUpdate {
	return func(buttonpressed bool, dt float64, level tdcgame.Level, p *tdcgame.MovingSquare) {
		r.runtime += dt
		if r.runtime >= 120 {
			r.gameState = tdcgame.GameOver
		}
		
		if p.Onground && buttonpressed {
			p.Vy = 300
		}

		p.Onground = false

		iter := level.NewCollisionIterator(p, dt)

		coins := 0
		for iter.Next(p.Square) {
			coins += handleCollision(p, dt, iter)
			iter.RegisterCollision()
		}

		// Collide with ground
		if p.P.Y <= 0 {
			p.P.Y = 0
			p.Vy = 0
			p.Onground = true
		}

		// TODO: Don't hard code gravity
		if !p.Onground {
			p.Vy -= 700 * dt // Always apply gravity to avoid shenanigans
		}

		// Turn around at start flag
		if p.P.X < 0 {
			p.P.X = 0
			p.Direction = p.Direction * -1.0
		}
		
		r.currentScore += coins
	}
}

func handleCollision(pSquare *tdcgame.MovingSquare, dt float64, i *tdcgame.CollisionIterator) int {
	c := i.CollisionResult

	if c.T == tdcgame.Flag {
		pSquare.Direction = pSquare.Direction * -1.0

		playerCenter := pSquare.CenterX()
		flagCenter := c.GameObjSquare.CenterX()
		if playerCenter < flagCenter { // player is to the left of block
			pSquare.P.X = c.GameObjSquare.Left() - pSquare.W - 1
		} else { // player is to the right of block
			pSquare.P.X = c.GameObjSquare.Right() + 1
		}
	}
	if c.T == tdcgame.Coin {
		return 1
	}
	if c.T == tdcgame.Platform {
		if c.Overlap.W > c.Overlap.H { // y is shallowest so solve y collision first
			pSquare.CollideY(c.GameObjSquare)
			pSquare.NextPos(dt) // dont do this,get new pos

			if i.CheckCollisionObj(*pSquare.Square, c.Idx) {
				pSquare.CollideX(c.GameObjSquare)
				pSquare.NextPos(dt)
			}
		} else {
			pSquare.CollideX(c.GameObjSquare)
			pSquare.NextPos(dt)

			if i.CheckCollisionObj(*pSquare.Square, c.Idx) {
				pSquare.CollideY(c.GameObjSquare)

				pSquare.NextPos(dt)
			}
		}
	}
	return 0
}
