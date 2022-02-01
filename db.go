package deckard

import (
	"database/sql"
	"path"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	STATE_NEW = "new"
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
	_, err = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS index_fetch_states ON fetch_states (project)")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS commits (project TEXT NOT NULL, hash TEXT NOT NULL, message TEXT NOT NULL, author TEXT NOT NULL, author_when INTEGER, slat_score INTEGER, state TEXT NOT NULL, comment TEXT)")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS index_commits ON commits (project, hash)")
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
	_, err := db.Exec("INSERT INTO fetch_states (project, since) VALUES (?1, ?2) ON CONFLICT(project) DO UPDATE SET since = ?2", project, t.UnixMilli())
	if err != nil {
		return err
	}
	return nil
}

func StoreCommits(db *sql.DB, commits []*Commit) error {
	for _, commit := range commits {
		_, err := db.Exec("INSERT OR IGNORE INTO commits (project, hash, message, author, author_when, slat_score, state, comment) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8)",
			commit.Project, commit.Hash, commit.Message, commit.Author, commit.AuthorWhen.UnixMilli(), commit.SlatScore, commit.State, commit.Comment)
		if err != nil {
			return err
		}
	}
	return nil
}

func UpdateFromDB(db *sql.DB, ui *DeckardUI) error {

	rows, err := db.Query("SELECT project, hash, message, author, author_when, slat_score, state, comment FROM commits")
	if err != nil {
		return err
	}

	commits := make([]*Commit, 0)
	var project string
	var hash string
	var message string
	var author string
	var authorWhen int64
	var slatScore int
	var state string
	var comment *string
	for rows.Next() {
		err = rows.Scan(&project, &hash, &message, &author, &authorWhen, &slatScore, &state, &comment)
		if err != nil {
			return err
		}
		commits = append(commits, &Commit{
			Project:    project,
			Hash:       hash,
			Message:    message,
			Author:     author,
			AuthorWhen: time.UnixMilli(authorWhen),
			SlatScore:  slatScore,
			State:      state,
			Comment:    comment,
		})
	}

	ui.AddCommits(commits)

	return nil
}
