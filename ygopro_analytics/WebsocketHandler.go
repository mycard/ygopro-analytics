package ygopro_analytics

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	ygopro_data "github.com/iamipanda/ygopro-data"
)

var websocket_upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func WebsocketMain(context *gin.Context) {
	websocket, err := websocket_upgrader.Upgrade(context.Writer, context.Request, nil)
	if err != nil {
		Logger.Error("Failed to upgrade: ")
		Logger.Error(err)
		return
	}
	websocket.WriteMessage(1, []byte("Start to accept deck post."))
	for {
		_, messageBytes, err := websocket.ReadMessage()
		if err != nil {
			Logger.Error("Failed to receive message: ")
			Logger.Error(err)
			break
		}
		var message map[string]string = make(map[string]string)
		json.Unmarshal(messageBytes, &message)
		source, sourceOk := message["arena"]
		deckString, deckStringOk := message["deck"]
		playerName, playerNameOk := message["playername"]
		if !sourceOk {
			source = "unknown"
		}
		if !deckStringOk {
			deckString = ""
		}
		if !playerNameOk {
			playerName = "unknown"
		}
		deck := ygopro_data.LoadYdkFromString(deckString)
		Analyze(&deck, source, playerName)
		err = websocket.WriteMessage(1, []byte("analyzing"))
		if err != nil {
			Logger.Error("Failed to response message: ")
			Logger.Error(err)
			break
		}
	}
	websocket.Close()
}
