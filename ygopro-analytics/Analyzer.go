package ygopro_analytics

import (
	"github.com/iamipanda/ygopro-data"
	"./analyzers"
	"github.com/go-pg/pg"
)

var onlineAnalyzers = make([]analyzers.Analyzer, 0)
var environment *ygopro_data.Environment
var db *pg.DB

func initializeAnalyzers() {
	environment = ygopro_data.GetEnvironment("zh-CN")
	environment.LoadAllCards()
	countAnalyzer := analyzers.NewCountAnalyzer()
	singleAnalyzer := analyzers.NewSingleCardAnalyzer(environment)
	deckAnalyzer := analyzers.NewDeckAnalyzer(Config.DeckIdentifierHost)
	onlineAnalyzers = append(onlineAnalyzers, &countAnalyzer)
	onlineAnalyzers = append(onlineAnalyzers, &singleAnalyzer)
	onlineAnalyzers = append(onlineAnalyzers, &deckAnalyzer)
}

func initializeDatabaseConnection() {
	db = pg.Connect(&Config.Postgres)
}

func Analyze(deck *ygopro_data.Deck, source string, playerName string) {
	deck.SeparateExFromMainFromCache(environment)
	deck.Summary()
	deck.Classify()
	for _, analyzer := range onlineAnalyzers {
		analyzer.Analyze(deck, source, playerName)
	}
}

func Push() {
	for _, analyzer := range onlineAnalyzers {
		analyzer.Push(db)
	}
}
