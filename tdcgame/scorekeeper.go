package tdcgame

type ScoreKeeper struct {
	scores map[string]GameScore
}

type GameScore struct {
	gameName string
	scores   []int
}
