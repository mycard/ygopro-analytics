package main

import (
	"./ygopro-analytics"
	"github.com/jasonlvhit/gocron"
)

func main() {
	ygopro_analytics.Initialize()
	ygopro_analytics.StartServer()
	gocron.Every(5).Minutes().Do(ygopro_analytics.Push)
}