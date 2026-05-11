package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	ghLaneCount = 4
	ghLaneW     = 60
	ghNoteH     = 12
	ghNoteW     = 54
	ghHitY      = 205.0  // hit zone center y
	ghSpeed     = 180.0  // note fall speed px/sec
	ghPerfect   = 12.0   // px from hit center for PERFECT
	ghGood      = 26.0   // px from hit center for GOOD
	ghLaneOff   = (ScreenW - ghLaneCount*ghLaneW) / 2 // = 93
)

var (
	ghColors = [ghLaneCount]color.RGBA{
		{50, 210, 50, 255},   // green
		{220, 50, 50, 255},   // red
		{220, 200, 50, 255},  // yellow
		{60, 120, 230, 255},  // blue
	}
	ghKeys = [ghLaneCount]ebiten.Key{
		ebiten.KeyD, ebiten.KeyF, ebiten.KeyJ, ebiten.KeyK,
	}
	ghKeyLabels = [ghLaneCount]string{"D", "F", "J", "K"}
)

type guitarState int

const (
	guitarPlaying  guitarState = iota
	guitarGameOver
)

type noteEvent struct {
	time float64
	lane int
}

type fallingNote struct {
	targetTime float64
	lane       int
	hit        bool
	missed     bool
}

type hitEffect struct {
	x, y  float64
	text  string
	timer float64
	clr   color.RGBA
}

type GuitarScene struct {
	assets   *Assets
	song     []noteEvent
	songLen  float64
	songLoop int
	songIdx  int
	timer    float64
	notes    []*fallingNote
	effects  []hitEffect
	score    int
	combo    int
	maxCombo int
	health   float64 // 0.0–1.0
	state    guitarState
}

func NewGuitarScene(assets *Assets) *GuitarScene {
	song, songLen := makeGuitarSong()
	return &GuitarScene{
		assets:  assets,
		song:    song,
		songLen: songLen,
		health:  1.0,
	}
}

// makeGuitarSong returns a 64-note, ~15s looping pattern at 130 BPM.
func makeGuitarSong() ([]noteEvent, float64) {
	bpm := 130.0
	e := 60.0 / bpm / 2.0 // 8th note

	lanes := []int{
		// Bar 1: ascending
		0, 1, 2, 3, 0, 1, 2, 3,
		// Bar 2: descending
		3, 2, 1, 0, 3, 2, 1, 0,
		// Bar 3: alternating
		0, 2, 1, 3, 0, 2, 1, 3,
		// Bar 4: grouped pairs
		0, 0, 1, 1, 2, 2, 3, 3,
		// Bar 5: crossing
		0, 3, 1, 2, 3, 0, 2, 1,
		// Bar 6: green anchor
		0, 1, 0, 2, 0, 3, 0, 2,
		// Bar 7: blue anchor
		3, 2, 3, 1, 3, 0, 3, 1,
		// Bar 8: wave
		0, 1, 2, 3, 3, 2, 1, 0,
	}

	events := make([]noteEvent, len(lanes))
	for i, lane := range lanes {
		events[i] = noteEvent{time: float64(i) * e, lane: lane}
	}
	return events, float64(len(lanes)) * e
}

