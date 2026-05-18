package ygopro_analytics

import (
	"encoding/base64"
	"encoding/json"
	"main/ygopro_analytics/analyzers"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	ygopro_data "github.com/iamipanda/ygopro-data"
	"github.com/op/go-logging"
)

func Initialize() {
	initializeConfig()
	ygopro_data.DatabasePath = Config.DatabasePath[0]
	if len(Config.DatabasePath) > 1 {
		for _, path := range Config.DatabasePath[1:] {
			ygopro_data.GetEnvironment("zh-CN").AppendFolder(path)
		}
	}
	initializeLogger()
	initializeAnalyzers()
	initializeDatabaseConnection()
	Logger.Info("Analytics server started.")
}

func StartServer() {
	router := gin.New()
	router.Use(gin.Recovery())
	if gin.IsDebugging() {
		router.Use(gin.Logger())
	}
	router.GET("/", func(context *gin.Context) {
		context.String(200, "MCPro Analyzer is working.")
	})

	router.POST("/push", func(context *gin.Context) {
		Push()
		context.String(200, "ok")
	})

	router.POST("/deck", func(context *gin.Context) {
		source := context.DefaultPostForm("arena", "unknown")
		deckString := context.DefaultPostForm("deck", "")
		playerName := context.DefaultPostForm("playername", "Unknown")
		deck := ygopro_data.LoadYdkFromString(deckString)
		AnalyzeDeck(&deck, source, playerName)
		context.String(200, "analyzed")
	})

	router.POST("/message", func(context *gin.Context) {
		report := analyzers.MatchReport{}
		report.Arena = context.DefaultPostForm("arena", "unknown")
		report.UsernameA = context.DefaultPostForm("usernameA", "Unknown")
		report.UsernameB = context.DefaultPostForm("usernameB", "Unknown")
		report.UserdeckA = ygopro_data.LoadYdkFromString(context.DefaultPostForm("userdeckA", ""))
		report.UserdeckB = ygopro_data.LoadYdkFromString(context.DefaultPostForm("userdeckB", ""))
		report.UserscoreA, _ = strconv.Atoi(context.DefaultPostForm("userscoreA", "-5"))
		report.UserscoreB, _ = strconv.Atoi(context.DefaultPostForm("userscoreB", "-5"))

		firstList := context.DefaultPostForm("first", "[]")
		var first []string
		json.Unmarshal([]byte(firstList), &first)
		report.First = first

		winsList := context.DefaultPostForm("wins", "[]")
		var wins []string
		json.Unmarshal([]byte(winsList), &wins)
		report.Wins = wins

		replaysList := context.DefaultPostForm("replays", "[]")
		var replaysBase64 []string
		json.Unmarshal([]byte(replaysList), &replaysBase64)
		for _, b64 := range replaysBase64 {
			raw, err := base64.StdEncoding.DecodeString(b64)
			if err != nil {
				Logger.Warningf("Failed to decode replay base64: %v", err)
				continue
			}
			replay, err := ygopro_data.ReadReplayFromBytes(raw)
			if err != nil {
				Logger.Warningf("Failed to parse replay: %v", err)
				continue
			}
			report.Replays = append(report.Replays, *replay)
		}

		AnalyzeMatch(report)
		context.String(200, "analyzed")
	})

	router.POST("/reload", func(context *gin.Context) {
		Logger.Info("Reloading database.")
		ygopro_data.LoadAllEnvironmentCards()
		context.String(200, "ok")
	})

	router.PATCH("/reload", func(context *gin.Context) {
		Logger.Info("Reloading database.")
		ygopro_data.LoadAllEnvironmentCards()
		context.String(200, "ok")
	})

	router.Run(":8081")
}

// ===================Logger===================
var Logger = logging.MustGetLogger("standard")
var NormalLoggingBackend logging.Backend

func initializeLogger() {
	format := logging.MustStringFormatter(
		`%{color} %{id:05x} %{time:15:04:05.000} ▶ %{level:.4s}%{color:reset} %{message} from [%{shortfunc}] `,
	)
	backendPrototype := logging.NewLogBackend(os.Stderr, "", 0)
	fBackend := logging.NewBackendFormatter(backendPrototype, format)
	lBackend := logging.AddModuleLevel(fBackend)
	if gin.IsDebugging() {
		lBackend.SetLevel(logging.DEBUG, "")
	} else {
		lBackend.SetLevel(logging.INFO, "")
	}
	NormalLoggingBackend = lBackend
	logging.SetBackend(NormalLoggingBackend)
	analyzers.Logger = Logger
}
