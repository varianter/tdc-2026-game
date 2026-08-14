package tdcgame

import (
	"bytes"
	"embed"
	"fmt"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (

	// Should not be changeable by implementers
	GroundY          = 186
	ScreenW          = 426
	ScreenH          = 240
	GroundDrawOffset = 6.0

	MaxRunTime = 120
	GameEnd    = 1200 // default end-flag X for games that don't implement TdcGameEndX
)

// GameRunner interface with helpers to make it easier to make a game
type GameRunner struct {
	player            *Player
	camera            Camera
	assets            *Assets
	level             *Level
	game              TdcGame
	params            *GameParameters
	currentScore      int
	previousGameState GameState
	scoreKeeper       *ScoreKeeper
	gameName          string
	runtime           float64
	started           bool
	viewport          *ebiten.Image // fixed-res game canvas; scaled to fill screen each frame
}

func NewGameFrameworkWithPlayer(embed embed.FS, game TdcGameWithPlayer, scoreKeeper *ScoreKeeper, gameName string) *GameRunner {
	params := game.GetGameParameters()
	assets := LoadAssets(embed)
	sheet := LoadSpriteSheet(assets, 64, 64)
	player := Newplayer(sheet, params)
	objs := game.GetGameObjects()

	return &GameRunner{
		player:            player,
		assets:            assets,
		level:             NewLevelFromObjects(objs),
		game:              game,
		params:            params,
		previousGameState: game.GetGameState(),
		scoreKeeper:       scoreKeeper,
		gameName:          gameName,
	}
}

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

func (c *Canvas) TilingGround(img *ebiten.Image, cameraX, cameraY, worldWidth float64) {
	w := float64(img.Bounds().Dx()) // w is 32 for our ground texture
	startX := math.Floor(cameraX/w) * w

	for x := startX; x < cameraX+ScreenW; x += w {
		c.DrawImage(img, x-cameraX, (GroundY-GroundDrawOffset)-cameraY)
	}
}

type Camera struct {
	x, y float64
}

func (cam *Camera) Follow(player *Player, screenW, screenH int) {
	// camera stays on player
	cam.x = player.x - float64(screenW)/2 + float64(player.sheet.FrameW)/2
	cam.y = -player.y
}

func (g *GameRunner) State() GameState {
	return g.game.GetGameState()
}

func (g *GameRunner) Update() error {
	dt := 1.0 / float64(ebiten.TPS()) // calculate deltatime based on TPS, ~0.0166 at 60 TPS
	g.runtime += dt

	if !g.started {
		g.player.UpdateStartScreen(dt)
		if g.params.ShouldCameraFollowPlayer {
			g.camera.Follow(g.player, ScreenH, ScreenW)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.started = true
		}
		return nil
	}

	if g.previousGameState != GameOver {
		if g.runtime >= MaxRunTime {
			g.previousGameState = GameOver
		}
		if gp, ok := g.game.(TdcGameWithPlayer); ok {
			err := g.player.Update(dt, *g.level, gp.GetPlayerUpdateFunc())
			if err != nil {
				return err
			}

			g.currentScore = g.game.GetCurrentScore()
			if g.params.ShouldCameraFollowPlayer {
				g.camera.Follow(g.player, ScreenH, ScreenW)
			}
		}
	}

	state := g.game.GetGameState()
	if state != g.previousGameState {
		if state == GameOver {
			if g.scoreKeeper != nil {
				if err := g.scoreKeeper.AddScore(g.gameName, g.currentScore); err != nil {
					log.Printf("failed to save score: %v", err)
				}
			}
		}
		g.previousGameState = state
	}

	return nil
}

func (g *GameRunner) Draw(screen *ebiten.Image) {
	// scale: how many device pixels correspond to one game pixel
	scale := float64(screen.Bounds().Dx()) / ScreenW

	if cd, ok := g.game.(GameWithCustomDraw); ok {
		cd.CustomDraw(screen)
		return
	}

	// ── Render game world into a fixed-resolution viewport ─────────────────
	if g.viewport == nil {
		g.viewport = ebiten.NewImage(ScreenW, ScreenH)
	}
	vp := g.viewport
	vp.Fill(color.RGBA{30, 30, 30, 255})

	c := &Canvas{screen: vp}
	c.Rect(0, 0, float32(ScreenW), float32(ScreenH), color.RGBA{135, 206, 235, 255}) // sky

	if gb, ok := g.game.(TdcGameWithBackground); ok {
		gb.DrawBackground(vp, g.camera.x, g.camera.y)
	}

	if _, ok := g.game.(TdcGameWithPlayer); ok {
		c.TilingGround(g.assets.Sprites["ground"], g.camera.x, g.camera.y, 5000)

		g.DrawImage(c, g.player.currentAnimation.CurrentFrame(), g.player.x, g.player.y)

		g.RectXY(c, float32(-20), float32(0), float32(20), float32(40), color.RGBA{105, 76, 0, 255})

		for _, gObj := range g.level.gameObjects {
			if gObj.removed {
				continue
			}
			if gObj.t == Flag {
				g.drawFlag(c, float32(gObj.s.P.X), float32(gObj.s.P.Y), gObj.Color())
			} else {
				g.RectXY(c, float32(gObj.s.P.X), float32(gObj.s.P.Y), float32(gObj.s.W), float32(gObj.s.H), gObj.Color())
			}
		}
		endX := float64(GameEnd)
		if ge, ok := g.game.(TdcGameEndX); ok {
			endX = ge.EndX()
		}
		g.RectXY(c, float32(endX+4), float32(0), float32(3), float32(64), color.RGBA{30, 31, 30, 255})
		g.RectXY(c, float32(endX+4), float32(64), float32(30), float32(20), color.RGBA{255, 56, 147, 255})
	}

	if gd, ok := g.game.(TdcGameWithDraw); ok {
		gd.Draw(vp, g.camera.x, g.camera.y)
	}

	// ── Scale viewport to fill screen (nearest-neighbour keeps pixel art sharp) ──
	vpOp := &ebiten.DrawImageOptions{}
	vpOp.GeoM.Scale(scale, scale)
	vpOp.Filter = ebiten.FilterNearest
	screen.DrawImage(vp, vpOp)

	// ── Game overlay (crisp text drawn by the game itself) ──────────────────
	if god, ok := g.game.(TdcGameWithOverlayDraw); ok {
		god.DrawOverlay(screen, scale, g.camera.x, g.camera.y)
	}

	s := scale

	if !g.started {
		const pad = 12
		lineY := int(float64(ScreenH/2-6) * s)
		lineH := int(14 * s)
		bgX := float32(float64(ScreenW/2)*s - 120*s)
		bgY := float32(float64(lineY) - float64(pad)*s)
		bgW := float32(240 * s)
		bgH := float32(float64(lineH+pad*2) * s)
		vector.FillRect(screen, bgX, bgY, bgW, bgH, color.RGBA{40, 40, 40, 200}, false)
		WriteCentered(screen, "Press the big red button to start", lineY, lineH)
	}

	if g.game.GetGameState() == GameOver {
		const pad = 12
		line1Y := int(float64(ScreenH/2-36) * s)
		line2Y := int(float64(ScreenH/2-12) * s)
		line3Y := int(float64(ScreenH/2+12) * s)
		line4Y := int(float64(ScreenH/2+28) * s)
		lineSmH := int(10 * s)

		bgX := float32(float64(ScreenW/2)*s - 150*s)
		bgY := float32(float64(line1Y) - float64(pad)*s)
		bgW := float32(300 * s)
		bgH := float32(line4Y+lineSmH) + float32(pad)*float32(s) - bgY

		vector.FillRect(screen, bgX, bgY, bgW, bgH, color.RGBA{40, 40, 40, 200}, false)
		WriteCentered(screen, "GAME OVER", line1Y, int(16*s))
		WriteCentered(screen, fmt.Sprintf("Score: %d", g.currentScore), line2Y, int(16*s))
		WriteCentered(screen, "Press button to return to the game wheel", line3Y, lineSmH)
		WriteCentered(screen, "Double press button to replay the same game", line4Y, lineSmH)
	} else {
		Write(screen, fmt.Sprintf("Score: %d.    Time left: %.1fs", g.game.GetCurrentScore(), MaxRunTime-g.runtime), int(5*s), int(5*s), int(8*s))
	}
}

// Startflag
func (g *GameRunner) drawFlag(c *Canvas, x, y float32, flagColor color.RGBA) {
	g.RectXY(c, float32(x), float32(y), float32(4), float32(64), color.RGBA{30, 31, 30, 255})
	g.RectXY(c, float32(x), float32(y+64), float32(30), float32(20), flagColor)
}

func (g *GameRunner) DrawImage(c *Canvas, img *ebiten.Image, x float64, y float64) {
	// TODO: Such casting
	c.DrawImage(img, x-g.camera.x, float64(zeroY(float32(y), float32(img.Bounds().Dy()))-float32(g.camera.y)))
}

func (g *GameRunner) RectXY(c *Canvas, x, y, w, h float32, clr color.Color) {
	vector.FillRect(c.screen, x-float32(g.camera.x), zeroY(y, h)-float32(g.camera.y), w, h, clr, false)
}

// 0,0 on grid system is bottom left of players frame on game start
func zeroY(y, h float32) float32 {
	return GroundY - h - y
}

// WriteAt draws msg at device-pixel coordinates x,y with the given size, color
// and alignment, using the shared M+ font. It is the primitive that all other
// Write* helpers delegate to, so every game renders text through one code path.
func WriteAt(s *ebiten.Image, msg string, x, y, size float64, clr color.Color, primary, secondary text.Align) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.PrimaryAlign = primary
	op.SecondaryAlign = secondary
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(s, msg, &text.GoTextFace{Source: mplusFaceSource, Size: size}, op)
}

// Write draws left-aligned white text at x,y.
func Write(s *ebiten.Image, msg string, x, y int, size int) {
	WriteAt(s, msg, float64(x), float64(y), float64(size), color.White, text.AlignStart, text.AlignStart)
}

// WriteCentered draws white text horizontally centered on the image width.
func WriteCentered(s *ebiten.Image, msg string, y int, size int) {
	WriteAt(s, msg, float64(s.Bounds().Dx()/2), float64(y), float64(size), color.White, text.AlignCenter, text.AlignStart)
}

// WriteCenteredAt draws white text horizontally centered on x (all in device pixels).
func WriteCenteredAt(s *ebiten.Image, msg string, x, y, size int) {
	WriteAt(s, msg, float64(x), float64(y), float64(size), color.White, text.AlignCenter, text.AlignStart)
}

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.MPlus1pRegular_ttf))
	if err != nil {
		log.Fatal(err)
	}
	mplusFaceSource = s
}

var mplusFaceSource *text.GoTextFaceSource
