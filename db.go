package deckard

import (
	"database/sql"
	"path"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	STATE_NEW      = "new"
	STATE_REVIEWED = "rev"
)

type CommitState string

var migrations = [][]string{
	{
		"CREATE TABLE IF NOT EXISTS fetch_states (project TEXT NOT NULL, since INTEGER)",
		"CREATE UNIQUE INDEX IF NOT EXISTS index_fetch_states ON fetch_states (project)",
		"CREATE TABLE IF NOT EXISTS commits (project TEXT NOT NULL, hash TEXT NOT NULL, message TEXT NOT NULL, author_name TEXT NOT NULL, committer_name TEXT NOT NULL, commit_when INTEGER, slat_score INTEGER, state TEXT NOT NULL, comment TEXT)",
		"CREATE UNIQUE INDEX IF NOT EXISTS index_commits ON commits (project, hash)",
	},
}

func InitDB(config *Config) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path.Join(config.CodeFolder, "deckard.db"))
	if err != nil {
		return nil, err
	}

	row := db.QueryRow("SELECT Count(*) FROM sqlite_master WHERE type='table' AND name='migrations'")
	var tabCount int
	err = row.Scan(&tabCount)
	if err != nil {
		return nil, err
	}

	var version int
	if tabCount == 0 {
		_, errCreate := db.Exec("CREATE TABLE IF NOT EXISTS migrations (version INTEGER NOT NULL)")
		if errCreate != nil {
			return nil, errCreate
		}
		_, errCreate = db.Exec("INSERT INTO migrations (version) VALUES (0)")
		if errCreate != nil {
			return nil, errCreate
		}
		version = 0
	} else {
		row := db.QueryRow("SELECT version FROM migrations")
		errVersion := row.Scan(&version)
		if errVersion != nil {
			return nil, errVersion
		}
	}

	for i := version; i < len(migrations); i++ {
		migrationList := migrations[i]
		for _, migration := range migrationList {
			_, err := db.Exec(migration)
			if err != nil {
				return nil, err
			}
		}
		_, err := db.Exec("UPDATE migrations SET version = ?", i+1)
		if err != nil {
			return nil, err
		}
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
		_, err := db.Exec("INSERT OR IGNORE INTO commits (project, hash, message, author_name, committer_name, commit_when, slat_score, state, comment) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9)",
			commit.Project, commit.Hash, commit.Subject, commit.AuthorName, commit.CommitterName, commit.CommitWhen.UnixMilli(), commit.SlatScore, commit.State, commit.Comment)
		if err != nil {
			return err
		}
	}
	return nil
}

func UpdateCommitState(db *sql.DB, project, hash string, state CommitState) error {
	_, err := db.Exec("UPDATE commits SET state = ?1 WHERE project = ?2 AND hash = ?3", state, project, hash)
	if err != nil {
		return err
	}
	return nil
}

func UpdateFromDB(db *sql.DB, ui *DeckardUI) error {

	rows, err := db.Query("SELECT project, hash, message, author_name, committer_name, commit_when, slat_score, state, comment FROM commits WHERE state = ?1", STATE_NEW)
	if err != nil {
		return err
	}

	commits := make([]*Commit, 0)
	var project string
	var hash string
	var message string
	var authorName string
	var committerName string
	var commitWhen int64
	var slatScore int
	var state string
	var comment *string
	for rows.Next() {
		err = rows.Scan(&project, &hash, &message, &authorName, &committerName, &commitWhen, &slatScore, &state, &comment)
		if err != nil {
			return err
		}
		commits = append(commits, &Commit{
			Project:       project,
			Hash:          hash,
			Subject:       message,
			AuthorName:    authorName,
			CommitterName: committerName,
			CommitWhen:    time.UnixMilli(commitWhen),
			SlatScore:     slatScore,
			State:         state,
			Comment:       comment,
		})
	}

	ui.AddCommits(commits)

	return nil
}
