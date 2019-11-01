package analyzers

import (
	"github.com/iamipanda/ygopro-data"
	"github.com/go-pg/pg"
	"github.com/op/go-logging"
)

type Analyzer interface {
	Analyze(deck *ygopro_data.Deck, source string, playerName string)
	Push(db *pg.DB)
}

type MessageAnalyzer interface {
	Analyze(playerAName string, playerBName string, source string, playerADeck *ygopro_data.Deck, playerBDeck *ygopro_data.Deck, winner int, first []string)
	Push(db *pg.DB)
}

type deckInfo struct {
	Deck string
	Tag []string
}

type AnalyzerWithDeckInfo interface {
	AnalyzeWithInfo(deck *ygopro_data.Deck, info *deckInfo, source string, playerName string)
	Push(db *pg.DB)
}

var Logger *logging.Logger