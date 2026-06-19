package analyzers

import (
	"bytes"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-pg/pg"
)

type SourceTransformer func(*string)

type MatchUpAnalyzer struct {
	matchCache  sync.Map // map[string]*map[matchUp]*matchUpResult
	duelCache   sync.Map // map[string]*map[matchUp]*matchUpResult
	Next        []*DeckAnalyzer
	Transformer SourceTransformer
}

func NewMatchUpAnalyzer() MatchUpAnalyzer {
	return MatchUpAnalyzer{sync.Map{}, sync.Map{}, make([]*DeckAnalyzer, 0), nil}
}

type matchUp struct {
	deckA string
	deckB string
}

func newMatchUp(deckA string, deckB string) (m matchUp) {
	return matchUp{deckA, deckB}
}

type matchUpResult struct {
	win  int
	lose int
	draw int
}

func (analyzer *MatchUpAnalyzer) Analyze(report *MatchReport) {
	var matchCacheTarget *sync.Map
	if untypedMatchCacheTarget, ok := analyzer.matchCache.Load(report.Arena); !ok {
		matchCacheTarget = &sync.Map{}
		analyzer.matchCache.Store(report.Arena, matchCacheTarget)
	} else {
		matchCacheTarget = untypedMatchCacheTarget.(*sync.Map)
	}

	playerADeckInfo, playerBDeckInfo := report.DeckInfoA, report.DeckInfoB
	analyzer.recordMatch(report, matchCacheTarget, playerADeckInfo, playerBDeckInfo)

	if len(report.Wins) > 0 {
		var duelCacheTarget *sync.Map
		if untypedCacheTarget, ok := analyzer.duelCache.Load(report.Arena); !ok {
			duelCacheTarget = &sync.Map{}
			analyzer.duelCache.Store(report.Arena, duelCacheTarget)
		} else {
			duelCacheTarget = untypedCacheTarget.(*sync.Map)
		}
		analyzer.recordPerDuel(report, duelCacheTarget, playerADeckInfo, playerBDeckInfo)
	}

	analyzer.passToNext(report, playerADeckInfo, playerBDeckInfo)
}

func (analyzer *MatchUpAnalyzer) recordMatch(report *MatchReport, matchCacheTarget *sync.Map, playerADeckInfo *deckInfo, playerBDeckInfo *deckInfo) {
	winner := JudgeWinLose(report.UserscoreA, report.UserscoreB)
	Logger.Debugf("%s(%s) vs %s(%s) win %v/%v", report.UsernameA, playerADeckInfo.Deck, report.UsernameB, playerBDeckInfo.Deck, winner, len(report.First))

	if report.First[0] != report.UsernameA {
		playerADeckInfo, playerBDeckInfo = playerBDeckInfo, playerADeckInfo
		winner *= -1
	}

	_matchUp := newMatchUp(playerADeckInfo.Deck, playerBDeckInfo.Deck)
	winNumber, loseNumber, drawNumber := winLoseNumberAccordingToWinner(winner)
	if untypedMatchUpResult, ok := matchCacheTarget.Load(_matchUp); !ok {
		matchCacheTarget.Store(_matchUp, &matchUpResult{winNumber, loseNumber, drawNumber})
	} else {
		result := untypedMatchUpResult.(*matchUpResult)
		result.win += winNumber
		result.lose += loseNumber
		result.draw += drawNumber
	}
}

func (analyzer *MatchUpAnalyzer) recordPerDuel(report *MatchReport, matchCacheTarget *sync.Map, playerADeckInfo *deckInfo, playerBDeckInfo *deckInfo) {
	duelCount := min(len(report.Wins), len(report.First))

	for i := 0; i < duelCount; i++ {
		firstPlayer := report.First[i]
		duelWinner := report.Wins[i]

		deckAInfo, deckBInfo := playerADeckInfo, playerBDeckInfo
		if firstPlayer != report.UsernameA {
			deckAInfo, deckBInfo = deckBInfo, deckAInfo
		}

		winNumber, loseNumber, drawNumber := 0, 0, 0
		switch duelWinner {
		case "":
			drawNumber = 1
		case firstPlayer:
			winNumber, loseNumber = 1, 0
		default:
			winNumber, loseNumber = 0, 1
		}

		_matchUp := newMatchUp(deckAInfo.Deck, deckBInfo.Deck)
		if untypedMatchUpResult, ok := matchCacheTarget.Load(_matchUp); !ok {
			matchCacheTarget.Store(_matchUp, &matchUpResult{winNumber, loseNumber, drawNumber})
		} else {
			result := untypedMatchUpResult.(*matchUpResult)
			result.win += winNumber
			result.lose += loseNumber
			result.draw += drawNumber
		}
	}
}

