package deckard

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
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
	Project    string
	Hash       string
	Message    string
	Author     string
	AuthorWhen time.Time
	State      string
	Comment    *string
}

func (c *Commit) Headline() string {
	lines := strings.Split(c.Message, "\n")
	if len(lines) > 0 {
		return lines[0]
	}
	return ""
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
		return ui.state.commits[i].AuthorWhen.Before(ui.state.commits[j].AuthorWhen)
	})

	updateCommitTable(ui)
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
			AddItem(commits, 0, 70, true).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(tview.NewBox().SetBorder(true).SetTitle("Commit Details"), 0, 66, false).
				AddItem(tview.NewBox().SetBorder(true).SetTitle("Project Metric"), 0, 34, false),
				0, 30, false),
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
		if event.Rune() >= '0' && event.Rune() <= '9' {
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
	}
	return event
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
			table.SetCellSimple(tablePos, 0, lookupProjectIcon(ui, commit.Project))
			table.SetCellSimple(tablePos, 1, commit.AuthorWhen.Format("02.01 15:04"))
			table.SetCellSimple(tablePos, 2, commit.Hash[0:6])
			table.SetCellSimple(tablePos, 3, commit.Author)
			table.SetCellSimple(tablePos, 4, commit.Headline())
			tablePos++
		}
	}
}

func lookupProjectIcon(ui *DeckardUI, project string) string {
	for prj, conf := range ui.config.Projects {
		if prj == project {
			return conf.Icon
		}
	}
	return ""
}
