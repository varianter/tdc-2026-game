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
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (

	// Should not be changeable by implementers
	GroundY          = 186
	ScreenW          = 426
	ScreenH          = 240
	GroundDrawOffset = 6.0
	GameEnd          = 1200
)

// GameRunner interface with helpers to make it easier to make a game
type GameRunner struct {
	player       *Player
	camera       Camera
	assets       *Assets
	level        *Level
	game         TdcGame
	params       *GameParameters
	currentScore int
}

func NewGameFrameworkWithPlayer(embed embed.FS, game TdcGameWithPlayer) *GameRunner {
	params := game.GetGameParameters()
	assets := LoadAssets(embed)
	sheet := LoadSpriteSheet(assets, 64, 64)
	player := Newplayer(sheet, params)
	objs := game.GetGameObjects()

	return &GameRunner{player: player, assets: assets, level: NewLevelFromObjects(objs), game: game, params: params}
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
	// TODO: Detect type of game and call correct update function

	if gp, ok := g.game.(TdcGameWithPlayer); ok {
		coins, err := g.player.Update(dt, *g.level, gp.GetPlayerUpdateFunc())
		if err != nil {
			return err
		}

		g.currentScore += coins
		if g.params.ShouldCameraFollowPlayer {
			g.camera.Follow(g.player, ScreenH, ScreenW)
		}
	}

	return nil
}

func (g *GameRunner) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{30, 30, 30, 255})

	c := &Canvas{
		screen: screen,
	}
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()

	c.Rect(0, 0, float32(w), float32(h), color.RGBA{135, 206, 235, 255}) // sky is light blue, should be replaced with some background

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
		// EndFlag
		g.RectXY(c, float32(GameEnd+4), float32(0), float32(3), float32(64), color.RGBA{30, 31, 30, 255})
		g.RectXY(c, float32(GameEnd+4), float32(64), float32(30), float32(20), color.RGBA{255, 56, 147, 255})
	}

	if g.game.GetGameState() == GameOver {
		const pad = 12
		line1Y := ScreenH/2 - 28
		line2Y := ScreenH/2 - 4
		line3Y := ScreenH/2 + 20
		line3H := 12 // font size of line 3

		bgX := float32(ScreenW/2) - 140
		bgY := float32(line1Y - pad)
		bgW := float32(280)
		bgH := float32(line3Y+line3H+pad) - bgY

		vector.FillRect(screen, bgX, bgY, bgW, bgH, color.RGBA{40, 40, 40, 200}, false)

		WriteCentered(screen, "GAME OVER", line1Y, 16)
		WriteCentered(screen, fmt.Sprintf("Score: %d", g.currentScore), line2Y, 16)
		WriteCentered(screen, "Press button to return to the game wheel", line3Y, line3H)
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

func Write(s *ebiten.Image, msg string, x, y int, size int) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(color.White)
	text.Draw(s, msg, &text.GoTextFace{
		Source: mplusFaceSource,
		Size:   float64(size),
	}, op)
}

func WriteCentered(s *ebiten.Image, msg string, y int, size int) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(ScreenW/2), float64(y))
	op.PrimaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(color.White)
	text.Draw(s, msg, &text.GoTextFace{Source: mplusFaceSource, Size: float64(size)}, op)
}

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.MPlus1pRegular_ttf))
	if err != nil {
		log.Fatal(err)
	}
	mplusFaceSource = s
}

var mplusFaceSource *text.GoTextFaceSource
