package main

import (
	"github.com/Ragnaroek/deckard"
)

func main() {
	config, err := deckard.LoadConfig()
	if err != nil {
		panic(err)
	}

	ui, err := deckard.BuildUI(config)
	if err != nil {
		panic(err)
	}
	err = ui.Run()
	if err != nil {
		panic(err)
	}
}
