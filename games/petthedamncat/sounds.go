package petthedamncat

import (
	"embed"
	"io"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"variant.dev/tdcgame/tdcgame"
)

const sampleRate = 44100

var (
	audioOnce          sync.Once
	audioCtx           *audio.Context
	meowPCM            [3][]byte // per-variant regular meow
	alertPCM           []byte    // alert.mp3 — played when cats are scared away
	scaredPCM          []byte    // scared.mp3 — regular cat scare
	scaredBigPCM       []byte    // scaredBig.mp3 — big cat scare
	bigCatMeowPCM      [3][]byte // per-variant mid-pat big cat meow
	bigCatFinalMeowPCM [3][]byte // per-variant final-pat big cat meow
)

// initAudio pre-loads / pre-synthesises all one-shot sound effects.
// Uses the shared tdcgame audio context to avoid creating a second one.
// Safe to call multiple times; only runs once.
func initAudio(assets embed.FS) {
	audioOnce.Do(func() {
		audioCtx = tdcgame.GetAudioContext()

		// Load raw OGG once; derive all per-variant sounds from it.
		raw := loadOGG(assets, "assets/cats/Meow.ogg")
		if raw != nil {
			// Regular cats — subtle pitch/speed personality per variant.
			meowPCM[0] = raw                           // normal
			meowPCM[1] = pitchShift(raw, 1.12, 1.0)   // higher, punchier
			meowPCM[2] = pitchShift(raw, 0.90, 0.95)  // a touch lower and softer

			// Big cat mid-pat — noticeably deeper, each variant distinct.
			bigCatMeowPCM[0] = pitchShift(raw, 0.62, 1.3)  // deep
			bigCatMeowPCM[1] = pitchShift(raw, 0.55, 1.4)  // deeper
			bigCatMeowPCM[2] = pitchShift(raw, 0.70, 1.2)  // deep-ish, slightly shorter

			// Big cat final pat — even lower for impact.
			bigCatFinalMeowPCM[0] = pitchShift(raw, 0.48, 1.6)
			bigCatFinalMeowPCM[1] = pitchShift(raw, 0.42, 1.7)  // most imposing
			bigCatFinalMeowPCM[2] = pitchShift(raw, 0.54, 1.5)
		}

		alertPCM = loadMP3(assets, "assets/cats/alert.mp3")
		scaredPCM = loadMP3(assets, "assets/cats/scared.mp3")
		scaredBigPCM = loadMP3(assets, "assets/cats/scaredBig.mp3")
	})
}

// newMusicPlayer decodes radioactive.wav and returns an infinitely-looping
// player ready to be started/paused by the game. Returns nil on any error.
func newMusicPlayer(assets embed.FS) *audio.Player {
	if audioCtx == nil {
		return nil
	}
	f, err := assets.Open("assets/cats/radioactive.wav")
	if err != nil {
		return nil
	}
	stream, err := wav.DecodeWithSampleRate(sampleRate, f)
	if err != nil {
		return nil
	}
	loop := audio.NewInfiniteLoop(stream, stream.Length())
	p, err := audioCtx.NewPlayer(loop)
	if err != nil {
		return nil
	}
	return p
}

// playSound fires a one-shot sound. The audio context keeps the player alive
// until playback finishes; discarding the pointer is safe.
func playSound(pcm []byte) {
	if audioCtx == nil || len(pcm) == 0 {
		return
	}
	audioCtx.NewPlayerFromBytes(pcm).Play()
}

// loadMP3 decodes an MP3 and returns stereo-interleaved 16-bit PCM.
func loadMP3(assets embed.FS, path string) []byte {
	f, err := assets.Open(path)
	if err != nil {
		return nil
	}
	stream, err := mp3.DecodeWithSampleRate(sampleRate, f)
	if err != nil {
		return nil
	}
	data, err := io.ReadAll(stream)
	if err != nil || len(data) == 0 {
		return nil
	}
	return data
}

// loadOGG decodes a Vorbis OGG and returns stereo-interleaved 16-bit PCM.
// ebiten's vorbis decoder always outputs stereo, so we read it directly.
func loadOGG(assets embed.FS, path string) []byte {
	f, err := assets.Open(path)
	if err != nil {
		return nil
	}
	stream, err := vorbis.DecodeWithSampleRate(sampleRate, f)
	if err != nil {
		return nil
	}
	data, err := io.ReadAll(stream)
	if err != nil || len(data) == 0 {
		return nil
	}
	return data
}

// pitchShift lowers the pitch of stereo 16-bit PCM by stretching it.
// ratio < 1 → lower pitch (e.g. 0.62 = ~38% lower); gain > 1 → amplify.
func pitchShift(src []byte, ratio, gain float64) []byte {
	srcN := len(src) / 4
	dstN := int(float64(srcN) / ratio)
	dst := make([]byte, dstN*4)

	read := func(idx int) (int16, int16) {
		if idx >= srcN {
			return 0, 0
		}
		l := int16(uint16(src[idx*4+0]) | uint16(src[idx*4+1])<<8)
		r := int16(uint16(src[idx*4+2]) | uint16(src[idx*4+3])<<8)
		return l, r
	}
	clamp := func(v float64) int16 {
		if v > 32767 {
			return 32767
		}
		if v < -32768 {
			return -32768
		}
		return int16(v)
	}

	for i := 0; i < dstN; i++ {
		srcPos := float64(i) * ratio
		idx := int(srcPos)
		frac := srcPos - float64(idx)
		l0, r0 := read(idx)
		l1, r1 := read(idx + 1)
		l := clamp((float64(l0) + frac*(float64(l1)-float64(l0))) * gain)
		r := clamp((float64(r0) + frac*(float64(r1)-float64(r0))) * gain)
		dst[i*4+0] = byte(uint16(l))
		dst[i*4+1] = byte(uint16(l) >> 8)
		dst[i*4+2] = byte(uint16(r))
		dst[i*4+3] = byte(uint16(r) >> 8)
	}
	return dst
}

