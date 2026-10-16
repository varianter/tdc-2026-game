//go:build ignore

// gen_barista.go re-dresses the TDC character's idle animation as a barista:
// the red sweater becomes a black tee, a brown apron hangs from the neck over
// the torso and the hair is tucked under a beanie. The conference badge stays
// on, pinned to the apron.
//
// Only the idle row (10 frames of 64×64) is written, since the barista never
// walks. The output is a plain PNG that can be touched up in Aseprite.
//
// Run with:  go run gen_barista.go
package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

const (
	frame     = 64
	idleRow   = 2
	idleCount = 10
)

// Source palette of tdcgjenger.png.
var (
	outline     = color.RGBA{9, 10, 20, 255}
	sweater     = color.RGBA{207, 87, 60, 255}
	sweaterShad = color.RGBA{165, 48, 48, 255}
	sweaterDark = color.RGBA{117, 36, 56, 255}
	hair        = color.RGBA{222, 158, 65, 255}
	hairShad    = color.RGBA{190, 119, 43, 255}
	badge       = color.RGBA{168, 202, 88, 255}
)

var (
	tee       = color.RGBA{52, 50, 60, 255}
	teeShad   = color.RGBA{36, 34, 44, 255}
	teeDark   = color.RGBA{26, 24, 34, 255}
	apron     = color.RGBA{132, 84, 48, 255}
	apronShad = color.RGBA{100, 62, 36, 255}
	apronTie  = color.RGBA{232, 220, 196, 255}
	beanie    = color.RGBA{58, 128, 96, 255}
	beanieRib = color.RGBA{40, 98, 72, 255}
)

func main() {
	src := load("../../../assets/tdcgjenger.png")
	out := image.NewRGBA(image.Rect(0, 0, frame*idleCount, frame))
	for i := range idleCount {
		f := image.NewRGBA(image.Rect(0, 0, frame, frame))
		for y := range frame {
			for x := range frame {
				f.Set(x, y, src.At(i*frame+x, idleRow*frame+y))
			}
		}
		dress(f)
		for y := range frame {
			for x := range frame {
				out.Set(i*frame+x, y, f.At(x, y))
			}
		}
	}
	save("barista.png", out)
}

func dress(f *image.RGBA) {
	// Locate the torso from the badge, which hangs at the chest centre.
	cx, badgeTop := badgeAnchor(f)
	hairTop := firstRowOf(f, hair, hairShad)

	// Sweater → black tee.
	recolor(f, map[color.RGBA]color.RGBA{sweater: tee, sweaterShad: teeShad, sweaterDark: teeDark})

	// Apron: a bib from just above the badge down past the waist, 9px wide,
	// painted over anything that isn't outline or transparent.
	top, bottom := badgeTop-2, badgeTop+22
	for y := top; y <= bottom; y++ {
		half := 4
		if y < badgeTop+2 {
			half = 3 // narrower bib at the chest
		}
		for x := cx - half; x <= cx+half; x++ {
			c := f.RGBAAt(x, y)
			if c.A == 0 || c == outline || c == badge {
				continue
			}
			shade := apron
			if x == cx+half || y == bottom {
				shade = apronShad
			}
			f.SetRGBA(x, y, shade)
		}
	}
	// Neck straps up to the collar and a waist tie.
	for y := top - 3; y < top; y++ {
		for _, x := range []int{cx - 3, cx + 3} {
			if c := f.RGBAAt(x, y); c.A != 0 && c != outline {
				f.SetRGBA(x, y, apronShad)
			}
		}
	}
	waist := badgeTop + 12
	for x := cx - 6; x <= cx+6; x++ {
		if c := f.RGBAAt(x, waist); c.A != 0 && c != outline && (x < cx-4 || x > cx+4) {
			f.SetRGBA(x, waist, apronTie)
		}
	}

	// Beanie: the top four rows of hair, with a ribbed fold on the last one,
	// plus a row poking out above the head.
	for y := hairTop; y < hairTop+4; y++ {
		for x := range frame {
			c := f.RGBAAt(x, y)
			if c == hair || c == hairShad {
				if y == hairTop+3 {
					f.SetRGBA(x, y, beanieRib)
				} else {
					f.SetRGBA(x, y, beanie)
				}
			}
		}
	}
	for x := cx - 2; x <= cx+2; x++ {
		f.SetRGBA(x, hairTop-2, outline)
		f.SetRGBA(x, hairTop-1, beanie)
	}
	f.SetRGBA(cx-3, hairTop-1, outline)
	f.SetRGBA(cx+3, hairTop-1, outline)
}

// badgeAnchor returns the badge's centre column and top row.
func badgeAnchor(f *image.RGBA) (cx, top int) {
	minX, maxX, top := frame, 0, frame
	for y := range frame {
		for x := range frame {
			if f.RGBAAt(x, y) == badge {
				minX, maxX, top = min(minX, x), max(maxX, x), min(top, y)
			}
		}
	}
	return (minX + maxX) / 2, top
}

func firstRowOf(f *image.RGBA, cs ...color.RGBA) int {
	for y := range frame {
		for x := range frame {
			for _, c := range cs {
				if f.RGBAAt(x, y) == c {
					return y
				}
			}
		}
	}
	return 0
}

func recolor(f *image.RGBA, m map[color.RGBA]color.RGBA) {
	for y := range frame {
		for x := range frame {
			if to, ok := m[f.RGBAAt(x, y)]; ok {
				f.SetRGBA(x, y, to)
			}
		}
	}
}

func load(path string) image.Image {
	r, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer r.Close()
	img, err := png.Decode(r)
	if err != nil {
		panic(err)
	}
	return img
}

func save(name string, img image.Image) {
	w, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	defer w.Close()
	if err := png.Encode(w, img); err != nil {
		panic(err)
	}
}
