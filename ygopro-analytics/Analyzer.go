package ygopro_analytics

import (
	"github.com/iamipanda/ygopro-data"
	"./analyzers"
	"github.com/go-pg/pg"
	"strings"
)

var onlineAnalyzers = make([]analyzers.Analyzer, 0)
var onlineMessageAnalyzers = make([]analyzers.MessageAnalyzer, 0)
var environment *ygopro_data.Environment
var db *pg.DB
var deckAnalyzer analyzers.DeckAnalyzer

func initializeAnalyzers() {
	environment = ygopro_data.GetEnvironment("zh-CN")
	environment.LoadAllCards()
	countAnalyzer := analyzers.NewCountAnalyzer()
	singleAnalyzer := analyzers.NewSingleCardAnalyzer(environment)
	deckAnalyzer = analyzers.NewDeckAnalyzer(Config.DeckIdentifierHost)
	onlineAnalyzers = append(onlineAnalyzers, &countAnalyzer)
	onlineAnalyzers = append(onlineAnalyzers, &singleAnalyzer)
	// onlineAnalyzers = append(onlineAnalyzers, &deckAnalyzer)
	matchUpAnalyzer := analyzers.NewMatchUpAnalyzer(Config.DeckIdentifierHost)
	matchUpAnalyzer.Next = append(matchUpAnalyzer.Next, &deckAnalyzer)
	matchUpAnalyzer.Transformer = func(source *string) {
		*source = "mycard-" + *source
	}
	onlineMessageAnalyzers = append(onlineMessageAnalyzers, &matchUpAnalyzer)
}

func initializeDatabaseConnection() {
	db = pg.Connect(&Config.Postgres)
}

func Analyze(deck *ygopro_data.Deck, source string, playerName string) {
	deck.RemoveAlias(environment)
	deck.SeparateExFromMainFromCache(environment)
	deck.Classify()
	for _, analyzer := range onlineAnalyzers {
		analyzer.Analyze(deck, source, playerName)
	}
	if !strings.HasPrefix(source, "mycard") {
		deckAnalyzer.Analyze(deck, source, playerName)
	}
}

func AnalyzeMessage(playerAName string, playerBName string, playerADeck *ygopro_data.Deck, playerBDeck *ygopro_data.Deck, playerAScore int, playerBScore int, source string, first []string) {
	if playerAScore == -5 || playerBScore == -5 {
		return
	}
	// not strict.
	if len(playerADeck.Main) == 0 || len(playerBDeck.Main) == 0 {
		return
	}
	playerADeck.RemoveAlias(environment)
	playerBDeck.RemoveAlias(environment)
	playerADeck.SeparateExFromMainFromCache(environment)
	playerBDeck.SeparateExFromMainFromCache(environment)
	playerADeck.Classify()
	playerBDeck.Classify()
	for _, analyzer := range onlineMessageAnalyzers {
		analyzer.Analyze(playerAName, playerBName, source, playerADeck, playerBDeck, analyzers.JudgeWinLose(playerAScore, playerBScore), first)
	}
}

func Push() {
	for _, analyzer := range onlineAnalyzers {
		analyzer.Push(db)
	}
	for _, analyzer := range onlineMessageAnalyzers {
		analyzer.Push(db)
	}
}
