package analyzers

import (
	"bytes"
	"encoding/json"
	"github.com/go-pg/pg"
	"github.com/iamipanda/ygopro-data"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// deckCache map[string]map[string]int
// tagCache map[string]map[string]int
type DeckAnalyzer struct {
	deckCache sync.Map
	tagCache sync.Map
	DeckIdentifierHost string
	UnknownDecks []unknownDeckDetail
}

type deckInfo struct {
	Deck string
	Tag []string
}

type unknownDeckDetail struct {
	Deck *ygopro_data.Deck
	Source string
	User string
	Time time.Time
}

func NewDeckAnalyzer(deckIdentifierHost string) DeckAnalyzer {
	return DeckAnalyzer { sync.Map{}, sync.Map{}, deckIdentifierHost, make([]unknownDeckDetail, 0) }
}

func (analyzer *DeckAnalyzer) addDeckInfoToCache(source string, info *deckInfo) {
	var deckCacheTarget *sync.Map
	var tagCacheTarget *sync.Map
	if untypedDeckCacheTarget, ok := analyzer.deckCache.Load(source); !ok {
		deckCacheTarget = &sync.Map{}
		analyzer.deckCache.Store(source, deckCacheTarget)
	} else {
		deckCacheTarget = untypedDeckCacheTarget.(*sync.Map)
	}
	if untypedTagCacheTarget, ok := analyzer.tagCache.Load(source); !ok {
		tagCacheTarget = &sync.Map{}
		analyzer.tagCache.Store(source, tagCacheTarget)
	} else {
		tagCacheTarget = untypedTagCacheTarget.(*sync.Map)
	}
	if untypedCount, ok := deckCacheTarget.Load(info.Deck); ok {
		deckCacheTarget.Store(info.Deck, untypedCount.(int) + 1)
	} else {
		deckCacheTarget.Store(info.Deck, 1)
	}
	for _, tag := range info.Tag {
		tag = info.Deck + "-" + tag
		if untypedCount, ok := tagCacheTarget.Load(tag); ok {
			tagCacheTarget.Store(tag, untypedCount.(int) + 1)
		} else {
			tagCacheTarget.Store(tag, 1)
		}
	}
}

func (analyzer *DeckAnalyzer) Analyze(deck *ygopro_data.Deck, source string, playerName string) {
	ch := make(chan *deckInfo)
	go analyzer.fetchDeckInfo(deck, ch)
	info := <- ch
	if info.Deck == "迷之卡组" {
		analyzer.UnknownDecks = append(analyzer.UnknownDecks, unknownDeckDetail{deck, source, playerName, time.Now()})
	}
	analyzer.addDeckInfoToCache(source, info)
}

func (analyzer *DeckAnalyzer) fetchDeckInfo(deck *ygopro_data.Deck, channel chan *deckInfo) {
	resp, err := http.PostForm(analyzer.DeckIdentifierHost, url.Values{ "deck": { deck.ToYdk() } })
	var info deckInfo
	if err != nil {
		Logger.Warningf("Deck Analyzer failed fetching identifier header: %v\n", err)
		info.Deck = "No name due to network"
		channel <- &info
		return
	}
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(&info); err != nil {
		Logger.Warningf("Deck Analyzer failed fetching identifier content: %v\n", err)
		info.Deck = "No name due to parsing"
		channel <- &info
		return
	}
	channel <- &info
}

func (analyzer *DeckAnalyzer) Push(db *pg.DB) {
	var tempBuffer bytes.Buffer
	var deckBuffer bytes.Buffer
	var tagBuffer bytes.Buffer
	var unknownBuffer bytes.Buffer
	var deckValues []string
	var tagValues []string
	var unknownValues []string

	currentTime := time.Now().Format("2006-01-02")

	analyzer.deckCache.Range(func(untypedSource, untypedData interface{}) bool {
		data := untypedData.(*sync.Map)
		data.Range(func(untypedName, untypedCount interface{}) bool {
			tempBuffer.Reset()
			tempBuffer.WriteString("('")
			tempBuffer.WriteString(untypedName.(string))
			tempBuffer.WriteString("', '")
			tempBuffer.WriteString(currentTime)
			tempBuffer.WriteString("', 1, '")
			tempBuffer.WriteString(untypedSource.(string))
			tempBuffer.WriteString("', ")
			tempBuffer.WriteString(strconv.Itoa(untypedCount.(int)))
			tempBuffer.WriteString(")")
			deckValues = append(deckValues, tempBuffer.String())
			return true
		})
		return true
	})

	analyzer.tagCache.Range(func(untypedSource, untypedData interface{}) bool {
		data := untypedData.(*sync.Map)
		data.Range(func(untypedName, untypedCount interface{}) bool {
			tempBuffer.Reset()
			tempBuffer.WriteString("('")
			tempBuffer.WriteString(untypedName.(string))
			tempBuffer.WriteString("', '")
			tempBuffer.WriteString(currentTime)
			tempBuffer.WriteString("', 1, '")
			tempBuffer.WriteString(untypedSource.(string))
			tempBuffer.WriteString("', ")
			tempBuffer.WriteString(strconv.Itoa(untypedCount.(int)))
			tempBuffer.WriteString(")")
			tagValues = append(tagValues, tempBuffer.String())
			return true
		})
		return true
	})

	for _, detail := range analyzer.UnknownDecks {
		tempBuffer.Reset()
		tempBuffer.WriteString("('")
		tempBuffer.WriteString(detail.Deck.ToYdk())
		tempBuffer.WriteString("', '")
		tempBuffer.WriteString(detail.User)
		tempBuffer.WriteString("', '")
		tempBuffer.WriteString(detail.Source)
		tempBuffer.WriteString("', '")
		tempBuffer.WriteString(detail.Time.Format("2006-01-02 03:04:05"))
		tempBuffer.WriteString("')")
		unknownValues = append(unknownValues, tempBuffer.String())
	}

	analyzer.deckCache = sync.Map{}
	analyzer.tagCache = sync.Map{}
	analyzer.UnknownDecks = make([]unknownDeckDetail, 0)

	if len(deckValues) > 0 {
		deckBuffer.WriteString("insert into deck values")
		deckBuffer.WriteString(strings.Join(deckValues, ", "))
		deckBuffer.WriteString("  on conflict on constraint card_environment_deck do update set " +
			"count = deck.count + excluded.count")
		sql := deckBuffer.String()
		Logger.Debugf("Deck sql exec: %v", sql)
		if _, err := db.Exec(sql); err != nil {
			Logger.Errorf("Deck Analyzer failed pushing deck information to database: %v\n", err)
		}
	}
	if len(tagValues) > 0 {
		tagBuffer.WriteString("insert into tag values")
		tagBuffer.WriteString(strings.Join(tagValues, ", "))
		tagBuffer.WriteString("  on conflict on constraint card_environment_tag do update set " +
			"count = tag.count + excluded.count")
		sql := tagBuffer.String()
		Logger.Debugf("Tag sql exec: %v", sql)
		if _, err := db.Exec(sql); err != nil {
			Logger.Errorf("Deck Analyzer failed pushing tag information to database: %v\n", err)
		}
	}

	if len(unknownValues) > 0 {
		unknownBuffer.WriteString("insert into unknown_decks values")
		unknownBuffer.WriteString(strings.Join(unknownValues, ", "))
		sql := unknownBuffer.String()
		Logger.Debugf("Unknown sql exec: %v", sql)
		if _, err := db.Exec(sql); err != nil {
			Logger.Errorf("Deck Analyzer failed pushing unknown deck information to database: %v\n", err)
		}
	}
}


