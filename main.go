package main

import (
	"./ygopro-analytics"
)

func main() {
	ygopro_analytics.Initialize()
	ygopro_analytics.StartServer()
}