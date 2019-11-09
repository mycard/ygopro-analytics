package analyzers

import (
	"bytes"
	"github.com/go-pg/pg"
	"github.com/iamipanda/ygopro-data"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SourceTransformer func(*string)

type MatchUpAnalyzer struct {
	matchCache sync.Map // map[string]*map[matchUp]*matchUpResult
	Next []AnalyzerWithDeckInfo
	IdentifierHost string
	Transformer SourceTransformer
}

func NewMatchUpAnalyzer(IdentifierHost string) MatchUpAnalyzer {
	return MatchUpAnalyzer{sync.Map{}, make([]AnalyzerWithDeckInfo, 0), IdentifierHost, nil}
}

type matchUp struct {
	deckA string
	deckB string
}

func newMatchUp(deckA string, deckB string) (m matchUp) {
	return matchUp{ deckA, deckB }
}

type matchUpResult struct {
	 win int
	 lose int
	 draw int
}

func (analyzer *MatchUpAnalyzer) Analyze(playerAName string, playerBName string, source string, playerADeck *ygopro_data.Deck, playerBDeck *ygopro_data.Deck, winner int, first []string) {
	var matchCacheTarget *sync.Map
	if untypedMatchCacheTarget, ok := analyzer.matchCache.Load(source); !ok {
		matchCacheTarget = &sync.Map{}
		analyzer.matchCache.Store(source, matchCacheTarget)
	} else {
		matchCacheTarget = untypedMatchCacheTarget.(*sync.Map)
	}
	// fetch Deck
	channelA := make(chan *deckInfo)
	channelB := make(chan *deckInfo)
	go fetchDeckInfo(analyzer.IdentifierHost, playerADeck, channelA)
	go fetchDeckInfo(analyzer.IdentifierHost, playerBDeck, channelB)
	playerADeckInfo, playerBDeckInfo := <-channelA, <-channelB
	// win/lose switch
	if playerADeckInfo.Deck == playerBDeckInfo.Deck {
		winner = MATCH_RESULT_PLAYERS_DRAW
	} else if len(first) == 0 {
		winner = MATCH_RESULT_PLAYERS_DROP
	} else if first[0] != playerAName {
		tmp := playerADeckInfo
		playerADeckInfo = playerBDeckInfo
		playerBDeckInfo = tmp
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
	// pass to next
	if analyzer.Transformer != nil {
		analyzer.Transformer(&source)
	}
	for _, nextAnalyzer := range analyzer.Next {
		nextAnalyzer.AnalyzeWithInfo(playerBDeck, playerBDeckInfo, source, playerBName)
	}
}

func winLoseNumberAccordingToWinner(winner int) (winNumber int, loseNumber int, drawNumber int) {
	switch winner {
	case MATCH_RESULT_PLAYERS_DRAW:
		return 0,0,1
	case MATCH_RESULT_PLAYER_A_WIN:
		return 1,0,0
	case MATCH_RESULT_PLAYER_B_WIN:
		return 0,1,0
	case MATCH_RESULT_PLAYERS_DROP:
		return 0,0,0
	default:
		return 0,0, 0
	}
}

func (analyzer *MatchUpAnalyzer) Push(db *pg.DB) {
	var tempBuffer bytes.Buffer
	var matchupCache bytes.Buffer
	matchupValues := make([]string, 0)

	currentTime := time.Now().Format("2006-01")

	analyzer.matchCache.Range(func(key, value interface{}) bool {
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
			tempBuffer.WriteString(")")
			matchupValues = append(matchupValues, tempBuffer.String())
			return true
		})
		return true
	})

	analyzer.matchCache = sync.Map{}

	if len(matchupValues) > 0 {
		matchupCache.Reset()
		matchupCache.WriteString("insert into matchup values")
		matchupCache.WriteString(strings.Join(matchupValues, ", "))
		matchupCache.WriteString("on conflict on constraint matchup_pk do update set draw = matchup.draw + excluded.draw, win = matchup.win + excluded.win, lose = matchup.lose + excluded.lose")
		sql := matchupCache.String()
		Logger.Debugf("Matchup sql exec: %v", sql)
		if _, err := db.Exec(sql); err != nil {
			Logger.Errorf("Deck Analyzer failed pushing match-up information to database: %v\n", err)
		}
	}

	for _, nextAnalyzer := range analyzer.Next {
		nextAnalyzer.Push(db)
	}
}