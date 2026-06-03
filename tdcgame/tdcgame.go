// Package tdcgame package containing whats needed to make a cool game
package tdcgame

type TdcGame interface {
	GetGameObjects() []GameObject
	GetPlayerUpdateFunc() PlayerUpdate

	// TODO: getGameParameters() GameParameters
}

type PlayerUpdate func(buttonpressed bool, dt float64, level Level, player *MovingSquare) int
