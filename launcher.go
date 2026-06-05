package main

import (
	"image/color"
	"log"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type launcherState int

const (
	launcherIdle launcherState = iota
	launcherSpinning
	launcherResult
	launcherFading
)

type gameEntry struct {
	name     string
	color    color.RGBA
	newScene func() Scene
}

var wheelGames []gameEntry

var whitePx *ebiten.Image

type LauncherScene struct {
	angle     float64
	velocity  float64
	state     launcherState
	winner    int
	timer     float64
	fadeAlpha float64
}

func NewLauncherScene() *LauncherScene {
	if whitePx == nil {
		whitePx = ebiten.NewImage(1, 1)
		whitePx.Fill(color.White)
	}
	return &LauncherScene{}
}

func (l *LauncherScene) Update(dt float64) (Scene, error) {
	if ebiten.IsKeyPressed(ebiten.KeyR) {
		log.Print("jakjaskdj")
		l.winner = 0
		entry := wheelGames[l.winner] // HACK: Use enum for indexes? idk
		if entry.newScene != nil {
			return entry.newScene(), nil
		}
	}

	switch l.state {
	case launcherIdle:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			l.velocity = 15.0 + rand.Float64()*10.0
			l.state = launcherSpinning
		}
	case launcherSpinning:
		l.angle += l.velocity * dt
		l.velocity *= math.Pow(0.98, 60*dt)
		if l.velocity < 0.1 {
			l.velocity = 0
			l.winner = l.calcWinner()
			l.state = launcherResult
			l.timer = 1.5
		}
	case launcherResult:
		l.timer -= dt
		if l.timer <= 0 || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			l.state = launcherFading
		}
	case launcherFading:
		l.fadeAlpha += dt * 255
		if l.fadeAlpha >= 255 {
			entry := wheelGames[l.winner]
			if entry.newScene != nil {
				return entry.newScene(), nil
			}
		}
	}
	return nil, nil
}

func (l *LauncherScene) calcWinner() int {
	N := len(wheelGames)
	sliceAngle := 2 * math.Pi / float64(N)
	// Pointer is at the top (-π/2). Find which segment is under it.
	a := math.Mod(-math.Pi/2-l.angle, 2*math.Pi)
	if a < 0 {
		a += 2 * math.Pi
	}
	return int(a/sliceAngle) % N
}

func (l *LauncherScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{15, 15, 30, 255})

	cx := float32(ScreenW) / 4.0
	cy := float32(ScreenH)/2.0 + 10
	r := float32(80.0)

	N := len(wheelGames)
	sliceAngle := 2 * math.Pi / float64(N)

	for i, game := range wheelGames {
		start := float32(l.angle + float64(i)*sliceAngle)
		end := float32(l.angle + float64(i+1)*sliceAngle)
		clr := game.color
		if l.state == launcherResult || l.state == launcherFading {
			if i == l.winner {
				clr = brightenColor(clr, 70)
			} else {
				clr = dimColor(clr, 0.3)
			}
		}
		drawPieSlice(screen, cx, cy, r, start, end, clr)
	}

	// Divider lines between slices
	for i := range N {
		a := l.angle + float64(i)*sliceAngle
		ex := cx + float32(math.Cos(a))*r
		ey := cy + float32(math.Sin(a))*r
		vector.StrokeLine(screen, cx, cy, ex, ey, 1.5, color.RGBA{10, 10, 20, 220}, true)
	}

	vector.StrokeCircle(screen, cx, cy, r, 2.0, color.RGBA{200, 200, 220, 255}, true)
	vector.FillCircle(screen, cx, cy, 5, color.RGBA{220, 220, 240, 255}, true)

	// Pointer arrow at top of wheel
	drawWheelPointer(screen, cx, cy-r)

	// Title
	title := "TDC 2026 GAME LAUNCHER"
	ebitenutil.DebugPrintAt(screen, title, (ScreenW-len(title)*6)/2, 4)

	// Game list (right half)
	listX := int(ScreenW/2) + 8
	ebitenutil.DebugPrintAt(screen, "GAMES:", listX, 20)
	for i, game := range wheelGames {
		y := 36 + i*15
		isWinner := i == l.winner && (l.state == launcherResult || l.state == launcherFading)
		if isWinner {
			vector.FillRect(screen, float32(listX-2), float32(y-1), 125, 12, color.RGBA{255, 255, 200, 35}, false)
		}
		vector.FillRect(screen, float32(listX), float32(y+2), 7, 7, game.color, false)
		prefix := "  "
		if isWinner {
			prefix = "> "
		}
		ebitenutil.DebugPrintAt(screen, prefix+game.name, listX+9, y)
	}

	// Status at bottom (centered under wheel)
	var status string
	switch l.state {
	case launcherIdle:
		status = "SPACE to spin"
	case launcherSpinning:
		status = "Spinning..."
	case launcherResult:
		status = wheelGames[l.winner].name + "! [SPACE]"
	case launcherFading:
		status = "Launching " + wheelGames[l.winner].name + "..."
	}
	statusX := int(cx) - len(status)*3
	ebitenutil.DebugPrintAt(screen, status, statusX, ScreenH-14)

	// Fade-to-black overlay
	if l.fadeAlpha > 0 {
		alpha := uint8(math.Min(l.fadeAlpha, 255))
		vector.FillRect(screen, 0, 0, float32(ScreenW), float32(ScreenH), color.RGBA{0, 0, 0, alpha}, false)
	}
}

