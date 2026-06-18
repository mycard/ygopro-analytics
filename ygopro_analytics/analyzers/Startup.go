package analyzers

import (
	"bytes"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-pg/pg"
)

type StartupAnalyzer struct {
	startupCache sync.Map // map[string]*sync.Map (inner: map[startupCacheKey]*startupResult)
	catchupCache sync.Map // map[string]*sync.Map (inner: map[catchupCacheKey]*startupResult)
}

func NewStartupAnalyzer() StartupAnalyzer {
	return StartupAnalyzer{sync.Map{}, sync.Map{}}
}

type startupResult struct {
	win   int
	lose  int
	draw  int
	nWin  int
	nLose int
	nDraw int
}

type startupCacheKey struct {
	cardID int
	first  bool
}

type catchupCacheKey struct {
	cardID       int
	opponentDeck string
}

func (analyzer *StartupAnalyzer) Analyze(report *MatchReport) {
	if len(report.Wins) == 0 || len(report.Replays) == 0 {
		return
	}

	var startupSourceData *sync.Map
	if untypedSourceData, ok := analyzer.startupCache.Load(report.Arena); !ok {
		startupSourceData = &sync.Map{}
		analyzer.startupCache.Store(report.Arena, startupSourceData)
	} else {
		startupSourceData = untypedSourceData.(*sync.Map)
	}

	var catchupSourceData *sync.Map
	if untypedSourceData, ok := analyzer.catchupCache.Load(report.Arena); !ok {
		catchupSourceData = &sync.Map{}
		analyzer.catchupCache.Store(report.Arena, catchupSourceData)
	} else {
		catchupSourceData = untypedSourceData.(*sync.Map)
	}

	// Make sure A is always host
	deckA, deckB := report.DeckInfoA, report.DeckInfoB
	nameA, nameB := report.UsernameA, report.UsernameB
	if report.Replays[0].HostName != report.UsernameA {
		deckA, deckB = deckB, deckA
		nameA, nameB = nameB, nameA
	}

	for i := 0; i < len(report.Replays); i++ {
		replay := &report.Replays[i]
		if replay.StartHand == 0 {
			continue
		}

		aDeck := deckA.Deck
		bDeck := deckB.Deck
		aWon := report.Wins[i] == nameA
		bWon := report.Wins[i] == nameB
		isDraw := report.Wins[i] == ""
		aFirst := report.First[i] == nameA
		bFirst := report.First[i] == nameB

		hostStart := len(replay.HostDeck.Main) - replay.StartHand
		clientStart := len(replay.ClientDeck.Main) - replay.StartHand
		for j := 0; j < len(replay.HostDeck.Main); j++ {
			inHand := j >= hostStart
			analyzer.recordStartupCard(startupSourceData, replay.HostDeck.Main[j], aWon, isDraw, aFirst, inHand)
			analyzer.recordCatchupCard(catchupSourceData, replay.HostDeck.Main[j], aWon, isDraw, bDeck, inHand)
		}
		for j := 0; j < len(replay.ClientDeck.Main); j++ {
			inHand := j >= clientStart
			analyzer.recordStartupCard(startupSourceData, replay.ClientDeck.Main[j], bWon, isDraw, bFirst, inHand)
			analyzer.recordCatchupCard(catchupSourceData, replay.ClientDeck.Main[j], bWon, isDraw, aDeck, inHand)
		}
	}
}

func (analyzer *StartupAnalyzer) recordStartupCard(sourceData *sync.Map, cardID int, won bool, isDraw bool, first bool, inHand bool) {
	key := startupCacheKey{cardID, first}
	var data *startupResult
	if untypedData, ok := sourceData.Load(key); !ok {
		data = &startupResult{}
		sourceData.Store(key, data)
	} else {
		data = untypedData.(*startupResult)
	}
	applyResult(data, won, isDraw, inHand)
}

func (analyzer *StartupAnalyzer) recordCatchupCard(sourceData *sync.Map, cardID int, won bool, isDraw bool, opponentDeck string, inHand bool) {
	key := catchupCacheKey{cardID, opponentDeck}
	var data *startupResult
	if untypedData, ok := sourceData.Load(key); !ok {
		data = &startupResult{}
		sourceData.Store(key, data)
	} else {
		data = untypedData.(*startupResult)
	}
	applyResult(data, won, isDraw, inHand)
}

func applyResult(result *startupResult, won bool, isDraw bool, inHand bool) {
	var drawPointer, winPointer, losePointer *int
	if inHand {
		drawPointer, winPointer, losePointer = &result.draw, &result.win, &result.lose
	} else {
		drawPointer, winPointer, losePointer = &result.nDraw, &result.nWin, &result.nLose
	}
	if isDraw {
		*drawPointer++
	} else if won {
		*winPointer++
	} else {
		*losePointer++
	}
}

func (analyzer *StartupAnalyzer) Push(db *pg.DB) {
	analyzer.pushStartup(db)
	analyzer.pushCatchup(db)
}

