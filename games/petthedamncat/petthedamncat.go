// Package petthedamncat — walk the park path and pet cats you encounter.
package petthedamncat

import (
	"bytes"
	"embed"
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"variant.dev/tdcgame/tdcgame"
)

const (
	catSpriteW = 64
	catSpriteH = 64
	sheetCols  = 14

	// Sprite sheet rows (each row = one animation, sheetCols frames wide).
	rowWalkRight       = 4  // 6 frames
	rowWalkLeft        = 5  // 6 frames
	rowTailWagLieRight = 28 // 3 frames — used for petted cats

	// Lowest visible pixel within the 64-px frame for each animation state.
	walkFeetY  = 47.0
	lyingFeetY = 45.0 // tail wag lie right & sleep frames

	popupDuration = 1.4
	sleepDuration = 2.2
	baseWander    = 28.0
	baseRange     = 38.0
	scareRadius   = 220.0 // miss within this range of any cat scares it away

	pathLength  = 10000.0
	minSpeed    = 90.0
	maxSpeed    = 280.0
	maxPetRange = 62.0
	minPetRange = 16.0

	// Big cat constants.
	bigCatScale    = 1.5
	bigCatPats     = 3
	bigCatReward   = 50
	bigCatPetMult  = 1.5  // pet range multiplier (bigger target)
	bigCatSpeedMul = 1.35 // speed relative to player
	bigCatFwdBound = 200.0
	bigCatBckBound = 60.0
)

var catSpawnXs = []float64{
	// Dense 0–3 000: solo cats every ~150 px
	100, 250, 400, 560, 720, 880, 1050, 1220, 1390, 1570,
	1750, 1940, 2130, 2330, 2530, 2740, 2950,
	// Medium 3 000–6 500: solo cats every ~280 px
	3220, 3510, 3800, 4100, 4400, 4720, 5040, 5380, 5720, 6080, 6430,
	// Sparse 6 500–10 000: grouped cats with clear gaps between clusters
	// Cluster 1
	6680, 6760, 6840,
	// Cluster 2
	7300, 7380,
	// Cluster 3
	7820, 7900, 7980,
	// Cluster 4 (flanks the big cat at 8700)
	8300, 8380,
	// Cluster 5
	8950, 9030, 9110,
	// Final rush
	9380, 9460, 9540, 9640,
}

var bigCatSpawnXs = []float64{2800, 5800, 8700}

// Cat is a regular wandering cat.
type Cat struct {
	X, Y        float64
	SpawnX      float64
	VX          float64
	WanderTimer float64
	Variant     int
	animElapsed float64
	Petted      bool
	PettedTimer float64
	Scared      bool // bolts right and despawns after a missed press nearby
}

// BigCat needs three pats before falling asleep, and runs faster than the player.
type BigCat struct {
	X           float64
	SpawnX      float64
	VX          float64
	SprintV     float64 // extra forward speed after each pat, decays over ~1s
	Pats        int     // pats received so far (0‒2; 3 = fully petted)
	Petted      bool
	PettedTimer float64
	PatCooldown float64 // brief lockout after each pat
	animElapsed float64
	Variant     int
	active      bool // starts following player once it gets close enough
	Scared      bool // bolts right after a missed press nearby
}

type Popup struct {
	X, Y   float64
	Timer  float64
	Amount int // points to display (+10 or +50)
}

type PetTheDamnCat struct {
	Cats          []*Cat
	BigCats       []*BigCat
	Popups        []Popup
	Score         int
	CatsPetted    int // regular cats + fully-petted big cats
	ScaredAway    int // cats that bolted and despawned
	SpaceWas      bool
	nearestCat    *Cat
	nearestBigCat *BigCat
	progress      float64
	gameOver      bool
	playerX       float64
	currSpeed     float64
	timeLeft      float64
	catSheets    [3]*tdcgame.SpriteSheet
	font         *text.GoTextFaceSource
	musicPlayer  *audio.Player // loops radioactive.wav while a big cat is active
}

