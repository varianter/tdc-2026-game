package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"variant.dev/tdcgame/tdcgame"
)

// championHoldDuration is how long the button must be held to leave the
// champion screen, so a stray tap can't skip the celebration.
const championHoldDuration = 1.0

var crownJewelColor = color.RGBA{214, 58, 52, 255}

// ChampionScene celebrates a score that just took #1 on a game's leaderboard.
type ChampionScene struct {
	gameName  string
	score     int
	t         float64
	holdTimer float64
	holdArmed bool // false until the button carried over from the game is released
}

func NewChampionScene(gameName string, score int) *ChampionScene {
	return &ChampionScene{gameName: gameName, score: score}
}

func (c *ChampionScene) Update(dt float64) (Scene, error) {
	c.t += dt
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		return NewLauncherScene(), nil
	}

	pressed := ebiten.IsKeyPressed(ebiten.KeyEnter)
	if !c.holdArmed {
		c.holdArmed = !pressed
		return nil, nil
	}
	if !pressed {
		c.holdTimer = 0
		return nil, nil
	}
	c.holdTimer += dt
	if c.holdTimer >= championHoldDuration {
		return NewLauncherScene(), nil
	}
	return nil, nil
}

// gameEntryFor returns the launcher entry (display name, color) for gameName.
func gameEntryFor(gameName string) gameEntry {
	for _, g := range launcherGames {
		if g.gameName == gameName {
			return g
		}
	}
	return gameEntry{name: gameName, color: rankYellowColor}
}

func (c *ChampionScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)
	s := float64(screen.Bounds().Dx()) / ScreenW
	cx := float64(ScreenW) / 2

	c.drawCrown(screen, s, cx, 14+2*math.Sin(c.t*3))

	tdcgame.WriteAt(screen, "NEW CHAMPION!", cx*s, 62*s, 14*s, rankYellowColor, text.AlignCenter, text.AlignStart)
	entry := gameEntryFor(c.gameName)
	tdcgame.WriteAt(screen, entry.name, cx*s, 82*s, 7*s, entry.color, text.AlignCenter, text.AlignStart)

	// Leaderboard; the new #1 (always row 0) sits on a pulsing gold pill.
	const boardW, rowH, rowsY, fontSize = 200.0, 18.0, 100.0, 7.0
	x0, x1 := cx-boardW/2, cx+boardW/2
	scores := topScores(c.gameName)
	for i := 0; i < scoreboardMaxRows; i++ {
		y := rowsY + float64(i)*rowH
		clr := rankColor(i)
		value := "0"
		if i < len(scores) {
			value = fmt.Sprintf("%d", scores[i])
		} else {
			clr = scoreboardEmptyColor
		}
		if i == 0 {
			pulse := 0.75 + 0.25*math.Sin(c.t*6)
			gold := color.RGBA{uint8(255 * pulse), uint8(212 * pulse), uint8(47 * pulse), 255}
			fillRoundedRect(screen, float32((x0-8)*s), float32((y-4)*s), float32((boardW+16)*s), float32((rowH-2)*s), float32(7*s), gold)
			clr = color.RGBA{0, 0, 0, 255}
			tdcgame.WriteAt(screen, "YOU", cx*s, y*s, fontSize*s, clr, text.AlignCenter, text.AlignStart)
		}
		tdcgame.WriteAt(screen, fmt.Sprintf("%d.", i+1), x0*s, y*s, fontSize*s, clr, text.AlignStart, text.AlignStart)
		tdcgame.WriteAt(screen, value, x1*s, y*s, fontSize*s, clr, text.AlignEnd, text.AlignStart)
	}

	// Hold-to-continue hint with a fill bar that tracks the hold.
	tdcgame.WriteAt(screen, "HOLD BUTTON TO CONTINUE", cx*s, 202*s, 6*s, whiteTextColor, text.AlignCenter, text.AlignStart)
	const barW, barH, barY = 120.0, 5.0, 216.0
	bx := float32((cx - barW/2) * s)
	strokeRoundedRect(screen, bx, float32(barY*s), float32(barW*s), float32(barH*s), float32(barH/2*s), float32(s), whiteTextColor)
	if p := math.Min(c.holdTimer/championHoldDuration, 1); p > 0 {
		fillRoundedRect(screen, bx, float32(barY*s), float32(barW*p*s), float32(barH*s), float32(barH/2*s), rankYellowColor)
	}
}

// drawCrown draws a five-point gold crown, 60x40 logical px, top-centered at (cx, top).
func (c *ChampionScene) drawCrown(screen *ebiten.Image, s, cx, top float64) {
	pt := func(dx, dy float64) (float32, float32) { return float32((cx + dx) * s), float32((top + dy) * s) }

	var p vector.Path
	p.MoveTo(pt(-30, 40))
	p.LineTo(pt(-30, 8))
	p.LineTo(pt(-15, 24))
	p.LineTo(pt(0, 2))
	p.LineTo(pt(15, 24))
	p.LineTo(pt(30, 8))
	p.LineTo(pt(30, 40))
	p.Close()
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(rankYellowColor)
	vector.FillPath(screen, &p, nil, op)

	// Darker band along the base, with jewels on the band and the three tips.
	bx, by := pt(-30, 31)
	vector.FillRect(screen, bx, by, float32(60*s), float32(9*s), color.RGBA{204, 160, 20, 255}, true)
	for _, d := range []float64{-15, 0, 15} {
		x, y := pt(d, 35.5)
		vector.FillCircle(screen, x, y, float32(2.5*s), crownJewelColor, true)
	}
	for _, tip := range [][2]float64{{-30, 8}, {0, 2}, {30, 8}} {
		x, y := pt(tip[0], tip[1])
		vector.FillCircle(screen, x, y, float32(3*s), rankYellowColor, true)
	}
}
