package main

import (
	"./ygopro-analytics"
	"github.com/jasonlvhit/gocron"
)

func main() {
	ygopro_analytics.Initialize()
	gocron.Every(5).Minutes().Do(ygopro_analytics.Push)
	go gocron.Start()
	ygopro_analytics.StartServer()
}