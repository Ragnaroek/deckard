package main

import (
	"fmt"

	"github.com/rivo/tview"

	"github.com/Ragnaroek/deckard"
)

func main() {

	config, err := deckard.LoadConfig()
	if err != nil {
		panic(err)
	}

	fmt.Printf("config = %#v", config)
	return

	box := tview.NewBox().
		SetBorder(true).
		SetTitle("Box Demo")
	if err := tview.NewApplication().SetRoot(box, true).Run(); err != nil {
		panic(err)
	}
}
