package main

import (
	"image/color"
	"log"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"variant.dev/tdcgame/tdcgame"
)

const (
	maxNameLen = 12
	// nameHoldDuration is how long a hold takes to pick the focused cell. Taps
	// (released within tapMaxDuration) move the cursor instead.
	nameHoldDuration = 0.6
	nameGridCols     = 10
)

// nameCells is the single-button keyboard, laid out left to right in rows of
// nameGridCols. OK is last so the cursor can start on it: hold to skip, or
// tap once to wrap around to A.
var nameCells = append(strings.Split("ABCDEFGHIJKLMNOPQRSTUVWXYZ", ""), "SPC", "DEL", "OK")

// editName applies a picked letter, SPC or DEL to name. Names are capped at
// maxNameLen and never start with or double up a space.
func editName(name, cell string) string {
	switch cell {
	case "DEL":
		if name == "" {
			return name
		}
		return name[:len(name)-1]
	case "SPC":
		if name == "" || strings.HasSuffix(name, " ") || len(name) >= maxNameLen {
			return name
		}
		return name + " "
	default:
		if len(name) >= maxNameLen {
			return name
		}
		return name + cell
	}
}

// NameEntryScene lets a new #1 player put their name on the leaderboard,
// reached from the champion screen. OK saves the name and returns to the
// launcher.
type NameEntryScene struct {
	gameName  string
	scoreID   int64
	name      string
	cursor    int
	t         float64
	holdTimer float64
	consumed  bool // the current hold already picked a cell; wait for release
	holdArmed bool // false until the button carried over from the game is released
}

func NewNameEntryScene(gameName string, scoreID int64) *NameEntryScene {
	return &NameEntryScene{gameName: gameName, scoreID: scoreID, cursor: len(nameCells) - 1}
}

func (n *NameEntryScene) Update(dt float64) (Scene, error) {
	n.t += dt
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		return NewLauncherScene(), nil
	}

	pressed := ebiten.IsKeyPressed(ebiten.KeyEnter)
	if !n.holdArmed {
		n.holdArmed = !pressed
		return nil, nil
	}

	if pressed {
		n.holdTimer += dt
		if !n.consumed && n.holdTimer >= nameHoldDuration {
			n.consumed = true
			if nameCells[n.cursor] == "OK" {
				return n.finish(), nil
			}
			n.name = editName(n.name, nameCells[n.cursor])
		}
		return nil, nil
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyEnter) && !n.consumed && n.holdTimer <= tapMaxDuration {
		n.cursor = (n.cursor + 1) % len(nameCells)
	}
	n.holdTimer, n.consumed = 0, false
	return nil, nil
}

func (n *NameEntryScene) finish() Scene {
	if name := strings.TrimSpace(n.name); name != "" && scoreKeeper != nil {
		if err := scoreKeeper.SetName(n.gameName, n.scoreID, name); err != nil {
			log.Printf("failed to save name: %v", err)
		}
	}
	return NewLauncherScene()
}

func (n *NameEntryScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)
	s := float64(screen.Bounds().Dx()) / ScreenW
	cx := float64(ScreenW) / 2

	tdcgame.WriteAt(screen, "NEW #1!", cx*s, 10*s, 12*s, rankYellowColor, text.AlignCenter, text.AlignStart)
	tdcgame.WriteAt(screen, "ENTER YOUR NAME", cx*s, 28*s, 7*s, whiteTextColor, text.AlignCenter, text.AlignStart)

	// Name box, with a blinking cursor while there's room for more.
	const boxW, boxH, boxY = 240.0, 24.0, 42.0
	strokeRoundedRect(screen, float32((cx-boxW/2)*s), float32(boxY*s), float32(boxW*s), float32(boxH*s), float32(6*s), float32(s), whiteTextColor)
	shown := n.name
	if len(n.name) < maxNameLen && math.Mod(n.t, 1) < 0.5 {
		shown += "_"
	}
	tdcgame.WriteAt(screen, shown, (cx-boxW/2+10)*s, (boxY+boxH/2)*s, 12*s, whiteTextColor, text.AlignStart, text.AlignCenter)

	// Grid. The focused cell is outlined in gold and fills while held.
	const cellW, cellH, gap, gridY = 34.0, 30.0, 4.0, 78.0
	gridX := cx - (nameGridCols*cellW+(nameGridCols-1)*gap)/2
	progress := math.Max(0, math.Min(1, (n.holdTimer-tapMaxDuration)/(nameHoldDuration-tapMaxDuration)))
	for i, cell := range nameCells {
		x := gridX + float64(i%nameGridCols)*(cellW+gap)
		y := gridY + float64(i/nameGridCols)*(cellH+gap)
		fx, fy, fw, fh, r := float32(x*s), float32(y*s), float32(cellW*s), float32(cellH*s), float32(5*s)

		textClr := whiteTextColor
		if i == n.cursor {
			if p := progress; p > 0 && !n.consumed {
				fillRoundedRect(screen, fx, fy, fw*float32(p), fh, r, rankYellowColor)
				textClr = color.RGBA{0, 0, 0, 255}
			}
			strokeRoundedRect(screen, fx, fy, fw, fh, r, float32(2*s), rankYellowColor)
		} else {
			strokeRoundedRect(screen, fx, fy, fw, fh, r, float32(s/2), scoreboardEmptyColor)
		}

		size := 12.0
		if len(cell) > 1 {
			size = 7
		}
		tdcgame.WriteAt(screen, cell, (x+cellW/2)*s, (y+cellH/2)*s, size*s, textClr, text.AlignCenter, text.AlignCenter)
	}

	tdcgame.WriteAt(screen, "TAP: NEXT    HOLD: SELECT", cx*s, 196*s, 6*s, whiteTextColor, text.AlignCenter, text.AlignStart)
	tdcgame.WriteAt(screen, "HOLD OK WHEN DONE (OR TO SKIP)", cx*s, 212*s, 6*s, scoreboardEmptyColor, text.AlignCenter, text.AlignStart)
}