func (g *GuitarScene) Update(dt float64) (Scene, error) {
	if g.state == guitarGameOver {
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			return NewLauncherScene(g.assets), nil
		}
		return nil, nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return NewLauncherScene(g.assets), nil
	}

	g.timer += dt

	// Spawn notes that will be on-screen soon
	travelTime := ghHitY / ghSpeed
	loopTime := float64(g.songLoop) * g.songLen
	for g.songIdx < len(g.song) {
		targetTime := g.song[g.songIdx].time + loopTime
		if targetTime-g.timer <= travelTime+0.15 {
			g.notes = append(g.notes, &fallingNote{
				targetTime: targetTime,
				lane:       g.song[g.songIdx].lane,
			})
			g.songIdx++
		} else {
			break
		}
	}
	if g.songIdx >= len(g.song) {
		g.songLoop++
		g.songIdx = 0
	}

	// Update notes: check for auto-misses and remove dead ones
	alive := g.notes[:0]
	for _, n := range g.notes {
		y := ghHitY - (n.targetTime-g.timer)*ghSpeed
		if !n.hit && !n.missed && y > ghHitY+ghGood {
			n.missed = true
			g.combo = 0
			g.health -= 0.1
			if g.health <= 0 {
				g.health = 0
				g.state = guitarGameOver
			}
		}
		// Keep note until it scrolls off the bottom (or hit note scrolls off top)
		if !(n.hit && y < -float64(ghNoteH)) && !(n.missed && y > float64(ScreenH)) {
			alive = append(alive, n)
		}
	}
	g.notes = alive

	// Handle key presses
	for lane := 0; lane < ghLaneCount; lane++ {
		if !inpututil.IsKeyJustPressed(ghKeys[lane]) {
			continue
		}
		var best *fallingNote
		bestDist := ghGood + 1
		for _, n := range g.notes {
			if n.lane != lane || n.hit || n.missed {
				continue
			}
			y := ghHitY - (n.targetTime-g.timer)*ghSpeed
			dist := math.Abs(y - ghHitY)
			if dist <= ghGood && dist < bestDist {
				bestDist = dist
				best = n
			}
		}
		if best != nil {
			best.hit = true
			g.combo++
			if g.combo > g.maxCombo {
				g.maxCombo = g.combo
			}
			mult := g.comboMult()
			lx := float64(ghLaneOff+lane*ghLaneW+ghLaneW/2)
			if bestDist <= ghPerfect {
				g.score += 100 * mult
				g.health = math.Min(1.0, g.health+0.02)
				g.addEffect(lx, ghHitY-18, "PERFECT!", color.RGBA{255, 230, 50, 255})
			} else {
				g.score += 50 * mult
				g.addEffect(lx, ghHitY-18, "GOOD", color.RGBA{100, 220, 100, 255})
			}
		}
	}

	// Age out effects
	alive2 := g.effects[:0]
	for i := range g.effects {
		g.effects[i].timer -= dt
		g.effects[i].y -= 25 * dt
		if g.effects[i].timer > 0 {
			alive2 = append(alive2, g.effects[i])
		}
	}
	g.effects = alive2

	return nil, nil
}

func (g *GuitarScene) comboMult() int {
	switch {
	case g.combo >= 20:
		return 4
	case g.combo >= 10:
		return 3
	case g.combo >= 5:
		return 2
	default:
		return 1
	}
}

func (g *GuitarScene) addEffect(x, y float64, text string, clr color.RGBA) {
	g.effects = append(g.effects, hitEffect{x: x, y: y, text: text, timer: 0.55, clr: clr})
}

