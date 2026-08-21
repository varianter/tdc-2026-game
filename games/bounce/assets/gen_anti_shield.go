//go:build ignore

package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

func main() {
	f, err := os.Open("sokker.png")
	if err != nil {
		panic(err)
	}
	src, _, err := image.Decode(f)
	f.Close()
	if err != nil {
		panic(err)
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	dst := image.NewRGBA(bounds)

	// copy source with a red/dark tint
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, a := src.At(x, y).RGBA()
			if a == 0 {
				continue
			}
			nr := uint8(min(255, int(r>>8)*180/255+30))
			ng := uint8(int(g>>8) * 130 / 255)
			nb := uint8(int(b>>8) * 130 / 255)
			dst.SetRGBA(x, y, color.RGBA{nr, ng, nb, uint8(a >> 8)})
		}
	}

	// draw circle
	cx, cy := float64(w)/2, float64(h)/2
	radius := float64(w)/2 - 6
	thickness := 5.0
	red := color.RGBA{220, 35, 35, 230}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > radius-thickness/2 && dist < radius+thickness/2 {
				blendOver(dst, x, y, red)
			}
		}
	}

	// draw X lines
	lineThickness := 5.0
	margin := 16.0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			fx, fy := float64(x), float64(y)
			// line from top-left to bottom-right
			d1 := distToSegment(fx, fy, margin, margin, float64(w)-margin, float64(h)-margin)
			// line from top-right to bottom-left
			d2 := distToSegment(fx, fy, float64(w)-margin, margin, margin, float64(h)-margin)
			if d1 < lineThickness/2 || d2 < lineThickness/2 {
				blendOver(dst, x, y, red)
			}
		}
	}

	out, err := os.Create("anti_sokker.png")
	if err != nil {
		panic(err)
	}
	png.Encode(out, dst)
	out.Close()
}

func blendOver(dst *image.RGBA, x, y int, src color.RGBA) {
	bg := dst.RGBAAt(x, y)
	sa := float64(src.A) / 255
	da := float64(bg.A) / 255
	oa := sa + da*(1-sa)
	if oa == 0 {
		return
	}
	or := (float64(src.R)*sa + float64(bg.R)*da*(1-sa)) / oa
	og := (float64(src.G)*sa + float64(bg.G)*da*(1-sa)) / oa
	ob := (float64(src.B)*sa + float64(bg.B)*da*(1-sa)) / oa
	dst.SetRGBA(x, y, color.RGBA{uint8(or), uint8(og), uint8(ob), uint8(oa * 255)})
}

func distToSegment(px, py, x1, y1, x2, y2 float64) float64 {
	dx, dy := x2-x1, y2-y1
	lenSq := dx*dx + dy*dy
	t := ((px-x1)*dx + (py-y1)*dy) / lenSq
	t = math.Max(0, math.Min(1, t))
	cx, cy := x1+t*dx, y1+t*dy
	ddx, ddy := px-cx, py-cy
	return math.Sqrt(ddx*ddx + ddy*ddy)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
