// Package tdcgame package containing whats needed to make a cool game
package tdcgame

import "github.com/hajimehoshi/ebiten/v2"

type TdcGame interface {
	GetGameObjects() []GameObject

	GetGameParameters() *GameParameters

	GetGameState() GameState

	GetCurrentScore() int
}

type TdcGameWithPlayer interface {
	TdcGame

	GetPlayerUpdateFunc() PlayerUpdate
}

type PlayerUpdate func(buttonpressed bool, dt float64, level Level, player *MovingSquare)

// GameWithCustomDraw is optionally implemented by games that want full control
// over rendering instead of using the default GameRunner draw.
// The framework still draws the game over overlay on top.
type GameWithCustomDraw interface {
	CustomDraw(screen *ebiten.Image)
}

type GameState int

const (
	Running  GameState = iota
	GameOver GameState = iota
)

type GameParameters struct {
	WalkSpeed                float64
	JumpSpeed                float64
	Gravity                  float64
	JumpForce                float64
	AirControl               float64
	AnimIdleFPS              float64
	AnimWalkFPS              float64
	AnimRunFPS               float64
	ShouldCameraFollowPlayer bool
}

/*
	Ideas for other parameters:
	- Camerasettings (where the caemra puts the player)

*/

// WalkSpeed   = 90.0 // px/sec
// JumpSpeed   = 60
// Gravity     = 700 // px/sec^2
// JumpForce   = 300
// AirControl  = 1
// AutoRun     = true
// AnimIdleFPS = 5.0
// AnimWalkFPS = 10.0
// AnimRunFPS  = 14.0
