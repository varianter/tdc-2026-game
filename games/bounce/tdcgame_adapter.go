package bounce

import (
	"embed"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"variant.dev/tdcgame/tdcgame"
)

// Bounce implements tdcgame.TdcGameWithPlayer and tdcgame.GameWithCustomDraw,
// wrapping the internal Game so it can be used with the framework's GameRunner.
type Bounce struct {
	game *Game
}

func NewBounce(assets embed.FS) *Bounce {
	return &Bounce{game: New(assets)}
}

func (b *Bounce) GetGameObjects() []tdcgame.GameObject {
	return nil
}

func (b *Bounce) GetGameParameters() *tdcgame.GameParameters {
	return &tdcgame.GameParameters{
		WalkSpeed:                baseScrollSpeed,
		AnimWalkFPS:              12,
		AnimIdleFPS:              12,
		ShouldCameraFollowPlayer: false,
	}
}

func (b *Bounce) GetGameState() tdcgame.GameState {
	if b.game.GameOver {
		return tdcgame.GameOver
	}
	return tdcgame.Running
}

func (b *Bounce) GetCurrentScore() int {
	return b.game.score
}

func (b *Bounce) GetPlayerUpdateFunc() tdcgame.PlayerUpdate {
	return func(buttonpressed bool, dt float64, level tdcgame.Level, player *tdcgame.MovingSquare) {
		spaceJustPressed := inpututil.IsKeyJustPressed(ebiten.KeySpace)
		b.game.Update(dt, buttonpressed, spaceJustPressed)
	}
}

func (b *Bounce) CustomDraw(screen *ebiten.Image) {
	b.game.Draw(screen)
}
