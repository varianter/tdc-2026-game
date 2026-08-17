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
	"variant.dev/tdcgame/games/petthedamncat"
	"variant.dev/tdcgame/games/tdcrunner"
	"variant.dev/tdcgame/games/vclicker"
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
//go:embed assets/cats/cat1.png
//go:embed assets/cats/cat2.png
//go:embed assets/cats/cat3.png
//go:embed assets/cats/radioactive.wav
//go:embed assets/cats/Meow.ogg
//go:embed assets/cats/alert.mp3
//go:embed assets/cats/scared.mp3
//go:embed assets/cats/scaredBig.mp3
var assets embed.FS

const (
	ScreenW = 426
	ScreenH = 240
)

type Game struct {
	currentScene Scene
}

// Layout is required by ebiten.Game; LayoutF is called instead when present.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

// LayoutF returns the full device-pixel window size so the game renders at
// native resolution. Pixel-art content is upscaled inside Draw via a viewport.
func (g *Game) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	s := ebiten.Monitor().DeviceScaleFactor()
	return outsideWidth * s, outsideHeight * s
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
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
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
	case "vclicker":
		return tdcgame.NewGameFrameworkWithPlayer(assets, vclicker.NewVClickerScene(), scoreKeeper, gameName)
	case "flappy-guy":
		return tdcgame.NewGameFrameworkWithPlayer(assets, flappyguy.New(), scoreKeeper, gameName)
	case "petthedamncat":
		catGame := &petthedamncat.PetTheDamnCat{}
		catGame.Init(assets)
		return tdcgame.NewGameFrameworkWithPlayer(assets, catGame, scoreKeeper, gameName)
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

	launcherGames = append(launcherGames, gameEntry{
		name:     "TDCRUNNER",
		gameName: "tdcrunner",
		color:    color.RGBA{142, 32, 32, 255}, // #8E2020
		key:      ebiten.KeyR,
		newScene: func() Scene {
			return &GameRunnerScene{runner: createGameFramework("tdcrunner"), gameName: "tdcrunner", spaceTimer: -1}
		},
	})
	launcherGames = append(launcherGames, gameEntry{
		name:     "BOUNCE",
		gameName: "bounce",
		color:    color.RGBA{108, 42, 130, 255}, // #6C2A82
		key:      ebiten.KeyB,
		newScene: func() Scene {
			return &GameRunnerScene{runner: createGameFramework("bounce"), gameName: "bounce", spaceTimer: -1}
		},
	})
	launcherGames = append(launcherGames, gameEntry{
		name:     "FLAPPY-GUY",
		gameName: "flappy-guy",
		color:    color.RGBA{60, 180, 220, 255},
		key:      ebiten.KeyF,
		newScene: func() Scene {
			return &GameRunnerScene{runner: createGameFramework("flappy-guy"), gameName: "flappy-guy", spaceTimer: -1}
		},
	})
	launcherGames = append(launcherGames, gameEntry{
		name:     "Pet the Damn Cat!",
		gameName: "petthedamncat",
		color:    color.RGBA{160, 82, 28, 255}, // #A0521C
		key:      ebiten.KeyP,
		newScene: func() Scene {
			return &GameRunnerScene{runner: createGameFramework("petthedamncat"), gameName: "petthedamncat", spaceTimer: -1}
		},
	})
	launcherGames = append(launcherGames, gameEntry{
		name:     "V-CLICKER",
		gameName: "vclicker",
		color:    color.RGBA{163, 24, 64, 255}, // #A31840
		key:      ebiten.KeyV,
		newScene: func() Scene {
			return &GameRunnerScene{runner: createGameFramework("vclicker"), gameName: "vclicker", spaceTimer: -1}
		},
	})
}
