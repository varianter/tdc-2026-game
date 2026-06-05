package tdcgame

import (
	"database/sql"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
)

type ScoreKeeper struct {
	mu     sync.RWMutex
	scores map[string][]int
	db     *sql.DB
}

func NewScoreKeeper(dbPath string) (*ScoreKeeper, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open score database: %w", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS scores (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			game_name TEXT NOT NULL,
			score INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		db.Close()
		return nil, fmt.Errorf("init score table: %w", err)
	}

	sk := &ScoreKeeper{
		scores: make(map[string][]int),
		db:     db,
	}
	if err := sk.loadScores(); err != nil {
		db.Close()
		return nil, err
	}
	return sk, nil
}

func (sk *ScoreKeeper) loadScores() error {
	rows, err := sk.db.Query(`SELECT game_name, score FROM scores ORDER BY id`)
	if err != nil {
		return fmt.Errorf("load scores: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var gameName string
		var score int
		if err := rows.Scan(&gameName, &score); err != nil {
			return fmt.Errorf("scan score row: %w", err)
		}
		sk.scores[gameName] = append(sk.scores[gameName], score)
	}
	return rows.Err()
}

func (sk *ScoreKeeper) AddScore(gameName string, score int) error {
	sk.mu.Lock()
	sk.scores[gameName] = append(sk.scores[gameName], score)
	sk.mu.Unlock()

	if err := sk.SaveScore(gameName, score); err != nil {
		return err
	}
	return nil
}

func (sk *ScoreKeeper) SaveScore(gameName string, score int) error {
	_, err := sk.db.Exec(
		`INSERT INTO scores (game_name, score) VALUES (?, ?)`,
		gameName,
		score,
	)
	if err != nil {
		return fmt.Errorf("insert score: %w", err)
	}
	return nil
}

func (sk *ScoreKeeper) GetScores(gameName string) []int {
	sk.mu.RLock()
	defer sk.mu.RUnlock()

	scores := sk.scores[gameName]
	if len(scores) == 0 {
		return nil
	}
	out := make([]int, len(scores))
	copy(out, scores)
	return out
}

func (sk *ScoreKeeper) Close() error {
	return sk.db.Close()
}