func (p *PetTheDamnCat) Init(assets embed.FS) {
	p.SpaceWas = true // prevent the start-screen space press from scaring a cat
	p.timeLeft = tdcgame.MaxRunTime
	p.currSpeed = minSpeed
	for i, path := range []string{
		"assets/cats/cat1.png",
		"assets/cats/cat2.png",
		"assets/cats/cat3.png",
	} {
		img, _, err := ebitenutil.NewImageFromFileSystem(assets, path)
		if err != nil {
			log.Fatalf("load cat sprite %s: %v", path, err)
		}
		p.catSheets[i] = &tdcgame.SpriteSheet{
			Image: img, FrameW: catSpriteW, FrameH: catSpriteH, Columns: sheetCols,
		}
	}

	p.Cats = make([]*Cat, 0, len(catSpawnXs))
	for _, sx := range catSpawnXs {
		p.Cats = append(p.Cats, &Cat{
			X: sx, SpawnX: sx, VX: baseWander,
			WanderTimer: rand.Float64()*2 + 0.5,
			Variant:     rand.Intn(3),
			animElapsed: rand.Float64() * 10,
		})
	}

	p.BigCats = make([]*BigCat, 0, len(bigCatSpawnXs))
	for _, sx := range bigCatSpawnXs {
		p.BigCats = append(p.BigCats, &BigCat{
			X: sx, SpawnX: sx, VX: minSpeed,
			Variant:     rand.Intn(3),
			animElapsed: rand.Float64() * 10,
		})
	}

	initAudio(assets)
	p.musicPlayer = newMusicPlayer(assets)

	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.MPlus1pRegular_ttf))
	if err != nil {
		log.Fatal(err)
	}
	p.font = s
}

func (p *PetTheDamnCat) GetGameObjects() []tdcgame.GameObject { return nil }
func (p *PetTheDamnCat) GetCurrentScore() int                  { return p.Score }
func (p *PetTheDamnCat) EndX() float64 {
	return p.playerX + math.Max(0, p.timeLeft)*p.currSpeed
}

func (p *PetTheDamnCat) GetGameParameters() *tdcgame.GameParameters {
	return &tdcgame.GameParameters{
		WalkSpeed:                minSpeed, // overridden each frame in PlayerUpdateFunc
		Gravity:                  700,
		AnimWalkFPS:              10.0,
		AnimIdleFPS:              5.0,
		ShouldCameraFollowPlayer: true,
	}
}

func (p *PetTheDamnCat) GetGameState() tdcgame.GameState {
	if p.gameOver {
		return 1 // GameOver
	}
	return 0 // Running
}

