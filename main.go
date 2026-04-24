package main

import (
	"embed"
	"image"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

//go:embed assets/chick.png
//go:embed assets/ground.png
var assets embed.FS

type Canvas struct {
	screen *ebiten.Image
}

func (c *Canvas) DrawImage(img *ebiten.Image, x, y float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	c.screen.DrawImage(img, op)
}

type Assets struct {
	Sprites     map[string]*ebiten.Image
	Backgrounds map[string]*ebiten.Image
}

func LoadAssets() *Assets {
	a := &Assets{
		Sprites:     make(map[string]*ebiten.Image),
		Backgrounds: make(map[string]*ebiten.Image),
	}
	a.Sprites["chick"], _, _ = ebitenutil.NewImageFromFileSystem(assets, "assets/chick.png")
	a.Sprites["ground"], _, _ = ebitenutil.NewImageFromFileSystem(assets, "assets/ground.png")
	return a
}

type SpriteSheet struct {
	Image   *ebiten.Image
	FrameW  int
	FrameH  int
	Columns int
}

func LoadSpriteSheet(assets *Assets, frameW, frameH int) *SpriteSheet {
	img := assets.Sprites["chick"]
	w, _ := img.Bounds().Dx(), img.Bounds().Dy()
	return &SpriteSheet{
		Image:   img,
		FrameW:  frameW,
		FrameH:  frameH,
		Columns: w / frameW,
	}
}

func (s *SpriteSheet) Frame(index int) *ebiten.Image {
	col := index % s.Columns
	row := index / s.Columns
	x := col * s.FrameW
	y := row * s.FrameH
	return s.Image.SubImage(image.Rect(x, y, x+s.FrameW, y+s.FrameH)).(*ebiten.Image)
}

type Animation struct {
	Sheet      *SpriteSheet
	StartFrame int // first frame of this animation row
	FrameCount int // how many frames in the animation
	FPS        float64
	elapsed    float64
}

func (a *Animation) Update(dt float64) {
	a.elapsed += dt
}

func (a *Animation) CurrentFrame() *ebiten.Image {
	frame := int(a.elapsed*a.FPS) % a.FrameCount
	return a.Sheet.Frame(a.StartFrame + frame)
}

type Chick struct {
	sheet         *SpriteSheet
	walkRightAnim *Animation
	walkLeftAnim  *Animation
	idleRightAnim *Animation
	idleLeftAnim  *Animation
	current       *Animation
	orientation   rune
	x, y          float64
	vx, vy        float64
	onGround      bool
}

func (c *Chick) switchAnim(anim *Animation) {
	if c.current != anim {
		anim.elapsed = 0
		c.current = anim
	}
}

func (c *Chick) Update(dt float64) error {
	movementScale := 1.0
	if !c.onGround {
		movementScale = AirControl
	}

	// Calculate velocity
	c.vx = 0
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		c.vx = WalkSpeed * movementScale
		c.switchAnim(c.walkRightAnim)
		c.orientation = 'r'
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		c.vx = -WalkSpeed * movementScale
		c.switchAnim(c.walkLeftAnim) // TODO: Flying animation if not on ground
		c.orientation = 'l'
	}
	if c.onGround && (ebiten.IsKeyPressed(ebiten.KeySpace) || ebiten.IsKeyPressed(ebiten.KeyUp)) {
		c.vy = JumpForce
		c.onGround = false
	}
	if c.vx == 0 {
		if c.orientation == 'l' {
			c.switchAnim(c.idleLeftAnim)
		} else {
			c.switchAnim(c.idleRightAnim)
		}
	}

	// Gravity
	if !c.onGround {
		c.vy += Gravity * dt
	}

	// Update position based on velocity
	c.x += c.vx * dt
	c.y += c.vy * dt

	// Bodies hit the floor
	if c.y+float64(c.sheet.FrameH) >= GroundY {
		c.y = GroundY - float64(c.sheet.FrameH)
		c.vy = 0
		c.onGround = true
	}

	c.current.Update(dt)
	return nil
}

func (c *Chick) Draw(canvas *Canvas) {
	frame := c.current.CurrentFrame()
	canvas.DrawImage(frame, c.x, c.y)
}

type Game struct {
	chick  *Chick
	camera Camera
	assets *Assets
}

func (g *Game) Update() error {
	dt := 1.0 / float64(ebiten.TPS()) // ~0.0166 at 60 TPS
	err := g.chick.Update(dt)
	if err != nil {
		return err
	}
	g.camera.Follow(g.chick, ScreenH, ScreenW)
	return nil
}

func (c *Canvas) Rect(x, y, w, h float32, clr color.Color) {
	vector.FillRect(c.screen, x, y, w, h, clr, false)
}

func (c *Canvas) TilingGround(img *ebiten.Image, cameraX, cameraY, groundY float64, worldWidth float64) {
	w := float64(img.Bounds().Dx())
	startX := math.Floor(cameraX/w) * w

	for x := startX; x < cameraX+ScreenW; x += w {
		c.DrawImage(img, x-cameraX, (groundY-GroundDrawOffset)-cameraY)
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{30, 30, 30, 255})

	c := &Canvas{
		screen: screen,
	}
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()

	c.Rect(0, 0, float32(w), float32((h)), color.RGBA{135, 206, 235, 255}) // light blue

	c.TilingGround(g.assets.Sprites["ground"], g.camera.x, g.camera.y, GroundY, 5000)

	frame := g.chick.current.CurrentFrame()
	c.DrawImage(frame, g.chick.x-g.camera.x, g.chick.y-g.camera.y)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenW, ScreenH
}

type Camera struct {
	x, y float64
}

func (cam *Camera) Follow(player *Chick, screenW, screenH int) {
	cam.x = player.x - float64(screenW)/2 + float64(player.sheet.FrameW)/2
	cam.y = player.y + 60 - float64(screenH)/2

	maxCamY := GroundY - 190.0 // HACK: This is dumb but whatever
	if cam.y > maxCamY {
		cam.y = maxCamY
	}
}

const (
	WalkSpeed  = 100.0 // px/sec
	JumpSpeed  = 60
	Gravity    = 600  // px/sec^2
	JumpForce  = -250 // negative means up
	GroundY    = 148  // 0,0 is top left
	AirControl = 0.5

	AnimIdleFPS      = 5.0
	AnimWalkFPS      = 12.0
	AnimRunFPS       = 14.0
	ScreenW          = 426
	ScreenH          = 240
	GroundDrawOffset = 6.0
)

func main() {
	assets := LoadAssets()
	sheet := LoadSpriteSheet(assets, 32, 32)

	chick := &Chick{
		sheet: sheet,
		walkRightAnim: &Animation{
			Sheet:      sheet,
			StartFrame: 0,
			FrameCount: 6,
			FPS:        AnimWalkFPS,
		},
		idleRightAnim: &Animation{
			Sheet:      sheet,
			StartFrame: 6,
			FrameCount: 6,
			FPS:        10,
		},
		walkLeftAnim: &Animation{
			Sheet:      sheet,
			StartFrame: 12,
			FrameCount: 6,
			FPS:        AnimWalkFPS,
		},
		idleLeftAnim: &Animation{
			Sheet:      sheet,
			StartFrame: 18,
			FrameCount: 6,
			FPS:        10,
		},
		x: ScreenW / 2, y: GroundY,
		orientation: 'r',
	}
	chick.current = chick.idleRightAnim
	game := &Game{
		chick:  chick,
		assets: assets,
	}
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSize(426*4, 240*4)
	ebiten.SetWindowTitle("GOTY2026")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
