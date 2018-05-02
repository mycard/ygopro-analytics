package analyzers

import (
	"github.com/iamipanda/ygopro-data"
	"github.com/go-pg/pg"
	"bytes"
	"strings"
	"strconv"
	"time"
)

type CountAnalyzer struct {
	cache map[string]int
}

func NewCountAnalyzer() CountAnalyzer {
	return CountAnalyzer{ make(map[string]int) }
}

func (analyzer *CountAnalyzer) Analyze(deck *ygopro_data.Deck, source string) {
	if count, ok := analyzer.cache[source]; ok {
		analyzer.cache[source] = count + 1
	} else {
		analyzer.cache[source] = 1
	}
}

func (analyzer *CountAnalyzer) Push(db *pg.DB) {
	var buffer bytes.Buffer
	data := make([]string, 0)
	currentTime := time.Now().Format("2006-01-02")
	for source, count := range analyzer.cache {
		buffer.WriteString("('")
		buffer.WriteString(currentTime)
		buffer.WriteString("', 1, '")
		buffer.WriteString(source)
		buffer.WriteString("', ")
		buffer.WriteString(strconv.Itoa(count))
		buffer.WriteString(")")
		data = append(data, buffer.String())
		buffer.Reset()
	}
	analyzer.cache = make(map[string]int)
	if len(data) == 0 {
		return
	}
	buffer.Reset()
	buffer.WriteString("insert into count values ")
	buffer.WriteString(strings.Join(data, ", "))
	buffer.WriteString(" on conflict on constraint count_environment do update set count = counter.count + excluded.count")
	if _, err := db.Exec(buffer.String()); err != nil {
		Logger.Errorf("Counter failed pushing to database: %v\n", err)
	}
}
