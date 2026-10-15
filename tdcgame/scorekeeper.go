package tdcgame

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

// Score is one saved result. Name is empty when the player didn't enter one.
type Score struct {
	ID    int64
	Value int
	Name  string
}

type ScoreKeeper struct {
	mu     sync.RWMutex
	scores map[string][]Score
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
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			name TEXT NOT NULL DEFAULT ''
		);
	`); err != nil {
		db.Close()
		return nil, fmt.Errorf("init score table: %w", err)
	}
	// Databases created before names existed lack the column; add it in place.
	if _, err := db.Exec(`ALTER TABLE scores ADD COLUMN name TEXT NOT NULL DEFAULT ''`); err != nil &&
		!strings.Contains(err.Error(), "duplicate column") {
		db.Close()
		return nil, fmt.Errorf("add name column: %w", err)
	}

	sk := &ScoreKeeper{
		scores: make(map[string][]Score),
		db:     db,
	}
	if err := sk.loadScores(); err != nil {
		db.Close()
		return nil, err
	}
	return sk, nil
}

func (sk *ScoreKeeper) loadScores() error {
	rows, err := sk.db.Query(`SELECT id, game_name, score, name FROM scores ORDER BY id`)
	if err != nil {
		return fmt.Errorf("load scores: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var gameName string
		var sc Score
		if err := rows.Scan(&sc.ID, &gameName, &sc.Value, &sc.Name); err != nil {
			return fmt.Errorf("scan score row: %w", err)
		}
		sk.scores[gameName] = append(sk.scores[gameName], sc)
	}
	return rows.Err()
}

// AddScore saves a score and returns its ID, for naming it later with SetName.
func (sk *ScoreKeeper) AddScore(gameName string, score int) (int64, error) {
	res, err := sk.db.Exec(`INSERT INTO scores (game_name, score) VALUES (?, ?)`, gameName, score)
	if err != nil {
		return 0, fmt.Errorf("insert score: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("insert score id: %w", err)
	}

	sk.mu.Lock()
	sk.scores[gameName] = append(sk.scores[gameName], Score{ID: id, Value: score})
	sk.mu.Unlock()
	return id, nil
}

// SetName attaches a player name to a saved score.
func (sk *ScoreKeeper) SetName(gameName string, id int64, name string) error {
	if _, err := sk.db.Exec(`UPDATE scores SET name = ? WHERE id = ?`, name, id); err != nil {
		return fmt.Errorf("set score name: %w", err)
	}

	sk.mu.Lock()
	defer sk.mu.Unlock()
	for i := range sk.scores[gameName] {
		if sk.scores[gameName][i].ID == id {
			sk.scores[gameName][i].Name = name
		}
	}
	return nil
}

func (sk *ScoreKeeper) GetScores(gameName string) []Score {
	sk.mu.RLock()
	defer sk.mu.RUnlock()

	scores := sk.scores[gameName]
	if len(scores) == 0 {
		return nil
	}
	out := make([]Score, len(scores))
	copy(out, scores)
	return out
}

func (sk *ScoreKeeper) Close() error {
	return sk.db.Close()
}
