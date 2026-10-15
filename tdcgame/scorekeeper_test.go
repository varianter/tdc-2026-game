package tdcgame

import (
	"database/sql"
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

	if _, err := sk.AddScore("tdcrunner", 100); err != nil {
		t.Fatalf("AddScore: %v", err)
	}
	id, err := sk.AddScore("tdcrunner", 250)
	if err != nil {
		t.Fatalf("AddScore: %v", err)
	}
	if err := sk.SetName("tdcrunner", id, "ADA L"); err != nil {
		t.Fatalf("SetName: %v", err)
	}

	scores := sk.GetScores("tdcrunner")
	if len(scores) != 2 || scores[0].Value != 100 || scores[1].Value != 250 || scores[1].Name != "ADA L" {
		t.Fatalf("GetScores = %v, want [100 250]", scores)
	}

	sk2, err := NewScoreKeeper(dbPath)
	if err != nil {
		t.Fatalf("reload NewScoreKeeper: %v", err)
	}
	defer sk2.Close()

	reloaded := sk2.GetScores("tdcrunner")
	if len(reloaded) != 2 || reloaded[0].Value != 100 || reloaded[1].Value != 250 || reloaded[0].Name != "" || reloaded[1].Name != "ADA L" {
		t.Fatalf("reloaded scores = %v, want [100 250]", reloaded)
	}
}

// A database created before names existed gains the column on open.
func TestScoreKeeperMigratesNameColumn(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "scores.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE scores (id INTEGER PRIMARY KEY AUTOINCREMENT, game_name TEXT NOT NULL, score INTEGER NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		INSERT INTO scores (game_name, score) VALUES ('bounce', 7);`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	sk, err := NewScoreKeeper(dbPath)
	if err != nil {
		t.Fatalf("NewScoreKeeper on old schema: %v", err)
	}
	defer sk.Close()
	got := sk.GetScores("bounce")
	if len(got) != 1 || got[0].Value != 7 || got[0].Name != "" {
		t.Fatalf("migrated scores = %v, want [{7, \"\"}]", got)
	}
	if err := sk.SetName("bounce", got[0].ID, "BO"); err != nil {
		t.Fatalf("SetName after migration: %v", err)
	}
}
