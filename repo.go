package deckard

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
)

/// RepoUpdate refreshes all repo based resources after the UI has been started.
func UpdateFromRepo(ui *DeckardUI) {
	go backgroundUpdate(ui)
}

func backgroundUpdate(ui *DeckardUI) {
	updateRepos(ui)
	updateCommits(ui)
}

func updateCommits(ui *DeckardUI) {
	updateStatus(ui, "Updating commit list...")

	commits := make([]*Commit, 0)

	for prj, conf := range ui.config.Projects {
		since, err := GetFetchState(ui.db, prj)
		if err != nil {
			panic(err) //TODO show error in UI
		}
		if since == nil {
			fallback := time.Now().Add(60 * -24 * time.Hour)
			since = &fallback
		}

		folder := repoFolder(ui.config, conf)
		log, err := logRepo(folder, since)
		if err != nil {
			panic(err)
		}

		var lastCommitTime = since
		repoCommits := make([]*Commit, 0)

		for _, commit := range log {
			/*slatScore, err := slatScore(commit)
			if err != nil {
				return err
			}*/
			commit.Project = prj
			commit.State = STATE_NEW

			// TODO go back to AuthorWhen???
			if commit.CommitWhen.After(*lastCommitTime) {
				lastCommitTime = &commit.CommitWhen
			}

			repoCommits = append(repoCommits, commit)
		}

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
func updateRepos(ui *DeckardUI) {
	for _, conf := range ui.config.Projects {
		folder := repoFolder(ui.config, conf)
		_, err := os.Stat(folder)
		if os.IsNotExist(err) {
			updateStatus(ui, fmt.Sprintf("Cloning new repo: %s (this may take a while)", conf.Repo))
			err := cloneRepo(conf, folder)
			if err != nil {
				panic(err) //TODO show error in ui instead
			}
		} else {
			updateStatus(ui, fmt.Sprintf("Pulling repo: %s", conf.Repo))
			err := pullRepo(folder)
			if err != nil {
				panic(err) //TODO show error in ui instead if this fails
			}
		}
	}
	clearStatus(ui)
}

func repoFolder(conf *Config, prjConf ConfigProject) string {
	return path.Join(conf.CodeFolder, path.Base(prjConf.Repo))
}

func cloneRepo(prj ConfigProject, targetFolder string) error {
	cmd := exec.Command("git", "clone", prj.Repo, targetFolder)
	return cmd.Run()
}

func pullRepo(targetFolder string) error {
	cmd := exec.Command("git", "pull")
	cmd.Dir = targetFolder
	return cmd.Run()
}

func logRepo(targetFolder string, since *time.Time) ([]*Commit, error) {
	sinceArg := fmt.Sprintf("--since=%s", since.Format(time.RFC3339))
	cmd := exec.Command("git", "log", "--all", sinceArg, "--format=%H%x00%an%x00%cn%x00%ct%x00%s%x00%b%x00")
	cmd.Dir = targetFolder
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	split := strings.Split(string(out), "\x00")

	commits := make([]*Commit, 0)
	for i := 0; i < len(split); i += 6 {

		if i+6 > len(split) {
			break
		}

		cwUnix, err := strconv.Atoi(split[i+3])
		if err != nil {
			return nil, fmt.Errorf("illegal commit time: %s, folder = %s", split[i+3], targetFolder)
		}

		commits = append(commits, &Commit{
			Hash:          strings.TrimSpace(split[i]),
			AuthorName:    split[i+1],
			CommitterName: split[i+2],
			CommitWhen:    time.Unix(int64(cwUnix), 0),
			Subject:       split[i+4],
			Message:       split[i+5],
		})
	}
	return commits, nil
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
