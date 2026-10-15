package main

import (
	"math"
	"path/filepath"
	"strings"
	"testing"

	"variant.dev/tdcgame/tdcgame"
)

func TestIsNewLeader(t *testing.T) {
	sk, err := tdcgame.NewScoreKeeper(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer sk.Close()
	prev := scoreKeeper
	scoreKeeper = sk
	defer func() { scoreKeeper = prev }()

	steps := []struct {
		score int
		want  bool
	}{
		{0, false},  // zero never counts
		{10, true},  // first real score
		{5, false},  // below #1
		{10, false}, // tie with #1
		{11, true},  // beats #1
	}
	for _, st := range steps {
		sk.AddScore("g", st.score)
		if got := isNewLeader("g", st.score); got != st.want {
			t.Errorf("score %d: got %v, want %v", st.score, got, st.want)
		}
	}
}

func TestEditName(t *testing.T) {
	steps := []struct{ cell, want string }{
		{"DEL", ""}, // DEL on empty is a no-op
		{"SPC", ""}, // no leading space
		{"A", "A"},
		{"SPC", "A "},
		{"SPC", "A "}, // no double space
		{"B", "A B"},
		{"DEL", "A "},
		{"C", "A C"},
	}
	name := ""
	for _, st := range steps {
		name = editName(name, st.cell)
		if name != st.want {
			t.Fatalf("after %q: got %q, want %q", st.cell, name, st.want)
		}
	}

	full := strings.Repeat("Z", maxNameLen)
	if got := editName(full, "A"); got != full {
		t.Errorf("letter past max: got %q", got)
	}
	if got := editName(full, "SPC"); got != full {
		t.Errorf("space past max: got %q", got)
	}
}

func TestPressAmount(t *testing.T) {
	p := &pressIcon{period: 2, down: 0.5, up: 1, move: 0.2}
	for _, c := range []struct{ t, want float64 }{
		{0, 0}, {0.6, 0.5}, {0.8, 1}, {1.1, 0.5}, {1.5, 0}, {2.8, 1}, // 2.8 wraps to 0.8
	} {
		if got := p.pressAmount(c.t); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("pressAmount(%v) = %v, want %v", c.t, got, c.want)
		}
	}
}

func TestKeyOptions(t *testing.T) {
	if got := strings.Join(keyOptions(1), ""); got != "ABC"+nameBack {
		t.Errorf("keyOptions(1) = %q", got)
	}
	// Appending the back option must not write into nameKeys' backing array.
	keyOptions(8)
	if got := strings.Join(nameKeys[8].opts, ""); got != "WXYZ" {
		t.Errorf("nameKeys[8] changed to %q", got)
	}
}
