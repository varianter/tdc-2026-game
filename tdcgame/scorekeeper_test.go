package tdcgame

import (
	"path/filepath"
	"testing"
)

func TestScoreKeeper(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "scores.db")

	sk, err := NewScoreKeeper(dbPath)
	if err != nil {
		t.Fatalf("NewScoreKeeper: %v", err)
	}
	defer sk.Close()

	if err := sk.AddScore("tdcrunner", 100); err != nil {
		t.Fatalf("AddScore: %v", err)
	}
	if err := sk.AddScore("tdcrunner", 250); err != nil {
		t.Fatalf("AddScore: %v", err)
	}

	scores := sk.GetScores("tdcrunner")
	if len(scores) != 2 || scores[0] != 100 || scores[1] != 250 {
		t.Fatalf("GetScores = %v, want [100 250]", scores)
	}

	sk2, err := NewScoreKeeper(dbPath)
	if err != nil {
		t.Fatalf("reload NewScoreKeeper: %v", err)
	}
	defer sk2.Close()

	reloaded := sk2.GetScores("tdcrunner")
	if len(reloaded) != 2 || reloaded[0] != 100 || reloaded[1] != 250 {
		t.Fatalf("reloaded scores = %v, want [100 250]", reloaded)
	}
}
