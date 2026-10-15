package tdcgame

import (
	"math"
	"testing"
)

func TestPressAmount(t *testing.T) {
	p := &PressIcon{period: 2, presses: []press{{0.5, 1}}, move: 0.2}
	for _, c := range []struct{ t, want float64 }{
		{0, 0}, {0.6, 0.5}, {0.8, 1}, {1.1, 0.5}, {1.5, 0}, {2.8, 1}, // 2.8 wraps to 0.8
	} {
		if got := p.pressAmount(c.t); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("pressAmount(%v) = %v, want %v", c.t, got, c.want)
		}
	}
}

func TestPressAmountDouble(t *testing.T) {
	p := &PressIcon{period: 4, presses: []press{{0.5, 1}, {2, 2.5}}, move: 0.2}
	for _, c := range []struct{ t, want float64 }{
		{0.8, 1}, {1.5, 0}, {2.1, 0.5}, {2.3, 1}, {2.6, 0.5}, {3, 0},
	} {
		if got := p.pressAmount(c.t); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("pressAmount(%v) = %v, want %v", c.t, got, c.want)
		}
	}
}
