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
)

type DeckAnalyzer struct {
	deckCache map[string]map[string]int
	tagCache map[string]map[string]int
	DeckIdentifierHost string
}

type deckInfo struct {
	Deck string
	Tag []string
}

func NewDeckAnalyzer(deckIdentifierHost string) DeckAnalyzer {
	return DeckAnalyzer { make(map[string]map[string]int), make(map[string]map[string]int), deckIdentifierHost }
}

func (analyzer *DeckAnalyzer) addDeckInfoToCache(source string, info *deckInfo) {
	var count int
	var ok bool
	var deckCacheTarget map[string]int
	var tagCacheTarget map[string]int
	if deckCacheTarget, ok = analyzer.deckCache[source]; !ok {
		deckCacheTarget = make(map[string]int)
		analyzer.deckCache[source] = deckCacheTarget
	}
	if tagCacheTarget, ok = analyzer.tagCache[source]; !ok {
		tagCacheTarget = make(map[string]int)
		analyzer.tagCache[source] = tagCacheTarget
	}
	if count, ok = deckCacheTarget[info.Deck]; ok {
		deckCacheTarget[info.Deck] = count + 1
	} else {
		deckCacheTarget[info.Deck] = 1
	}
	for _, tag := range info.Tag {
		if count, ok = tagCacheTarget[tag]; ok {
			tagCacheTarget[tag] = count + 1
		} else {
			tagCacheTarget[tag] = 1
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
	if err != nil {
		Logger.Warningf("Deck Analyzer failed fetching identifier header: %v\n", err)
	}
	decoder := json.NewDecoder(resp.Body)
	var info deckInfo
	if err = decoder.Decode(&info); err != nil {
		Logger.Warningf("Deck Analyzer failed fetching identifier content: %v\n", err)
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
	for source, data := range analyzer.deckCache {
		for name, count := range data {
			tempBuffer.Reset()
			tempBuffer.WriteString("('")
			tempBuffer.WriteString(name)
			tempBuffer.WriteString("', '")
			tempBuffer.WriteString(currentTime)
			tempBuffer.WriteString("', 1, '")
			tempBuffer.WriteString(source)
			tempBuffer.WriteString("', ")
			tempBuffer.WriteString(strconv.Itoa(count))
			tempBuffer.WriteString(")")
			deckValues = append(deckValues, tempBuffer.String())
		}
	}
	for source, data := range analyzer.tagCache {
		for name, count := range data {
			tempBuffer.Reset()
			tempBuffer.WriteString("('")
			tempBuffer.WriteString(name)
			tempBuffer.WriteString("', '")
			tempBuffer.WriteString(currentTime)
			tempBuffer.WriteString("', 1, '")
			tempBuffer.WriteString(source)
			tempBuffer.WriteString("', ")
			tempBuffer.WriteString(strconv.Itoa(count))
			tempBuffer.WriteString(")")
			tagValues = append(tagValues, tempBuffer.String())
		}
	}
	analyzer.deckCache = make(map[string]map[string]int)
	analyzer.tagCache = make(map[string]map[string]int)
	if len(deckValues) > 0 {
		deckBuffer.WriteString("insert into deck values")
		deckBuffer.WriteString(strings.Join(deckValues, ", "))
		deckBuffer.WriteString("  on conflict on constraint card_environment_deck do update set " +
			"count = deck.count + excluded.count")
		if _, err := db.Exec(deckBuffer.String()); err != nil {
			Logger.Errorf("Deck Analyzer failed pushing deck information to database: %v\n", err)
		}
	}
	if len(tagValues) > 0 {
		tagBuffer.WriteString("insert into tag values")
		tagBuffer.WriteString(strings.Join(tagValues, ", "))
		tagBuffer.WriteString("  on conflict on constraint card_environment_tag do update set " +
			"count = tag.count + excluded.count")
		if _, err := db.Exec(tagBuffer.String()); err != nil {
			Logger.Errorf("Deck Analyzer failed pushing tag information to database: %v\n", err)
		}
	}
}


