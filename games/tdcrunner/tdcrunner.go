// Package tdcrunner
package tdcrunner

import "variant.dev/tdcgame/tdcgame"

type TdcRunner struct{}

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

func (r *TdcRunner) GetPlayerUpdateFunc() tdcgame.PlayerUpdate {
	return func(buttonpressed bool, dt float64, level tdcgame.Level, p *tdcgame.MovingSquare) int {
		if p.Onground && buttonpressed {
			p.Vy = 300
		}

		p.Onground = false

		coins := level.ResolveCollisions(p, dt)

		return coins
	}
}