func drawPieSlice(screen *ebiten.Image, cx, cy, r, startAngle, endAngle float32, clr color.RGBA) {
	var path vector.Path
	path.MoveTo(cx, cy)
	path.Arc(cx, cy, r, startAngle, endAngle, vector.Clockwise)
	path.Close()
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	cr := float32(clr.R) / 255
	cg := float32(clr.G) / 255
	cb := float32(clr.B) / 255
	ca := float32(clr.A) / 255
	for i := range vs {
		vs[i].ColorR = cr
		vs[i].ColorG = cg
		vs[i].ColorB = cb
		vs[i].ColorA = ca
	}
	screen.DrawTriangles(vs, is, whitePx, &ebiten.DrawTrianglesOptions{
		FillRule: ebiten.FillRuleNonZero,
	})
}

func drawWheelPointer(screen *ebiten.Image, tipX, tipY float32) {
	var path vector.Path
	path.MoveTo(tipX, tipY)
	path.LineTo(tipX-7, tipY-14)
	path.LineTo(tipX+7, tipY-14)
	path.Close()
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].ColorR = 1
		vs[i].ColorG = 1
		vs[i].ColorB = 1
		vs[i].ColorA = 1
	}
	screen.DrawTriangles(vs, is, whitePx, &ebiten.DrawTrianglesOptions{})
	vector.StrokeLine(screen, tipX, tipY, tipX-7, tipY-14, 1.0, color.RGBA{50, 50, 80, 255}, true)
	vector.StrokeLine(screen, tipX-7, tipY-14, tipX+7, tipY-14, 1.0, color.RGBA{50, 50, 80, 255}, true)
	vector.StrokeLine(screen, tipX+7, tipY-14, tipX, tipY, 1.0, color.RGBA{50, 50, 80, 255}, true)
}

func brightenColor(c color.RGBA, d int) color.RGBA {
	clamp := func(v int) uint8 {
		if v > 255 {
			return 255
		}
		return uint8(v)
	}
	return color.RGBA{clamp(int(c.R) + d), clamp(int(c.G) + d), clamp(int(c.B) + d), c.A}
}

func dimColor(c color.RGBA, f float64) color.RGBA {
	return color.RGBA{
		uint8(float64(c.R) * f),
		uint8(float64(c.G) * f),
		uint8(float64(c.B) * f),
		c.A,
	}
}
