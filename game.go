package main

import (
	"embed"
	"fmt"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"variant.dev/tdcgame/games/bounce"
	"variant.dev/tdcgame/games/flappyguy"
	"variant.dev/tdcgame/games/tdcrunner"
	"variant.dev/tdcgame/tdcgame"
)

type Canvas struct {
	screen *ebiten.Image
}

func (c *Canvas) DrawImage(img *ebiten.Image, x, y float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	c.screen.DrawImage(img, op)
}

func (c *Canvas) Rect(x, y, w, h float32, clr color.Color) {
	vector.FillRect(c.screen, x, y, w, h, clr, false)
}

//go:embed assets/tdcgjenger.png
//go:embed assets/ground.png
//go:embed assets/audio/coincollect.ogg
//go:embed assets/audio/wingflap.ogg
//go:embed assets/audio/scream.ogg
var assets embed.FS

const (
	ScreenW = 426
	ScreenH = 240
)

type Game struct {
	currentScene Scene
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenW, ScreenH
}

func (g *Game) Update() error {
	dt := 1.0 / float64(ebiten.TPS())
	next, err := g.currentScene.Update(dt)
	if err != nil {
		return err
	}
	if next != nil {
		g.currentScene = next
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.currentScene.Draw(screen)
}

// GameRunnerScene wraps a tdcgame.GameRunner as a Scene.
// Pressing Q returns to the launcher.
type GameRunnerScene struct {
	runner          *tdcgame.GameRunner
	gameName        string
	spaceTimer      float64 // time since first space press; -1 = not active
	doubleTapWindow float64
}

const doubleTapWindow = 0.35

func (s *GameRunnerScene) Update(dt float64) (Scene, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		return NewLauncherScene(), nil
	}

	if s.runner.State() == tdcgame.GameOver {
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			if s.spaceTimer >= 0 {
				// Second press within window: restart
				return &GameRunnerScene{runner: createGameFramework(s.gameName), gameName: s.gameName, spaceTimer: -1}, nil
			}
			// First press: start timer
			s.spaceTimer = 0
		}

		if s.spaceTimer >= 0 {
			s.spaceTimer += dt
			if s.spaceTimer > doubleTapWindow {
				return NewLauncherScene(), nil
			}
		}
	}

	return nil, s.runner.Update()
}

func (s *GameRunnerScene) Draw(screen *ebiten.Image) {
	s.runner.Draw(screen)
}

var scoreKeeper *tdcgame.ScoreKeeper

func createGameFramework(gameName string) *tdcgame.GameRunner {
	switch gameName {
	case "tdcrunner":
		return tdcgame.NewGameFrameworkWithPlayer(assets, &tdcrunner.TdcRunner{}, scoreKeeper, gameName)
	case "bounce":
		return tdcgame.NewGameFrameworkWithPlayer(assets, bounce.NewBounce(assets), scoreKeeper, gameName)
	case "flappy-guy":
		return tdcgame.NewGameFrameworkWithPlayer(assets, &flappyguy.FlappyGuy{}, scoreKeeper, gameName)
	default:
		panic(fmt.Sprintf("Unknown game: %s", gameName))
	}
}

func init() {
	var err error
	scoreKeeper, err = tdcgame.NewScoreKeeper("scores.db")
	if err != nil {
		log.Fatal(err)
	}

	wheelGames = append(wheelGames, gameEntry{
		name:  "TDCRUNNER",
		color: color.RGBA{220, 60, 60, 255},
		key:   ebiten.KeyR,
		newScene: func() Scene {
			return &GameRunnerScene{runner: createGameFramework("tdcrunner"), gameName: "tdcrunner", spaceTimer: -1}
		},
	})
	wheelGames = append(wheelGames, gameEntry{
		name:  "BOUNCE",
		color: color.RGBA{180, 50, 220, 255},
		key:   ebiten.KeyB,
		newScene: func() Scene {
			return &GameRunnerScene{runner: createGameFramework("bounce")}
		},
	})
	wheelGames = append(wheelGames, gameEntry{
		name:  "FLAPPY-GUY",
		color: color.RGBA{60, 180, 220, 255},
		key:   ebiten.KeyF,
		newScene: func() Scene {
			return &GameRunnerScene{runner: createGameFramework("flappy-guy"), gameName: "flappy-guy", spaceTimer: -1}
		},
	})
}
