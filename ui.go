package deckard

import (
	"database/sql"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/pkg/browser"
	"github.com/rivo/tview"
)

type DeckardUI struct {
	app      *tview.Application
	projects *tview.TextView
	status   *tview.TextView
	commits  *tview.Table

	config *Config
	db     *sql.DB

	state *uiState
}

type uiState struct {
	selectedProject int
	status          string
	commits         []*Commit
}

type Commit struct {
	Project       string
	Hash          string
	Subject       string
	Message       string
	AuthorName    string
	CommitterName string
	CommitWhen    time.Time
	State         string
	Comment       *string
	SlatScore     int // score between 0 and 100
}

func newDeckardUi(app *tview.Application, state *uiState, config *Config, db *sql.DB) *DeckardUI {
	return &DeckardUI{app: app, state: state, config: config, db: db}
}

func (ui *DeckardUI) Run() error {
	return ui.app.Run()
}

func (ui *DeckardUI) SelectProject(id int) {
	ui.state.selectedProject = id
	updateProjectText(ui.projects, ui.state, ui.config)
	updateCommitTable(ui)
}

// TODO Render spinner (optional parameter)
func (ui *DeckardUI) UpdateStatus(text string) {
	ui.state.status = text
	updateStatusText(ui.status, ui.state)
}

func (ui *DeckardUI) ClearStatus() {
	ui.state.status = ""
	updateStatusText(ui.status, ui.state)
}

func (ui *DeckardUI) MarkAsReviewed(commit *Commit) {
	err := UpdateCommitState(ui.db, commit.Project, commit.Hash, STATE_REVIEWED)
	if err != nil {
		fmt.Printf("ERR: %#v", err) //TODO proper error handling in UI
	}

	filtered := make([]*Commit, 0, len(ui.state.commits))
	for _, stateCommit := range ui.state.commits {
		if commit.Hash != stateCommit.Hash {
			filtered = append(filtered, stateCommit)
		}
	}
	ui.state.commits = filtered

	updateCommitTable(ui)
}

func (ui *DeckardUI) AddCommits(commits []*Commit) {
	//TODO make ui.state.commits a hashtable to prevent this O(n^2) check
ADD_COMMIT:
	for _, newCommit := range commits {
		for _, commit := range ui.state.commits {
			if commit.Hash == newCommit.Hash {
				continue ADD_COMMIT
			}
		}
		ui.state.commits = append(ui.state.commits, newCommit)
	}

	sort.Slice(ui.state.commits, func(i, j int) bool {
		return ui.state.commits[i].CommitWhen.Before(ui.state.commits[j].CommitWhen)
	})

	updateCommitTable(ui)
}

func (ui *DeckardUI) Quit() {
	ui.app.Stop()
}

func BuildUI(config *Config, db *sql.DB) (*DeckardUI, error) {

	initialState := &uiState{}

	header := tview.NewFlex().SetDirection(tview.FlexColumn)
	header.SetTitle("Deckard").SetTitleAlign(tview.AlignLeft).SetBorder(true)

	projects := buildProjects(initialState, config)
	status := buildStatus(initialState)
	commits := buildCommits(initialState)

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header.
			AddItem(projects, 0, 50, false).
			AddItem(status, 0, 50, false),
			3, 100, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(commits, 0, 70, true),
			//	AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			//		AddItem(tview.NewBox().SetBorder(true).SetTitle("Commit Details"), 0, 66, false).
			//		AddItem(tview.NewBox().SetBorder(true).SetTitle("Project Metric"), 0, 34, false),
			//		0, 30, false),
			0, 100, false)

	app := tview.NewApplication().SetRoot(flex, true).SetFocus(commits)
	ui := newDeckardUi(app, initialState, config, db)
	ui.projects = projects
	ui.status = status
	ui.commits = commits

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		return handleInput(ui, config, event)
	})

	return ui, nil
}

func handleInput(ui *DeckardUI, config *Config, event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyRune {
		if event.Rune() >= '0' && event.Rune() <= '9' { // select a project
			id, err := strconv.Atoi(string(event.Rune()))
			if err != nil {
				panic(err) //should not happen
			}
			if id > len(config.Projects) {
				return event
			}
			ui.SelectProject(id)
			return event
		}
		if event.Rune() == 'r' { // mark as reviewed
			ui.MarkAsReviewed(selectedCommit(ui))
		}
		if event.Rune() == 'q' { // mark as reviewed
			ui.Quit()
		}
		if event.Rune() == 'o' { // open commit in browser
			err := openCommit(ui, selectedCommit(ui))
			if err != nil {
				panic(err) //TODO proper ui dialog or status line
			}
		}
	}
	return event
}

