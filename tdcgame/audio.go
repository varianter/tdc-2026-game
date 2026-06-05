package tdcgame

import (
	"bytes"
	"embed"
	"log"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

const audioSampleRate = 44100

var (
	audioContext     *audio.Context
	audioContextOnce sync.Once
)

func getAudioContext() *audio.Context {
	audioContextOnce.Do(func() {
		audioContext = audio.NewContext(audioSampleRate)
	})
	return audioContext
}

type Audio struct {
	clips map[string][]byte
}

func LoadAudio(assets embed.FS, names ...string) *Audio {
	a := &Audio{clips: make(map[string][]byte)}
	for _, name := range names {
		data, err := assets.ReadFile("assets/audio/" + name + ".ogg")
		if err != nil {
			log.Printf("audio: failed to load %s: %v", name, err)
			continue
		}
		a.clips[name] = data
	}
	return a
}

func (a *Audio) Play(name string) {
	if a == nil {
		return
	}
	data, ok := a.clips[name]
	if !ok {
		return
	}

	stream, err := vorbis.DecodeF32(bytes.NewReader(data))
	if err != nil {
		log.Printf("audio: decode %s: %v", name, err)
		return
	}

	player, err := getAudioContext().NewPlayerF32(stream)
	if err != nil {
		log.Printf("audio: player %s: %v", name, err)
		return
	}
	player.Play()
}
