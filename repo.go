package deckard

import (
	"fmt"
	"os"
	"path"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

/// RepoUpdate refreshes all repo based resources after the UI has been started.
func UpdateFromRepo(ui *DeckardUI) {
	go backgroundUpdate(ui)
}

func backgroundUpdate(ui *DeckardUI) {
	repos := updateRepos(ui)
	updateCommits(ui, repos)
}

func updateCommits(ui *DeckardUI, repos map[string]*git.Repository) {
	updateStatus(ui, "Updating commit list...")

	commits := make([]*Commit, 0)

	for prj, repo := range repos {

		since, err := GetFetchState(ui.db, prj)
		if err != nil {
			panic(err) //TODO show error in UI
		}
		if since == nil {
			fallback := time.Now().Add(14 * -24 * time.Hour)
			since = &fallback
		}

		iter, err := repo.Log(&git.LogOptions{All: true, Since: since})
		if err != nil {
			panic(err) //TODO show error in UI
		}

		var lastCommitTime = since
		repoCommits := make([]*Commit, 0)
		iter.ForEach(func(commit *object.Commit) error {
			repoCommits = append(repoCommits, &Commit{
				Project:    prj,
				Hash:       commit.Hash.String(),
				Message:    commit.Message,
				Author:     commit.Author.Name,
				AuthorWhen: commit.Author.When,
				State:      STATE_NEW,
				SlatScore:  slatScore(commit),
			})
			if commit.Author.When.After(*lastCommitTime) {
				lastCommitTime = &commit.Author.When
			}
			return nil
		})

		err = StoreCommits(ui.db, repoCommits)
		if err != nil {
			panic(err) //TODO show error in UI
		}

		commits = append(commits, repoCommits...)

		err = UpdateFetchState(ui.db, prj, lastCommitTime)
		if err != nil {
			panic(err) //TODO show error in UI
		}
	}

	ui.app.QueueUpdateDraw(func() {
		ui.AddCommits(commits)
	})

	clearStatus(ui)
}

// check if repos are there, if not clones them. If they exist
// they are pulled to the latest state.
func updateRepos(ui *DeckardUI) map[string]*git.Repository {
	repos := make(map[string]*git.Repository)
	for prj, conf := range ui.config.Projects {
		folder := path.Join(ui.config.CodeFolder, path.Base(conf.Repo))
		_, err := os.Stat(folder)
		if os.IsNotExist(err) {
			updateStatus(ui, fmt.Sprintf("Cloning new repo: %s (this may take a while)", conf.Repo))
			repo, err := cloneRepo(conf, folder)
			if err != nil {
				panic(err) //TODO show error in ui instead
			}
			repos[prj] = repo
		} else {
			updateStatus(ui, fmt.Sprintf("Pulling repo: %s", conf.Repo))
			repo, err := pullRepo(folder)
			if err != nil {
				panic(err) //TODO show error in ui instead if this fails
			}
			repos[prj] = repo
		}
	}
	clearStatus(ui)
	return repos
}

func cloneRepo(prj ConfigProject, targetFolder string) (*git.Repository, error) {
	return git.PlainClone(targetFolder, false, &git.CloneOptions{
		URL:      prj.Repo,
		Progress: nil,
	})
}

func pullRepo(targetFolder string) (*git.Repository, error) {
	repo, err := git.PlainOpen(targetFolder)
	if err != nil {
		return nil, err
	}
	tree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}
	err = tree.Pull(&git.PullOptions{
		Progress: nil,
	})
	if err != git.NoErrAlreadyUpToDate {
		return nil, err
	}
	return repo, nil
}

func updateStatus(ui *DeckardUI, text string) {
	ui.app.QueueUpdateDraw(func() {
		ui.UpdateStatus(text)
	})
}

func clearStatus(ui *DeckardUI) {
	ui.app.QueueUpdateDraw(func() {
		ui.ClearStatus()
	})
}
