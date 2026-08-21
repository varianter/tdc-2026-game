//go:build ignore

// gen_castle_assets.go generates the bouncy-castle themed pixel art used by the
// game: a tileable inflatable wall panel for the background, the top canopy and
// bottom bounce-mattress bands, and a background turret for the parallax layer.
//
// Run with:  go run gen_castle_assets.go
package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

const (
	tileW = 64  // all bands tile horizontally on this width
	bandH = 14  // must match bounce.wallThickness
	playH = 212 // screenH - 2*wallThickness
)

var (
	vinylRed    = color.RGBA{214, 58, 52, 255}
	vinylBlue   = color.RGBA{45, 96, 190, 255}
	vinylYellow = color.RGBA{247, 200, 46, 255}
	vinylGreen  = color.RGBA{56, 160, 74, 255}
	vinylWhite  = color.RGBA{236, 236, 240, 255}
	shadowNavy  = color.RGBA{18, 16, 30, 255}
)

func main() {
	write("castle_wall.png", genWall())
	write("castle_top.png", genTop())
	write("castle_bottom.png", genBottom())
	write("castle_turret.png", genTurret())
}

// genWall builds the background: vertical inflatable vinyl columns with a
// highlight down the left of each tube, a shaded right side, and quilted
// horizontal seams with stitching. Kept deliberately dark so the player,
// powerups and boulders stay readable on top of it.
func genWall() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, tileW, playH))
	cols := []color.RGBA{vinylRed, vinylBlue, vinylYellow, vinylGreen}
	const colW = 16

	for x := 0; x < tileW; x++ {
		c := scale(cols[(x/colW)%len(cols)], 0.24)
		switch px := x % colW; {
		case px == 0 || px == colW-1:
			c = scale(c, 0.5) // seam between tubes
		case px == 2:
			c = scale(c, 1.6) // inflated highlight
		case px == 3:
			c = scale(c, 1.25)
		case px >= colW-4:
			c = scale(c, 0.72) // rolls away into shadow
		}
		for y := 0; y < playH; y++ {
			img.SetRGBA(x, y, c)
		}
	}

	const seamEvery = 22
	for y := 0; y < playH; y += seamEvery {
		for x := 0; x < tileW; x++ {
			img.SetRGBA(x, y, scale(img.RGBAAt(x, y), 0.55))
			if y+1 < playH {
				img.SetRGBA(x, y+1, scale(img.RGBAAt(x, y+1), 1.35))
			}
		}
		// stitching along the seam
		for x := 3; x < tileW; x += 8 {
			img.SetRGBA(x, y, scale(img.RGBAAt(x, y), 2.2))
		}
	}
	return img
}

// genTop builds the canopy: a bright rail along the very top with scalloped
// bunting hanging underneath, alternating red and white.
func genTop() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, tileW, bandH))
	fillAll(img, shadowNavy)

	const railH = 4
	for x := 0; x < tileW; x++ {
		img.SetRGBA(x, 0, scale(vinylYellow, 0.55))
		img.SetRGBA(x, 1, scale(vinylYellow, 1.15))
		img.SetRGBA(x, 2, vinylYellow)
		img.SetRGBA(x, 3, scale(vinylYellow, 0.7))
	}

	const scallopW = 16
	radius := 9.0
	for x := 0; x < tileW; x++ {
		idx := x / scallopW
		cx := float64(idx*scallopW + scallopW/2)
		base := vinylRed
		if idx%2 == 1 {
			base = vinylWhite
		}
		for y := railH; y < bandH; y++ {
			dx := float64(x) - cx
			dy := float64(y-railH) + 0.5
			d := math.Sqrt(dx*dx + dy*dy)
			if d > radius {
				continue
			}
			c := base
			switch {
			case d > radius-1.5:
				c = scale(base, 0.5) // rim
			case dx < -2 && dy < 4:
				c = scale(base, 1.25) // sheen
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// genBottom builds the bounce mattress: two fat air tubes running horizontally
// with rounded shading and seam ticks where the baffles are welded.
func genBottom() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, tileW, bandH))
	fillAll(img, shadowNavy)

	tubes := []struct {
		y0, y1 int
		clr    color.RGBA
	}{
		{0, 6, vinylBlue},
		{7, bandH - 1, vinylRed},
	}

	for _, t := range tubes {
		h := float64(t.y1 - t.y0)
		for y := t.y0; y <= t.y1; y++ {
			// 0 at the top of the tube, 1 at the bottom: bright where the vinyl
			// catches the light, dark where it curves away.
			f := 1.35 - 0.75*(float64(y-t.y0)/h)
			if y == t.y0 {
				f = 0.6 // welded seam
			}
			if y == t.y1 {
				f = 0.45
			}
			for x := 0; x < tileW; x++ {
				img.SetRGBA(x, y, scale(t.clr, f))
			}
		}
		// baffle ticks
		for x := 0; x < tileW; x += 16 {
			for y := t.y0; y <= t.y1; y++ {
				img.SetRGBA(x, y, scale(img.RGBAAt(x, y), 0.6))
			}
		}
	}
	return img
}

// genTurret builds a background castle turret: striped inflatable body, an
// inflated dome on top, an arched doorway and a little pennant.
func genTurret() *image.RGBA {
	const w, h = 40, 72
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	const bodyTop = 22
	const x0, x1 = 4, 35

	// striped body
	for x := x0; x <= x1; x++ {
		c := vinylWhite
		if ((x-x0)/6)%2 == 1 {
			c = vinylBlue
		}
		if x == x0 || x == x1 {
			c = scale(c, 0.45)
		}
		for y := bodyTop; y < h; y++ {
			img.SetRGBA(x, y, c)
		}
	}

	// inflated dome
	cx, cy, r := 19.5, float64(bodyTop), 15.5
	for y := 0; y < bodyTop; y++ {
		for x := 0; x < w; x++ {
			dx, dy := float64(x)-cx, float64(y)-cy
			d := math.Sqrt(dx*dx + dy*dy)
			if d > r {
				continue
			}
			c := vinylRed
			if d > r-1.5 {
				c = scale(vinylRed, 0.45)
			} else if dx < -3 && dy > -10 {
				c = scale(vinylRed, 1.3)
			}
			img.SetRGBA(x, y, c)
		}
	}

	// arched doorway
	doorCx, doorTop := 19.5, 48.0
	for y := 40; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := float64(x) - doorCx
			if math.Abs(dx) > 7 {
				continue
			}
			if float64(y) < doorTop {
				dy := doorTop - float64(y)
				if dx*dx+dy*dy > 49 {
					continue
				}
			}
			img.SetRGBA(x, y, color.RGBA{26, 22, 38, 255})
		}
	}

	// pennant on a pole
	for y := 0; y < 8; y++ {
		img.SetRGBA(20, y, color.RGBA{80, 74, 96, 255})
	}
	for y := 0; y < 5; y++ {
		for x := 21; x < 21+(5-y); x++ {
			img.SetRGBA(x, y, vinylYellow)
		}
	}
	return img
}

func fillAll(img *image.RGBA, c color.RGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func scale(c color.RGBA, f float64) color.RGBA {
	return color.RGBA{clamp(float64(c.R) * f), clamp(float64(c.G) * f), clamp(float64(c.B) * f), c.A}
}

func clamp(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func write(name string, img image.Image) {
	f, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}
