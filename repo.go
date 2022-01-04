package deckard

import (
	"fmt"
	"os"
	"path"
	"sort"
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
	since := time.Now().Add(14 * -24 * time.Hour)
	for prj, repo := range repos {
		iter, err := repo.Log(&git.LogOptions{All: true, Since: &since})
		if err != nil {
			panic(err) //TODO show error in UI
		}
		iter.ForEach(func(commit *object.Commit) error {
			commits = append(commits, &Commit{
				Project:    prj,
				Hash:       commit.Hash.String(),
				Message:    commit.Message,
				Author:     commit.Author.Name,
				AuthorWhen: commit.Author.When,
			})
			return nil
		})
	}

	sort.Slice(commits, func(i, j int) bool {
		return commits[i].AuthorWhen.Before(commits[j].AuthorWhen)
	})

	ui.app.QueueUpdateDraw(func() {
		ui.UpdateCommits(commits)
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
