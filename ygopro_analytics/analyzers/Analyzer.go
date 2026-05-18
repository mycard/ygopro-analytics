package analyzers

import (
	"github.com/go-pg/pg"
	ygopro_data "github.com/iamipanda/ygopro-data"
	"github.com/op/go-logging"
)

type DeckMessageAnalyzer interface {
	Analyze(deck *ygopro_data.Deck, source string, playerName string)
	Push(db *pg.DB)
}

type MatchReportAnalyzer interface {
	Analyze(report *MatchReport)
	Push(db *pg.DB)
}

type deckInfo struct {
	Deck string
	Tag  []string
}

type MatchReport struct {
	AccessKey        string
	UsernameA        string
	UsernameB        string
	UserscoreA       int
	UserscoreB       int
	UserdeckA        ygopro_data.Deck
	UserdeckB        ygopro_data.Deck
	UserdeckAHistory []ygopro_data.Deck
	UserdeckBHistory []ygopro_data.Deck
	First            []string
	Wins             []string
	Replays          []ygopro_data.Replay
	Start            string
	End              string
	Arena            string
	DeckInfoA        *deckInfo
	DeckInfoB        *deckInfo
}

var Logger *logging.Logger
