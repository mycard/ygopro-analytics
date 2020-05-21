package ygopro_analytics

import (
	"encoding/json"
	"os"

	"github.com/go-pg/pg"
)

type Configuration struct {
	Postgres           pg.Options
	DeckIdentifierHost string
	DatabasePath       string
}

var Config Configuration

func initializeConfig() {
	file, err := os.Open("./ygopro_analytics/Config.json")
	if err != nil {
		Logger.Errorf("Failed to open Config.json. %v", err)
		return
	}
	decoder := json.NewDecoder(file)
	Config = Configuration{}
	err = decoder.Decode(&Config)
	if err != nil {
		Logger.Errorf("Failed to load config: %v", err)
	}
}
