package flappyguy

import (
	"embed"

	"variant.dev/tdcgame/tdcgame"
)

//go:embed assets/audio/coincollect.ogg
//go:embed assets/audio/scream.ogg
//go:embed assets/audio/wingflap.ogg
var audioAssets embed.FS

const (
	soundCoinCollect = "coincollect"
	soundScream      = "scream"
	soundWingFlap    = "wingflap"
)

func loadAudio() *tdcgame.Audio {
	return tdcgame.LoadAudio(audioAssets, soundCoinCollect, soundScream, soundWingFlap)
}
