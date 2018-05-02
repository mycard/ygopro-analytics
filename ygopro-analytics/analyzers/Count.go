package analyzers

import (
	"github.com/iamipanda/ygopro-data"
	"github.com/go-pg/pg"
	"bytes"
	"strings"
	"strconv"
	"time"
	"sync"
)

// cache map[string]int
type CountAnalyzer struct {
	cache sync.Map
}

func NewCountAnalyzer() CountAnalyzer {
	return CountAnalyzer{ sync.Map{} }
}

func (analyzer *CountAnalyzer) Analyze(deck *ygopro_data.Deck, source string) {
	if untypedCount, ok := analyzer.cache.Load(source); ok {
		analyzer.cache.Store(source, untypedCount.(int) + 1)
	} else {
		analyzer.cache.Store(source, 1)
	}
}

func (analyzer *CountAnalyzer) Push(db *pg.DB) {
	var buffer bytes.Buffer
	data := make([]string, 0)
	currentTime := time.Now().Format("2006-01-02")
	analyzer.cache.Range(func(untypedSource, untypedCount interface{}) bool {
		source := untypedSource.(string)
		count := untypedCount.(int)
		buffer.WriteString("('")
		buffer.WriteString(currentTime)
		buffer.WriteString("', 1, '")
		buffer.WriteString(source)
		buffer.WriteString("', ")
		buffer.WriteString(strconv.Itoa(count))
		buffer.WriteString(")")
		data = append(data, buffer.String())
		buffer.Reset()
		return true
	})
	analyzer.cache = sync.Map{}
	if len(data) == 0 {
		return
	}
	buffer.Reset()
	buffer.WriteString("insert into counter values ")
	buffer.WriteString(strings.Join(data, ", "))
	buffer.WriteString(" on conflict on constraint counter_environment do update set count = counter.count + excluded.count")
	if _, err := db.Exec(buffer.String()); err != nil {
		Logger.Errorf("Counter failed pushing to database: %v\n", err)
	}
}
