package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// roundedRectPath builds a rounded-rectangle outline path for the rect at
// (x,y) with size (w,h) and corner radius r, using the same MoveTo/LineTo/
// ArcTo tangent-arc construction as HTML5 canvas's roundRect. Radius is
// clamped so it never exceeds half the shorter side (r == h/2 gives a pill).
func roundedRectPath(x, y, w, h, r float32) *vector.Path {
	if maxR := w / 2; r > maxR {
		r = maxR
	}
	if maxR := h / 2; r > maxR {
		r = maxR
	}

	var p vector.Path
	p.MoveTo(x+r, y)
	p.LineTo(x+w-r, y)
	p.ArcTo(x+w, y, x+w, y+r, r)
	p.LineTo(x+w, y+h-r)
	p.ArcTo(x+w, y+h, x+w-r, y+h, r)
	p.LineTo(x+r, y+h)
	p.ArcTo(x, y+h, x, y+h-r, r)
	p.LineTo(x, y+r)
	p.ArcTo(x, y, x+r, y, r)
	p.Close()
	return &p
}

// fillRoundedRect fills a rounded rectangle. Pass r == h/2 (or w/2, whichever
// is smaller) for a pill shape.
func fillRoundedRect(dst *ebiten.Image, x, y, w, h, r float32, clr color.Color) {
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(clr)
	vector.FillPath(dst, roundedRectPath(x, y, w, h, r), nil, op)
}

// strokeRoundedRect outlines a rounded rectangle with the given stroke width.
func strokeRoundedRect(dst *ebiten.Image, x, y, w, h, r, strokeWidth float32, clr color.Color) {
	strokeOp := &vector.StrokeOptions{Width: strokeWidth, LineJoin: vector.LineJoinRound}
	drawOp := &vector.DrawPathOptions{AntiAlias: true}
	drawOp.ColorScale.ScaleWithColor(clr)
	vector.StrokePath(dst, roundedRectPath(x, y, w, h, r), strokeOp, drawOp)
}

// withAlpha returns c with its alpha replaced, for the many translucent
// glow/halo passes the neon look layers on top of each other.
func withAlpha(c color.RGBA, a uint8) color.RGBA {
	c.A = a
	return c
}

// strokeNeonRect outlines a rounded rectangle as a glowing neon tube: a wide,
// translucent halo of the same hue underneath a narrow bright core. scale is the
// device-pixel scale so the tube keeps a constant apparent thickness at any
// window size.
func strokeNeonRect(dst *ebiten.Image, x, y, w, h, r float32, clr color.RGBA, scale float32) {
	strokeRoundedRect(dst, x, y, w, h, r, 4*scale, withAlpha(clr, 70))
	strokeRoundedRect(dst, x, y, w, h, r, 1.3*scale, clr)
}
