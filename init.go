package deckard

import (
	"fmt"
	"os"
	"path"

	git "github.com/go-git/go-git/v5"
)

/// Init refreshes all resources after the UI has been started.
func StartInit(ui *DeckardUI) {
	go backgroundInit(ui)
}

func backgroundInit(ui *DeckardUI) {
	updateRepos(ui)

	//TODO compute new commit list and display it!
}

// check if repos are there, if not clones them. If they exist
// they are pulled to the latest state.
func updateRepos(ui *DeckardUI) {
	for _, prj := range ui.config.Projects {
		folder := path.Join(ui.config.CodeFolder, path.Base(prj.Repo))
		_, err := os.Stat(folder)
		if os.IsNotExist(err) {
			updateStatus(ui, fmt.Sprintf("Cloning new repo: %s (this may take a while)", prj.Repo))
			err = cloneRepo(prj, folder)
			if err != nil {
				panic(err) //TODO show error in ui instead
			}
		} else {
			updateStatus(ui, fmt.Sprintf("Pulling repo: %s", prj.Repo))
			err = pullRepo(folder)
			if err != nil {
				panic(err) //TODO show error in ui instead if this fails
			}
		}
	}
	clearStatus(ui)
}

func cloneRepo(prj ConfigProject, targetFolder string) error {
	_, err := git.PlainClone(targetFolder, false, &git.CloneOptions{
		URL:      prj.Repo,
		Progress: nil,
	})
	return err
}

func pullRepo(targetFolder string) error {
	repo, err := git.PlainOpen(targetFolder)
	if err != nil {
		return err
	}
	tree, err := repo.Worktree()
	if err != nil {
		return err
	}
	err = tree.Pull(&git.PullOptions{
		Progress: nil,
	})
	if err != git.NoErrAlreadyUpToDate {
		return err
	}
	return nil
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
