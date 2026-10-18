package busybarista

import (
	"bytes"
	"io"
	"log"
	"math"
	"math/rand"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"variant.dev/tdcgame/games/busybarista/assets"
	"variant.dev/tdcgame/tdcgame"
)

const sampleRate = 44100

var (
	soundsOnce  sync.Once
	audioCtx    *audio.Context
	lidPCM      []byte
	shotPCM     []byte
	spillPCM    []byte
	missPCM     []byte
	levelUpPCM  []byte
	gameOverPCM []byte
)

// initSounds decodes the effects to raw PCM once, up front, so playing one
// mid-game never stalls on a decode.
func initSounds() {
	soundsOnce.Do(func() {
		audioCtx = tdcgame.GetAudioContext()
		lidPCM = decodeWAV(assets.LidSound)
		shotPCM = decodeWAV(assets.ShotSound)
		spillPCM = decodeWAV(assets.SpillSound)
		missPCM = decodeWAV(assets.MissSound)
		levelUpPCM = decodeWAV(assets.LevelUpSound)
		gameOverPCM = decodeWAV(assets.GameOverSound)
	})
}

func decodeWAV(data []byte) []byte {
	stream, err := wav.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
	if err != nil {
		log.Printf("busybarista audio: decode: %v", err)
		return nil
	}
	pcm, err := io.ReadAll(stream)
	if err != nil {
		log.Printf("busybarista audio: read: %v", err)
		return nil
	}
	return pcm
}

func play(pcm []byte) {
	if audioCtx == nil || len(pcm) == 0 {
		return
	}
	audioCtx.NewPlayerFromBytes(pcm).Play()
}

// newPour synthesises one pour, sized to this cup: noise through a resonance
// that rises as the cup fills (the "filling a bottle" sound), reaching
// lineFreq exactly when the coffee hits the line after fillDur seconds. The
// clip runs on past the line so overshooting keeps sounding. The returned
// player is started; the caller stops it on release.
func newPour(fillDur float64) *audio.Player {
	if audioCtx == nil {
		return nil
	}
	const startFreq, lineFreq = 320.0, 1300.0
	total := fillDur + holdWindows[2] + 0.1
	n := int(total * sampleRate)
	pcm := make([]byte, n*4)
	rng := rand.New(rand.NewSource(int64(fillDur * 1e6)))

	var x1, x2, y1, y2, b0, b2, a1, a2 float64
	for i := range n {
		t := float64(i) / sampleRate
		if i%64 == 0 {
			// Retune the bandpass along an exponential sweep, so each step of
			// fill sounds like the same musical interval.
			f := startFreq * math.Pow(lineFreq/startFreq, t/fillDur)
			w := 2 * math.Pi * math.Min(f, 6000) / sampleRate
			alpha := math.Sin(w) / (2 * 4.0)
			a0 := 1 + alpha
			b0, b2, a1, a2 = alpha/a0, -alpha/a0, -2*math.Cos(w)/a0, (1-alpha)/a0
		}
		x := rng.Float64()*2 - 1
		y := b0*x + b2*x2 - a1*y1 - a2*y2
		x2, x1 = x1, x
		y2, y1 = y1, y
		// Gurgle: the stream's loudness wobbles irregularly.
		gurgle := 0.75 + 0.25*math.Sin(2*math.Pi*17*t+3*math.Sin(2*math.Pi*2.3*t))
		env := math.Min(1, t/0.03)
		putSample(pcm, i, 2.2*y*gurgle*env)
	}
	p := audioCtx.NewPlayerFromBytes(pcm)
	p.Play()
	return p
}

func putSample(pcm []byte, i int, v float64) {
	s := uint16(int16(math.Max(-32768, math.Min(32767, v*32767))))
	pcm[i*4], pcm[i*4+1] = byte(s), byte(s>>8)
	pcm[i*4+2], pcm[i*4+3] = byte(s), byte(s>>8)
}
