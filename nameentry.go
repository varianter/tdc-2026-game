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
	// nameHoldDuration is how long a hold takes to type on the focused key.
	// Taps (released within tapMaxDuration) move to the next key instead.
	nameHoldDuration = 0.6
	nameKeyCols      = 3
)

// nameKey is one key on the name keypad: its printed label and what holding
// it types.
type nameKey struct {
	label string
	opts  []string
}

// nameKeys is the single-button keyboard, laid out like an old phone keypad.
// Tap to move to the next key, and hold to enter it. Inside a key, tap to
// cycle its options (and nameBack), and hold to type the selected one, which
// leaves the key again. The single-option keys (*, 0, #) act right away on a
// hold instead of being entered, so DEL can be held repeatedly.
var nameKeys = []nameKey{
	{"1", []string{".", "-", "!", "?"}},
	{"2", []string{"A", "B", "C"}},
	{"3", []string{"D", "E", "F"}},
	{"4", []string{"G", "H", "I"}},
	{"5", []string{"J", "K", "L"}},
	{"6", []string{"M", "N", "O"}},
	{"7", []string{"P", "Q", "R", "S"}},
	{"8", []string{"T", "U", "V"}},
	{"9", []string{"W", "X", "Y", "Z"}},
	{"*", []string{"DEL"}},
	{"0", []string{"SPC"}},
	{"#", []string{"OK"}},
}

// nameBack is the last option inside every entered key: holding it leaves
// the key without typing. It's drawn as a return-arrow icon, not this text.
const nameBack = "<"

// keyOptions returns the options to cycle through once inside key.
func keyOptions(key int) []string {
	opts := nameKeys[key].opts
	return append(opts[:len(opts):len(opts)], nameBack)
}

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
	key       int  // focused key, index into nameKeys
	inKey     bool // the key has been entered; taps cycle its options
	opt       int  // selected option inside the entered key, index into keyOptions
	t         float64
	holdTimer float64
	consumed  bool // the current hold already typed; wait for release
	holdArmed bool // false until the button carried over from the game is released
}

