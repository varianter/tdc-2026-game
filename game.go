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

//go:embed shaders/crt.kage
var crtShaderSrc []byte

const (
	ScreenW = 426
	ScreenH = 240
)

// crtMode selects the global CRT post-process. Toggled from the launcher.
type crtMode int

const (
	crtOff crtMode = iota
	crtSubtle
	crtFull
)

// currentCRT is the active CRT mode. Shared between the launcher (which toggles
// it) and Game.Draw (which applies it). Defaults to subtle so the effect is
// visible out of the box.
var currentCRT = crtSubtle

var crtShader *ebiten.Shader

// ensureCRTShader lazily compiles the shader on first use (after the graphics
// context exists), then caches it.
func ensureCRTShader() *ebiten.Shader {
	if crtShader == nil {
		s, err := ebiten.NewShader(crtShaderSrc)
		if err != nil {
			log.Fatalf("compile CRT shader: %v", err)
		}
		crtShader = s
	}
	return crtShader
}

// crtUniforms returns the shader uniforms for a mode. crtOff is handled before
// the shader runs, so only subtle/full are represented here.
func crtUniforms(m crtMode) map[string]any {
	u := map[string]any{"Logical": []float32{float32(ScreenW), float32(ScreenH)}}
	switch m {
	case crtFull:
		u["ScanDepth"] = float32(0.35)
		u["ScanHard"] = float32(-3.0)
		u["BrightBoost"] = float32(1.18)
		u["MaskDark"] = float32(0.90)
		u["MaskLight"] = float32(1.12)
		u["WarpX"] = float32(0.03)
		u["WarpY"] = float32(0.04)
		u["Vignette"] = float32(0.25)
	default: // crtSubtle
		u["ScanDepth"] = float32(0.18)
		u["ScanHard"] = float32(-2.0)
		u["BrightBoost"] = float32(1.06)
		u["MaskDark"] = float32(1.0) // 1.0/1.0 = mask disabled
		u["MaskLight"] = float32(1.0)
		u["WarpX"] = float32(0.0) // 0 = flat, no curvature
		u["WarpY"] = float32(0.0)
		u["Vignette"] = float32(0.12)
	}
	return u
}

type Game struct {
	currentScene Scene
	offscreen    *ebiten.Image // frame buffer the scene renders into before the CRT pass
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
	if currentCRT == crtOff {
		g.currentScene.Draw(screen)
		return
	}

	// Render the whole frame into an offscreen buffer, then blit it to the real
	// screen through the CRT shader so every game and the launcher get the
	// effect uniformly.
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	if w == 0 || h == 0 {
		return
	}
	if g.offscreen == nil || g.offscreen.Bounds().Dx() != w || g.offscreen.Bounds().Dy() != h {
		g.offscreen = ebiten.NewImage(w, h)
	}
	g.offscreen.Clear()
	g.currentScene.Draw(g.offscreen)

	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = g.offscreen
	op.Uniforms = crtUniforms(currentCRT)
	screen.DrawRectShader(w, h, ensureCRTShader(), op)
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
		name:     "BOUNCY CASTLE",
		gameName: "bounce",
		color:    color.RGBA{214, 58, 52, 255}, // #D63A34, bouncy-castle vinyl red
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
		name:     "SCRUM SMASHER 3000",
		gameName: "vclicker",
		color:    color.RGBA{163, 24, 64, 255}, // #A31840
		key:      ebiten.KeyV,
		newScene: func() Scene {
			return &GameRunnerScene{runner: createGameFramework("vclicker"), gameName: "vclicker", spaceTimer: -1}
		},
	})
}
