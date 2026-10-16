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

// TdcGameWithBackground lets a game draw into the background layer — after the
// sky rect but before the ground tiles and player. Use for scenery like trees.
type TdcGameWithBackground interface {
	TdcGame
	DrawBackground(screen *ebiten.Image, cameraX, cameraY float64)
}

// TdcGameWithDraw is an optional interface games can implement to draw custom
// content on top of the base rendering. cameraX and cameraY are the current
// camera offsets needed to convert world coordinates to screen coordinates.
type TdcGameWithDraw interface {
	TdcGame
	Draw(screen *ebiten.Image, cameraX, cameraY float64)
}

// TdcGameWithOverlayDraw lets a game draw HUD/text elements at full device-pixel
// resolution. Called after the viewport is blitted to screen. scale is the ratio
// of device pixels to game pixels (e.g. 4.0 on a Retina display at 426-wide).
// Multiply all coordinates and font sizes by scale.
type TdcGameWithOverlayDraw interface {
	TdcGame
	DrawOverlay(screen *ebiten.Image, scale, cameraX, cameraY float64)
}

// TdcGameEndX lets a game override the world X position of the end flag.
// If not implemented, GameEnd is used.
type TdcGameEndX interface {
	TdcGame
	EndX() float64
}

// TdcGameWithFinalScore is optionally implemented by games that keep reporting
// Running for a while after their score is final (e.g. a grace period that
// swallows button mashing on their score screen). Once ScoreFinal returns
// true the score is saved right away, so a new #1 is recognised without
// waiting for GameOver.
type TdcGameWithFinalScore interface {
	TdcGame
	ScoreFinal() bool
}

// TdcGameWithCleanup is optionally implemented by games that hold resources
// outliving a single frame, such as a looping ambience player. Cleanup is
// called once when the game's scene is left, including quitting mid-run.
type TdcGameWithCleanup interface {
	TdcGame
	Cleanup()
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
	IsFlying                 bool
	StartY                   float64
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
