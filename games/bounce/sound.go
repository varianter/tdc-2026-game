package bounce

import (
	"bytes"
	"io"
	"log"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"variant.dev/tdcgame/games/bounce/assets"
	"variant.dev/tdcgame/tdcgame"
)

const sampleRate = 44100

var (
	soundsOnce    sync.Once
	audioCtx      *audio.Context
	bouncePCM     []byte
	crushPCM      []byte
	powerupPCM    []byte
	antiShieldPCM []byte
)

// initSounds decodes the effects to raw PCM once, up front, so playing one
// mid-game never stalls on a decode. It shares tdcgame's audio context rather
// than creating a second one.
func initSounds() {
	soundsOnce.Do(func() {
		audioCtx = tdcgame.GetAudioContext()
		bouncePCM = decodeWAV(assets.BounceSound)
		crushPCM = decodeWAV(assets.CrushSound)
		powerupPCM = decodeWAV(assets.PowerupSound)
		antiShieldPCM = decodeWAV(assets.AntiShieldSound)
	})
}

func decodeWAV(data []byte) []byte {
	stream, err := wav.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
	if err != nil {
		log.Printf("bounce audio: decode: %v", err)
		return nil
	}
	pcm, err := io.ReadAll(stream)
	if err != nil {
		log.Printf("bounce audio: read: %v", err)
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

// Bounce playback rate limits: a slow drift into the wall is a soft, high
// little tap; slamming into it at full pelt is a deep, loud whump. Rates below
// 1 stretch the clip, which drops its pitch.
const (
	bounceRateSlow = 1.25
	bounceRateFast = 0.72
)

// onBounce plays the trampoline hit, pitched by impact speed so a hard landing
// sounds like a big one.
func (g *Game) onBounce() {
	if len(bouncePCM) == 0 {
		return
	}
	// How hard this hit was, 0 (opening bounce) .. 1 (top speed late in a run).
	// Dashing into the wall counts as harder still, so a lunge into the canopy
	// lands with more weight than drifting into it.
	hardness := (math.Abs(g.playerVY) - bounceSpeedBase) / (bounceSpeedMax - bounceSpeedBase)
	hardness += g.playerVX / dashMaxSpeed * 0.35
	hardness = math.Max(0, math.Min(1, hardness))

	rate := bounceRateSlow + (bounceRateFast-bounceRateSlow)*hardness
	gain := 0.55 + 0.45*hardness
	play(resample(bouncePCM, rate, gain))
}

func (g *Game) playSmash()      { play(crushPCM) }
func (g *Game) playPickup()     { play(powerupPCM) }
func (g *Game) playAntiShield() { play(antiShieldPCM) }

// resample stretches or squeezes stereo 16-bit PCM, which shifts its pitch:
// rate < 1 plays back slower and lower, rate > 1 faster and higher. gain
// scales the amplitude. Linear interpolation is plenty for short one-shots.
func resample(src []byte, rate, gain float64) []byte {
	srcN := len(src) / 4
	if srcN == 0 || rate <= 0 {
		return src
	}
	dstN := int(float64(srcN) / rate)
	dst := make([]byte, dstN*4)

	sample := func(idx, ch int) float64 {
		if idx >= srcN {
			return 0
		}
		o := idx*4 + ch*2
		return float64(int16(uint16(src[o]) | uint16(src[o+1])<<8))
	}
	clamp := func(v float64) uint16 {
		if v > 32767 {
			v = 32767
		} else if v < -32768 {
			v = -32768
		}
		return uint16(int16(v))
	}

	for i := 0; i < dstN; i++ {
		pos := float64(i) * rate
		idx := int(pos)
		frac := pos - float64(idx)
		for ch := 0; ch < 2; ch++ {
			a := sample(idx, ch)
			b := sample(idx+1, ch)
			v := clamp((a + frac*(b-a)) * gain)
			o := i*4 + ch*2
			dst[o] = byte(v)
			dst[o+1] = byte(v >> 8)
		}
	}
	return dst
}