func sanitizeRepoURL(raw string) (string, error) {
	if strings.HasPrefix(raw, "git") {
		split := strings.Split(raw, ":")
		prefix, path := split[0], split[1]

		if prefix == "git@ssh.dev.azure.com" {
			return sanitizeAzurePath(path)
		}

		return "", fmt.Errorf("unable to handle '%s'", prefix)
	}
	if strings.HasPrefix(raw, "http") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return "", err
		}
		parsed.User = nil
		return parsed.String(), nil
	}

	return "", fmt.Errorf("unable to parse '%s'", raw)
}

func sanitizeAzurePath(rawPath string) (string, error) {
	pathParts := strings.Split(rawPath, "/")
	version, org, project, repo := pathParts[0], pathParts[1], pathParts[2], pathParts[3]

	if version != "v3" {
		return "", fmt.Errorf("can only handle v3 urls but is '%s'", version)
	}

	return fmt.Sprintf("https://dev.azure.com/%s/%s/_git/%s", org, project, repo), nil
}

func openCommit(ui *DeckardUI, commit *Commit) error {
	config, found := ui.config.Projects[commit.Project]
	if !found {
		return fmt.Errorf("no config found for project %s, cannot open in browser", commit.Project)
	}

	repo, err := sanitizeRepoURL(config.Repo)
	if err != nil {
		return err
	}

	var url string
	if strings.Contains(repo, "dev.azure.com") {
		url = path.Join(repo, "commit", commit.Hash) + "?refName=refs%2Fheads%2Fmain"
	} else {
		url = path.Join(repo, "commit", commit.Hash)
	}

	browser.Stderr = nil
	browser.Stdout = nil
	return browser.OpenURL(url)
}

func selectedCommit(ui *DeckardUI) *Commit {
	row, _ := ui.commits.GetSelection()
	commit := ui.state.commits[row]
	return commit
}

// ## project view

func buildProjects(state *uiState, config *Config) *tview.TextView {
	text := tview.NewTextView()
	text.SetRegions(true)
	updateProjectText(text, state, config)
	return text
}

type project struct {
	name string
	icon string
}

const SelectionMarker = "sel"

func updateProjectText(text *tview.TextView, state *uiState, config *Config) {

	prjText := ""
	num := 0
	for _, prj := range getProjects(config) {

		sel := ""
		if state.selectedProject == num {
			sel = SelectionMarker
		}

		prjText += fmt.Sprintf(` | ["%s"](%d) %s %s[""]`, sel, num, prj.name, prj.icon)
		num++
	}

	text.SetText(prjText)
	text.Highlight(SelectionMarker)
}

func getProjects(config *Config) []project {
	projects := make([]project, 0)
	projects = append(projects, project{
		name: "all",
		icon: "",
	})
	for id, data := range config.Projects {
		projects = append(projects, project{name: id, icon: data.Icon})
	}
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].name < projects[j].name
	})
	return projects
}

// ## status view

func buildStatus(state *uiState) *tview.TextView {
	text := tview.NewTextView()
	text.SetDynamicColors(true)
	text.SetRegions(true)
	updateStatusText(text, state)
	return text
}

func updateStatusText(text *tview.TextView, state *uiState) {
	text.SetText(fmt.Sprintf("[yellow]%s[-]", state.status))
}

// ## commit table

func buildCommits(state *uiState) *tview.Table {
	table := tview.NewTable()
	table.SetBorder(true)
	table.SetSelectable(true, false)
	return table
}

func updateCommitTable(ui *DeckardUI) {
	table := ui.commits

	selectedPrjName := ""
	if ui.state.selectedProject != 0 {
		projects := getProjects(ui.config)
		selectedPrjName = projects[ui.state.selectedProject].name
	}

	table.Clear()
	tablePos := 0
	for _, commit := range ui.state.commits {
		if selectedPrjName == "" || selectedPrjName == commit.Project {
			colour := slatColour(commit.SlatScore)
			setCell(table, tablePos, 0, lookupProjectIcon(ui, commit.Project), colour)
			setCell(table, tablePos, 1, strconv.FormatInt(int64(commit.SlatScore), 10), colour)
			setCell(table, tablePos, 2, commit.CommitWhen.Format("02.01 15:04"), colour)
			setCell(table, tablePos, 3, commit.Hash[0:6], colour)
			setCell(table, tablePos, 4, commit.AuthorName, colour)
			setCell(table, tablePos, 5, commit.Subject, colour)
			tablePos++
		}
	}
}

func slatColour(score int) tcell.Color {
	// TODO Use HSV colour model for a nicer gradient between R and G
	return tcell.NewRGBColor(int32((255*score)/100), int32((255*(100-score))/100), 0)
}

func setCell(table *tview.Table, row, column int, text string, colour tcell.Color) {
	cell := tview.NewTableCell(text)
	cell.SetTextColor(colour)
	table.SetCell(row, column, cell)
}

func lookupProjectIcon(ui *DeckardUI, project string) string {
	for prj, conf := range ui.config.Projects {
		if prj == project {
			return conf.Icon
		}
	}
	return ""
}
