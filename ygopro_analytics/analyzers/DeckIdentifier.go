package analyzers

import (
	"github.com/go-pg/pg"
)

type DeckIdentifier struct {
	IdentifierHost string
}

func NewDeckIdentifier(identifierHost string) DeckIdentifier {
	return DeckIdentifier{identifierHost}
}

func (analyzer *DeckIdentifier) Analyze(report *MatchReport) {
	channelA := make(chan *deckInfo)
	channelB := make(chan *deckInfo)
	go fetchDeckInfo(analyzer.IdentifierHost, &report.UserdeckA, channelA)
	go fetchDeckInfo(analyzer.IdentifierHost, &report.UserdeckB, channelB)
	report.DeckInfoA = <-channelA
	report.DeckInfoB = <-channelB
}

func (analyzer *DeckIdentifier) Push(db *pg.DB) {}