func (analyzer *MatchUpAnalyzer) passToNext(report *MatchReport, playerADeckInfo *deckInfo, playerBDeckInfo *deckInfo) {
	source := report.Arena
	if analyzer.Transformer != nil {
		analyzer.Transformer(&source)
	}
	for _, nextAnalyzer := range analyzer.Next {
		nextAnalyzer.AnalyzeWithInfo(&report.UserdeckA, playerADeckInfo, source, report.UsernameA)
		nextAnalyzer.AnalyzeWithInfo(&report.UserdeckB, playerBDeckInfo, source, report.UsernameB)
	}
}

func winLoseNumberAccordingToWinner(winner int) (winNumber int, loseNumber int, drawNumber int) {
	switch winner {
	case MATCH_RESULT_PLAYERS_DRAW:
		return 0, 0, 1
	case MATCH_RESULT_PLAYER_A_WIN:
		return 1, 0, 0
	case MATCH_RESULT_PLAYER_B_WIN:
		return 0, 1, 0
	case MATCH_RESULT_PLAYERS_DROP:
		return 0, 0, 0
	default:
		return 0, 0, 0
	}
}

func (analyzer *MatchUpAnalyzer) Push(db *pg.DB) {
	var tempBuffer bytes.Buffer
	var matchupBuffer bytes.Buffer
	matchupValues := make([]string, 0)

	currentTime := time.Now().Format("2006-01")

	collectValues := func(cache *sync.Map, typ string) {
		cache.Range(func(key, value interface{}) bool {
			source := key.(string)
			hash := value.(*sync.Map)
			hash.Range(func(key, value interface{}) bool {
				matchup := key.(matchUp)
				result := value.(*matchUpResult)
				tempBuffer.Reset()
				tempBuffer.WriteString("('")
				tempBuffer.WriteString(source)
				tempBuffer.WriteString("', '")
				tempBuffer.WriteString(matchup.deckA)
				tempBuffer.WriteString("', '")
				tempBuffer.WriteString(matchup.deckB)
				tempBuffer.WriteString("', '")
				tempBuffer.WriteString(currentTime)
				tempBuffer.WriteString("', ")
				tempBuffer.WriteString(strconv.Itoa(result.draw))
				tempBuffer.WriteString(", ")
				tempBuffer.WriteString(strconv.Itoa(result.lose))
				tempBuffer.WriteString(", ")
				tempBuffer.WriteString(strconv.Itoa(result.win))
				tempBuffer.WriteString(", '")
				tempBuffer.WriteString(typ)
				tempBuffer.WriteString("')")
				matchupValues = append(matchupValues, tempBuffer.String())
				return true
			})
			return true
		})
	}

	collectValues(&analyzer.matchCache, "match")
	collectValues(&analyzer.duelCache, "duel")

	analyzer.matchCache = sync.Map{}
	analyzer.duelCache = sync.Map{}

	if len(matchupValues) > 0 {
		matchupBuffer.Reset()
		matchupBuffer.WriteString("insert into matchup values")
		matchupBuffer.WriteString(strings.Join(matchupValues, ", "))
		matchupBuffer.WriteString(" on conflict on constraint matchup_pk do update set draw = matchup.draw + excluded.draw, win = matchup.win + excluded.win, lose = matchup.lose + excluded.lose")
		sql := matchupBuffer.String()
		Logger.Debugf("Matchup sql exec: %v", sql)
		if _, err := db.Exec(sql); err != nil {
			Logger.Errorf("Deck Analyzer failed pushing match-up information to database: %v\n", err)
		}
	}

	for _, nextAnalyzer := range analyzer.Next {
		nextAnalyzer.Push(db)
	}
}
