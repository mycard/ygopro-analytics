package ygopro_analytics

import (
	"main/ygopro_analytics/analyzers"
	"strings"

	"github.com/go-pg/pg"
	ygopro_data "github.com/iamipanda/ygopro-data"
)

var onlineDeckAnalyzers = make([]analyzers.DeckMessageAnalyzer, 0)
var onlineMessageAnalyzers = make([]analyzers.MatchReportAnalyzer, 0)
var environment *ygopro_data.Environment
var db *pg.DB
var deckAnalyzer analyzers.DeckAnalyzer

func initializeAnalyzers() {
	environment = ygopro_data.GetEnvironment("zh-CN")
	environment.LoadAllCards()
	countAnalyzer := analyzers.NewCountAnalyzer()
	singleAnalyzer := analyzers.NewSingleCardAnalyzer(environment)
	deckAnalyzer = analyzers.NewDeckAnalyzer(Config.DeckIdentifierHost)
	onlineDeckAnalyzers = append(onlineDeckAnalyzers, &countAnalyzer)
	onlineDeckAnalyzers = append(onlineDeckAnalyzers, &singleAnalyzer)
	// onlineDeckAnalyzers = append(onlineDeckAnalyzers, &deckAnalyzer)
	deckIdentifier := analyzers.NewDeckIdentifier(Config.DeckIdentifierHost)
	onlineMessageAnalyzers = append(onlineMessageAnalyzers, &deckIdentifier)
	matchUpAnalyzer := analyzers.NewMatchUpAnalyzer()
	matchUpAnalyzer.Next = append(matchUpAnalyzer.Next, &deckAnalyzer)
	matchUpAnalyzer.Transformer = func(source *string) {
		*source = "mycard-" + *source
	}
	onlineMessageAnalyzers = append(onlineMessageAnalyzers, &matchUpAnalyzer)
	startupAnalyzer := analyzers.NewStartupAnalyzer()
	onlineMessageAnalyzers = append(onlineMessageAnalyzers, &startupAnalyzer)
}

func initializeDatabaseConnection() {
	db = pg.Connect(&Config.Postgres)
}

func AnalyzeDeck(deck *ygopro_data.Deck, source string, playerName string) {
	deck.RemoveAlias(environment)
	deck.SeparateExFromMainFromCache(environment)
	deck.Classify()
	for _, analyzer := range onlineDeckAnalyzers {
		analyzer.Analyze(deck, source, playerName)
	}
	if source != "mycard-athletic" && source != "mycard-entertain" {
		deckAnalyzer.Analyze(deck, source, playerName)
	}
}

func AnalyzeMatch(report analyzers.MatchReport) {
	if report.UserscoreA == -5 || report.UserscoreB == -5 {
		return
	}
	if len(report.UserdeckA.Main) == 0 || len(report.UserdeckB.Main) == 0 {
		return
	}
	report.UserdeckA.RemoveAlias(environment)
	report.UserdeckB.RemoveAlias(environment)
	report.UserdeckA.SeparateExFromMainFromCache(environment)
	report.UserdeckB.SeparateExFromMainFromCache(environment)
	report.UserdeckA.Classify()
	report.UserdeckB.Classify()
	for i := range report.Replays {
		report.Replays[i].HostDeck.RemoveAlias(environment)
		report.Replays[i].ClientDeck.RemoveAlias(environment)
		report.Replays[i].HostName = strings.TrimRight(report.Replays[i].HostName, "\x00")
		report.Replays[i].ClientName = strings.TrimRight(report.Replays[i].ClientName, "\x00")
	}
	for _, analyzer := range onlineMessageAnalyzers {
		analyzer.Analyze(&report)
	}
}

func Push() {
	for _, analyzer := range onlineDeckAnalyzers {
		analyzer.Push(db)
	}
	for _, analyzer := range onlineMessageAnalyzers {
		analyzer.Push(db)
	}
}
