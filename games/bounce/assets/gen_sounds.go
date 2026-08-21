//go:build ignore

// gen_sounds.go synthesises the Bouncy Castle sound effects as 16-bit stereo
// WAV files at 44100 Hz (the rate the game's audio context runs at).
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

// rng is seeded with a fixed value so regenerating the assets reproduces the
// same bytes. With the global (auto-seeded) source, every run rewrote the
// noise-based clips even when nothing about them had changed.
var rng = rand.New(rand.NewSource(20260821))

func main() {
	write("bounce.wav", boing())
	write("crush.wav", crush())
	write("powerup.wav", powerup())
	write("antishield.wav", antiShield())
}

// boing is the trampoline hit: a taut rubbery thump whose pitch dives fast,
// with a short noisy slap at the front for the vinyl smack. The game plays it
// back resampled, so this is the "neutral speed" version.
func boing() []float64 {
	const dur = 0.30
	n := int(dur * sampleRate)
	out := make([]float64, n)
	phase := 0.0
	for i := 0; i < n; i++ {
		t := float64(i) / sampleRate
		p := t / dur

		// pitch dives from 320 Hz to 90 Hz, with a slow wobble on top so it
		// reads as stretched rubber rather than a plain kick drum
		freq := 90 + 230*math.Exp(-7*p)
		freq *= 1 + 0.05*math.Sin(2*math.Pi*18*t)
		phase += 2 * math.Pi * freq / sampleRate

		body := math.Sin(phase) + 0.25*math.Sin(2*phase)
		env := math.Exp(-5.5*p) * math.Min(1, p/0.004)

		slap := 0.0
		if t < 0.02 {
			slap = (rng.Float64()*2 - 1) * math.Exp(-160*t) * 0.5
		}
		out[i] = 0.8*body*env + slap
	}
	return out
}

// crush is the boulder crumbling: a soft gravelly wash over a low thud. The
// noise runs through two cascaded low-pass poles and fades in over ~20ms, so it
// reads as heavy stone rather than a bright click — sharp broadband transients
// here are what make the sound hurt at close range.
func crush() []float64 {
	const dur = 0.55
	n := int(dur * sampleRate)
	out := make([]float64, n)

	// Two one-pole low-passes in series; the cutoff closes as the sound decays.
	var lp1, lp2 float64
	phase := 0.0
	for i := 0; i < n; i++ {
		t := float64(i) / sampleRate
		p := t / dur

		noise := rng.Float64()*2 - 1
		a := 0.16 * math.Exp(-3*p) // low cutoff: keeps the hiss out of the top end
		lp1 += a * (noise - lp1)
		lp2 += a * (lp1 - lp2)

		// Slow attack, so there is no click at the start.
		env := math.Exp(-4*p) * math.Min(1, p/0.05)

		// Low thud underneath for weight; this is where the impact lives now.
		phase += 2 * math.Pi * (68 + 38*math.Exp(-10*p)) / sampleRate
		thud := math.Sin(phase) * math.Exp(-7.5*p) * 0.55

		out[i] = 2.6*lp2*env + thud
	}
	return out
}

// powerup is the classic collect blip: a rising square-wave arpeggio.
func powerup() []float64 {
	steps := []float64{523.25, 659.25, 783.99, 1046.50} // C5 E5 G5 C6
	const step = 0.06
	n := int(float64(len(steps)) * step * sampleRate)
	out := make([]float64, n)
	for i := range out {
		t := float64(i) / sampleRate
		k := int(t / step)
		if k >= len(steps) {
			k = len(steps) - 1
		}
		local := t - float64(k)*step
		env := math.Min(1, local/0.004) * math.Exp(-6*local)
		out[i] = square(steps[k], t) * env * 0.35
	}
	return out
}

// antiShield is the "you lost it" sting: two detuned tones sagging downward in
// pitch, ending on a minor interval so it lands as a disappointment.
func antiShield() []float64 {
	const dur = 0.7
	n := int(dur * sampleRate)
	out := make([]float64, n)
	var p1, p2 float64
	for i := 0; i < n; i++ {
		t := float64(i) / sampleRate
		p := t / dur

		// both voices slide down; the second is detuned for a sour beating
		base := 330 * math.Pow(0.5, p*1.15)
		p1 += 2 * math.Pi * base / sampleRate
		p2 += 2 * math.Pi * (base * 1.19) / sampleRate

		env := math.Min(1, p/0.01) * math.Exp(-2.2*p)
		v := 0.5*saw(p1) + 0.4*saw(p2)
		out[i] = v * env * 0.6
	}
	return out
}

func square(freq, t float64) float64 {
	if math.Mod(t*freq, 1) < 0.5 {
		return 1
	}
	return -1
}

// saw takes an accumulated phase (radians) rather than a frequency, so callers
// can sweep pitch without phase discontinuities.
func saw(phase float64) float64 {
	return 2*math.Mod(phase/(2*math.Pi), 1) - 1
}

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
	f.WriteString("RIFF")
	w(uint32(36 + len(data)))
	f.WriteString("WAVEfmt ")
	w(uint32(16), uint16(1), uint16(channels), uint32(sampleRate), uint32(byteRate),
		uint16(channels*bitsPerSample/8), uint16(bitsPerSample))
	f.WriteString("data")
	w(uint32(len(data)))
	if _, err := f.Write(data); err != nil {
		panic(err)
	}
}
