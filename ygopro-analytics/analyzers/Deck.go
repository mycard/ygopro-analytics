package analyzers

import (
	"github.com/iamipanda/ygopro-data"
	"github.com/go-pg/pg"
	"net/http"
	"net/url"
	"github.com/gin-gonic/gin/json"
	"bytes"
	"time"
	"strings"
	"strconv"
	"sync"
)

// deckCache map[string]map[string]int
// tagCache map[string]map[string]int
type DeckAnalyzer struct {
	deckCache sync.Map
	tagCache sync.Map
	DeckIdentifierHost string
}

type deckInfo struct {
	Deck string
	Tag []string
}

func NewDeckAnalyzer(deckIdentifierHost string) DeckAnalyzer {
	return DeckAnalyzer { sync.Map{}, sync.Map{}, deckIdentifierHost }
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

func (analyzer *DeckAnalyzer) Analyze(deck *ygopro_data.Deck, source string) {
	ch := make(chan *deckInfo)
	go analyzer.fetchDeckInfo(deck, ch)
	info := <- ch
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
	var deckValues []string
	var tagValues []string
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
	analyzer.deckCache = sync.Map{}
	analyzer.tagCache = sync.Map{}
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
}


