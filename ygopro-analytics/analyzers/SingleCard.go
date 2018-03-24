package analyzers

import (
	"github.com/iamipanda/ygopro-data"
	"github.com/go-pg/pg"
	"time"
	"bytes"
	"strings"
	"strconv"
)

type SingleCardAnalyzer struct {
	cache map[string]singleCardSourceData
	environment *ygopro_data.Environment
}

func NewSingleCardAnalyzer(environment *ygopro_data.Environment) SingleCardAnalyzer {
	return SingleCardAnalyzer{make(map[string]singleCardSourceData), environment}
}

type singleCardSourceData struct{
	monster map[int]*singleCardData
	spell map[int]*singleCardData
	trap map[int]*singleCardData
	side map[int]*singleCardData
	ex map[int]*singleCardData
}

func newSingleCardSourceData() singleCardSourceData {
	return singleCardSourceData{make(map[int]*singleCardData), make(map[int]*singleCardData), make(map[int]*singleCardData), make(map[int]*singleCardData), make(map[int]*singleCardData)}
}

type singleCardData struct {
	frequency int
	numbers int
	putone int
	puttwo int
	putthree int
	putoverthree int
}

func addSingleCardDataTo(target *map[int]*singleCardData, id int, count int) {
	var data *singleCardData
	var ok bool
	if data, ok = (*target)[id]; !ok {
		data = &singleCardData{0,0,0,0,0,0}
		(*target)[id] = data
	}
	switch count {
	case 0:
		return
	case 1:
		data.putone += 1
	case 2:
		data.puttwo += 1
	case 3:
		data.putthree += 1
	default:
		data.putoverthree += 1
	}
	data.numbers += 1
	data.frequency += count
}

func (analyzer *SingleCardAnalyzer) Analyze(deck *ygopro_data.Deck, source string) {
	var target singleCardSourceData
	var ok bool
	if target, ok = analyzer.cache[source]; !ok {
		target = newSingleCardSourceData()
		analyzer.cache[source] = target
	}
	for id, count := range deck.ClassifiedMain {
		if card, ok := analyzer.environment.GetCard(id); ok {
			if card.IsType("monster") {
				addSingleCardDataTo(&target.monster, id, count)
			} else if card.IsType("spell") {
				addSingleCardDataTo(&target.spell, id, count)
			} else if card.IsType("trap") {
				addSingleCardDataTo(&target.trap, id, count)
			}
		}
	}
	for id, count := range deck.ClassifiedSide {
		addSingleCardDataTo(&target.side, id, count)
	}
	for id, count := range deck.ClassifiedEx {
		addSingleCardDataTo(&target.ex, id, count)
	}
}

func (analyzer *SingleCardAnalyzer) Push(db *pg.DB) {
	var buffer bytes.Buffer
	originData := make([]string, 0)
	currentTime := time.Now().Format("2006-01-02")
	for source, sourceData := range analyzer.cache {
		originData = append(originData, generateSingleCardSourceSQL(source, currentTime, "monster", sourceData.monster))
		originData = append(originData, generateSingleCardSourceSQL(source, currentTime, "spell", sourceData.spell))
		originData = append(originData, generateSingleCardSourceSQL(source, currentTime, "trap", sourceData.trap))
		originData = append(originData, generateSingleCardSourceSQL(source, currentTime, "side", sourceData.side))
		originData = append(originData, generateSingleCardSourceSQL(source, currentTime, "ex", sourceData.ex))
	}
	data := make([]string, 0)
	for _, item := range originData {
		if len(item) > 0 {
			data = append(data, item)
		}
	}
	analyzer.cache = make(map[string]singleCardSourceData)
	if len(data) == 0 {
		return
	}
	buffer.WriteString("insert into single values")
	buffer.WriteString(strings.Join(data, ", "))
	buffer.WriteString("  on conflict on constraint card_environment_single do update set " +
		"frequency = single.frequency + excluded.frequency, numbers = single.numbers + excluded.numbers, " +
		"putone = single.putone + excluded.putone, puttwo = single.puttwo + excluded.puttwo, " +
		"putthree = single.putthree + excluded.putthree, putoverthree = single.putoverthree + excluded.putoverthree")
	if _, err := db.Exec(buffer.String()); err != nil {
		Logger.Errorf("Single Analyzer failed pushing to database: %v\n", err)
	}
}

func generateSingleCardSourceSQL(source string, time string, category string, target map[int]*singleCardData) string {
	value := make([]string, 0)
	var buffer bytes.Buffer
	for id, data := range target {
		value = append(value, generateSingleCardDataValueSQL(source, id, category, time, data, buffer))
	}
	return strings.Join(value, ", ")
}

func generateSingleCardDataValueSQL(source string, id int, category string, time string, data *singleCardData, buffer bytes.Buffer) string {
	buffer.Reset()
	buffer.WriteString("(")
	buffer.WriteString(strconv.Itoa(id))
	buffer.WriteString(", '")
	buffer.WriteString(category)
	buffer.WriteString("', '")
	buffer.WriteString(time)
	buffer.WriteString("', 1, '")
	buffer.WriteString(source)
	buffer.WriteString("', ")
	buffer.WriteString(strconv.Itoa(data.frequency))
	buffer.WriteString(", ")
	buffer.WriteString(strconv.Itoa(data.numbers))
	buffer.WriteString(", ")
	buffer.WriteString(strconv.Itoa(data.putone))
	buffer.WriteString(", ")
	buffer.WriteString(strconv.Itoa(data.puttwo))
	buffer.WriteString(", ")
	buffer.WriteString(strconv.Itoa(data.putthree))
	buffer.WriteString(", ")
	buffer.WriteString(strconv.Itoa(data.putoverthree))
	buffer.WriteString(")")
	return buffer.String()
}