func (analyzer *StartupAnalyzer) pushStartup(db *pg.DB) {
	var buffer bytes.Buffer
	data := make([]string, 0)
	currentTime := time.Now().Format("2006-01")

	analyzer.startupCache.Range(func(untypedSource, untypedSourceData interface{}) bool {
		source := untypedSource.(string)
		sourceData := untypedSourceData.(*sync.Map)
		sourceData.Range(func(untypedCacheKey, untypedData interface{}) bool {
			cacheKey := untypedCacheKey.(startupCacheKey)
			result := untypedData.(*startupResult)
			buffer.Reset()
			buffer.WriteString("('")
			buffer.WriteString(source)
			buffer.WriteString("', ")
			buffer.WriteString(strconv.Itoa(cacheKey.cardID))
			buffer.WriteString(", ")
			if cacheKey.first {
				buffer.WriteString("true")
			} else {
				buffer.WriteString("false")
			}
			buffer.WriteString(", '")
			buffer.WriteString(currentTime)
			buffer.WriteString("', ")
			buffer.WriteString(strconv.Itoa(result.draw))
			buffer.WriteString(", ")
			buffer.WriteString(strconv.Itoa(result.lose))
			buffer.WriteString(", ")
			buffer.WriteString(strconv.Itoa(result.win))
			buffer.WriteString(", ")
			buffer.WriteString(strconv.Itoa(result.nDraw))
			buffer.WriteString(", ")
			buffer.WriteString(strconv.Itoa(result.nLose))
			buffer.WriteString(", ")
			buffer.WriteString(strconv.Itoa(result.nWin))
			buffer.WriteString(")")
			data = append(data, buffer.String())
			return true
		})
		return true
	})

	analyzer.startupCache = sync.Map{}
	if len(data) == 0 {
		return
	}

	buffer.Reset()
	buffer.WriteString("insert into startup values ")
	buffer.WriteString(strings.Join(data, ", "))
	buffer.WriteString(" on conflict on constraint card_period_startup do update set draw = startup.draw + excluded.draw, win = startup.win + excluded.win, lose = startup.lose + excluded.lose, n_draw = startup.n_draw + excluded.n_draw, n_win = startup.n_win + excluded.n_win, n_lose = startup.n_lose + excluded.n_lose")
	sql := buffer.String()
	Logger.Debugf("Startup sql exec: %v", sql)
	if _, err := db.Exec(sql); err != nil {
		Logger.Errorf("Startup Analyzer failed pushing to database: %v\n", err)
	}
}

func (analyzer *StartupAnalyzer) pushCatchup(db *pg.DB) {
	var buffer bytes.Buffer
	data := make([]string, 0)
	currentTime := time.Now().Format("2006-01")

	analyzer.catchupCache.Range(func(untypedSource, untypedSourceData interface{}) bool {
		source := untypedSource.(string)
		sourceData := untypedSourceData.(*sync.Map)
		sourceData.Range(func(untypedCacheKey, untypedData interface{}) bool {
			cacheKey := untypedCacheKey.(catchupCacheKey)
			result := untypedData.(*startupResult)
			buffer.Reset()
			buffer.WriteString("('")
			buffer.WriteString(source)
			buffer.WriteString("', ")
			buffer.WriteString(strconv.Itoa(cacheKey.cardID))
			buffer.WriteString(", '")
			buffer.WriteString(cacheKey.opponentDeck)
			buffer.WriteString("', '")
			buffer.WriteString(currentTime)
			buffer.WriteString("', ")
			buffer.WriteString(strconv.Itoa(result.draw))
			buffer.WriteString(", ")
			buffer.WriteString(strconv.Itoa(result.lose))
			buffer.WriteString(", ")
			buffer.WriteString(strconv.Itoa(result.win))
			buffer.WriteString(", ")
			buffer.WriteString(strconv.Itoa(result.nDraw))
			buffer.WriteString(", ")
			buffer.WriteString(strconv.Itoa(result.nLose))
			buffer.WriteString(", ")
			buffer.WriteString(strconv.Itoa(result.nWin))
			buffer.WriteString(")")
			data = append(data, buffer.String())
			return true
		})
		return true
	})

	analyzer.catchupCache = sync.Map{}
	if len(data) == 0 {
		return
	}

	buffer.Reset()
	buffer.WriteString("insert into catchup values ")
	buffer.WriteString(strings.Join(data, ", "))
	buffer.WriteString(" on conflict on constraint card_period_catchup do update set draw = catchup.draw + excluded.draw, win = catchup.win + excluded.win, lose = catchup.lose + excluded.lose, n_draw = catchup.n_draw + excluded.n_draw, n_win = catchup.n_win + excluded.n_win, n_lose = catchup.n_lose + excluded.n_lose")
	sql := buffer.String()
	Logger.Debugf("Catchup sql exec: %v", sql)
	if _, err := db.Exec(sql); err != nil {
		Logger.Errorf("Startup Analyzer failed pushing catchup to database: %v\n", err)
	}
}