func (p *PetTheDamnCat) GetPlayerUpdateFunc() tdcgame.PlayerUpdate {
	return func(spacePressed bool, dt float64, level tdcgame.Level, player *tdcgame.MovingSquare) {
		if p.gameOver {
			player.Vx = 0
			player.Direction = 1
			return
		}

		p.progress = math.Min(1.0, player.P.X/pathLength)
		currentSpeed := minSpeed + (maxSpeed-minSpeed)*p.progress
		player.Vx = currentSpeed
		p.playerX = player.P.X
		p.currSpeed = currentSpeed
		p.timeLeft -= dt

		currentPetRange := maxPetRange - (maxPetRange-minPetRange)*p.progress

		spaceJustPressed := spacePressed && !p.SpaceWas
		p.SpaceWas = spacePressed

		playerCenterX := player.P.X + player.W/2
		playerX := player.P.X

		// ── Regular cats ──────────────────────────────────────────────────
		curWander := baseWander * (1.0 + 2.5*p.progress)
		curRange := baseRange * (1.0 + 0.8*p.progress)
		scaredSpeed := currentSpeed * 4.0 // fast enough to exit screen quickly

		var nearestCat *Cat
		nearestCatDist := math.MaxFloat64

		alive := p.Cats[:0]
		for _, cat := range p.Cats {
			cat.animElapsed += dt

			if cat.Petted {
				cat.PettedTimer += dt
				if cat.PettedTimer < sleepDuration {
					alive = append(alive, cat)
				}
				continue
			}

			if cat.Scared {
				cat.X += cat.VX * dt
				halfW := float64(tdcgame.ScreenW) / 2
				if cat.X > playerX+halfW+catSpriteW || cat.X+catSpriteW < playerX-halfW {
					p.ScaredAway++
				} else {
					alive = append(alive, cat)
				}
				continue
			}

			cat.VX = math.Copysign(curWander, cat.VX)
			cat.X += cat.VX * dt
			cat.WanderTimer -= dt
			bounced := false
			if cat.X < cat.SpawnX-curRange {
				cat.X = cat.SpawnX - curRange
				bounced = true
			}
			if cat.X > cat.SpawnX+curRange {
				cat.X = cat.SpawnX + curRange
				bounced = true
			}
			if cat.X > pathLength-catSpriteW {
				cat.X = pathLength - catSpriteW
				bounced = true
			}
			if bounced || cat.WanderTimer <= 0 {
				cat.VX = -cat.VX
				timerMax := 2.0 - 1.5*p.progress
				cat.WanderTimer = rand.Float64()*timerMax + 0.2
			}

			dist := math.Abs(playerCenterX - (cat.X + catSpriteW/2))
			if dist < currentPetRange && dist < nearestCatDist {
				nearestCatDist = dist
				nearestCat = cat
			}
			alive = append(alive, cat)
		}
		p.Cats = alive
		p.nearestCat = nearestCat

		// ── Big cats ──────────────────────────────────────────────────────
		var nearestBigCat *BigCat
		nearestBigDist := math.MaxFloat64
		bigPetRange := currentPetRange * bigCatPetMult

		aliveBig := p.BigCats[:0]
		for _, bc := range p.BigCats {
			bc.animElapsed += dt
			if bc.PatCooldown > 0 {
				bc.PatCooldown -= dt
			}
			if bc.Petted {
				bc.PettedTimer += dt
				if bc.PettedTimer < sleepDuration {
					aliveBig = append(aliveBig, bc)
				}
				continue
			}
			if bc.Scared {
				bc.X += bc.VX * dt
				halfW := float64(tdcgame.ScreenW) / 2
				scaledW := float64(catSpriteW) * bigCatScale
				if bc.X > playerX+halfW+scaledW || bc.X+scaledW < playerX-halfW {
					p.ScaredAway++
				} else {
					aliveBig = append(aliveBig, bc)
				}
				continue
			}
			// Activate once the player walks within 400px of the spawn point.
			if !bc.active && playerX > bc.SpawnX-400 {
				bc.active = true
				bc.VX = currentSpeed * bigCatSpeedMul
			}
			if bc.active {
				bc.SprintV = math.Max(0, bc.SprintV-180*dt)
				speed := currentSpeed*bigCatSpeedMul + bc.SprintV
				relX := bc.X - playerX
				switch {
				case relX > bigCatFwdBound:
					bc.VX = -speed
				case relX < -bigCatBckBound:
					bc.VX = speed
				default:
					bc.VX = math.Copysign(speed, bc.VX)
				}
				bc.X += bc.VX * dt
				// Big cat also stays behind the finish line.
				if bc.X > pathLength-float64(catSpriteW)*bigCatScale {
					bc.X = pathLength - float64(catSpriteW)*bigCatScale
					bc.VX = -math.Abs(bc.VX) // bounce back
				}

				bcCenterX := bc.X + float64(catSpriteW)*bigCatScale/2
				dist := math.Abs(playerCenterX - bcCenterX)
				if dist < bigPetRange && bc.PatCooldown <= 0 && dist < nearestBigDist {
					nearestBigDist = dist
					nearestBigCat = bc
				}
			}
			aliveBig = append(aliveBig, bc)
		}
		p.BigCats = aliveBig
		p.nearestBigCat = nearestBigCat

		// ── SPACE press ───────────────────────────────────────────────────
		if spaceJustPressed {
			switch {
			case nearestBigCat != nil:
				nearestBigCat.Pats++
				nearestBigCat.PatCooldown = 0.5
				nearestBigCat.SprintV = 110
				nearestBigCat.VX = math.Abs(nearestBigCat.VX)
				if nearestBigCat.Pats >= bigCatPats {
					// Final pat — triumphant sound and full reward.
					playSound(bigCatFinalMeowPCM[nearestBigCat.Variant])
					reward := int(float64(bigCatReward) * (1 + p.progress))
					nearestBigCat.Petted = true
					p.Score += reward
					p.CatsPetted++
					p.Popups = append(p.Popups, Popup{
						X:      nearestBigCat.X + float64(catSpriteW)*bigCatScale/2,
						Y:      float64(catSpriteH)*bigCatScale + 12,
						Timer:  popupDuration,
						Amount: reward,
					})
				} else {
					playSound(bigCatMeowPCM[nearestBigCat.Variant])
				}
			case nearestCat != nil:
				nearestCat.Petted = true
				points := int(float64(10) * (1 + p.progress))
				p.Score += points
				p.CatsPetted++
				playSound(meowPCM[nearestCat.Variant])
				p.Popups = append(p.Popups, Popup{
					X: nearestCat.X + catSpriteW/2, Y: catSpriteH + 12, Timer: popupDuration, Amount: points,
				})
			default:
				// Missed — scare all nearby unprotected cats (regular and big).
				regularScared := false
				bigScared := false
				for _, cat := range p.Cats {
					if cat.Petted || cat.Scared {
						continue
					}
					dist := math.Abs(playerCenterX - (cat.X + catSpriteW/2))
					if dist < scareRadius {
						cat.Scared = true
						if cat.X+catSpriteW/2 < playerCenterX {
							cat.VX = -scaredSpeed
						} else {
							cat.VX = scaredSpeed
						}
						regularScared = true
					}
				}
				for _, bc := range p.BigCats {
					if bc.Petted || bc.Scared {
						continue
					}
					bcCenterX := bc.X + float64(catSpriteW)*bigCatScale/2
					dist := math.Abs(playerCenterX - bcCenterX)
					if dist < scareRadius {
						bc.Scared = true
						if bcCenterX < playerCenterX {
							bc.VX = -scaredSpeed
						} else {
							bc.VX = scaredSpeed
						}
						bigScared = true
					}
				}
				if bigScared {
					playSound(alertPCM)
					playSound(scaredBigPCM)
				} else if regularScared {
					playSound(alertPCM)
					playSound(scaredPCM)
				}
			}
		}

		// ── Popups ────────────────────────────────────────────────────────
		live := p.Popups[:0]
		for i := range p.Popups {
			p.Popups[i].Timer -= dt
			p.Popups[i].Y += 22 * dt
			if p.Popups[i].Timer > 0 {
				live = append(live, p.Popups[i])
			}
		}
		p.Popups = live

		// ── Physics ───────────────────────────────────────────────────────
		player.Onground = false
		player.NextPos(dt)
		if player.P.Y <= 0 {
			player.P.Y = 0
			player.Vy = 0
			player.Onground = true
		}
		if !player.Onground {
			player.Vy -= 700 * dt
		}
		if player.P.X < 0 {
			player.P.X = 0
		}
		if player.P.X >= pathLength {
			p.gameOver = true
		}

		// Start or stop the dramatic music based on big-cat activity.
		if p.musicPlayer != nil {
			anyActive := false
			for _, bc := range p.BigCats {
				if bc.active && !bc.Petted && !bc.Scared {
					anyActive = true
					break
				}
			}
			shouldPlay := anyActive && !p.gameOver
			if shouldPlay && !p.musicPlayer.IsPlaying() {
				p.musicPlayer.Play()
			} else if !shouldPlay && p.musicPlayer.IsPlaying() {
				p.musicPlayer.Pause()
			}
		}

	}
}

