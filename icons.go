package main

import (
	"image/color"

	"variant.dev/tdcgame/tdcgame"
)

// backIcon is a "return" arrow (like ⏎), drawn tinted to match the text
// around it.
var backIcon = &tdcgame.PixelArt{Rows: []string{
	".......x",
	"..x....x",
	".xx....x",
	"xxxxxxxx",
	".xx.....",
	"..x.....",
}}

// gameIconPalette colors the launcher's per-game icons.
var gameIconPalette = map[byte]color.RGBA{
	'r': {214, 58, 52, 255},   // red
	'y': {255, 212, 47, 255},  // yellow
	'b': {60, 140, 230, 255},  // blue
	'd': {60, 80, 160, 255},   // denim
	'o': {235, 130, 40, 255},  // ginger
	'p': {255, 150, 170, 255}, // pink
	'e': {110, 210, 90, 255},  // cat-eye green
	'k': {20, 14, 30, 255},    // near-black
	'w': {236, 240, 255, 255}, // white
	's': {255, 204, 150, 255}, // skin
	'h': {120, 70, 35, 255},   // hair
	'n': {170, 110, 60, 255},  // wood
	'N': {110, 65, 30, 255},   // wood shadow
	'g': {160, 160, 175, 255}, // steel
	'G': {105, 105, 120, 255}, // steel shadow
	'c': {92, 52, 30, 255},    // coffee
	'C': {196, 140, 80, 255},  // crema
}

// Launcher icons, one per game, each listIconSize pixels square.
var (
	bouncyCastleIcon = &tdcgame.PixelArt{Palette: gameIconPalette, Rows: []string{
		".y........y.",
		"yyy......yyy",
		"rrry.yy.yrrr",
		"rrryyyyyyrrr",
		"rrrrrrrrrrrr",
		"rrrrrrrrrrrr",
		"rrrrbbbbrrrr",
		"rrrbbbbbbrrr",
		"rrrbbbbbbrrr",
		"rrrbbbbbbrrr",
		"yyyyyyyyyyyy",
		"yyyyyyyyyyyy",
	}}
	flappyGuyIcon = &tdcgame.PixelArt{Palette: gameIconPalette, Rows: []string{
		"....hhhh....",
		"....ssss....",
		"....sksk....",
		"....ssss....",
		"ww..bbbb..ww",
		"www.bbbb.www",
		".wwwbbbbwww.",
		"..wwbbbbww..",
		"....bbbb....",
		"....dddd....",
		"....d..d....",
		"....d..d....",
	}}
	catIcon = &tdcgame.PixelArt{Palette: gameIconPalette, Rows: []string{
		"o..........o",
		"oo........oo",
		"opo......opo",
		"oooooooooooo",
		"oooooooooooo",
		"ooeeooooeeoo",
		"ooekooooekoo",
		"oooooppooooo",
		"woooowwoooow",
		".oooooooooo.",
		"..oooooooo..",
		"............",
	}}
	hammerIcon = &tdcgame.PixelArt{Palette: gameIconPalette, Rows: []string{
		"y..........y",
		".gggggggggg.",
		".gwwwwwwwwg.",
		".gggggggggg.",
		".GGGGGGGGGG.",
		".....nN.....",
		".....nN.....",
		".....nN.....",
		".....nN.....",
		".....nN.....",
		".....nN.....",
		".....NN.....",
	}}
	coffeeIcon = &tdcgame.PixelArt{Palette: gameIconPalette, Rows: []string{
		"...w..w.....",
		"....w..w....",
		"...w..w.....",
		"............",
		"wwwwwwwwww..",
		"wCCCCCCCCwww",
		"wccccccccw.w",
		"wccccccccw.w",
		".wccccccwwww",
		".wwwwwwww...",
		"..wwwwww....",
		"nnnnnnnnnnnn",
	}}
)
