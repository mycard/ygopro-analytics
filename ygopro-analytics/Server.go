package ygopro_analytics

import (
	"./analyzers"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/iamipanda/ygopro-data"
	"github.com/op/go-logging"
	"os"
	"strconv"
)

func Initialize() {
	initializeConfig()
	ygopro_data.DatabasePath = Config.DatabasePath
	ygopro_data.InitializeStaticEnvironment()
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
		Analyze(&deck, source, playerName)
		context.String(200, "analyzing")
	})

	router.POST("/message", func(context *gin.Context) {
		source := context.DefaultPostForm("arena", "unknown")
		deckAString := context.DefaultPostForm("userdeckA", "")
		deckBString := context.DefaultPostForm("userdeckB", "")
		playerAName := context.DefaultPostForm("usernameA", "Unknown")
		playerBName := context.DefaultPostForm("usernameB", "Unknown")
		firstList := context.DefaultPostForm("first", "[]")
		var first []string
		json.Unmarshal([]byte(firstList), &first)
		deckA := ygopro_data.LoadYdkFromString(deckAString)
		deckB := ygopro_data.LoadYdkFromString(deckBString)
		playerAScore, errA := strconv.Atoi(context.DefaultPostForm("userscoreA", "-5"))
		playerBScore, errB := strconv.Atoi(context.DefaultPostForm("userscoreB", "-5"))
		if errA != nil || errB != nil {
			Logger.Warning("Can't recognize score message.")
			context.String(504, "wrong score message")
			return
		}
		AnalyzeMessage(playerAName, playerBName, &deckA, &deckB, playerAScore, playerBScore, source, first)
		context.String(200, "analyzing")
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