// groundSY returns the screen Y for drawing a sprite so that the lowest
// visible pixel in the frame lands on the ground line.
// feetY is the Y coordinate of the lowest content pixel within the source frame.
// scale is the draw scale (1.0 for regular cats, bigCatScale for big cats).
func groundSY(feetY, scale, cameraY float64) float32 {
	return float32(tdcgame.GroundY) - float32(feetY*scale) - float32(cameraY)
}

func (p *PetTheDamnCat) Draw(screen *ebiten.Image, cameraX, cameraY float64) {
	toSX := func(wx float64) float32 { return float32(wx - cameraX) }

	// Regular cats.
	for _, cat := range p.Cats {
		feetY := walkFeetY
		if cat.Petted {
			feetY = lyingFeetY
		}
		p.drawCat(screen, toSX(cat.X), groundSY(feetY, 1, cameraY), cat)
	}

	// Big cats.
	for _, bc := range p.BigCats {
		if !bc.active && !bc.Petted && !bc.Scared {
			continue
		}
		feetY := walkFeetY
		if bc.Petted {
			feetY = lyingFeetY
		}
		p.drawBigCat(screen, toSX(bc.X), groundSY(feetY, bigCatScale, cameraY), bc)
	}

	p.drawSpeedBar(screen, p.progress)

}