func NewNameEntryScene(gameName string, scoreID int64) *NameEntryScene {
	return &NameEntryScene{gameName: gameName, scoreID: scoreID}
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
			opts := nameKeys[n.key].opts
			switch {
			case n.inKey:
				if opt := keyOptions(n.key)[n.opt]; opt != nameBack {
					n.name = editName(n.name, opt)
				}
				n.inKey = false
			case opts[0] == "OK":
				return n.finish(), nil
			case len(opts) == 1:
				n.name = editName(n.name, opts[0])
			default:
				n.inKey, n.opt = true, 0
			}
		}
		return nil, nil
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyEnter) && !n.consumed && n.holdTimer <= tapMaxDuration {
		if n.inKey {
			n.opt = (n.opt + 1) % len(keyOptions(n.key))
		} else {
			n.key = (n.key + 1) % len(nameKeys)
		}
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
	tdcgame.WriteAt(screen, "ENTER YOUR NAME", cx*s, 26*s, 7*s, whiteTextColor, text.AlignCenter, text.AlignStart)

	// Name box, with a blinking cursor while there's room for more.
	const boxW, boxH, boxY = 240.0, 24.0, 38.0
	strokeRoundedRect(screen, float32((cx-boxW/2)*s), float32(boxY*s), float32(boxW*s), float32(boxH*s), float32(6*s), float32(s), whiteTextColor)
	// Inside a key, the selected option previews in gold where it would go.
	nameX, nameY := (cx-boxW/2+10)*s, (boxY+boxH/2)*s
	tdcgame.WriteAt(screen, n.name, nameX, nameY, 12*s, whiteTextColor, text.AlignStart, text.AlignCenter)
	if len(n.name) < maxNameLen {
		cursorX := nameX + tdcgame.MeasureWidth(n.name, 12*s)
		if n.inKey {
			if opt := keyOptions(n.key)[n.opt]; opt != nameBack {
				tdcgame.WriteAt(screen, opt, cursorX, nameY, 12*s, rankYellowColor, text.AlignStart, text.AlignCenter)
			}
		}
		if math.Mod(n.t, 1) < 0.5 {
			tdcgame.WriteAt(screen, "_", cursorX, nameY+3*s, 12*s, whiteTextColor, text.AlignStart, text.AlignCenter)
		}
	}

	// Keypad. The focused key is outlined in gold and fills while a hold
	// enters it. Once entered, the other keys dim, its back option shows, and
	// the selected option is boxed and fills while held.
	const keyW, keyH, gap, padX, gridY = 112.0, 25.0, 4.0, 8.0, 68.0
	gridX := cx - (nameKeyCols*keyW+(nameKeyCols-1)*gap)/2
	progress := math.Max(0, math.Min(1, (n.holdTimer-tapMaxDuration)/(nameHoldDuration-tapMaxDuration)))
	if n.consumed {
		progress = 0
	}
	black := color.RGBA{0, 0, 0, 255}
	for k, key := range nameKeys {
		x := gridX + float64(k%nameKeyCols)*(keyW+gap)
		y := gridY + float64(k/nameKeyCols)*(keyH+gap)
		fx, fy, fw, fh, r := float32(x*s), float32(y*s), float32(keyW*s), float32(keyH*s), float32(6*s)
		focused := k == n.key
		opts := key.opts
		if focused && n.inKey {
			opts = keyOptions(k)
		}

		textClr, numClr := whiteTextColor, scoreboardEmptyColor
		switch {
		case focused:
			numClr = rankYellowColor
			if !n.inKey && progress > 0 {
				fillRoundedRect(screen, fx, fy, fw*float32(progress), fh, r, rankYellowColor)
				textClr, numClr = black, black
			}
			strokeRoundedRect(screen, fx, fy, fw, fh, r, float32(2*s), rankYellowColor)
		case n.inKey:
			textClr = scoreboardEmptyColor
			strokeRoundedRect(screen, fx, fy, fw, fh, r, float32(s/2), scoreboardEmptyColor)
		default:
			strokeRoundedRect(screen, fx, fy, fw, fh, r, float32(s/2), scoreboardEmptyColor)
		}
		tdcgame.WriteAt(screen, key.label, (x+5)*s, (y+3)*s, 5*s, numClr, text.AlignStart, text.AlignStart)

		size := 11.0
		if len(opts) == 1 {
			size = 7
		}
		optW := (keyW - 2*padX) / float64(len(opts))
		midY := y + keyH/2 + 1
		for i, opt := range opts {
			ox := x + padX + float64(i)*optW
			clr := textClr
			if focused && n.inKey && i == n.opt {
				hx, hy, hw, hh, hr := float32((ox+2)*s), float32((midY-8)*s), float32((optW-4)*s), float32(16*s), float32(4*s)
				clr = rankYellowColor
				if progress > 0 {
					fillRoundedRect(screen, hx, hy, hw*float32(progress), hh, hr, rankYellowColor)
					clr = black
				}
				strokeRoundedRect(screen, hx, hy, hw, hh, hr, float32(s), rankYellowColor)
			}
			if opt == nameBack {
				// Pixels sized to match the font's, which is an 8px grid.
				px := size / 8
				iw, ih := float64(len(backIcon.rows[0]))*px, float64(len(backIcon.rows))*px
				backIcon.DrawColored(screen, (ox+optW/2-iw/2)*s, (midY-ih/2)*s, px*s, clr)
				continue
			}
			tdcgame.WriteAt(screen, opt, (ox+optW/2)*s, midY*s, size*s, clr, text.AlignCenter, text.AlignCenter)
		}
	}

	// Action hints: tap icon + label, then hold icon + label.
	const iconScale, iconY, labelGap, groupGap, labelSize = 1.5, 190.0, 5.0, 36.0, 6.0
	iconW := func(p *pixelArt) float64 { return float64(len(p.rows[0])) * iconScale }
	tapLabel, holdLabel := "NEXT KEY", "OPEN KEY"
	switch {
	case n.inKey && keyOptions(n.key)[n.opt] == nameBack:
		tapLabel, holdLabel = "NEXT LETTER", "BACK"
	case n.inKey:
		tapLabel, holdLabel = "NEXT LETTER", "TYPE IT"
	case nameKeys[n.key].opts[0] == "OK":
		holdLabel = "SAVE NAME"
	case nameKeys[n.key].opts[0] == "DEL":
		holdLabel = "DELETE"
	case nameKeys[n.key].opts[0] == "SPC":
		holdLabel = "SPACE"
	}
	tapW := iconW(tapIcon.hand) + labelGap + tdcgame.MeasureWidth(tapLabel, labelSize*s)/s
	holdW := iconW(holdIcon.hand) + labelGap + tdcgame.MeasureWidth(holdLabel, labelSize*s)/s
	labelY := (iconY + 11*iconScale) * s

	gx := cx - (tapW+groupGap+holdW)/2
	tapIcon.Draw(screen, gx*s, iconY*s, s*iconScale, n.t)
	tdcgame.WriteAt(screen, tapLabel, (gx+iconW(tapIcon.hand)+labelGap)*s, labelY, labelSize*s, whiteTextColor, text.AlignStart, text.AlignCenter)

	gx += tapW + groupGap
	holdIcon.Draw(screen, gx*s, iconY*s, s*iconScale, n.t)
	tdcgame.WriteAt(screen, holdLabel, (gx+iconW(holdIcon.hand)+labelGap)*s, labelY, labelSize*s, whiteTextColor, text.AlignStart, text.AlignCenter)
}
