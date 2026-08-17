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

// roundedTopRectPath builds a rectangle path with only its top two corners
// rounded and square bottom corners — a header cap that seats flush against
// a body drawn below it.
func roundedTopRectPath(x, y, w, h, r float32) *vector.Path {
	if maxR := w / 2; r > maxR {
		r = maxR
	}
	if r > h {
		r = h
	}

	var p vector.Path
	p.MoveTo(x+r, y)
	p.LineTo(x+w-r, y)
	p.ArcTo(x+w, y, x+w, y+r, r)
	p.LineTo(x+w, y+h)
	p.LineTo(x, y+h)
	p.LineTo(x, y+r)
	p.ArcTo(x, y, x+r, y, r)
	p.Close()
	return &p
}

// fillRoundedTopRect fills a top-rounded rectangle (see roundedTopRectPath).
func fillRoundedTopRect(dst *ebiten.Image, x, y, w, h, r float32, clr color.Color) {
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(clr)
	vector.FillPath(dst, roundedTopRectPath(x, y, w, h, r), nil, op)
}