func (p *PetTheDamnCat) drawCat(screen *ebiten.Image, sx, sy float32, cat *Cat) {
	sheet := p.catSheets[cat.Variant]
	pettable := cat == p.nearestCat && !cat.Petted

	row, frameCount, fps := animParams(cat.Petted, cat.VX)
	img := sheet.Frame(row*sheetCols + int(cat.animElapsed*fps)%frameCount)

	if cat.Scared {
		p.drawStar(screen, sx+catSpriteW/2, sy+catSpriteH/2, catSpriteW*0.68, 10, color.RGBA{255, 40, 40, 110})
	} else if pettable {
		p.drawHeartGlow(screen, sx+catSpriteW/2, sy+catSpriteH/2, 3, p.progress)
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(sx), float64(sy))
	screen.DrawImage(img, op)

	if cat.Petted {
		p.drawHearts(screen, sx+catSpriteW/2, sy+catSpriteH*0.55, cat.PettedTimer, 1.0)
	}
}

func (p *PetTheDamnCat) drawBigCat(screen *ebiten.Image, sx, sy float32, bc *BigCat) {
	sheet := p.catSheets[bc.Variant]
	pettable := bc == p.nearestBigCat && !bc.Petted
	scaledW := float32(catSpriteW) * bigCatScale

	row, frameCount, fps := animParams(bc.Petted, bc.VX)
	img := sheet.Frame(row*sheetCols + int(bc.animElapsed*fps)%frameCount)

	if bc.Scared {
		p.drawStar(screen, sx+scaledW/2, sy+float32(catSpriteH)*bigCatScale/2, scaledW*0.68, 10, color.RGBA{255, 40, 40, 110})
	} else if pettable {
		p.drawHeartGlow(screen, sx+scaledW/2, sy+float32(catSpriteH)*bigCatScale/2, 5, p.progress)
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(bigCatScale, bigCatScale)
	op.GeoM.Translate(float64(sx), float64(sy))
	screen.DrawImage(img, op)

	if !bc.Petted {
		p.drawPatMeter(screen, sx+scaledW/2, sy-12, bc.Pats)
	}
	if bc.Petted {
		p.drawHearts(screen, sx+scaledW/2, sy+float32(catSpriteH)*bigCatScale*0.55, bc.PettedTimer, 1.4)
	}
}

// animParams returns the sprite sheet row, frame count, and FPS for a cat state.
// Note: the sprite sheet labels are inverted relative to facing direction —
// row rowWalkRight (4) shows a cat facing LEFT, rowWalkLeft (5) faces RIGHT.
func animParams(petted bool, vx float64) (row, frameCount int, fps float64) {
	switch {
	case petted:
		return rowTailWagLieRight, 3, 4
	case vx > 0:
		// Moving right → sprite that faces right = rowWalkLeft (row 5)
		return rowWalkLeft, 6, 9
	default:
		// Moving left → sprite that faces left = rowWalkRight (row 4)
		return rowWalkRight, 6, 9
	}
}

// drawHeartGlow draws a heart centred at (cx, cy) as the pet-range indicator.
// scale controls size (3 = regular cat, 5 = big cat). Color shifts yellow→red with progress.
func (p *PetTheDamnCat) drawHeartGlow(screen *ebiten.Image, cx, cy float32, scale float32, progress float64) {
	glowG := uint8(math.Max(0, 255-progress*200))
	glowB := uint8(math.Max(0, 120-progress*120))
	clr := color.RGBA{255, glowG, glowB, 90}

	s := scale * 3
	// Centre the heart bounding box on (cx, cy):
	// heart X spans [x-s/2, x+4.5s], centre at x+2s → x = cx-2s
	// heart Y spans [y-s, y+3s],     centre at y+s   → y = cy-s
	p.drawHeart(screen, cx-2*s, cy-s, scale)
	// draw a second, slightly smaller heart for a glow effect
	p.drawHeartColored(screen, cx-2*s+s*0.15, cy-s+s*0.15, scale*0.7, clr)
}

func (p *PetTheDamnCat) drawHeartColored(screen *ebiten.Image, x, y float32, scale float32, clr color.RGBA) {
	s := float32(math.Max(1, float64(scale*3)))
	vector.FillRect(screen, x, y-s, s*2, s, clr, false)
	vector.FillRect(screen, x+s*2, y-s, s*2, s, clr, false)
	vector.FillRect(screen, x-s/2, y, s*5, s, clr, false)
	vector.FillRect(screen, x, y+s, s*4, s, clr, false)
	vector.FillRect(screen, x+s, y+s*2, s*2, s, clr, false)
}

// drawPatMeter draws the three-segment pat progress bar centered at (cx, top).
func (p *PetTheDamnCat) drawPatMeter(screen *ebiten.Image, cx, top float32, pats int) {
	const segW, segH, gap = 13.0, 6.0, 3.0
	totalW := float32(bigCatPats)*segW + float32(bigCatPats-1)*gap
	x0 := cx - totalW/2
	for i := 0; i < bigCatPats; i++ {
		x := x0 + float32(i)*(segW+gap)
		vector.FillRect(screen, x, top, segW, segH, color.RGBA{50, 50, 50, 220}, false)
		if i < pats {
			vector.FillRect(screen, x, top, segW, segH, color.RGBA{255, 80, 180, 255}, false)
		}
	}
}

func (p *PetTheDamnCat) drawHearts(screen *ebiten.Image, cx, baseY float32, timer, scale float64) {
	rise := float32(timer * 16)
	p.drawHeart(screen, cx-5, baseY-rise, float32(scale))
	if timer > 0.5 {
		p.drawHeart(screen, cx+7, baseY-rise*0.65, float32(scale*0.7))
	}
	if timer > 1.0 {
		p.drawHeart(screen, cx-13, baseY-rise*0.4, float32(scale*0.5))
	}
}

func (p *PetTheDamnCat) drawHeart(screen *ebiten.Image, x, y float32, scale float32) {
	s := float32(math.Max(1, float64(scale*3)))
	clr := color.RGBA{255, 80, 130, 200}
	vector.FillRect(screen, x, y-s, s*2, s, clr, false)
	vector.FillRect(screen, x+s*2, y-s, s*2, s, clr, false)
	vector.FillRect(screen, x-s/2, y, s*5, s, clr, false)
	vector.FillRect(screen, x, y+s, s*4, s, clr, false)
	vector.FillRect(screen, x+s, y+s*2, s*2, s, clr, false)
}

// drawStar draws an n-pointed star centred at (cx, cy) with the given outer radius.
func (p *PetTheDamnCat) drawStar(screen *ebiten.Image, cx, cy, outerR float32, points int, clr color.RGBA) {
	innerR := outerR * 0.38
	var path vector.Path
	for i := 0; i < points*2; i++ {
		angle := float64(i)*math.Pi/float64(points) - math.Pi/2
		r := outerR
		if i%2 == 1 {
			r = innerR
		}
		x := cx + r*float32(math.Cos(angle))
		y := cy + r*float32(math.Sin(angle))
		if i == 0 {
			path.MoveTo(x, y)
		} else {
			path.LineTo(x, y)
		}
	}
	path.Close()
	drawOp := &vector.DrawPathOptions{}
	drawOp.ColorScale.ScaleWithColor(clr)
	vector.FillPath(screen, &path, nil, drawOp)
}


// treePositions describes background trees along the path.
// x = world position, scale = size multiplier (1.4–2.2, bigger = closer).
var treePositions = []struct{ x, scale float64 }{
	// Opening cluster
	{60, 2.0}, {130, 1.5}, {190, 2.2},
	// Scattered early
	{430, 1.8},
	{680, 2.1}, {740, 1.6},
	{920, 2.3},
	// Midgame group
	{1150, 1.7}, {1210, 2.0}, {1280, 1.5},
	{1550, 1.9},
	{1820, 1.8}, {1870, 2.2},
	{2100, 1.6},
	{2380, 2.0},
	{2650, 1.7}, {2720, 1.9},
	// Big cat region 2800
	{3050, 2.1},
	{3300, 1.6},
	{3600, 1.9}, {3660, 1.7},
	{3950, 2.3},
	{4250, 1.8},
	{4550, 1.9}, {4610, 1.5},
	{4900, 2.1},
	{5200, 1.7},
	// Big cat region 5800
	{5450, 1.9}, {5510, 1.6},
	{5900, 2.1},
	{6150, 1.5}, {6210, 1.9},
	// Cluster before sparse section
	{6500, 2.0}, {6560, 1.6}, {6620, 2.3},
	{6900, 1.8},
	{7200, 1.9}, {7270, 1.7},
	{7600, 2.2},
	{7900, 1.5}, {7960, 1.9},
	// Big cat region 8700
	{8300, 1.8},
	{8550, 2.0}, {8620, 1.6},
	{8950, 2.3},
	{9200, 1.7}, {9260, 1.9},
	// Final stretch
	{9500, 2.1},
	{9750, 1.8}, {9820, 1.5},
}

// cloudPositions describes background clouds. x = world position (large range for parallax).
// y = fraction of sky height from top (0=top, 1=horizon). w = cloud width multiplier.
var cloudPositions = []struct{ x, y, w float64 }{
	{200, 0.15, 1.0},
	{800, 0.25, 0.75},
	{1400, 0.10, 1.2},
	{2100, 0.30, 0.9},
	{2900, 0.18, 1.1},
	{3700, 0.08, 0.8},
	{4500, 0.22, 1.3},
	{5300, 0.12, 1.0},
	{6100, 0.28, 0.85},
	{7000, 0.06, 1.15},
	{7800, 0.20, 0.95},
	{8700, 0.14, 1.2},
	{9600, 0.25, 1.0},
	// Extra clouds for the far background (very slow parallax)
	{500, 0.05, 0.6},
	{1800, 0.32, 0.7},
	{3200, 0.10, 0.65},
	{5800, 0.28, 0.7},
	{8100, 0.08, 0.6},
}

// DrawBackground draws clouds and park trees into the background layer.
func (p *PetTheDamnCat) DrawBackground(screen *ebiten.Image, cameraX, cameraY float64) {
	groundY := float32(tdcgame.GroundY - tdcgame.GroundDrawOffset) // ~180
	skyH := float32(groundY)                                        // usable sky height

	// ── Clouds (parallax factor 0.2 — move at 20% of camera speed) ──────────
	const cloudParallax = 0.2
	for _, c := range cloudPositions {
		cx := float32(c.x - cameraX*cloudParallax)
		// Wrap clouds so they tile across the visible world.
		// Map to screen space and cull generously.
		if cx < -120 || cx > float32(tdcgame.ScreenW)+120 {
			continue
		}
		cy := skyH * float32(c.y)
		w := float32(c.w * 55)
		h := w * 0.45

		// Simple puffy cloud: three overlapping ellipses.
		vector.FillCircle(screen, cx, cy, h*0.9, color.RGBA{240, 245, 255, 210}, false)
		vector.FillCircle(screen, cx+w*0.28, cy+h*0.15, h*0.72, color.RGBA{240, 245, 255, 210}, false)
		vector.FillCircle(screen, cx-w*0.22, cy+h*0.2, h*0.65, color.RGBA{240, 245, 255, 200}, false)
		// Flat bottom rect to clean up the underside.
		vector.FillRect(screen, cx-w*0.55, cy, w*1.1, h*0.55, color.RGBA{240, 245, 255, 190}, false)
	}

	// ── Trees (no parallax — same layer as ground) ────────────────────────────
	for _, t := range treePositions {
		screenX := float32(t.x - cameraX)
		if screenX < -120 || screenX > float32(tdcgame.ScreenW)+120 {
			continue
		}

		s := float32(t.scale)
		trunkW := s * 9
		trunkH := s * 44
		canopyR := s * 28

		trunkX := screenX - trunkW/2
		trunkTopY := groundY - trunkH
		canopyCX := screenX
		canopyCY := trunkTopY + canopyR*0.25

		// Canopy — two overlapping circles for a rounder look.
		vector.FillCircle(screen, canopyCX, canopyCY, canopyR, color.RGBA{28, 90, 28, 255}, false)
		vector.FillCircle(screen, canopyCX-canopyR*0.25, canopyCY-canopyR*0.2, canopyR*0.6, color.RGBA{45, 120, 40, 200}, false)

		// Trunk.
		vector.FillRect(screen, trunkX, trunkTopY, trunkW, trunkH, color.RGBA{95, 60, 30, 255}, false)
	}
}

func (p *PetTheDamnCat) drawSpeedBar(screen *ebiten.Image, progress float64) {
	const barW, barH = 60.0, 6.0
	const barX, barY = float32(tdcgame.ScreenW-barW-8), float32(8)
	vector.FillRect(screen, barX, barY, barW, barH, color.RGBA{40, 40, 40, 200}, false)
	r := uint8(math.Min(255, progress*2*255))
	g := uint8(math.Min(255, (1-progress)*2*255))
	vector.FillRect(screen, barX, barY, float32(progress*barW), barH, color.RGBA{r, g, 30, 220}, false)
}


func (p *PetTheDamnCat) writeText(screen *ebiten.Image, msg string, x, y float64, size int, clr color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(screen, msg, &text.GoTextFace{Source: p.font, Size: float64(size)}, op)
}

// DrawOverlay draws text elements at full device-pixel resolution so they
// appear crisp on HiDPI screens. All viewport-space coordinates are multiplied
// by scale before drawing.
func (p *PetTheDamnCat) DrawOverlay(screen *ebiten.Image, scale, cameraX, cameraY float64) {
	toSX := func(wx float64) float64 { return (wx - cameraX) * scale }
	toSY := func(vpY float32) float64 { return float64(vpY) * scale }

	// "PET!" prompt above nearest pettable target.
	if p.nearestBigCat != nil && !p.nearestBigCat.Petted {
		sx := toSX(p.nearestBigCat.X + float64(catSpriteW)*bigCatScale/2 - 14)
		sy := toSY(groundSY(walkFeetY, bigCatScale, cameraY) - 14)
		p.writeText(screen, "PET!", sx, sy, int(9*scale), color.RGBA{255, 255, 255, 220})
	} else if p.nearestCat != nil && !p.nearestCat.Petted {
		sx := toSX(p.nearestCat.X + catSpriteW/2 - 14)
		sy := toSY(groundSY(walkFeetY, 1, cameraY) - 14)
		p.writeText(screen, "PET!", sx, sy, int(9*scale), color.RGBA{255, 255, 255, 220})
	}

	// Score popups.
	for _, pop := range p.Popups {
		sx := toSX(pop.X - 10)
		sy := (float64(tdcgame.GroundY) - pop.Y - cameraY) * scale
		alpha := uint8(255 * (pop.Timer / popupDuration))
		p.writeText(screen, fmt.Sprintf("+%d", pop.Amount), sx, sy, int(10*scale), color.RGBA{255, 230, 60, alpha})
	}

	// Speed multiplier label next to the speed bar.
	const barW, barH = 60.0, 6.0
	const barX, barY = float64(tdcgame.ScreenW-barW-8), float64(8)
	p.writeText(screen, fmt.Sprintf("×%.1f", 1+(maxSpeed/minSpeed-1)*p.progress),
		(barX-26)*scale, (barY-1)*scale, int(8*scale), color.RGBA{200, 200, 200, 200})
}