func (g *GuitarScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{10, 10, 20, 255})

	// Lane tracks
	for i := 0; i < ghLaneCount; i++ {
		lx := float32(ghLaneOff + i*ghLaneW)
		vector.FillRect(screen, lx, 0, float32(ghLaneW), float32(ScreenH), color.RGBA{22, 22, 38, 255}, false)
		vector.StrokeLine(screen, lx, 0, lx, float32(ScreenH), 1, color.RGBA{45, 45, 75, 255}, false)
	}
	rx := float32(ghLaneOff + ghLaneCount*ghLaneW)
	vector.StrokeLine(screen, rx, 0, rx, float32(ScreenH), 1, color.RGBA{45, 45, 75, 255}, false)

	// Hit zone guideline
	vector.StrokeLine(screen,
		float32(ghLaneOff), ghHitY,
		float32(ghLaneOff+ghLaneCount*ghLaneW), ghHitY,
		1, color.RGBA{120, 120, 180, 100}, false)

	// Falling notes
	for _, n := range g.notes {
		if n.hit {
			continue
		}
		y := float32(ghHitY - (n.targetTime-g.timer)*ghSpeed)
		if y < -float32(ghNoteH) || y > float32(ScreenH) {
			continue
		}
		lx := float32(ghLaneOff + n.lane*ghLaneW + (ghLaneW-ghNoteW)/2)
		clr := ghColors[n.lane]
		if n.missed {
			clr = dimColor(clr, 0.25)
		}
		vector.FillRect(screen, lx, y-float32(ghNoteH)/2, float32(ghNoteW), float32(ghNoteH), clr, false)
		vector.StrokeRect(screen, lx, y-float32(ghNoteH)/2, float32(ghNoteW), float32(ghNoteH), 1, color.RGBA{255, 255, 255, 160}, false)
	}

	// Hit zone buttons (drawn over notes)
	for i := 0; i < ghLaneCount; i++ {
		lx := float32(ghLaneOff + i*ghLaneW + (ghLaneW-ghNoteW)/2)
		clr := ghColors[i]
		if ebiten.IsKeyPressed(ghKeys[i]) {
			clr = brightenColor(clr, 90)
			// Glow effect
			glowClr := clr
			glowClr.A = 60
			vector.FillRect(screen, lx-4, ghHitY-float32(ghNoteH)-2, float32(ghNoteW)+8, float32(ghNoteH)*3, glowClr, false)
		} else {
			clr = dimColor(clr, 0.45)
		}
		vector.FillRect(screen, lx, ghHitY-float32(ghNoteH)/2, float32(ghNoteW), float32(ghNoteH), clr, false)
		vector.StrokeRect(screen, lx, ghHitY-float32(ghNoteH)/2, float32(ghNoteW), float32(ghNoteH), 1.5, color.RGBA{200, 200, 230, 220}, false)
	}

	// Key labels below buttons
	for i, label := range ghKeyLabels {
		kx := ghLaneOff + i*ghLaneW + (ghLaneW-6)/2
		ebitenutil.DebugPrintAt(screen, label, kx, int(ghHitY)+ghNoteH/2+4)
	}

	// Hit effects (floating text)
	for _, ef := range g.effects {
		textW := float32(len(ef.text) * 6)
		textX := float32(ef.x) - textW/2
		bgClr := ef.clr
		bgClr.A = uint8(float64(ef.timer) / 0.55 * 120)
		vector.FillRect(screen, textX-2, float32(ef.y)-1, textW+4, 13, bgClr, false)
		ebitenutil.DebugPrintAt(screen, ef.text, int(textX), int(ef.y))
	}

	// Health bar (top-left)
	const hpBarW = 82.0
	vector.FillRect(screen, 5, 5, hpBarW, 7, color.RGBA{50, 20, 20, 255}, false)
	fillW := float32(hpBarW * g.health)
	hpClr := color.RGBA{50, 220, 50, 255}
	if g.health < 0.3 {
		hpClr = color.RGBA{220, 50, 50, 255}
	} else if g.health < 0.6 {
		hpClr = color.RGBA{220, 180, 50, 255}
	}
	if fillW > 0 {
		vector.FillRect(screen, 5, 5, fillW, 7, hpClr, false)
	}
	vector.StrokeRect(screen, 5, 5, hpBarW, 7, 1, color.RGBA{150, 150, 200, 255}, false)
	ebitenutil.DebugPrintAt(screen, "HP", 5, 15)

	// Score / combo (right side)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("SCORE\n%06d", g.score), ScreenW-68, 5)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("COMBO\n  x%d", g.combo), ScreenW-68, 33)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("MULT\n  x%d", g.comboMult()), ScreenW-68, 61)

	// Title
	title := "-- GUITAR HERO --"
	ebitenutil.DebugPrintAt(screen, title, (ScreenW-len(title)*6)/2, 4)

	// ESC hint
	ebitenutil.DebugPrintAt(screen, "ESC: menu", 5, ScreenH-14)

	// Game over overlay
	if g.state == guitarGameOver {
		vector.FillRect(screen, 0, 0, float32(ScreenW), float32(ScreenH), color.RGBA{0, 0, 0, 185}, false)
		lines := []string{
			"GAME OVER",
			fmt.Sprintf("SCORE: %d", g.score),
			fmt.Sprintf("MAX COMBO: %d", g.maxCombo),
			"",
			"SPACE to return",
		}
		y := ScreenH/2 - len(lines)*8
		for _, line := range lines {
			ebitenutil.DebugPrintAt(screen, line, (ScreenW-len(line)*6)/2, y)
			y += 16
		}
	}
}
