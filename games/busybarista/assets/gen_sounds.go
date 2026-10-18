//go:build ignore

// gen_sounds.go synthesises the Busy Barista sound effects as 16-bit stereo WAV
// files at 44100 Hz. The pour itself is not here: it is synthesised at runtime
// so its pitch sweep matches each cup's fill time.
//
// Run with:  go run gen_sounds.go
package main

import (
	"encoding/binary"
	"math"
	"math/rand"
	"os"
)

const sampleRate = 44100

// Fixed seed so regenerating reproduces the same bytes.
var rng = rand.New(rand.NewSource(20261016))

func main() {
	write("lid.wav", lid())
	write("shot.wav", shot())
	write("spill.wav", spill())
	write("miss.wav", miss())
	write("levelup.wav", chime([]float64{1046.5, 1568.0}, 0.11))
	write("gameover.wav", chime([]float64{784.0, 659.3, 523.3, 392.0}, 0.16))
}

// lid is the lid going on the cup: a dull, papery knock, a short low-mid
// thud with no ring. It has its own seeded noise source and is normalised
// to a 0.7 peak, so it matches the sketch it was picked from.
func lid() []float64 {
	r := rand.New(rand.NewSource(13))
	out := make([]float64, secs(0.1))
	bp := newBandpass(420, 2.5)
	peak := 0.0
	for i := range out {
		t := float64(i) / sampleRate
		x := r.Float64()*2 - 1
		out[i] = (2.2*bp.step(x) + 0.5*math.Sin(2*math.Pi*300*t)) * math.Exp(-t*55)
		peak = math.Max(peak, math.Abs(out[i]))
	}
	for i := range out {
		out[i] *= 0.7 / peak
	}
	return out
}

// shot is the espresso machine: a steamy hiss over the pump's low buzz.
func shot() []float64 {
	n := secs(0.38)
	out := make([]float64, n)
	var lp float64
	for i := range out {
		t := float64(i) / sampleRate
		env := math.Min(1, t/0.02) * math.Exp(-t*6)
		noise := rng.Float64()*2 - 1
		lp += (noise - lp) * 0.25
		hiss := noise - lp // high-passed
		pump := math.Copysign(1, math.Sin(2*math.Pi*55*t)) * 0.18
		out[i] = env * (0.55*hiss + pump*math.Exp(-t*4))
	}
	return out
}

// spill is coffee slopping over the rim: a low wet splash and a few drips.
func spill() []float64 {
	n := secs(0.55)
	out := make([]float64, n)
	var lp float64
	for i := range out {
		t := float64(i) / sampleRate
		noise := rng.Float64()*2 - 1
		lp += (noise - lp) * 0.08
		out[i] = 1.6 * lp * math.Min(1, t/0.01) * math.Exp(-t*7)
	}
	for _, at := range []float64{0.18, 0.29, 0.41} {
		start := secs(at)
		f := 900 + rng.Float64()*500
		for i := 0; i < secs(0.06) && start+i < n; i++ {
			t := float64(i) / sampleRate
			// A drip is a quick upward chirp.
			out[start+i] += 0.3 * math.Sin(2*math.Pi*(f+f*8*t)*t) * math.Exp(-t*70)
		}
	}
	return out
}

// miss is a dull "nope" thunk for an empty or lidless cup.
func miss() []float64 {
	n := secs(0.22)
	out := make([]float64, n)
	for i := range out {
		t := float64(i) / sampleRate
		f := 150 - 60*t/0.22
		sq := math.Copysign(1, math.Sin(2*math.Pi*f*t))
		out[i] = 0.35 * sq * math.Exp(-t*14)
	}
	return out
}

// chime plays a sequence of bell notes, each step seconds apart.
func chime(notes []float64, step float64) []float64 {
	n := secs(step*float64(len(notes)) + 0.6)
	out := make([]float64, n)
	for k, f := range notes {
		start := secs(step * float64(k))
		for i := 0; start+i < n; i++ {
			t := float64(i) / sampleRate
			env := math.Exp(-t * 5)
			out[start+i] += 0.28 * env * (math.Sin(2*math.Pi*f*t) + 0.35*math.Sin(2*math.Pi*f*2.01*t))
		}
	}
	return out
}

type bandpass struct{ b0, b2, a1, a2, x1, x2, y1, y2 float64 }

func newBandpass(f, q float64) *bandpass {
	w := 2 * math.Pi * f / sampleRate
	alpha := math.Sin(w) / (2 * q)
	a0 := 1 + alpha
	return &bandpass{b0: alpha / a0, b2: -alpha / a0, a1: -2 * math.Cos(w) / a0, a2: (1 - alpha) / a0}
}

func (b *bandpass) step(x float64) float64 {
	y := b.b0*x + b.b2*b.x2 - b.a1*b.y1 - b.a2*b.y2
	b.x2, b.x1 = b.x1, x
	b.y2, b.y1 = b.y1, y
	return y
}

func secs(s float64) int { return int(s * sampleRate) }

// write encodes mono float samples as a 16-bit stereo WAV, the format the
// game's audio context decodes without resampling.
func write(name string, samples []float64) {
	f, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	data := make([]byte, len(samples)*4)
	for i, s := range samples {
		v := int16(math.Max(-32768, math.Min(32767, s*32767)))
		binary.LittleEndian.PutUint16(data[i*4+0:], uint16(v))
		binary.LittleEndian.PutUint16(data[i*4+2:], uint16(v))
	}

	const (
		channels      = 2
		bitsPerSample = 16
	)
	byteRate := sampleRate * channels * bitsPerSample / 8
	w := func(v ...any) {
		for _, x := range v {
			if err := binary.Write(f, binary.LittleEndian, x); err != nil {
				panic(err)
			}
		}
	}
	w([]byte("RIFF"), uint32(36+len(data)), []byte("WAVE"),
		[]byte("fmt "), uint32(16), uint16(1), uint16(channels), uint32(sampleRate),
		uint32(byteRate), uint16(channels*bitsPerSample/8), uint16(bitsPerSample),
		[]byte("data"), uint32(len(data)))
	if _, err := f.Write(data); err != nil {
		panic(err)
	}
}
