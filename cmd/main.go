package main

import (
	"github.com/Ragnaroek/deckard"
)

func main() {
	config, err := deckard.LoadConfig()
	if err != nil {
		panic(err)
	}

	db, err := deckard.InitDB(config)
	if err != nil {
		panic(err)
	}

	ui, err := deckard.BuildUI(config, db)
	if err != nil {
		panic(err)
	}

	// update UI from DB first (much faster than a repo update)
	err = deckard.UpdateFromDB(db, ui)
	if err != nil {
		panic(err)
	}
	deckard.UpdateFromRepo(ui)

	err = ui.Run()
	if err != nil {
		panic(err)
	}
}
