package deckard

import (
	"database/sql"
	"path"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB(config *Config) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path.Join(config.CodeFolder, "deckard.db"))
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS fetch_states (project TEXT NOT NULL, since INTEGER)")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS index_fetch_states ON fetch_states (project, since)")
	if err != nil {
		return nil, err
	}

	return db, nil
}

func GetFetchState(db *sql.DB, project string) (*time.Time, error) {
	row := db.QueryRow("SELECT since FROM fetch_states where project = ?", project)
	if row == nil {
		return nil, nil
	}
	var ts sql.NullInt64
	err := row.Scan(&ts)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	if !ts.Valid {
		return nil, nil
	}
	time := time.UnixMilli(ts.Int64)
	return &time, nil
}

func UpdateFetchState(db *sql.DB, project string, t *time.Time) error {
	_, err := db.Exec("INSERT INTO fetch_states (project, since) VALUES (?1, ?2) ON CONFLICT(project, since) DO UPDATE SET since = ?2", project, t.UnixMilli())
	if err != nil {
		return err
	}
	return nil
}

func UpdateFromDB(ui *DeckardUI) error {

	// TODO read fetch states from db and fetch from there, record them in DB

	return nil
}
