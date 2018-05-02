package ygopro_analytics

import (
	"github.com/gin-gonic/gin"
	"github.com/iamipanda/ygopro-data"
	"github.com/op/go-logging"
	"./analyzers"
	"os"
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
		deck := ygopro_data.LoadYdkFromString(deckString)
		Analyze(&deck, source)
		context.String(200, "analyzing")
	})

	router.POST("/reload", func(context *gin.Context) {

	})

	router.Run(":8081")
}

// ===================Logger===================
var Logger = logging.MustGetLogger("standard")
var NormalLoggingBackend logging.Backend
func initializeLogger()  {
	format := logging.MustStringFormatter(
		`%{color} %{id:05x} %{time:15:04:05.000} ▶ %{level:.4s}%{color:reset} %{message} from [%{shortfunc}] `,
	)
	backendPrototype := logging.NewLogBackend(os.Stderr, "", 0)
	fBackend := logging.NewBackendFormatter(backendPrototype, format)
	lBackend := logging.AddModuleLevel(fBackend)
	lBackend.SetLevel(logging.INFO, "")
	NormalLoggingBackend = lBackend
	logging.SetBackend(NormalLoggingBackend)
	analyzers.Logger = Logger
}
