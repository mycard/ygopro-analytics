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

var Logger *logging.Logger