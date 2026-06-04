// Package tdcgame package containing whats needed to make a cool game
package tdcgame

type TdcGame interface {
	GetGameObjects() []GameObject
	// TODO: getGameParameters() GameParameters returns params for the game, gravity and all that jazz
}

type TdcGameWithPlayer interface {
	TdcGame

	GetPlayerUpdateFunc() PlayerUpdate
}

type PlayerUpdate func(buttonpressed bool, dt float64, level Level, player *MovingSquare) int
