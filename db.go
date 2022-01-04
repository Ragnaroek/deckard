package deckard

import (
	"database/sql"
	"path"

	_ "github.com/mattn/go-sqlite3"
)

func UpdateFromDB(ui *DeckardUI) error {
	db, err := sql.Open("sqlite3", path.Join(ui.config.CodeFolder, "deckard.db"))
	if err != nil {
		return err
	}
	// TODO Read state from db and update ui
	defer db.Close()
	return nil
}
