package analyzers

import (
	"encoding/json"
	"github.com/iamipanda/ygopro-data"
	"net/http"
	"net/url"
)

func fetchDeckInfo(identifierHost string, deck *ygopro_data.Deck, channel chan *deckInfo) {
	resp, err := http.PostForm(identifierHost, url.Values{ "deck": { deck.ToYdk() } })
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

const MATCH_RESULT_PLAYER_A_WIN int = 1
const MATCH_RESULT_PLAYER_B_WIN int = -1
const MATCH_RESULT_PLAYERS_DRAW int = 0
const MATCH_RESULT_PLAYERS_DROP int = 999

func JudgeWinLose(playerAScore int, playerBScore int) (result int) {
	switch {
	case playerAScore == -5 || playerBScore == -5:
		return MATCH_RESULT_PLAYERS_DROP
	case playerAScore == -9:
		return MATCH_RESULT_PLAYER_B_WIN
	case playerBScore == -9:
		return MATCH_RESULT_PLAYER_A_WIN
	case playerAScore == playerBScore:
		return MATCH_RESULT_PLAYERS_DRAW
	case playerAScore > playerBScore:
		return MATCH_RESULT_PLAYER_A_WIN
	case playerAScore < playerBScore:
		return MATCH_RESULT_PLAYER_B_WIN
	default:
		return MATCH_RESULT_PLAYERS_DROP
	}
}