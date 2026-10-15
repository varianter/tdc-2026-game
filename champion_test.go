package main

import (
	"path/filepath"
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
